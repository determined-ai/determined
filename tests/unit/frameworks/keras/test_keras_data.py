from typing import Dict, List, Tuple, Union

import numpy as np
import pytest
from tensorflow.keras.utils import Sequence

from determined import keras
from determined_common import check
from tests.unit.frameworks import utils  # noqa: I100


def unwrap_data(tup: Tuple) -> keras.KerasBatch:
    if len(tup) != 2 and len(tup) != 3:
        raise TypeError(
            "Inputs to Keras trials must be of the structure `(data, labels)`"
            " or `(data, labels, sample_weight)`"
        )

    data = None
    labels = None
    sample_weight = None

    data = tup[0]
    labels = tup[1]

    if len(tup) == 3:
        sample_weight = tup[2]

    return keras.KerasBatch(data, labels, sample_weight)


class Empty(Sequence):
    def __getitem__(self, index: int) -> list:
        return []

    def __len__(self) -> int:
        return 0


SEQ = utils.make_xor_data_sequences()[0]
MULTITHREADING_MULTIPROCESS_SUITE = [
    (0, False, SEQ),
    (1, False, SEQ),
    (1, True, SEQ),
    (2, False, SEQ),
    (2, True, SEQ),
]


def test_make_keras_data_adapter_from_sequence() -> None:
    seqs = utils.make_xor_data_sequences()
    train = keras.make_keras_data_adapter(seqs[0], 1)
    test = keras.make_keras_data_adapter(seqs[1], 1)
    assert seqs[0] is train._sequence._sequence
    assert seqs[1] is test._sequence._sequence

    assert train is keras.make_keras_data_adapter(train, 1)
    assert test is keras.make_keras_data_adapter(test, 1)

    with pytest.raises(ValueError) as err:
        keras.make_keras_data_adapter((None, test), 1)
        assert err is not None


def test_make_keras_data_adapter_with_empty_sequence() -> None:
    sequence = Empty()
    with pytest.raises(ValueError):
        keras.make_keras_data_adapter(sequence, 1)


def test_sequence_offset() -> None:
    seq = utils.make_xor_data_sequences()[0]
    offset_seq = keras.SequenceWithOffset(seq, batch_offset=1)
    assert len(seq) == len(offset_seq)

    for i in range(len(offset_seq)):
        a = offset_seq[i]
        b = seq[(i + 1) % len(seq)]
        assert len(a) == len(b)
        for i in range(len(a)):
            assert np.equal(a[i], b[i]).all()


@pytest.mark.parametrize("workers,use_multiprocessing,seq", MULTITHREADING_MULTIPROCESS_SUITE)
def test_data(workers: int, use_multiprocessing: bool, seq: Sequence) -> None:
    data = keras.KerasDataAdapter(seq, workers=workers, use_multiprocessing=use_multiprocessing)
    assert len(data) == len(seq)

    data.start()
    iterator = data.get_iterator()
    assert iterator is not None

    for i in range(len(seq)):
        a = seq[i]
        b = next(iterator)
        assert len(a) == len(b)
        for i in range(len(a)):
            assert np.equal(a[i], b[i]).all()
    data.stop()


n = np.array([0])
lst = [n.copy()]
d = {"a": n.copy()}


UNWRAP_DATA_SUITE = [
    (n, n, None, len(n)),
    (n, lst, None, len(n)),
    (n, d, None, len(n)),
    (lst, n, None, len(n)),
    (d, n, None, len(n)),
    (n, n, n, len(n)),
]


KerasValidTypes = Union[Dict, List, np.ndarray]


@pytest.mark.parametrize("data,labels,sample_weight,length", UNWRAP_DATA_SUITE)
def test_unwrap_data(
    data: KerasValidTypes, labels: KerasValidTypes, sample_weight: np.ndarray, length: int
) -> None:
    batch = unwrap_data((data, labels, sample_weight))
    assert data is batch.data
    assert labels is batch.labels
    assert sample_weight is batch.sample_weight
    assert len(batch) == length


def test_unwrap_data_fails() -> None:
    with pytest.raises(TypeError):
        unwrap_data(())

    with pytest.raises(TypeError):
        unwrap_data((n, n, n, n))

    batch = unwrap_data((0, 0))
    with pytest.raises(TypeError):
        len(batch)


def test_minimal_in_memory_sequence() -> None:
    seq = keras.InMemorySequence(np.array([0]), np.array([1]), batch_size=1)
    assert len(seq) == 1
    assert seq[0] == (np.array([0]), np.array([1]))


def test_in_memory_sequence() -> None:
    seq = keras.InMemorySequence(
        np.arange(0, 100), np.arange(100, 200), batch_size=16, drop_leftovers=False
    )
    assert len(seq) == 7  # ceil(100/16)
    assert np.array_equal(seq[6], (np.arange(96, 100), np.arange(196, 200)))

    seq = keras.InMemorySequence(
        np.arange(0, 100), np.arange(100, 200), batch_size=16, drop_leftovers=True
    )
    assert len(seq) == 6  # floor(100/16)
    assert np.array_equal(seq[3], (np.arange(48, 64), np.arange(148, 164)))


def test_fail_in_memory_sequence() -> None:
    with pytest.raises(check.CheckFailedError):
        keras.InMemorySequence(np.arange(0, 16), np.arange(0, 16), batch_size=32)
