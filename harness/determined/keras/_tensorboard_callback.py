from typing import Any

import tensorflow as tf

from determined import tensorboard


class TFKerasTensorBoard(tf.keras.callbacks.TensorBoard):  # type: ignore
    """
    This is a thin wrapper over the TensorBoard callback that ships with ``tf.keras``.  For more
    information, see the :ref:`TensorBoard Guide <how-to-tensorboard>` or the upstream docs for
    `tf.keras.callbacks.TensorBoard
    <https://www.tensorflow.org/api_docs/python/tf/keras/callbacks/TensorBoard>`__.

    Note that if a ``log_dir`` argument is passed to the constructor, it will be ignored.
    """

    def __init__(self, *args: Any, **kwargs: Any):
        log_dir = str(tensorboard.get_base_path({}).resolve())
        super().__init__(log_dir=log_dir, *args, **kwargs)

    def _write_logs(self, *args: Any) -> None:
        """
        _write_logs calls the original _write_logs() function from the Keras
        TensorBoard callback. After the logs are flushed to disk, we close and
        reopen the tf event writer so that it serializes the next set of logs
        to a new file. This allows the tensorboard manager to treat the
        written files as immutable and upload them to persistent storage
        without later having to append to them. This behavior is useful for
        tensorboard backed by S3.
        """
        super()._write_logs(*args)
        self.writer.close()
        self.writer.reopen()
