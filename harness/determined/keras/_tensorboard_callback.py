import logging
from typing import Any

from determined.keras import callbacks


class TFKerasTensorBoard(callbacks.TensorBoard):
    def __init__(self, *args: Any, **kwargs: Any):
        logging.warning(
            "det.keras.TFKerasTensorBoard is a deprecated name for "
            "det.keras.callbacks.TensorBoard, please update your code."
        )
        # Avoid using super() due to a diamond inheritance pattern.
        callbacks.TensorBoard.__init__(self, *args, **kwargs)
