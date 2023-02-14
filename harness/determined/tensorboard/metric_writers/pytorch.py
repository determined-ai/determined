import warnings

from determined.pytorch.tensorboard_writer import TorchWriter

warnings.warn(
    "'tensorboard.pytorch' is deprecated in favor of 'determined.pytorch.tensorboard_writer'",
    DeprecationWarning,
    stacklevel=2,
)
