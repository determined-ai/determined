import logging
import queue
import typing
from logging import handlers

import numpy as np
import pytest
import torch

import determined as det
from determined.pytorch import data_length, samplers, to_device


def make_dataset() -> torch.utils.data.Dataset:
    training_data = np.array([[0, 0], [0, 1], [1, 0], [1, 1]], dtype=np.float32)
    training_data = torch.Tensor(training_data)
    training_labels = np.array([0, 1, 1, 0], dtype=np.float32)
    training_labels = torch.Tensor(training_labels)
    return torch.utils.data.TensorDataset(training_data, training_labels)


def make_data_loader() -> torch.utils.data.DataLoader:
    return torch.utils.data.DataLoader(make_dataset(), batch_size=1)


def test_skip_sampler():
    skip = 2
    sampler = torch.utils.data.SequentialSampler(range(15))
    skip_sampler = samplers.SkipSampler(sampler, skip)

    assert len(skip_sampler) == 15

    for samp, skip_samp in zip(range(skip, 15), iter(skip_sampler)):
        assert samp == skip_samp


def test_skip_batch_sampler():
    skip = 2
    sampler = torch.utils.data.SequentialSampler(range(15))
    batch_sampler = torch.utils.data.BatchSampler(sampler, batch_size=2, drop_last=False)
    skip_batch_sampler = samplers.SkipBatchSampler(batch_sampler, skip)

    assert len(skip_batch_sampler) == 8

    iterator = iter(batch_sampler)

    # Advance the iterator by skip batches.
    for _ in range(skip):
        next(iterator)

    for samp, skip_samp in zip(iterator, iter(skip_batch_sampler)):
        assert samp == skip_samp


def test_repeat_sampler():
    sampler = torch.utils.data.SequentialSampler(range(10))
    repeat_sampler = samplers.RepeatSampler(sampler)

    assert len(repeat_sampler) == 10

    one_pass = list(sampler)

    iterator = iter(repeat_sampler)
    for _ in range(3):
        assert one_pass == [next(iterator) for _ in range(len(one_pass))]


def test_repeat_batch_sampler():
    sampler = torch.utils.data.SequentialSampler(range(10))
    batch_sampler = torch.utils.data.BatchSampler(sampler, 3, False)
    repeat_batch_sampler = samplers.RepeatBatchSampler(batch_sampler)

    assert len(repeat_batch_sampler) == 4

    one_pass = list(batch_sampler)

    iterator = iter(repeat_batch_sampler)
    for _ in range(3):
        assert one_pass == [next(iterator) for _ in range(len(one_pass))]


def test_distributed_sampler():
    sampler = torch.utils.data.SequentialSampler(range(19))

    num_replicas = 4

    expected_samples = []
    expected_samples.append([0, 4, 8, 12, 16])
    expected_samples.append([1, 5, 9, 13, 17])
    expected_samples.append([2, 6, 10, 14, 18])
    expected_samples.append([3, 7, 11, 15])

    for rank in range(num_replicas):
        dist_sampler = samplers.DistributedSampler(sampler, 4, rank)
        samples = list(dist_sampler)
        assert len(dist_sampler) == len(samples)
        assert samples == expected_samples[rank]


def test_distributed_batch_sampler():
    worker_batch_size = 2
    sampler = torch.utils.data.SequentialSampler(range(19))
    sampler = torch.utils.data.BatchSampler(sampler, batch_size=worker_batch_size, drop_last=False)

    num_replicas = 4

    expected_samples = []
    expected_samples.append([[0, 1], [8, 9], [16, 17]])
    expected_samples.append([[2, 3], [10, 11], [18]])
    expected_samples.append([[4, 5], [12, 13]])
    expected_samples.append([[6, 7], [14, 15]])

    for rank in range(num_replicas):
        dist_sampler = samplers.DistributedBatchSampler(sampler, 4, rank)
        samples = list(dist_sampler)
        assert len(dist_sampler) == len(samples)
        assert samples == expected_samples[rank]


def test_reproducible_shuffle_sampler():
    sampler = torch.utils.data.SequentialSampler(range(5))
    sampler = samplers.ReproducibleShuffleSampler(sampler, 777)

    assert list(sampler) == [0, 4, 1, 2, 3]
    assert list(sampler) == [2, 0, 1, 3, 4]


def test_reproducible_shuffle_batch_sampler():
    sampler = torch.utils.data.SequentialSampler(range(10))
    batch_sampler = torch.utils.data.BatchSampler(sampler, batch_size=2, drop_last=False)
    shuffle_batch_sampler = samplers.ReproducibleShuffleSampler(batch_sampler, 777)

    assert list(shuffle_batch_sampler) == [[0, 1], [8, 9], [2, 3], [4, 5], [6, 7]]
    assert list(shuffle_batch_sampler) == [[4, 5], [0, 1], [2, 3], [6, 7], [8, 9]]


def test_pytorch_adapt_batch_sampler():
    def check_equality(batch0, batch1):
        for a, b in zip(batch0, batch1):
            assert torch.eq(a, b)

    offset = 2

    dataloader = det.pytorch.DataLoader(make_dataset())
    data_adapter = dataloader.get_data_loader(repeat=True, skip=offset)

    data = make_data_loader()
    iterator = iter(data)
    inf_iterator = iter(data_adapter)

    # Advance the iterator by offset batches.
    for _ in range(offset):
        next(iterator)

    # Verify indefinite generator with offset.

    for batch in iterator:
        n = next(inf_iterator)
        for pair in zip(batch, n):
            assert torch.all(torch.eq(pair[0], pair[1]))

    for _ in range(3):
        for batch in iter(data):
            n = next(inf_iterator)
            for pair in zip(batch, n):
                assert torch.all(torch.eq(pair[0], pair[1]))


def test_pytorch_batch_sampler_mutual_exclusion():
    dataloader = det.pytorch.DataLoader(make_dataset(), drop_last=True, shuffle=True, batch_size=2)
    assert dataloader.get_data_loader() is not None


def make_input(inp: typing.List) -> typing.Iterator[typing.Tuple]:
    return zip(inp, [length for _ in range(len(inp))])


n = np.array([0])
t = torch.Tensor(n)
length = 1

lists = [[n], [t], [[n]]]
tuples = [(n,), (t,), ({"a": t})]
dicts = [{"a": n}, {"a": t}, {"a": [[t]]}]

TEST_DATA_LENGTH_SUITE = [*make_input(lists), *make_input(tuples), *make_input(dicts)]

Array = typing.Union[np.ndarray, torch.Tensor]
Data = typing.Union[typing.Dict[str, Array], Array]


@pytest.mark.parametrize("data,length", TEST_DATA_LENGTH_SUITE)
def test_data_length(data: Data, length: int):
    assert data_length(data) == length


@pytest.mark.parametrize("data,error", [({}, ValueError), (0, TypeError)])
def test_data_type_error(data: typing.Any, error: typing.Any) -> None:
    with pytest.raises(error):
        data_length(data)


def test_to_device() -> None:
    """
    There doesn't seem to be an easy way to mock out PyTorch devices, so ignore
    testing that the data makes it *on* to the device.
    """
    data_structure = {
        "input_1": torch.Tensor(1),
        "input_3": "str",
        "input_4": 1,
    }

    assert to_device(data_structure, "cpu") == data_structure
    assert np.array_equal(to_device(np.array([0, 1, 2]), "cpu"), np.array([0, 1, 2]))


@pytest.mark.parametrize("dedup_between_calls", [True, False])
def test_to_device_warnings(dedup_between_calls) -> None:
    # Capture warning logs as elements in a queue.
    logger = logging.getLogger()
    q = queue.Queue()
    handler = handlers.QueueHandler(q)
    logger.addHandler(handler)
    try:
        warned_types = set() if dedup_between_calls else None
        to_device(["string_data", "string_data"], "cpu", warned_types)
        to_device(["string_data", "string_data"], "cpu", warned_types)

        assert q.qsize() == 1 if dedup_between_calls else 2
        while q.qsize():
            msg = q.get().message
            assert "not able to move data" in msg
    finally:
        # Restore logging as it was before.
        logger.removeHandler(handler)
