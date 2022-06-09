import enum


class TensorboardMode(enum.Enum):
    """
    ```TensorboardMode`` defines how Tensorboard metrics and profiling data are retained.
    In ``Auto`` mode only the chief uploads its metrics and profiling data to checkpoint
    storage. This is the same behavior that existed prior to 0.18.2.
    In ``Manual`` mode neither metrics nor profiling data are reported. It is up to the user
    to call ``TrainContext.upload_tensorboard_files()``
    """

    AUTO = "AUTO"
    MANUAL = "MANUAL"
