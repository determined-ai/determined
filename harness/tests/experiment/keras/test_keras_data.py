import numpy as np
import pytest
from tensorflow.keras.utils import Sequence

import determined as det
from determined import keras
from determined_common import check
from tests.experiment import utils  # noqa: I100


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


def test_sequence_offset() -> None:
    seq = utils.make_xor_data_sequences()[0]
    offset_seq = keras._SequenceWithOffset(seq, batch_offset=1)
    assert len(seq) == len(offset_seq)

    for i in range(len(offset_seq)):
        a = offset_seq[i]
        b = seq[(i + 1) % len(seq)]
        assert len(a) == len(b)
        for i in range(len(a)):
            assert np.equal(a[i], b[i]).all()


@pytest.mark.parametrize("workers,use_multiprocessing,seq", MULTITHREADING_MULTIPROCESS_SUITE)
def test_sequence_adapter(workers: int, use_multiprocessing: bool, seq: Sequence) -> None:
    data = keras.SequenceAdapter(seq, workers=workers, use_multiprocessing=use_multiprocessing)
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


def test_minial_arraylike_data_adapter() -> None:
    seq = keras._ArrayLikeAdapter(np.array([0]), np.array([1]), batch_size=1)
    assert len(seq) == 1
    assert seq[0] == (np.array([0]), np.array([1]))


def test_arraylike_data_adapter_drop_leftovers() -> None:
    seq = keras._ArrayLikeAdapter(
        np.arange(0, 100), np.arange(100, 200), batch_size=16, drop_leftovers=False
    )
    assert len(seq) == 7  # ceil(100/16)
    assert np.array_equal(seq[6], (np.arange(96, 100), np.arange(196, 200)))

    seq = keras._ArrayLikeAdapter(
        np.arange(0, 100), np.arange(100, 200), batch_size=16, drop_leftovers=True
    )
    assert len(seq) == 6  # floor(100/16)
    assert np.array_equal(seq[3], (np.arange(48, 64), np.arange(148, 164)))


def test_arraylike_data_adapter_with_unmatched_batch_size() -> None:
    with pytest.raises(check.CheckFailedError):
        keras._ArrayLikeAdapter(np.arange(0, 16), np.arange(0, 16), batch_size=32)


def test_adapt_invalid_data_type() -> None:
    seqs = utils.make_xor_data_sequences()
    test = keras._adapt_keras_data(seqs[1], batch_size=1)
    with pytest.raises(det.errors.InvalidDataTypeException) as err:
        keras._adapt_keras_data((None, test), batch_size=1)
        assert err is not None


def test_adapt_list_of_np_arrays_as_x() -> None:
    adapted = keras._adapt_keras_data(
        x=[np.arange(0, 100), np.arange(100, 200)],
        y=np.arange(200, 300),
        batch_size=16,
        drop_leftovers=False,
    )
    assert isinstance(adapted, keras.SequenceAdapter)
    assert len(adapted) == 7
    batch_x, batch_y = adapted._sequence[3]
    assert np.array_equal(batch_x[0], np.arange(48, 64))
    assert np.array_equal(batch_x[1], np.arange(148, 164))
    assert np.array_equal(batch_y, np.arange(248, 264))


def test_adapt_list_of_np_arrays_as_y() -> None:
    adapted = keras._adapt_keras_data(
        x=np.arange(0, 100),
        y=[np.arange(100, 200), np.arange(200, 300)],
        batch_size=16,
        drop_leftovers=False,
    )
    assert isinstance(adapted, keras.SequenceAdapter)
    assert len(adapted) == 7
    batch_x, batch_y = adapted._sequence[3]
    assert np.array_equal(batch_x, np.arange(48, 64))
    assert np.array_equal(batch_y[0], np.arange(148, 164))
    assert np.array_equal(batch_y[1], np.arange(248, 264))


def test_adapt_dict_of_np_arrays_as_x() -> None:
    adapted = keras._adapt_keras_data(
        x={"k1": np.arange(0, 100), "k2": np.arange(100, 200)},
        y=np.arange(200, 300),
        batch_size=16,
        drop_leftovers=False,
    )
    assert isinstance(adapted, keras.SequenceAdapter)
    assert len(adapted) == 7
    batch_x, batch_y = adapted._sequence[3]
    assert np.array_equal(batch_x["k1"], np.arange(48, 64))
    assert np.array_equal(batch_x["k2"], np.arange(148, 164))
    assert np.array_equal(batch_y, np.arange(248, 264))


def test_adapt_empty_sequence() -> None:
    sequence = Empty()
    with pytest.raises(ValueError):
        keras._adapt_keras_data(sequence, batch_size=1)


def test_adapt_sequence() -> None:
    seqs = utils.make_xor_data_sequences()
    train = keras._adapt_keras_data(seqs[0], batch_size=1)
    assert isinstance(train, keras.SequenceAdapter)
    test = keras._adapt_keras_data(seqs[1], batch_size=1)
    assert isinstance(test, keras.SequenceAdapter)
    assert seqs[0] is train._sequence._sequence
    assert seqs[1] is test._sequence._sequence

    assert train is keras._adapt_keras_data(train, batch_size=1)
    assert test is keras._adapt_keras_data(test, batch_size=1)
