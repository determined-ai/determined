import pathlib
import typing

from transformers import (
    TrainerCallback,
    TrainerState,
    TrainerControl,
    TrainingArguments,
)

import os
import determined as det
import logging
import torch
import uuid
import weakref
import importlib.util
import shutil


class DetCallback(TrainerCallback):
    def __init__(
        self,
        args: TrainingArguments,
        filter_metrics: typing.List[str] = None,
        tokenizer: typing.Any = None,
        tokenizer_options: typing.Dict = None,
        checkpoint_metadata: typing.Dict = None,
    ) -> None:
        super().__init__()

        assert (
            is_determined_available()
        ), "DetCallback requires determined to be installed. Run `pip install determined`."
        import determined

        self._det = determined
        if args.deepspeed:
            distributed = self._det.core.DistributedContext.from_deepspeed()
        else:
            distributed = self._det.core.DistributedContext.from_torch_distributed()
        self.core_context = self._det.core.init(distributed=distributed)
        self.core_context.__enter__()
        weakref.finalize(self, exit_context, self.core_context)

        self.filter_metrics = filter_metrics
        self.user_checkpoint_metadata = checkpoint_metadata
        self.tokenizer = tokenizer
        self.tokenizer_options = tokenizer_options
        self.load_last_checkpoint(args)

        self.last_metrics: typing.Dict[str, float] = {}

        searcher_config = self._det.get_cluster_info().trial._config["searcher"]
        self.searcher_metric = searcher_config["metric"]
        self.searcher_unit = list(searcher_config["max_length"].keys())[0]
        self.searcher_max_length = list(searcher_config["max_length"].values())[0]
        self.searcher_ops = self.core_context.searcher.operations()
        self.current_op = next(self.searcher_ops)

        self._check_searcher_compatibility(args)

    def on_log(
        self,
        args: TrainingArguments,
        state: TrainerState,
        control: TrainerControl,
        logs=None,
        **kwargs,
    ):
        if state.is_world_process_zero:
            metrics, metric_type = self._get_metrics(logs)
            if metric_type == TRAIN:
                if (
                    "train_step" not in self.last_metrics
                    or self.last_metrics["eval_step"] != state.global_step
                ):
                    self.core_context.train.report_training_metrics(
                        steps_completed=state.global_step, metrics=metrics
                    )
                    metrics["train_step"] = state.global_step

            elif metric_type == EVAL:
                if (
                    "eval_step" not in self.last_metrics
                    or self.last_metrics["eval_step"] != state.global_step
                ):
                    self.core_context.train.report_validation_metrics(
                        steps_completed=state.global_step, metrics=metrics
                    )
                    metrics["eval_step"] = state.global_step
                    self.last_metrics.update(metrics)
            else:
                logging.warning(f"Metrics not reported: metric type = {metric_type}.")

            self.last_metrics.update(metrics)

    def _get_metrics(self, logs: typing.Dict) -> typing.Tuple[typing.Dict, str]:
        metrics = logs
        metric_type = get_metric_type(logs)
        if self.filter_metrics:
            metrics = {}
            for k, v in logs.items():
                if any(m in k for m in self.filter_metrics) is True:
                    metrics[k] = v

        return metrics, metric_type

    def on_save(
        self,
        args: TrainingArguments,
        state: TrainerState,
        control: TrainerControl,
        **kwargs,
    ):
        info = self._det.get_cluster_info()
        if info is None:
            # TODO: modify to support local mode
            logging.warning("ClusterInfo is None: not running in a task. Skip saving.")
            return

        local_path = os.path.join(args.output_dir, f"checkpoint-{state.global_step}")

        storage_manager = self.core_context.checkpoint._storage_manager

        if state.is_world_process_zero:

            det_checkpoint_metadata = {
                "steps_completed": state.global_step,
                "trial_id": info.trial.trial_id,
            }
            if self.tokenizer_options is not None:
                det_checkpoint_metadata["tokenizer_options"] = self.tokenizer_options

            storage_id = str(uuid.uuid4())
            with storage_manager.store_path(storage_id) as path:
                self.core_context.distributed.broadcast((storage_id, path))
                self._save(path, local_path)

                storage = self._det.common.storage
                resources = storage.StorageManager._list_directory(path)
                if isinstance(storage_manager, storage.SharedFSStorageManager):
                    all_resources = [resources]
                else:
                    # Gather resources across nodes.
                    all_resources = self.core_context.distributed.gather(resources)

            resources = {k: v for d in all_resources for k, v in d.items()}

            self.core_context.checkpoint._report_checkpoint(
                storage_id, resources, det_checkpoint_metadata
            )

        else:
            storage_id, path = self.core_context.distributed.broadcast(None)
            self._save(path, local_path)

            storage = self._det.common.storage
            if not isinstance(storage_manager, storage.SharedFSStorageManager):
                # Gather resources across nodes.
                if self.core_context.distributed.local_rank == 0:
                    resources = storage.StorageManager._list_directory(path)
                else:
                    resources = {}

                _ = self.core_context.distributed.gather(resources)
            if self.core_context.distributed.local_rank == 0:
                storage_manager.post_store_path(str(path), storage_id)

        if self.core_context.preempt.should_preempt():
            raise Exception("Process preempted / killed")

    def _save(self, path: pathlib.Path, local_path: str) -> None:
        if self.core_context.distributed.local_rank == 0:
            path.mkdir(parents=True, exist_ok=True)

        _ = self.core_context.distributed.gather_local(None)  # sync

        if self.core_context.distributed.local_rank == 0:
            # only local_rank=0 should copy the content of the local path
            shutil.copytree(local_path, path, dirs_exist_ok=True)

        if self.core_context.distributed.rank == 0:
            if self.tokenizer is not None:
                self.tokenizer.save_pretrained(os.path.join(path, "tokenizer"))

            if self.user_checkpoint_metadata is not None:
                self._on_save_user_data(path)

    def _on_save_user_data(self, save_path) -> None:
        """
        User-defined saving of objects from self.checkpoint_metadata under save_path.
        After objects are saved, Determined handles uploading and downloading objects to/from selected storage.
        """
        raise NotImplementedError(
            "No implementation for _on_save_user_data. "
            "Objects passed to the callback via checkpoint_metadata will not be saved."
        )

    def load_last_checkpoint(self, args: TrainingArguments) -> None:
        info = self._det.get_cluster_info()

        if info is None:
            # TODO: modify to support local mode
            logging.warning("ClusterInfo is None: not running in a task. Skip loading.")
            return

        latest_checkpoint = info.latest_checkpoint
        if latest_checkpoint is not None:
            metadata = self.core_context.checkpoint.get_metadata(latest_checkpoint)
            prev_trial_id = metadata["trial_id"]
            trial_id = info.trial.trial_id
            if trial_id != prev_trial_id:
                resume_step = 0
            else:
                resume_step = metadata["steps_completed"]
            checkpoint_path = os.path.join(args.output_dir, f"checkpoint-{resume_step}")

            if self.core_context.distributed.local_rank == 0:
                self.core_context.checkpoint.download(
                    latest_checkpoint, checkpoint_path
                )
                torch.distributed.barrier()
            else:
                # wait until local rank 0 finishes downloading data
                torch.distributed.barrier()

            args.resume_from_checkpoint = checkpoint_path

    def on_step_end(
        self,
        args: TrainingArguments,
        state: TrainerState,
        control: TrainerControl,
        **kwargs,
    ):
        # state.epoch is not None only during training
        if state.epoch and self.searcher_unit == "batches":
            if state.is_world_process_zero:
                self.current_op.report_progress(state.global_step)

            if state.global_step >= self.current_op.length:
                self._update_searcher(state, control)

        if self.core_context.preempt.should_preempt():
            control.should_save = True

    def on_epoch_end(
        self,
        args: TrainingArguments,
        state: TrainerState,
        control: TrainerControl,
        **kwargs,
    ):
        # state.epoch is not None only during training
        if state.epoch and self.searcher_unit == "epochs":
            if state.is_world_process_zero:
                self.current_op.report_progress(state.epoch)

            if round(state.epoch) >= self.current_op.length:
                self._update_searcher(state, control)

    def _update_searcher(self, state: TrainerState, control: TrainerControl) -> None:
        if state.is_world_process_zero:
            if self.last_metrics is None:
                logging.warning(
                    f"No training or evaluation metrics has been recorded. Please check your settings for "
                    f"training metrics (--logging_strategy and --logging_steps) or "
                    f"evaluation metrics (--evaluation_strategy and --eval_steps). "
                    f"Reporting trainer_state.best_metric to the searcher."
                )
                self.current_op.report_completed(state.best_metric)
            elif self.searcher_metric not in self.last_metrics:
                logging.warning(
                    f"Searcher metric {self.searcher_metric} from the yaml config file does not match any "
                    f"of the recorded metrics in {self.last_metrics}. "
                    f"Reporting trainer_state.best_metric to the searcher."
                )
                self.current_op.report_completed(state.best_metric)
            else:
                self.current_op.report_completed(
                    self.last_metrics[self.searcher_metric]
                )

        try:
            self.current_op = next(self.searcher_ops)
        except StopIteration:
            control.should_training_stop = True

    def _check_searcher_compatibility(self, args: TrainingArguments) -> None:
        if hasattr(args, "num_train_epochs"):
            if (
                self.searcher_unit == "batches"
                or args.num_train_epochs != self.searcher_max_length
            ):
                self._log_config_mismatch("epochs", args.num_train_epochs)

        elif hasattr(args, "max_steps"):
            if (
                self.searcher_unit == "epochs"
                or args.max_steps != self.searcher_max_length
            ):
                self._log_config_mismatch("batches", args.max_steps)

    def _log_config_mismatch(
        self,
        trainer_units: str,
        trainer_len: int,
    ) -> None:
        logging.warning(
            f"Searcher configuration does not match HF Trainer configuration. "
            f"Searcher uses {self.searcher_unit}={self.searcher_max_length}, "
            f"while HF Trainer uses {trainer_units}={trainer_len}. "
            f"Continuing this run may cause Searcher not to behave correctly. "
            f"Make sure to match the units between HF Trainer and Searcher: "
            f"use (--num_train_epochs and searcher.max_length.epochs) OR "
            f"(--max_steps and searcher.max_length.batches)."
        )


def is_determined_available() -> bool:
    return importlib.util.find_spec("determined") is not None


def exit_context(context: det.core.Context) -> None:
    context.__exit__(None, None, None)


def set_hyperparameters(training_args: TrainingArguments):
    hparams = det.get_cluster_info().trial.hparams
    for k, v in hparams.items():
        if hasattr(training_args, k):
            setattr(training_args, k, v)


EVAL = "eval_"
TEST = "test_"
TRAIN_AVG = "train_"
TRAIN = "train_progress"


def get_metric_type(d):
    for k, v in d.items():
        if k.startswith(EVAL):
            return EVAL
        elif k.startswith(TEST):
            return TEST
        elif k.startswith(TRAIN_AVG):
            return TRAIN_AVG
        else:
            return TRAIN
