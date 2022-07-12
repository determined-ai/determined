import enum


class TensorboardMode(enum.Enum):
    """
    ``TensorboardMode`` defines how Tensorboard artifacts are handled.

    In ``AUTO`` mode the chief automatically writes any reported training or validation
    metrics to the Tensorboard path (see :meth:`TrainContext.get_tensorboard_path()
    <determined.core.TrainContext.get_tensorboard_path>`), and automatically
    uploads all of its own tensorboard artifacts to checkpoint storage.  Tensorboard
    artifacts written by non-chief workers will not be uploaded at all. This is the
    same behavior that existed prior to 0.18.3.

    In ``MANUAL`` mode no Tensorboard artifacts are written or uploaded at all.
    It is entirely up to the user to write their desired metrics and upload
    them with calls to :meth:`TrainContext.upload_tensorboard_files()
    <determined.core.TrainContext.upload_tensorboard_files>`.
    """

    AUTO = "AUTO"
    MANUAL = "MANUAL"
