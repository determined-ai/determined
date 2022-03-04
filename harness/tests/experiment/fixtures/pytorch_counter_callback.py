# type: ignore
import logging
from typing import Any, Dict

from determined import pytorch


class Counter(pytorch.PyTorchCallback):
    def __init__(self) -> None:
        self.validation_steps_started = 0
        self.validation_steps_ended = 0
        self.checkpoints_ended = 0
        self.training_started_times = 0
        self.training_epochs_started = 0
        self.training_epochs_ended = 0

    def on_validation_start(self) -> None:
        self.validation_steps_started += 1

    def on_validation_end(self, metrics: Dict[str, Any]) -> None:
        self.validation_steps_ended += 1

    def on_checkpoint_end(self, checkpoint_dir: str):
        self.checkpoints_ended += 1

    def on_training_start(self) -> None:
        logging.debug("starting training")
        self.training_started_times += 1

    def on_training_epoch_start(self, epoch_idx) -> None:
        logging.debug(f"starting epoch {epoch_idx}")
        self.training_epochs_started += 1

    def on_training_epoch_end(self, epoch_idx: int) -> None:
        logging.debug(f"end of epoch {epoch_idx}")
        self.training_epochs_ended += 1

    def state_dict(self) -> Dict[str, Any]:
        return self.__dict__

    def load_state_dict(self, state_dict: Dict[str, Any]) -> None:
        self.__dict__ = state_dict
