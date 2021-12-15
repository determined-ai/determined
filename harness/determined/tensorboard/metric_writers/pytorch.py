from typing import Any, Union

import numpy as np

from determined import tensorboard

# As of torch v1.9.0, torch.utils.tensorboard has a bug that is exposed by setuptools 59.6.0.  The
# bug is that it attempts to import distutils then access distutils.version without actually
# importing distutils.version.  We can workaround this by prepopulating the distutils.version
# submodule in the distutils module.
import distutils.version  # isort:skip  # noqa: F401
from torch.utils.tensorboard import SummaryWriter  # isort:skip


class TorchWriter(tensorboard.MetricWriter):
    """
    TorchWriter uses PyTorch file writers and summary operations to write
    out tfevent files containing scalar batch metrics. It creates
    an instance of ``torch.utils.tensorboard.SummaryWriter`` which can be
    accessed via the ``writer`` field and configures the SummaryWriter
    to write to the correct directory inside the trial container.

    Usage example:

     .. code-block:: python

        from determined.tensorboard.metric_writers.pytorch import TorchWriter

        class MyModel(PyTorchTrial):
            def __init__(self, context):
                ...
                self.logger = TorchWriter()

            def train_batch(self, batch, epoch_idx, batch_idx):
                self.logger.writer.add_scalar('my_metric', np.random.random(), batch_idx)
    """

    def __init__(self) -> None:
        super().__init__()

        self.writer: Any = SummaryWriter(log_dir=tensorboard.get_base_path({}))  # type: ignore

    def add_scalar(self, name: str, value: Union[int, float, np.number], step: int) -> None:
        self.writer.add_scalar(name, value, step)

    def reset(self) -> None:
        if "flush" in dir(self.writer):
            self.writer.flush()
