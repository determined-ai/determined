from typing import Any, Union

import numpy as np
from torch.utils.tensorboard import SummaryWriter

from determined import tensorboard


class TorchWriter(tensorboard.MetricWriter):
    def __init__(self) -> None:
        """
        TorchWriter uses pytorch file writers and summary operations to write
        out tfevent files containing scalar batch metrics.
        """
        super().__init__()

        self.writer: Any = SummaryWriter(log_dir=tensorboard.get_base_path({}))  # type: ignore

    def add_scalar(self, name: str, value: Union[int, float, np.number], step: int) -> None:
        self.writer.add_scalar(name, value, step)

    def reset(self) -> None:
        if "flush" in dir(self.writer):
            self.writer.flush()
