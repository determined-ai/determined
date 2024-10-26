import json
import logging
import os
from typing import Any, Dict, List, Optional, Tuple

import transformers
from transformers import trainer_utils

import determined as det

logger = logging.getLogger("det.transformers")


class DetCallback(transformers.TrainerCallback):  # type: ignore
    """
    ``DetCallback`` integrates a training loop built around ``transformers.Trainer`` with the
    Determined cluster.  It reports metrics, uploads checkpoints, and handles preemption signals.
    It also automatically restores training from the latest checkpoint after pauses or crashes.

    Simply include ``DetCallback`` as in the list of ``callbacks`` that you pass to your
    ``Trainer``.

    Args:
        core_context: the result of a ``det.core.init()`` call.
        args: ``TrainingArgs`` from a ``transformers.HfArgumentParser``, the same ``args`` to be
            passed to the ``Trainer``.
        filter_metrics: a list of metric names to report to Determined.  Default: ``None`` (all
            metrics are reported).
        user_data: an optional dict of metadata to be stored in every checkpoint.
            Default: ``None``.
    """

    def __init__(
        self,
        core_context: det.core.Context,
        args: transformers.TrainingArguments,
        filter_metrics: Optional[List[str]] = None,
        user_data: Optional[Dict[str, Any]] = None,
    ) -> None:
        super().__init__()
        self.core_context = core_context
        self.filter_metrics = filter_metrics
        self.user_data = user_data

        self.last_train_metrics = -1
        self.last_eval_metrics = -1
        self.last_save = -1
        self.last_progress = 0

        info = det.get_cluster_info()
        if not info:
            raise RuntimeError("det.transformers.DetCallback must be run on a Determined cluster")
        self.info = info

        self.load_last_checkpoint(args)

        self.searcher_metric = None
        self.time_metric = None
        if self.info.task_type == "TRIAL":
            searcher_config = self.info.trial._config["searcher"]
            self._check_searcher_config(searcher_config, args)
            self.searcher_metric = searcher_config["metric"]
            self.time_metric = searcher_config.get("time_metric")
            # Don't allow filtering of the searcher or time_metric metrics.
            if self.filter_metrics:
                self.filter_metrics.append(self.searcher_metric)
                if self.time_metric:
                    self.filter_metrics.append(self.time_metric)

        # Undocumented workarounds in case forcing the checkpoint and validations at the end of
        # non-preempted training is a bad idea somehow.
        self._force_final_save = True
        self._force_final_evaluate = True

    def load_last_checkpoint(self, args: transformers.TrainingArguments) -> None:
        latest_checkpoint = self.info.latest_checkpoint
        if latest_checkpoint is None:
            return
        if args.overwrite_output_dir is True:
            logger.info(
                "Skipping downloading last checkpoint from Determined due "
                "to overwrite_output_dir=True."
            )
            return

        # To resume DeepSpeed, each node requires ALL sharded model/optimizer states,
        # so we can skip using selector and just download all files.
        self.core_context.checkpoint.download(latest_checkpoint, args.output_dir)

        checkpoint_path = trainer_utils.get_last_checkpoint(args.output_dir)
        args.resume_from_checkpoint = checkpoint_path

        logger.info(f"Latest checkpoint downloaded to {checkpoint_path}.")

    def _check_searcher_config(
        self, cfg: Dict[str, Any], args: transformers.TrainingArguments
    ) -> None:
        if args.max_steps > -1:
            args_unit = "batches"
            args_len = args.max_steps
            len_arg = "--max_steps"
        else:
            args_unit = "epochs"
            args_len = args.num_train_epochs
            len_arg = "--num_train_epochs"

        if isinstance(cfg.get("max_length"), int):
            # Legacy searcher config (unitless).  Has never been supported, actually.
            raise ValueError(
                "HF trainer no longer respects the deprecated searcher.max_length "
                "field.  searcher.max_length is deprecated; please remove it and rely "
                f"on {len_arg} instead to avoid ambiguous training specifications."
            )
        elif isinstance(cfg.get("max_length"), dict):
            # Legacy searcher config; max_length must match provided args.
            search_unit, search_len = next(iter(cfg["max_length"].items()))
            if (search_unit, search_len) != (args_unit, args_len):
                raise ValueError(
                    "HF trainer units does not match configured searcher.max_length "
                    f"({args_unit}={args_len} != {search_unit}={search_len}).  The "
                    "searcher.max_length field is deprecated; please remove it and avoid "
                    "ambiguous training specifications."
                )
        elif cfg["name"] in ["adaptive_asha", "async_halving"]:
            # ASHA search: check time_metric and max_time are sane.
            self.required_metrics.append(cfg["time_metric"])
            search_unit = cfg["time_metric"]
            search_len = cfg["max_time"]
            if search_unit not in ("batches", "epochs"):
                self.required_metrics.append(search_unit)
            elif (search_unit, search_len) != (args_unit, args_len):
                name = cfg["name"]
                raise ValueError(
                    "HF trainer units does not match configured the max_time configured for "
                    f"{name} searcher ({args_unit}={args_len} != {search_unit}={search_len}.  "
                    f"Please update one of the searcher.max_time config field or the {len_arg} "
                    "to match the other."
                )

    def _check_eval_metrics(self, metrics: Dict[str, Any]) -> None:
        search_ok = self.searcher_metric is None or self.searcher_metric in metrics
        time_ok = self.time_metric is None or self.time_metric in metrics
        if not search_ok and not time_ok:
            raise ValueError(
                f"Searcher metric '{self.searcher_metric}' set by searcher.metric config field "
                f"and time metric '{self.time_metric}' from searcher.time_metric config field are "
                "both missing; you must emit those metrics for the hyperparameter search to work."
            )
        if not search_ok:
            raise ValueError(
                f"Searcher metric '{self.searcher_metric}' set by searcher.metric config field "
                "is missing; you must emit that metric for features like hyperparameter search, "
                "checkpoint garbage collection, and selecting the best checkpoint to work."
            )
        if not time_ok:
            raise ValueError(
                f"Time metric '{self.time_metric}' set by searcher.time_metric config field is "
                "missing; you must emit that metric for the hyperparameter search to work."
            )

    def on_log(
        self,
        args: transformers.TrainingArguments,
        state: transformers.TrainerState,
        control: transformers.TrainerControl,
        logs: Optional[Dict[str, Any]] = None,
        **kwargs: Any,
    ) -> None:
        if logs is None:
            logger.warning("on_log called with empty logs")
            return
        metrics, metric_type = self._get_metrics(logs)
        logger.debug(f"on_log metrics, global_step {state.global_step}", metrics)
        metrics["batches"] = metrics.get("batches", state.global_step)
        metrics["epochs"] = metrics.get("epochs", state.epoch)
        if metric_type == TRAIN:
            # Prevents reporting metrics for the same step twice. This happens after
            # training is completed and average training metrics are reported with
            # the same step as the in-progress training metrics.
            if self.last_train_metrics != state.global_step:
                self.last_train_metrics = state.global_step
                if state.is_world_process_zero:
                    # Note: state.global_step represents steps_completed, not step index
                    self.core_context.train.report_metrics(
                        group="training", steps_completed=state.global_step, metrics=metrics
                    )

        elif metric_type == EVAL:
            # Prevents reporting metrics for the same step twice. This happens when
            # after-training evaluation is completed, and it is reported with the same
            # step as the last during-training evaluation.
            if self.last_eval_metrics != state.global_step:
                self.last_eval_metrics = state.global_step
                if state.is_world_process_zero:
                    self._check_eval_metrics(metrics)
                    # Note: state.global_step represents steps_completed, not step index
                    self.core_context.train.report_metrics(
                        group="validation", steps_completed=state.global_step, metrics=metrics
                    )
        else:
            logger.warning(f"Metrics not reported: metric type = {metric_type}.")

        # If we've been preempted, save a checkpoint and shut down training.
        if self.core_context.preempt.should_preempt():
            control.should_training_stop = True
            # Don't set control.should_save now, or it can trigger multiple saves, if we trigger
            # in a training on_log and arrive here again in an evaluate on_log.  We would not cause
            # that to happen, but other callbacks could, such as if it were just naturally time for
            # an evaluation.  So just let the save-at-end logic handle it.

    def _get_metrics(self, logs: Dict[str, Any]) -> Tuple[Dict[str, Any], str]:
        metric_type = get_metric_type(logs)
        if not self.filter_metrics:
            metrics = logs
        else:
            metrics = {k: v for k, v in logs.items() if any(m in k for m in self.filter_metrics)}
        # Remove the default rounded 'epoch' metric.
        metrics.pop("epoch", None)
        # Also remove speed metrics.
        speed_suffixes = ["_runtime", "_per_second", "_compilation_time"]
        speed_metrics = [m for m in metrics if any(m.endswith(s) for s in speed_suffixes)]
        for m in speed_metrics:
            metrics.pop(m, None)
        return metrics, metric_type

    def on_save(
        self,
        args: transformers.TrainingArguments,
        state: transformers.TrainerState,
        control: transformers.TrainerControl,
        **kwargs: Any,
    ) -> None:
        self.last_save = state.global_step
        # local_path is where HF Trainer saves model and tokenizer in a given step.
        local_path = os.path.join(args.output_dir, f"checkpoint-{state.global_step}")
        if state.is_world_process_zero:
            if self.user_data is not None:
                self._on_save_user_data(local_path)

        metadata = {
            "steps_completed": state.global_step,
        }
        if self.info.task_type == "TRIAL":
            metadata["trial_id"] = self.info.trial.trial_id

        def selector(x: str) -> bool:
            return x.startswith((f"checkpoint-{state.global_step}/", "runs/"))

        self.core_context.checkpoint.upload(
            args.output_dir, metadata=metadata, shard=True, selector=selector
        )

    def _on_save_user_data(self, save_path: str) -> None:
        """
        User-defined saving of objects from self.checkpoint_metadata under save_path.
        After objects are saved, Determined handles uploading and downloading objects
        to/from selected storage.
        """
        with open(os.path.join(save_path, "my_data.json"), "w") as f:
            json.dump(self.user_data, f)

    def on_step_end(
        self,
        args: transformers.TrainingArguments,
        state: transformers.TrainerState,
        control: transformers.TrainerControl,
        **kwargs: Any,
    ) -> None:
        if state.is_world_process_zero and args.max_steps > -1:
            # There needs to be at least 1% increase in progress to report progress (maximum 100
            # report_progress API calls in per trial).
            progress = state.global_step / args.max_steps
            percent = int(progress * 100)
            if percent > self.last_progress:
                self.last_progress = percent
                self.core_context.train.report_progress(progress)

    def on_epoch_end(
        self,
        args: transformers.TrainingArguments,
        state: transformers.TrainerState,
        control: transformers.TrainerControl,
        **kwargs: Any,
    ) -> None:
        # Decide if we're about to shut down training.
        is_end = False
        if control.should_training_stop:
            is_end = True
        elif args.max_steps > -1:
            is_end = state.global_step >= args.max_steps
        else:
            is_end = state.epoch >= args.num_train_epochs

        # If training is ending, this is our last chance to ask for a eval and/or save.
        if is_end:
            # Avoid stale evaluate-at-end.
            if state.global_step > self.last_eval_metrics:
                # Also avoid evaluate-at-end if we have been preempted.
                if self._force_final_evaluate and not self.core_context.preempt.should_preempt():
                    control.should_evaluate = True
            # Avoid stale save-at-end.
            if state.global_step > self.last_save:
                # You can't disable save-after-preemption.
                if self._force_final_save or self.core_context.preempt.should_preempt():
                    control.should_save = True

        if state.is_world_process_zero and args.max_steps == -1:
            self.core_context.train.report_progress(state.epoch / args.num_train_epochs)


EVAL = "eval_"
TEST = "test_"
TRAIN = "train_"


def get_metric_type(d: Dict[str, Any]) -> str:
    if any(k.startswith(EVAL) for k in d):
        return EVAL
    if any(k.startswith(TEST) for k in d):
        return TEST
    return TRAIN


def get_ds_config_path_from_args(args: List[str]) -> Optional[str]:
    for idx in range(len(args)):
        if args[idx] == "--deepspeed":
            ds_config_idx = idx + 1
            ds_config_path = args[ds_config_idx]
            return ds_config_path
    return None
