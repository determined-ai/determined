from abc import ABCMeta, abstractmethod
from typing import Optional, Union

import tensorflow
from packaging import version

# Handle TensorFlow compatibility issues.
if version.parse(tensorflow.__version__) >= version.parse("1.14.0"):
    import tensorflow.compat.v1 as tf
    from tensorflow.compat.v1.data import Dataset, Iterator
else:
    import tensorflow as tf
    from tensorflow.data import Dataset, Iterator


class TensorFlowDatasetAdapter(metaclass=ABCMeta):
    """
    A class to assist with restoring and saving iterators for a dataset.
    This class may be subclassed for users with very customized data
    pipelines.

    Arguments:
        dataset:
            The tf.data.Dataset object from which data should be fetched.
    """

    @abstractmethod
    def get_iterator(self, repeat: bool = False, skip_batches: int = 0) -> Iterator:
        """
        Return a TensorFlow tf.data.Dataset, or restore one from the files
        of a checkpoint. The first element the returned Dataset produces should
        be `skip_batches` number of batches into the Dataset.

        Arguments:
            repeat:
                Indicate if dataset should be pre-transformed with a repeat().
            skip_batches:
                Indicate how many batches should be skipped from the beginning
                of the dataset before starting. With normal tf.data.Datasets,
                `skip_batches` is ignored; the iterator is created but its
                state is restored during restore_iterator(). Adapters which
                wrap non-Dataset generators using from_generator() should use
                `skip_batches` and ignore the call to restore_iterator().
        """
        pass

    def save_iterator(
        self, iterator: tf.data.Iterator, save_path: str, save_session: tf.Session
    ) -> None:
        """
        Save an iterator to a checkpoint.

        Arguments:
            iterator:
                The iterator to be saved.
            save_path:
                The path to a checkpoint used for restoring an iterator.
            save_session:
                The TensorFlow session which should be used for restoring an
                iterator from a checkpoint.
        """
        pass

    def restore_iterator(
        self,
        iterator: tf.data.Iterator,
        restore_path: str,
        restore_session: tf.Session,
        run_options: tf.RunOptions = None,
    ) -> Iterator:
        """
        Restore an iterator from a checkpoint.

        Arguments:
            iterator:
                The iterator to be restored.
            restore_path:
                The path to a checkpoint used for restoring an iterator.
            restore_session:
                The TensorFlow session which should be used for restoring an
                iterator from a checkpoint.
            run_options:
                The tf.RunOptions to pass to the tf.Session during
                tf.Saver.restore().
        """
        return iterator

    def initialize_iterator(
        self, iterator: tf.data.Iterator, initialize_session: tf.Session
    ) -> None:
        """
        Initialize an iterator produced by this TensorFlowDatasetAdapter, if
        necessary.

        This method is part of experimental support for some Datasets which
        are incompatible with make_one_shot_iterator().  This method is
        likely change or disappear in future versions.
        """
        pass


class DatasetToTensorFlowDatasetAdapter(TensorFlowDatasetAdapter):
    def __init__(self, dataset: Dataset, prefetch_buffer: int = 1) -> None:
        self.dataset = dataset
        self.prefetch_buffer = prefetch_buffer

    def get_iterator(self, repeat: bool = False, skip_batches: int = 0) -> Iterator:
        # Ignore the skip_batches argument; instead of skipping batches at
        # startup, this iterator will restore its old state from a checkpoint.

        temp = self.dataset
        if repeat:
            # Having an extra repeat should be ok, so we don't need to check if
            # the dataset already has one.
            temp = temp.repeat()

        if self.prefetch_buffer > 0:
            temp = temp.prefetch(self.prefetch_buffer)

        return temp.make_one_shot_iterator()

    def save_iterator(
        self, iterator: tf.data.Iterator, save_path: str, save_session: tf.Session
    ) -> None:
        saveable = tf.data.experimental.make_saveable_from_iterator(iterator)
        saver = tf.train.Saver({"iterator": saveable})
        saver.save(save_session, save_path)

    def restore_iterator(
        self,
        iterator: tf.data.Iterator,
        restore_path: str,
        restore_session: tf.Session,
        run_options: tf.RunOptions = None,
    ) -> Iterator:
        saveable = tf.data.experimental.make_saveable_from_iterator(iterator)
        restorer = tf.train.Saver({"iterator": saveable})
        restorer.restore(restore_session, restore_path, options=run_options)
        return iterator


def make_tensorflow_dataset_adapter(
    data_loader: Union[Dataset, "TensorFlowDatasetAdapter"], batch_size: Optional[int]
) -> TensorFlowDatasetAdapter:
    if isinstance(data_loader, TensorFlowDatasetAdapter):
        return data_loader
    elif isinstance(data_loader, Dataset):
        return DatasetToTensorFlowDatasetAdapter(data_loader)
    else:
        raise ValueError(
            "Unable to treat data loader of type {} as a TensorFlow Dataset.".format(
                type(data_loader).__name__
            )
        )
