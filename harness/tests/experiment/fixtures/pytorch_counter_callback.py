# type: ignore
import logging
from typing import Any, Dict

from determined import pytorch


class Counter(pytorch.PyTorchCallback):
    def __init__(self) -> None:
        self.validation_steps_started = 0
        self.validation_steps_ended = 0
        self.checkpoints_written = 0
        self.checkpoints_uploaded = 0
        self.training_started_times = 0
        self.training_epochs_started = 0
        self.training_epochs_ended = 0
        self.training_workloads_ended = 0
        self.trial_startups = 0
        self.trial_shutdowns = 0

    def on_validation_start(self) -> None:
        self.validation_steps_started += 1

    def on_validation_end(self, metrics: Dict[str, Any]) -> None:
        self.validation_steps_ended += 1

    def on_checkpoint_write_end(self, checkpoint_dir: str):
        self.checkpoints_written += 1

    def on_checkpoint_upload_end(self, uuid: str) -> None:
        logging.debug(f"checkpoint upload uuid {uuid}")
        self.checkpoints_uploaded += 1

    def on_training_start(self) -> None:
        logging.debug("starting training")
        self.training_started_times += 1

    def on_training_epoch_start(self, epoch_idx) -> None:
        logging.debug(f"starting epoch {epoch_idx}")
        self.training_epochs_started += 1

    def on_training_epoch_end(self, epoch_idx: int) -> None:
        logging.debug(f"end of epoch {epoch_idx}")
        self.training_epochs_ended += 1

    def on_training_workload_end(
        self, avg_metrics: Dict[str, Any], batch_metrics: Dict[str, Any]
    ) -> None:
        logging.debug(f"training workload avg_metrics {avg_metrics}")
        logging.debug(f"training workload batch_metrics {batch_metrics}")
        self.training_workloads_ended += 1

    def state_dict(self) -> Dict[str, Any]:
        return self.__dict__

    def load_state_dict(self, state_dict: Dict[str, Any]) -> None:
        self.__dict__ = state_dict

    def on_trial_startup(self, *arg):
        self.trial_startups += 1

    def on_trial_shutdown(self):
        self.trial_shutdowns += 1
