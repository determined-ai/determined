import warnings
from typing import Any, Union

import numpy as np
import torch
from packaging import version

from determined import tensorboard as det_tensorboard

# As of torch v1.9.0, torch.utils.tensorboard has a bug that is exposed by setuptools 59.6.0.  The
# bug is that it attempts to import distutils then access distutils.version without actually
# importing distutils.version.  We can workaround this by prepopulating the distutils.version
# submodule in the distutils module.
if version.parse("1.9.0") <= version.parse(torch.__version__) < version.parse("1.11.0"):
    # Except, starting with python 3.12 distutils isn't available at all.
    try:
        import distutils.version  # isort:skip  # noqa: F401
    except ImportError:
        pass

from torch.utils import tensorboard  # isort:skip


class _TorchWriter(det_tensorboard.MetricWriter):
    def __init__(self) -> None:
        super().__init__()

        self.writer: Any = tensorboard.SummaryWriter(
            log_dir=det_tensorboard.get_base_path({})
        )  # type: ignore

    def add_scalar(self, name: str, value: Union[int, float, np.number], step: int) -> None:
        self.writer.add_scalar(name, value, step)

    def reset(self) -> None:
        # flush AND close the writer so that the next attempt to write will create a new file
        self.writer.close()

    def flush(self) -> None:
        self.writer.flush()


class TorchWriter(_TorchWriter):
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

    .. warning::

        TorchWriter() has been deprecated and will be removed in a future version.

        Users are encouraged to switch to one of the following depending on use case:
            - Trials Users, see ``determined.pytorch.PyTorchTrialContext.get_tensorboard_writer()``
            - CoreAPI Users, create a ``torch.utils.tensorboard.SummaryWriter()`` object
              and pass ``core_context.train.get_tensorboard_path()`` as the log_dir
    """

    def __init__(self) -> None:
        warnings.warn(
            "TorchWriter() has been deprecated and will be removed in a future version.\n \
            Users are encouraged to switch to one of the following depending on use case:\n \
            - Trials Users, see determined.pytorch.PyTorchTrialContext.get_tensorboard_writer()\n \
            - CoreAPI Users, create a torch.utils.tensorboard.SummaryWriter() object\n \
              and pass core_context.train.get_tensorboard_path() as the log_dir",
            FutureWarning,
            stacklevel=2,
        )
        super().__init__()
