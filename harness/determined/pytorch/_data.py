import logging
from typing import (
    Any,
    Callable,
    Dict,
    Iterator,
    List,
    Optional,
    Sequence,
    Set,
    Type,
    TypeVar,
    Union,
    cast,
)

import numpy as np
import torch
from torch.utils.data import (
    BatchSampler,
    Dataset,
    RandomSampler,
    Sampler,
    SequentialSampler,
    _utils,
)

from determined.pytorch import samplers

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
        # Don't allow IterableDatasets in our DataLoader.
        if isinstance(dataset, torch.utils.data.IterableDataset):
            raise ValueError(
                "IterableDatasets are not supported through det.pytorch.DataLoader().  Read about "
                "the difference between IterableDatasets and MapDatasets at: "
                "pytorch.org/docs/stable/data. "
                "You can use an IterableDataset in Determined, but you will be responsible for "
                "shuffling, sharding, repeating, and skipping to ensure reproducibility and "
                "correctness of training.  See the in-depth guide at: "
                "docs.determined.ai/latest/reference/api/pytorch-samplers.html"
            )

        # Don't allow this rare combination of inputs that we would puke on later.
        if batch_sampler is None and batch_size is None:
            raise ValueError(
                "The case of batch_sampler=None and batch_size=None is not supported through "
                "det.pytorch.DataLoader(). For customizing your data loader beyond what is "
                "supported by det.pytorch.DataLoader, see the in-depth guide at: "
                "docs.determined.ai/latest/reference/api/pytorch-samplers.html"
            )

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
            if shuffle:
                sampler = RandomSampler(dataset)  # type: ignore
            else:
                sampler = SequentialSampler(dataset)  # type: ignore

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
            worker_init_fn=self.worker_init_fn,
        )

    def __iter__(self) -> Iterator:
        """Compatibiliy with the real DataLoader when using a PyTorchTrial outside of Determined."""
        return iter(self.get_data_loader())

    def __len__(self) -> int:
        """Compatibiliy with the real DataLoader when using a PyTorchTrial outside of Determined."""
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
        batch_sampler = samplers.RepeatBatchSampler(batch_sampler)

    if num_replicas > 1:
        batch_sampler = samplers.DistributedBatchSampler(batch_sampler, num_replicas, rank)

    # SkipBatchSampler is used when we are continuing training. SkipBatchSampler must be applied
    # after DistributedBatchSampler, since the number of batches to skip is based on how many
    # global batches should be skipped, which corresponds with how many batches are emitted by the
    # DistributedBatchSampler, not the initial BatchSampler.
    if skip > 0:
        batch_sampler = samplers.SkipBatchSampler(batch_sampler, skip)

    return batch_sampler


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


def to_device(
    data: _Data, device: torch.device, warned_types: Optional[Set[Type]] = None
) -> TorchData:
    """
    Accept np.ndarray, torch.Tensor, list, or dictionary. Recursively convert any ndarrays to
    tensors and call .to() on any tensors or data types that have custom serialization logic
    defined via a callable to() attribute.

    If the data cannot be moved to device, log a warning (only once per type) and return the
    original data.
    """
    # Never print errors recursively.
    if warned_types is None:
        warned_types = set()

    if isinstance(data, dict):
        return {k: to_device(v, device, warned_types) for k, v in data.items()}  # type: ignore
    elif isinstance(data, list):
        return [to_device(d, device, warned_types) for d in data]  # type: ignore
    elif isinstance(data, tuple):
        return tuple(to_device(d, device, warned_types) for d in data)  # type: ignore
    elif isinstance(data, np.ndarray):
        # Torch supports floats, complex floats, ints, uints, and bools as tensors.
        # Those correspond to numpy dtype kinds: "f", "c", "i", "u", and "b", respectively.
        # Do not attempt to convert any other kinds to tensors.
        if data.dtype.kind in "fciub":
            return torch.from_numpy(data).to(device)
    elif hasattr(data, "to") and callable(data.to):  # type: ignore
        return data.to(device)  # type: ignore

    if type(data) not in warned_types:
        warned_types.add(type(data))
        logging.warning(
            f"Was not able to move data item of type '{type(data).__name__}' to device."
        )

    return data  # type:ignore
