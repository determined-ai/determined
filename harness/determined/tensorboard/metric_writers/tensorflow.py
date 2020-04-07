import time
from typing import Optional, Set, Union

import numpy as np
import tensorflow
from packaging import version
from tensorflow.core.framework import summary_pb2
from tensorflow.core.util import event_pb2
from tensorflow.python.summary.writer.event_file_writer import EventFileWriter

from determined import tensorboard

# TODO(ryan): remove this check after removing support for TensorFlow 1.13.1.
if version.parse(tensorflow.__version__) >= version.parse("1.14.0"):
    import tensorflow.compat.v1 as tf
else:
    import tensorflow as tf


class TFWriter(tensorboard.MetricWriter):
    """
    TFWriter uses tensorflow file writers and summary operations to write out
    tfevent files containing scalar batch metrics.
    """

    def __init__(self) -> None:
        super().__init__()
        self.writer = EventFileWriter(
            logdir=str(tensorboard.get_base_path({})), filename_suffix=None
        )
        self.createSummary = tf.Summary

        # _seen_summary_tags is vendored from TensorFlow: tensorflow/python/summary/writer/writer.py
        # This set contains tags of Summary Values that have been encountered
        # already. The motivation here is that the SummaryWriter only keeps the
        # metadata property (which is a SummaryMetadata proto) of the first Summary
        # Value encountered for each tag. The SummaryWriter strips away the
        # SummaryMetadata for all subsequent Summary Values with tags seen
        # previously. This saves space.
        self._seen_summary_tags: Set[str] = set()

    def add_scalar(self, name: str, value: Union[int, float, np.number], step: int) -> None:
        summary = self.createSummary()
        summary_value = summary.value.add()
        summary_value.tag = name
        summary_value.simple_value = value
        self._add_summary(summary, step)

    def _add_summary(
        self, summary: Union[str, summary_pb2.Summary], global_step: Optional[int] = None
    ) -> None:
        """
        _add_summary is vendored from TensorFlow: tensorflow/python/summary/writer/writer.py

        Adds a `Summary` protocol buffer to the event file.

        This method wraps the provided summary in an `Event` protocol buffer
        and adds it to the event file.

        You can pass the result of evaluating any summary op, using
        `tf.Session.run` or
        `tf.Tensor.eval`, to this
        function. Alternatively, you can pass a `tf.compat.v1.Summary` protocol
        buffer that you populate with your own data. The latter is
        commonly done to report evaluation results in event files.

        Args:
          summary: A `Summary` protocol buffer, optionally serialized as a string.
          global_step: Number. Optional global step value to record with the
            summary.
        """
        if isinstance(summary, bytes):
            summ = summary_pb2.Summary()
            summ.ParseFromString(summary)
            summary = summ

        # We strip metadata from values with tags that we have seen before in order
        # to save space - we just store the metadata on the first value with a
        # specific tag.
        for value in summary.value:
            if not value.metadata:
                continue

            if value.tag in self._seen_summary_tags:
                # This tag has been encountered before. Strip the metadata.
                value.ClearField("metadata")
                continue

            # We encounter a value with a tag we have not encountered previously. And
            # it has metadata. Remember to strip metadata from future values with this
            # tag string.
            self._seen_summary_tags.add(value.tag)

        event = event_pb2.Event(summary=summary)
        self._add_event(event, global_step)

    def _add_event(self, event: event_pb2.Event, step: Optional[int]) -> None:
        # _add_event is vendored from TensorFlow: tensorflow/python/summary/writer/writer.py
        event.wall_time = time.time()
        if step is not None:
            event.step = int(step)
        self.writer.add_event(event)

    def reset(self) -> None:
        self.writer.close()
        self.writer.reopen()
