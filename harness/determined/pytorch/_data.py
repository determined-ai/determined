import logging
from typing import (
    Any,
    Callable,
    Dict,
    Generator,
    Iterator,
    List,
    Optional,
    Sequence,
    TypeVar,
    Union,
    cast,
)

import numpy as np
import torch

# from torch.utils.data.dataloader import _InfiniteConstantSampler
from determined_common.check import check_gt, check_lt

# TODO(DET-1524): Uncomment inports.
from torch.utils.data import (  # _DatasetKind,; IterableDataset,
    BatchSampler,
    Dataset,
    RandomSampler,
    Sampler,
    SequentialSampler,
    _utils,
)


_Array = Union[np.ndarray, torch.Tensor]
_Data = Union[Dict[str, _Array], Sequence[_Array], _Array]
TorchData = Union[Dict[str, torch.Tensor], Sequence[torch.Tensor], torch.Tensor]

_worker_init_fn_t = Optional[Callable[[int], None]]
T = TypeVar("T")
_collate_fn_t = Optional[Callable[[List[T]], Any]]


class DataLoader:
    """
    DataLoader is meant to contain a user's `Dataset`, configuration for
    sampling data in batches, and performance configuration like
    multiprocessing.

    The __init__ function determines the defaults in the same way as a
    `torch.utils.data.DataLoader` would, so the behavior should be familiar.
    However, the `torch.utils.data.Dataloader` that is used for training and
    validation is not created until `get_data_loader(...)` is called. This is
    done so that Determined can ensure that sampling restarts from the right location
    and distributed sampling is handled correctly.

    <ARGUMENTS DOCUMENTATION FROM PYTORCH>
    Arguments:
        dataset (Dataset): dataset from which to load the data.
        batch_size (int, optional): how many samples per batch to load
            (default: ``1``).
        shuffle (bool, optional): set to ``True`` to have the data reshuffled
            at every epoch (default: ``False``).
        sampler (Sampler, optional): defines the strategy to draw samples from
            the dataset. If specified, :attr:`shuffle` must be ``False``.
        batch_sampler (Sampler, optional): like :attr:`sampler`, but returns a batch of
            indices at a time. Mutually exclusive with :attr:`batch_size`,
            :attr:`shuffle`, :attr:`sampler`, and :attr:`drop_last`.
        num_workers (int, optional): how many subprocesses to use for data
            loading. ``0`` means that the data will be loaded in the main process.
            (default: ``0``)
        collate_fn (callable, optional): merges a list of samples to form a
            mini-batch of Tensor(s).  Used when using batched loading from a
            map-style dataset.
        pin_memory (bool, optional): If ``True``, the data loader will copy Tensors
            into CUDA pinned memory before returning them.  If your data elements
            are a custom type, or your :attr:`collate_fn` returns a batch that is a custom type,
            see the example below.
        drop_last (bool, optional): set to ``True`` to drop the last incomplete batch,
            if the dataset size is not divisible by the batch size. If ``False`` and
            the size of dataset is not divisible by the batch size, then the last batch
            will be smaller. (default: ``False``)
        timeout (numeric, optional): if positive, the timeout value for collecting a batch
            from workers. Should always be non-negative. (default: ``0``)
        worker_init_fn (callable, optional): If not ``None``, this will be called on each
            worker subprocess with the worker id (an int in ``[0, num_workers - 1]``) as
            input, after seeding and before data loading. (default: ``None``)
    """

    def __init__(
        self,
        dataset: Dataset,
        batch_size: Optional[int] = 1,
        shuffle: bool = False,
        sampler: Optional[Sampler] = None,
        batch_sampler: Optional[BatchSampler] = None,
        num_workers: int = 0,
        collate_fn: _collate_fn_t = None,
        pin_memory: bool = False,
        drop_last: bool = False,
        timeout: float = 0,
        worker_init_fn: _worker_init_fn_t = None,
    ):

        # BEGIN VENDORED CODE FROM PYTORCH
        # https://github.com/pytorch/pytorch/blob/v1.3.1/torch/utils/data/dataloader.py#L120
        if num_workers < 0:
            raise ValueError(
                "num_workers option should be non-negative; "
                "use num_workers=0 to disable multiprocessing."
            )

        if timeout < 0:
            raise ValueError("timeout option should be non-negative")

        self.dataset = dataset
        self.num_workers = num_workers
        self.pin_memory = pin_memory
        self.timeout = timeout
        self.worker_init_fn = worker_init_fn

        # TODO(DET-1524): uncomment this as we do not currently support IterableDataset
        # if isinstance(dataset, IterableDataset):
        #     raise AssertionError("`IterableDataset`s are not currently supported.")
        # else:
        #     self._dataset_kind = _DatasetKind.Map

        if sampler is not None and shuffle:
            raise ValueError("sampler option is mutually exclusive with " "shuffle")

        if batch_sampler is not None:
            # auto_collation with custom batch_sampler
            if batch_size != 1 or shuffle or sampler is not None or drop_last:
                raise ValueError(
                    "batch_sampler option is mutually exclusive "
                    "with batch_size, shuffle, sampler, and "
                    "drop_last"
                )
            batch_size = None
            drop_last = False
        elif batch_size is None:
            # no auto_collation
            if shuffle or drop_last:
                raise ValueError(
                    "batch_size=None option disables auto-batching "
                    "and is mutually exclusive with "
                    "shuffle, and drop_last"
                )

        if sampler is None:  # give default samplers
            # TODO(DET-1524): uncomment this logic and delete the one after it.
            # if self._dataset_kind == _DatasetKind.Iterable:
            #    # See NOTE [ Custom Samplers and IterableDataset ]
            #    sampler = _InfiniteConstantSampler()
            # else:  # map-style
            #    if shuffle:
            #        sampler = RandomSampler(dataset)
            #    else:
            #        sampler = SequentialSampler(dataset)
            if shuffle:
                sampler = RandomSampler(dataset)
            else:
                sampler = SequentialSampler(dataset)

        if batch_size is not None and batch_sampler is None:
            # auto_collation without custom batch_sampler
            batch_sampler = BatchSampler(sampler, batch_size, drop_last)

        self.batch_size = batch_size
        self.drop_last = drop_last
        self.sampler = sampler
        self.batch_sampler = batch_sampler

        if collate_fn is None:
            if self._auto_collation:
                collate_fn = _utils.collate.default_collate
            else:
                collate_fn = _utils.collate.default_convert

        self.collate_fn = collate_fn
        # END VENDORED CODE FROM PYTORCH

    # BEGIN VENDORED CODE FROM PYTORCH
    # https://github.com/pytorch/pytorch/blob/v1.3.1/torch/utils/data/dataloader.py#L280
    @property
    def _auto_collation(self) -> bool:
        return self.batch_sampler is not None

    # END VENDORED CODE FROM PYTORCH

    def get_data_loader(
        self, repeat: bool = False, skip: int = 0, num_replicas: int = 1, rank: int = 0
    ) -> torch.utils.data.DataLoader:
        batch_sampler = cast(BatchSampler, self.batch_sampler)
        batch_sampler = adapt_batch_sampler(
            batch_sampler, repeat=repeat, skip=skip, num_replicas=num_replicas, rank=rank
        )
        return torch.utils.data.DataLoader(
            self.dataset,
            batch_sampler=batch_sampler,
            num_workers=self.num_workers,
            collate_fn=self.collate_fn,
            pin_memory=self.pin_memory,
            timeout=self.timeout,
            worker_init_fn=self.worker_init_fn,  # type: ignore
        )

    def __iter__(self) -> Iterator:
        """Compatibiliy with the real DataLoader when using a PyTorchTrial outside of Determined."""
        return iter(self.get_data_loader())

    def __len__(self) -> int:
        """Compatibiliy with the real DataLoader when using a PyTorchTrial outside of Determined."""
        # TODO(DET-1524): uncomment this as we do not currently support IterableDataset
        # if isinstance(dataset, IterableDataset):
        #     return len(self.dataset)
        batch_sampler = cast(BatchSampler, self.batch_sampler)
        return len(batch_sampler)


def adapt_batch_sampler(
    batch_sampler: torch.utils.data.BatchSampler,
    repeat: bool = False,
    skip: int = 0,
    num_replicas: int = 1,
    rank: int = 0,
) -> torch.utils.data.BatchSampler:
    """
    Modify the underlying BatchSampler of a constructed DataLoader to account
    for repeating on training datasets, skipping when continuing training, and
    sharding for distributed training.
    """
    if repeat:
        batch_sampler = RepeatBatchSampler(batch_sampler)

    if num_replicas > 1:
        batch_sampler = DistributedBatchSampler(batch_sampler, num_replicas, rank)

    # SkipBatchSampler is used when we are continuing training. SkipBatchSampler must be applied
    # after DistributedBatchSampler, since the number of batches to skip is based on how many
    # global batches should be skipped, which corresponds with how many batches are emitted by the
    # DistributedBatchSampler, not the initial BatchSampler.
    if skip > 0:
        batch_sampler = SkipBatchSampler(batch_sampler, skip, same_length=repeat)

    return batch_sampler


class RepeatBatchSampler(torch.utils.data.BatchSampler):
    """
    RepeatBatchSampler yields infinite batches indices by repeatedly iterating
    through the batches of another BatchSampler. __len__ is just the length of
    the underlying BatchSampler.
    """

    def __init__(self, batch_sampler: torch.utils.data.BatchSampler) -> None:
        self.batch_sampler = batch_sampler

    def __len__(self) -> int:
        return len(self.batch_sampler)

    def __iter__(self) -> Generator:
        while True:
            yield from self.batch_sampler


class DistributedBatchSampler(torch.utils.data.BatchSampler):
    """
    DistributedBatchSampler is meant to wrap any BatchSampler to pass every nth
    batch to a worker, using the worker's rank as the initial offset.

    DistributedBatchSampler is different than the PyTorch built-in
    torch.utils.data.distributed.DistributedSampler, because that
    DistributedSampler expects to bbe called before the BatchSampler, and
    additionally the DistributedSampler is meant to be a stand-alone sampler.

    DistributedBatchSampler has the potential gotcha that when wrapping a
    non-repeating BatchSampler, if the length of the BatchSampler is not
    divisible by the number of replicas the length of the resulting
    DistributedBatchSampler will differ based on the rank. In that case, the
    divergent paths of multiple workers could cause problems during training.
    PyTorchTrial always uses RepeatBatchSampler during training, PyTorchTrial
    does not require that the workers stay in-step during validation, so this
    potential gotcha is not a problem in Determined.
    """

    def __init__(
        self, batch_sampler: torch.utils.data.BatchSampler, num_replicas: int, rank: int
    ) -> None:
        check_gt(rank, -1, "rank must be non-negative")
        check_gt(num_replicas, 0, "num_replicas must be positive")
        check_lt(rank, num_replicas, "rank must be less than num_replicas")

        self.batch_sampler = batch_sampler
        self.num_replicas = num_replicas
        self.rank = rank

    def __len__(self) -> int:
        full_global_batches = len(self.batch_sampler) // self.num_replicas
        worker_gets_partial_batch = int(len(self.batch_sampler) % self.num_replicas > self.rank)
        return full_global_batches + worker_gets_partial_batch

    def __iter__(self) -> Generator:
        if self.num_replicas == 1:
            yield from self.batch_sampler
        else:
            for i, batch in enumerate(self.batch_sampler):
                if i % self.num_replicas == self.rank:
                    yield batch


class SkipBatchSampler(torch.utils.data.BatchSampler):
    """
    SkipBatchSampler skips some batches from an underlying BatchSampler, and
    yield the rest. By default, the length of the new BatchSampler is reported
    to be the length of the base BatchSampler minus the amount skipped.

    In some cases, such as if the base BatchSampler is known to contain a
    RepeatBatchSampler, it makes more sense to report the full length of the
    base BatchSampler. This behavior is controlled using the same_length
    parameter.
    """

    def __init__(
        self, batch_sampler: torch.utils.data.BatchSampler, skip: int, same_length: bool = False
    ) -> None:
        self.batch_sampler = batch_sampler
        self.skip = skip
        self.length = len(batch_sampler)
        if not same_length:
            self.length -= self.skip

    def __len__(self) -> int:
        return self.length

    def __iter__(self) -> Generator:
        iterator = iter(self.batch_sampler)
        for _ in range(self.skip):
            try:
                next(iterator)
            except StopIteration:
                return
        yield from iterator


def data_length(data: _Data) -> int:
    """
    Calculate length of data input.

    Accepts np.ndarray, torch.tensor, dictionary, or list. Recursively traverses the tree to find
    the first np.ndarray or torch.Tensor and calculates the length of it. Assumes that every "leaf"
    of the data structure is batched to the same length.
    """
    if isinstance(data, (np.ndarray, torch.Tensor)):
        return len(data)
    if isinstance(data, dict):
        if len(data) == 0:
            raise ValueError(
                "`PyTorchTrial` must have at least one `np.ndarray` or `torch.Tensor` in it's dict "
                "of inputs."
            )
        return len(next(iter(data.values())))
    if isinstance(data, list):
        if len(data) == 0:
            raise ValueError(
                "`PyTorchTrial` must have at least one `np.ndarray` or `torch.Tensor` in it's list "
                "of inputs."
            )
        return len(data[0])
    if isinstance(data, tuple):
        if len(data) == 0:
            raise ValueError(
                "`PyTorchTrial` must have at least one `np.ndarray` or `torch.Tensor` in it's tuple"
                " of inputs."
            )
        return len(data[0])
    raise TypeError("Data of incorrect type: {}".format(type(data)))


def to_device(data: _Data, device: torch.device, log_warning: bool = False) -> TorchData:
    """
    Accept np.ndarray, torch.Tensor, list, or dictionary. Recursively convert
    any ndarrays to tensors and call .to() on any tensors or data types that
    have custom serialization logic defined via a callable to() attribute.

    If the data cannot be moved to device, optionally log a warning and return
    the original data.
    """
    if isinstance(data, dict):
        return {name: to_device(d, device, log_warning) for name, d in data.items()}  # type: ignore
    if isinstance(data, list):
        return [to_device(d, device, log_warning) for d in data]  # type: ignore
    if isinstance(data, tuple):
        return tuple(to_device(d, device, log_warning) for d in data)  # type: ignore
    if isinstance(data, np.ndarray):
        return torch.from_numpy(data).to(device)
    if hasattr(data, "to") and callable(data.to):  # type: ignore
        return data.to(device)  # type: ignore

    if log_warning:
        logging.warning(f"Was not able to move data item of type '{type(data)}' to device.")
        log_warning = False

    return data
