import logging
import multiprocessing
import sys
import typing
from logging import handlers

import numpy as np
import pytest
import torch

import determined as det
from determined.pytorch import (
    DistributedBatchSampler,
    RepeatBatchSampler,
    SkipBatchSampler,
    data_length,
    to_device,
)


def make_dataset() -> torch.utils.data.Dataset:
    training_data = np.array([[0, 0], [0, 1], [1, 0], [1, 1]], dtype=np.float32)
    training_data = torch.Tensor(training_data)
    training_labels = np.array([0, 1, 1, 0], dtype=np.float32)
    training_labels = torch.Tensor(training_labels)
    return torch.utils.data.TensorDataset(training_data, training_labels)


def make_data_loader() -> torch.utils.data.DataLoader:
    return torch.utils.data.DataLoader(make_dataset(), batch_size=1)


def test_skip_batch_sampler():
    skip = 2
    sampler = torch.utils.data.BatchSampler(
        torch.utils.data.SequentialSampler(range(15)), batch_size=2, drop_last=False
    )
    skip_sampler = SkipBatchSampler(sampler, skip)

    assert len(skip_sampler) == 6

    iterator = iter(sampler)

    # Advance the iterator by skip batches.
    for _ in range(skip):
        next(iterator)

    for samp, skip_samp in zip(iterator, iter(skip_sampler)):
        assert samp == skip_samp

    # Confirm that same_length works.
    same_size_skip_sampler = SkipBatchSampler(sampler, skip, same_length=True)
    assert len(same_size_skip_sampler) == len(sampler)


def test_repeat_batch_sampler():
    sampler = torch.utils.data.BatchSampler(torch.utils.data.SequentialSampler(range(10)), 3, False)
    repeat_sampler = RepeatBatchSampler(sampler)

    assert len(repeat_sampler) == 4

    one_pass = list(sampler)

    iterator = iter(repeat_sampler)
    for _ in range(3):
        assert one_pass == [next(iterator) for _ in range(len(one_pass))]


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
        dist_sampler = DistributedBatchSampler(sampler, 4, rank)
        samples = list(dist_sampler)
        assert len(dist_sampler) == len(samples)
        assert samples == expected_samples[rank]


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
@pytest.mark.skipif(
    sys.platform == "darwin", reason="Test relies on feature unimplemented on Mac OS X"
)
# Not implemented feature:
# https://stackoverflow.com/questions/65609529/python-multiprocessing-queue-notimplementederror-macos
def test_to_device_warnings(dedup_between_calls) -> None:
    queue = multiprocessing.Queue()

    logger = logging.getLogger()
    # Capture warning logs as elements in a queue.
    logger.addHandler(handlers.QueueHandler(queue))

    warned_types = set() if dedup_between_calls else None
    to_device(["string_data", "string_data"], "cpu", warned_types)
    to_device(["string_data", "string_data"], "cpu", warned_types)

    assert queue.qsize() == 1 if dedup_between_calls else 2
    while queue.qsize():
        msg = queue.get().message
        assert "not able to move data" in msg
