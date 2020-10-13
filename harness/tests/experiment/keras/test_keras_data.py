import numpy as np
import pytest
from tensorflow.keras.utils import Sequence

import determined as det
from determined import keras
from determined_common import check
from tests.experiment import utils  # noqa: I100


class IdentitySequence(Sequence):
    def __init__(self, length: int) -> None:
        self._length = length

    def __len__(self) -> int:
        return self._length

    def __getitem__(self, index: int) -> int:
        assert index < self._length
        return index


def test_minimal_arraylike_data_adapter() -> None:
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
    test = keras._adapt_data_from_data_loader(seqs[1], batch_size=1)
    with pytest.raises(det.errors.InvalidDataTypeException) as err:
        keras._adapt_data_from_data_loader((None, test), batch_size=1)
        assert err is not None


def test_adapt_list_of_np_arrays_as_x() -> None:
    adapted = keras._adapt_data_from_fit_args(
        x=[np.arange(0, 100), np.arange(100, 200)],
        y=np.arange(200, 300),
        sample_weight=None,
        batch_size=16,
    )
    assert isinstance(adapted, Sequence)
    assert len(adapted) == 7
    batch_x, batch_y = adapted[3]
    assert np.array_equal(batch_x[0], np.arange(48, 64))
    assert np.array_equal(batch_x[1], np.arange(148, 164))
    assert np.array_equal(batch_y, np.arange(248, 264))


def test_adapt_list_of_np_arrays_as_y() -> None:
    adapted = keras._adapt_data_from_fit_args(
        x=np.arange(0, 100),
        y=[np.arange(100, 200), np.arange(200, 300)],
        sample_weight=None,
        batch_size=16,
    )
    assert isinstance(adapted, Sequence)
    assert len(adapted) == 7
    batch_x, batch_y = adapted[3]
    assert np.array_equal(batch_x, np.arange(48, 64))
    assert np.array_equal(batch_y[0], np.arange(148, 164))
    assert np.array_equal(batch_y[1], np.arange(248, 264))


def test_adapt_dict_of_np_arrays_as_x() -> None:
    adapted = keras._adapt_data_from_fit_args(
        x={"k1": np.arange(0, 100), "k2": np.arange(100, 200)},
        y=np.arange(200, 300),
        sample_weight=None,
        batch_size=16,
    )
    assert isinstance(adapted, Sequence)
    assert len(adapted) == 7
    batch_x, batch_y = adapted[3]
    assert np.array_equal(batch_x["k1"], np.arange(48, 64))
    assert np.array_equal(batch_x["k2"], np.arange(148, 164))
    assert np.array_equal(batch_y, np.arange(248, 264))


def test_adapt_short_sequence() -> None:
    sequence = IdentitySequence(3)
    with pytest.raises(check.CheckFailedError):
        _ = keras._DeterminedSequenceWrapper(
            sequence=sequence,
            shard_rank=0,
            shard_size=4,
            training=False,
        )


@pytest.mark.parametrize("skip_batches", [0, 50, 350])
@pytest.mark.parametrize("rank_size", [(0, 1), (0, 3), (1, 3), (2, 3)])
@pytest.mark.parametrize("shuffle", [False, True])
def test_sequence_wrapper_training(shuffle, rank_size, skip_batches) -> None:
    num_epochs = 3
    epoch_len = 100
    shuffle_seed = 777
    rank, size = rank_size

    # Calculate the indices we expect to see.
    epoch_indices = list(range(epoch_len))
    if shuffle:
        rng = np.random.RandomState(shuffle_seed)
        expect = []
        for _ in range(num_epochs * 10):
            rng.shuffle(epoch_indices)
            expect += [*epoch_indices]
    else:
        expect = epoch_indices * (num_epochs * 10)

    wrapped_seq = keras._DeterminedSequenceWrapper(
        sequence=IdentitySequence(epoch_len),
        shard_rank=rank,
        shard_size=size,
        training=True,
        shuffle=shuffle,
        shuffle_seed=shuffle_seed,
        prior_batches_trained=skip_batches,
    )
    # Simulate the behavior of the OrderedEnqueuer for a few epochs of data.
    got = []
    for _ in range(num_epochs):
        for i in range(len(wrapped_seq)):
            got.append(wrapped_seq[i])
        wrapped_seq.on_epoch_end()

    shard_expect = expect[rank::size]
    skipped_shard_expect = shard_expect[skip_batches:]

    assert got == skipped_shard_expect[: len(got)]


@pytest.mark.parametrize("rank_size", [(0, 1), (0, 3), (1, 3), (2, 3)])
def test_sequence_wrapper_validation(rank_size) -> None:
    epoch_len = 100
    rank, size = rank_size

    shard_expect = list(range(rank, epoch_len, size))

    wrapped_seq = keras._DeterminedSequenceWrapper(
        sequence=IdentitySequence(epoch_len),
        shard_rank=rank,
        shard_size=size,
        training=False,
    )

    # Simulate the behavior of the OrderedEnqueuer for one epoch of data.
    got = [wrapped_seq[i] for i in range(len(wrapped_seq))]
    assert got == shard_expect

    # Confirm that on_epoch_end is ignored for validation datasets.
    wrapped_seq.on_epoch_end()
    got = [wrapped_seq[i] for i in range(len(wrapped_seq))]
    assert got == shard_expect, "on_epoch_end() should not affect a non-training-mode sequence"
