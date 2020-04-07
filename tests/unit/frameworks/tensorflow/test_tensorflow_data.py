import os
import shutil
import sys
import tempfile

import pytest
import tensorflow as tf

from determined.keras import DatasetToTensorFlowDatasetAdapter


@pytest.fixture()
def tempdir():
    directory = tempfile.mkdtemp("test_tensorflow_data")
    yield directory
    shutil.rmtree(directory)


# Use a user-defined dataset transformation function with a pure-python map
# inside of it to assert that iterator saving and restoring can handle user-
# defined dataset transformations.
def user_transformation(dataset):
    return dataset.map(lambda x: 2 * x)


class TestTensorFlowDatasetAdapter:
    def test_iterator_saving_restoring(self, tempdir):
        tf.reset_default_graph()
        chkpt_name = os.path.join(tempdir, "chkpt")

        with tf.Session() as sess:
            # Iterate through 10 batches of 3 numbers, multiplied by 2.
            ds = tf.data.Dataset.range(30).batch(3).apply(user_transformation)
            adapter = DatasetToTensorFlowDatasetAdapter(ds)
            iterator = adapter.get_iterator()
            next_item = iterator.get_next()

            # Read in the first half of the dataset
            values = [sess.run(next_item) for i in range(5)]
            expect = [[i * 3, i * 3 + 2, i * 3 + 4] for i in range(0, 10, 2)]
            print("comparing {} to {}".format(values, expect), file=sys.stderr)
            assert len(values) == len(expect)
            for v, e in zip(values, expect):
                assert (v == e).all()

            adapter.save_iterator(iterator, chkpt_name, sess)

            # Confirm we can read the second half of the dataset.
            values = [sess.run(next_item) for i in range(5)]
            expect = [[i * 3, i * 3 + 2, i * 3 + 4] for i in range(10, 20, 2)]
            print("comparing {} to {}".format(values, expect), file=sys.stderr)
            assert len(values) == len(expect)
            for v, e in zip(values, expect):
                assert (v == e).all()

        tf.reset_default_graph()

        with tf.Session() as sess:
            ds = tf.data.Dataset.range(30).batch(3).apply(user_transformation)
            adapter = DatasetToTensorFlowDatasetAdapter(ds)
            iterator = adapter.get_iterator()

            # Reload the iterator and read the second half of the batches again.
            new_iterator = adapter.restore_iterator(iterator, chkpt_name, sess)
            new_next_item = new_iterator.get_next()

            # Confirm we can read the second half of the dataset again.
            values = [sess.run(new_next_item) for i in range(5)]
            expect = [[i * 3, i * 3 + 2, i * 3 + 4] for i in range(10, 20, 2)]
            print("comparing {} to {}".format(values, expect), file=sys.stderr)
            assert len(values) == len(expect)
            for v, e in zip(values, expect):
                assert (v == e).all()
