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
        with keras._build_enqueuer(
            sequence=sequence,
            workers=0,
            use_multiprocessing=False,
            max_queue_size=10,
            shard_rank=0,
            num_shards=4,
            repeat=False,
            shuffle=False,
            shuffle_seed=0,
            prior_batches_trained=0,
        ):
            pass


@pytest.mark.parametrize("rank_size", [(0, 1), (0, 3), (1, 3), (2, 3)])
@pytest.mark.parametrize("skip", [0, 50, 350])
@pytest.mark.parametrize("shuffle", [False, True])
def test_sampler(shuffle, skip, rank_size):
    epoch_len = 100
    rank, size = rank_size
    seed = 777

    # Build a list of globally expected indices; just a stream of indices, shuffled every epoch.
    rng = np.random.RandomState(seed)
    all_indices = []
    one_epoch_indices = list(range(epoch_len))
    for _ in range(15):
        if shuffle:
            rng.shuffle(one_epoch_indices)
        all_indices += one_epoch_indices

    # Expect the appropriate shard of the stream for ourselves.
    expect_shard = [all_indices[i] for i in range(rank, len(all_indices), size)]

    # Respect the number of batches we have already trained for.
    expect_indices = expect_shard[skip:]

    sampler = keras._Sampler(epoch_len, rank, size, shuffle, seed, skip)

    got_indices = list(sampler.yield_epoch())
    got_indices += list(sampler.yield_epoch())
    got_indices += list(sampler.yield_epoch())

    # Ensure we got an appropriate length of indices.
    shard_len = epoch_len // size
    exp_len = 3 * shard_len - (skip % epoch_len) % shard_len
    got_len = len(got_indices)
    assert abs(exp_len - got_len) <= 3

    assert got_indices == expect_indices[: len(got_indices)]


@pytest.mark.parametrize("use_multiprocessing", [False, True])
@pytest.mark.parametrize("workers", [0, 1, 5])
@pytest.mark.parametrize("rank_size", [(0, 1), (0, 3), (1, 3), (2, 3)])
@pytest.mark.parametrize("skip", [0, 50, 350])
@pytest.mark.parametrize("shuffle", [False, True])
def test_enqueuer(shuffle, skip, rank_size, workers, use_multiprocessing) -> None:
    epoch_len = 100
    rank, size = rank_size

    # Ensure that the enqueuer reliably returns what the sampler is returning.
    sampler = keras._Sampler(epoch_len, rank, size, shuffle, 777, skip)

    with keras._build_enqueuer(
        sequence=IdentitySequence(100),
        workers=workers,
        use_multiprocessing=use_multiprocessing,
        max_queue_size=10,
        shard_rank=rank,
        num_shards=size,
        repeat=False,
        shuffle=shuffle,
        shuffle_seed=777,
        prior_batches_trained=skip,
    ) as enqueuer:
        assert list(enqueuer.data()) == list(sampler.yield_epoch()), "first epoch was wrong"
        assert list(enqueuer.data()) == list(sampler.yield_epoch()), "second epoch was wrong"
        assert list(enqueuer.data()) == list(sampler.yield_epoch()), "third epoch was wrong"
