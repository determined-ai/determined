from argparse import Namespace
from dataclasses import dataclass
import glob
import logging
import os
from pathlib import Path
import tempfile
from typing import Any, Callable, cast, Dict, Iterator, List, Optional, Tuple, Type, Union


import determined as det
from determined._info import ClusterInfo
from determined.core._searcher import SearcherOperation
import pytorch_lightning as pl
from pytorch_lightning.utilities.deepspeed import convert_zero_checkpoint_to_fp32_state_dict
from pytorch_lightning.utilities.distributed import rank_zero_only

CHECKPOINT_DOWNLOAD_PATH = "determined_checkpoint_download"
TEMP_CHECKPOINT_FILE = "determined.ckpt"


def flatten(xs: List[List]) -> List:
    return [item for items in xs for item in items]


def get_cluster_info_with_assert() -> ClusterInfo:
    """
    Raise an exception if not run on a Determined cluster.  Returns ClusterInfo.
    """
    info = det.get_cluster_info()
    assert info, "This code can only be run on-cluster."
    return info


def download_checkpoint(core_context: det.core.Context, module_load_only: bool) -> Optional[str]:
    info = det.get_cluster_info()
    if info:
        ckpt_id = info.latest_checkpoint
        if ckpt_id:
            core_context.checkpoint.download(ckpt_id, CHECKPOINT_DOWNLOAD_PATH)
    if os.path.isdir(CHECKPOINT_DOWNLOAD_PATH):
        if "latest" in os.listdir(CHECKPOINT_DOWNLOAD_PATH):
            if module_load_only:
                # DeepSpeed checkpoint; convert to a .ckpt file.
                convert_zero_checkpoint_to_fp32_state_dict(
                    CHECKPOINT_DOWNLOAD_PATH, TEMP_CHECKPOINT_FILE
                )
                return TEMP_CHECKPOINT_FILE
            else:
                return CHECKPOINT_DOWNLOAD_PATH
        else:
            ckpt_files = glob.glob(os.path.join(CHECKPOINT_DOWNLOAD_PATH, "*.ckpt"))
            assert len(ckpt_files) == 1, "Checkpoint must contain exactly one .ckpt file."
            return ckpt_files[0]
    return None


def get_checkpoint_metadata(core_context: det.core.Context) -> Optional[Dict]:
    info = det.get_cluster_info()
    if info:
        ckpt_id = info.latest_checkpoint
        if ckpt_id:
            return core_context.checkpoint.get_metadata(ckpt_id)
    return None


def get_searcher_metric() -> str:
    return cast(str, get_cluster_info_with_assert().trial._config["searcher"]["metric"])


def get_searcher_max_length() -> int:
    max_length_entry = get_cluster_info_with_assert().trial._config["searcher"]["max_length"]
    if isinstance(max_length_entry, dict):
        assert tuple(max_length_entry.keys()) == (
            "epochs",
        ), "Must express max training length in epochs."
        return cast(int, max_length_entry["epochs"])
    else:
        return cast(int, max_length_entry)


@dataclass
class DeterminedIntegrationSharedState:
    """
    State shared between the components of the Determined integration on a single Trainer.
    """

    core_context: det.core.Context
    searcher_ops: Iterator[SearcherOperation]
    current_op: SearcherOperation
    global_step: int = 0
    last_metric: Optional[float] = None


class DeterminedLogger(pl.loggers.logger.Logger):  # type: ignore
    def __init__(self, shared: DeterminedIntegrationSharedState) -> None:
        self.shared = shared

    def log_hyperparams(
        self, params: Union[Dict[str, Any], Namespace], *args: Any, **kwargs: Any
    ) -> None:
        pass

    @rank_zero_only  # type: ignore
    def log_metrics(self, metrics: Dict, step: int) -> None:
        searcher_metric = get_searcher_metric()
        if searcher_metric in metrics:
            self.shared.last_metric = metrics[searcher_metric]

    @property
    def name(self) -> Optional[str]:
        pass

    @property
    def version(self) -> Optional[Union[int, str]]:
        pass


def upload_determined_checkpoint(
    path: Union[str, Path], shared: DeterminedIntegrationSharedState
) -> None:
    if shared.core_context.distributed.rank == 0:
        det_checkpoint_metadata = {
            "steps_completed": shared.global_step,
            "trial_id": get_cluster_info_with_assert().trial.trial_id,
        }
        if os.path.isfile(path):
            # Create a temporary directory with a symbolic link to the saved file,
            # so we can upload it without making a copy.
            # If path is a directory terminated with /, basename will return empty string --
            # we use normpath to ensure it returns the last directory.
            ckpt_name = os.path.basename(os.path.normpath(path))
            with tempfile.TemporaryDirectory() as temp_dir:
                temp_ckpt_path = os.path.join(temp_dir, ckpt_name)
                os.symlink(os.path.abspath(path), os.path.abspath(temp_ckpt_path))
                shared.core_context.checkpoint.upload(temp_dir, det_checkpoint_metadata)
        else:
            shared.core_context.checkpoint.upload(path, det_checkpoint_metadata)


class DeterminedCheckpointIO(pl.plugins.io.CheckpointIO):  # type: ignore
    def __init__(
        self,
        shared: DeterminedIntegrationSharedState,
        base_ckpt_io: Optional[pl.plugins.io.CheckpointIO],
    ) -> None:
        self.shared = shared
        if base_ckpt_io:
            self.base_ckpt_io = base_ckpt_io
        else:
            self.base_ckpt_io = pl.plugins.io.TorchCheckpointIO()

    def save_checkpoint(
        self,
        checkpoint: Dict[str, Any],
        path: Union[str, Path],
        storage_options: Optional[Any] = None,
    ) -> None:
        self.base_ckpt_io.save_checkpoint(checkpoint, path, storage_options)
        upload_determined_checkpoint(path, self.shared)

    def load_checkpoint(
        self,
        path: Union[str, Path],
        map_location: Optional[Callable] = lambda storage, loc: storage,
    ) -> Dict[str, Any]:
        return cast(Dict[str, Any], self.base_ckpt_io.load_checkpoint(path, map_location))

    def remove_checkpoint(self, path: Union[str, Path]) -> None:
        self.base_ckpt_io.remove_checkpoint(path)


class DeterminedCallback(pl.callbacks.Callback):  # type: ignore
    def __init__(self, shared: DeterminedIntegrationSharedState) -> None:
        self.shared = shared
        self.core_context = shared.core_context
        self.val_epoch_outputs: List[pl.utilities.types.STEP_OUTPUT] = []
        self.test_epoch_outputs: List[pl.utilities.types.STEP_OUTPUT] = []

    def setup(self, trainer: pl.Trainer, pl_module: pl.LightningModule, stage: str) -> None:
        # If fitting/testing multiple times, keep a monotonically increasing global step for
        # for reporting Determined metrics and checkpoints.
        self.shared.global_step += 1
        self.initial_global_step = self.shared.global_step

    def on_train_batch_end(
        self,
        trainer: pl.Trainer,
        pl_module: pl.LightningModule,
        outputs: pl.utilities.types.STEP_OUTPUT,
        batch: Any,
        batch_idx: int,
    ) -> None:
        self.shared.global_step = self.initial_global_step + trainer.global_step
        if self.core_context.distributed.rank == 0:
            outputs = {k: v.item() for k, v in outputs.items()}
            # We only report training metrics from rank 0 to avoid too many blocking syncs.
            self.core_context.train.report_training_metrics(
                steps_completed=self.shared.global_step, metrics=outputs
            )

    # def on_train_epoch_end(self, trainer: pl.Trainer, pl_module: pl.LightningModule) -> None:
    #     if self.core_context.distributed.rank == 0:
    #         self.shared.current_op.report_progress(trainer.current_epoch + 1)
    #     if (trainer.current_epoch + 1) >= self.shared.current_op.length:
    #         if self.core_context.distributed.rank == 0:
    #             if self.shared.last_metric is None:
    #                 logging.warning(
    #                     f"Searcher metric {get_searcher_metric()} was not "
    #                     "logged during training.  Reporting as 0.",
    #                 )
    #                 self.shared.current_op.report_completed(0)
    #             else:
    #                 self.shared.current_op.report_completed(self.shared.last_metric)
    #         try:
    #             self.shared.current_op = next(self.shared.searcher_ops)
    #         except StopIteration:
    #             logging.info("Reached end of searcher operations.")
    #             trainer.should_stop = True
    #     if self.core_context.preempt.should_preempt():
    #         trainer.save_checkpoint(TEMP_CHECKPOINT_FILE)
    #         raise Exception("Training pre-empted.")

    def on_validation_epoch_start(self, trainer: pl.Trainer, pl_module: pl.LightningModule) -> None:
        self.val_epoch_outputs = []

    def on_validation_batch_end(
        self,
        trainer: pl.Trainer,
        pl_module: pl.LightningModule,
        outputs: Optional[pl.utilities.types.STEP_OUTPUT],
        batch: Any,
        batch_idx: int,
        dataloader_idx: int,
    ) -> None:
        if outputs:
            self.val_epoch_outputs.append({k: v.item() for k, v in outputs.items()})

    def on_validation_epoch_end(self, trainer: pl.Trainer, pl_module: pl.LightningModule) -> None:
        outputs = self.core_context.distributed.gather(self.val_epoch_outputs)
        self.val_epoch_outputs = []
        if self.core_context.distributed.rank == 0:
            outputs = flatten(cast(List[List[pl.utilities.types.STEP_OUTPUT]], outputs))
            if outputs:
                avg_results = {
                    k: sum([x[k] for x in outputs]) / len(outputs) for k in outputs[0].keys()
                }
                self.core_context.train.report_validation_metrics(
                    steps_completed=self.shared.global_step, metrics=avg_results
                )
        if self.core_context.distributed.rank == 0:
            self.shared.current_op.report_progress(trainer.current_epoch + 1)
        if (trainer.current_epoch + 1) >= self.shared.current_op.length:
            if self.core_context.distributed.rank == 0:
                if self.shared.last_metric is None:
                    logging.warning(
                        f"Searcher metric {get_searcher_metric()} was not "
                        "logged.  Reporting as 0.",
                    )
                    self.shared.current_op.report_completed(0)
                else:
                    self.shared.current_op.report_completed(self.shared.last_metric)
            try:
                self.shared.current_op = next(self.shared.searcher_ops)
            except StopIteration:
                logging.info("Reached end of searcher operations.")
                trainer.should_stop = True
        if self.core_context.preempt.should_preempt():
            trainer.save_checkpoint(TEMP_CHECKPOINT_FILE)
            raise Exception("Training pre-empted.")

    def on_test_epoch_start(self, trainer: pl.Trainer, pl_module: pl.LightningModule) -> None:
        self.test_epoch_outputs = []

    def on_test_batch_end(
        self,
        trainer: pl.Trainer,
        pl_module: pl.LightningModule,
        outputs: Optional[pl.utilities.types.STEP_OUTPUT],
        batch: Any,
        batch_idx: int,
        dataloader_idx: int,
    ) -> None:
        if outputs:
            self.test_epoch_outputs.append({k: v.item() for k, v in outputs.items()})

    def on_test_epoch_end(self, trainer: pl.Trainer, pl_module: pl.LightningModule) -> None:
        outputs = self.core_context.distributed.gather(self.test_epoch_outputs)
        self.test_epoch_outputs = []
        if self.core_context.distributed.rank == 0:
            outputs = flatten(cast(List[List[pl.utilities.types.STEP_OUTPUT]], outputs))
            if outputs:
                avg_results = {
                    k: sum([x[k] for x in outputs]) / len(outputs) for k in outputs[0].keys()
                }
                self.core_context.train.report_validation_metrics(
                    steps_completed=self.shared.global_step, metrics=avg_results
                )


# This class inserts a save_checkpoint hook into DeepSpeedStrategy.
class DeterminedDeepSpeedStrategy(pl.strategies.DeepSpeedStrategy):  # type: ignore
    def __init__(self, *args: List, **kwargs: Dict) -> None:
        super().__init__(*args, **kwargs)
        self._shared: Optional[DeterminedIntegrationSharedState] = None

    def _set_shared(self, shared: DeterminedIntegrationSharedState) -> None:
        self._shared = shared

    def save_checkpoint(
        self, checkpoint: Dict, filepath: Union[str, Path], storage_options: Optional[Any] = None
    ) -> None:
        assert self._shared
        super().save_checkpoint(checkpoint, filepath, storage_options)
        upload_determined_checkpoint(filepath, self._shared)


def build_determined_trainer(
    core_context: det.core.Context,
    module_cls: Type[pl.LightningModule],
    *args: List[Any],
    base_ckpt_io: Optional[pl.plugins.io.CheckpointIO] = None,
    **kwargs: Any,
) -> Tuple[pl.Trainer, pl.LightningModule]:
    searcher_ops = core_context.searcher.operations()
    current_op = next(searcher_ops)
    shared = DeterminedIntegrationSharedState(
        core_context=core_context,
        searcher_ops=searcher_ops,
        current_op=current_op,
    )
    if "strategy" in kwargs:
        strategy = kwargs["strategy"]
        if isinstance(strategy, DeterminedDeepSpeedStrategy):
            strategy._set_shared(shared)
    module_load_only = False
    ckpt_metadata = get_checkpoint_metadata(core_context)
    if ckpt_metadata and ckpt_metadata["trial_id"] != get_cluster_info_with_assert().trial.trial_id:
        # New trial, experiment hyperparameters may have changed.  Instead of fully loading
        # checkpoint, just load the module.
        logging.info("New trial -- only loading module weights and not training state.")
        module_load_only = True
    ckpt_path = download_checkpoint(core_context, module_load_only)
    if module_load_only:
        module = module_cls.load_from_checkpoint(ckpt_path)
    else:
        module = module_cls()

    if "max_epochs" in kwargs:
        logging.warning(
            "Overwriting max_epochs argument with Determined configured searcher max_length"
        )
    kwargs["max_epochs"] = get_searcher_max_length()
    callbacks: List[pl.callbacks.Callback] = kwargs.get("callbacks", [])
    callbacks.append(DeterminedCallback(shared))
    kwargs["callbacks"] = callbacks
    plugins: List[pl.plugins.PLUGIN] = kwargs.get("plugins", [])
    plugins.append(DeterminedCheckpointIO(shared, base_ckpt_io))
    kwargs["plugins"] = plugins
    loggers = kwargs.get("logger", [])
    if not (isinstance(loggers, list)):
        loggers = [loggers]
    loggers.append(DeterminedLogger(shared))
    kwargs["logger"] = loggers
    if not (module_load_only):
        if "resume_from_checkpoint" in kwargs and kwargs["resume_from_checkpoint"]:
            logging.warning(
                "Overwriting resume_from_checkpoint argument with Determined checkpoint."
            )
        kwargs["resume_from_checkpoint"] = ckpt_path
    return (pl.Trainer(*args, **kwargs), module)
