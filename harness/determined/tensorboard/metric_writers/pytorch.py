import warnings

from determined.pytorch.tensorboard_writer import TorchWriter  # noqa: F401

warnings.warn(
    "'tensorboard.pytorch.TorchWriter' is deprecated in favor \
     of 'determined.pytorch.tensorboard_writer.TorchWriter'",
    DeprecationWarning,
    stacklevel=2,
)
