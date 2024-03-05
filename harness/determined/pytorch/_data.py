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
from packaging import version
from torch.utils import data
from torch.utils.data._utils import collate

from determined.pytorch import samplers

logger = logging.getLogger("determined.pytorch")

_Array = Union[np.ndarray, torch.Tensor]
_Data = Union[Dict[str, _Array], Sequence[_Array], _Array]
TorchData = Union[Dict[str, torch.Tensor], Sequence[torch.Tensor], torch.Tensor]

_worker_init_fn_t = Optional[Callable[[int], None]]
T = TypeVar("T")
_collate_fn_t = Optional[Callable[[List[T]], Any]]


def _dataset_repro_warning(fn: str, data_obj: Any, is_deepspeed_trial: bool = False) -> str:
    disable_repro_check_method = "context.experimental.disable_dataset_reproducibility_checks()"
    if is_deepspeed_trial:
        disable_repro_check_method = "context.disable_dataset_reproducibility_checks()"

    return (
        f"{fn}() returned an instance of {type(data_obj).__name__}, which is not a "
        "subclass of det.pytorch.DataLoader.  For most non-Iterable DataSets, "
        "det.pytorch.DataLoader is a drop-in replacement for torch.utils.data.DataLoader "
        "but which offers easy and transparent reproducibility in Determined experiments. "
        "It is highly recommended that you use det.pytorch.DataLoader if possible.  If "
        f"not, you can disable this check by calling {disable_repro_check_method} at some point "
        "in your trial's __init__() method."
    )


class DataLoader:
    """
    DataLoader is meant to contain a user's ``data.Dataset``, configuration for
    sampling data in batches, and performance configuration like
    multiprocessing.

    The __init__ function determines the defaults in the same way as a
    ``torch.utils.data.DataLoader`` would, so the behavior should be familiar.
    However, the ``torch.utils.data.Dataloader`` that is used for training and
    validation is not created until ``get_data_loader(...)`` is called. This is
    done so that Determined can ensure that sampling restarts from the right location
    and distributed sampling is handled correctly.

    Note that the arguments are from PyTorch.

    Arguments:
        dataset (Dataset): dataset from which to load the data.
        batch_size (int, optional): how many samples per batch to load (default: ``1``).
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
        generator (torch.Generator, optional): If not ``None``, this RNG will be used
            by RandomSampler to generate random indexes and multiprocessing to generate
            ``base_seed`` for workers. (default: ``None``)
        prefetch_factor (int, optional, keyword-only arg): Number of samples loaded
            in advance by each worker. ``2`` means there will be a total of
            2 * num_workers samples prefetched across all workers. (default: ``2``)
        persistent_workers (bool, optional): If ``True``, the data loader will not shut down
            the worker processes after a dataset has been consumed once. This allows to
            maintain the workers ``Dataset`` instances alive. (default: ``False``)
    """

    def __init__(
        self,
        dataset: data.Dataset,
        batch_size: Optional[int] = 1,
        shuffle: bool = False,
        sampler: Optional[data.Sampler] = None,
        batch_sampler: Optional[data.BatchSampler] = None,
        num_workers: int = 0,
        collate_fn: _collate_fn_t = None,
        pin_memory: bool = False,
        drop_last: bool = False,
        timeout: float = 0,
        worker_init_fn: _worker_init_fn_t = None,
        multiprocessing_context: Any = None,
        generator: Any = None,
        *,
        prefetch_factor: Optional[int] = None,
        persistent_workers: bool = False,
    ):
        # Don't allow IterableDatasets in our DataLoader.
        if isinstance(dataset, data.IterableDataset):
            raise ValueError(
                "IterableDatasets are not supported through det.pytorch.DataLoader().  Read "
                "about the difference between IterableDatasets and Mapdata.Datasets at: "
                "pytorch.org/docs/stable/data. "
                "You can use an IterableDataset in Determined, but you will be responsible"
                " for shuffling, sharding, repeating, and skipping to ensure reproducibility and "
                "correctness of training.  See the in-depth guide at: "
                "https://docs.determined.ai/latest/training-apis/api-pytorch-advanced.html"
            )

        # Don't allow this rare combination of inputs that we would puke on later.
        if batch_sampler is None and batch_size is None:
            raise ValueError(
                "The case of batch_sampler=None and batch_size=None is not supported through "
                "det.pytorch.DataLoader(). For customizing your data loader beyond what is "
                "supported by det.pytorch.DataLoader, see the in-depth guide at: "
                "https://docs.determined.ai/latest/training-apis/api-pytorch-advanced.html"
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

        if num_workers == 0 and prefetch_factor is not None:
            raise ValueError(
                "prefetch_factor option could only be specified in multiprocessing. "
                "let num_workers > 0 to enable multiprocessing."
            )
        elif num_workers > 0 and prefetch_factor is None:
            prefetch_factor = 2
        elif prefetch_factor is not None and prefetch_factor < 0:
            raise ValueError("prefetch_factor option should be non-negative")

        if persistent_workers and num_workers == 0:
            raise ValueError("persistent_workers option needs num_workers > 0")

        self.dataset = dataset
        self.num_workers = num_workers
        self.prefetch_factor = prefetch_factor
        self.pin_memory = pin_memory
        self.timeout = timeout
        self.worker_init_fn = worker_init_fn
        self.multiprocessing_context = multiprocessing_context

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
                if version.parse(torch.__version__) >= version.parse("1.6.0"):
                    sampler = data.RandomSampler(dataset, generator=generator)  # type: ignore
                else:
                    sampler = data.RandomSampler(dataset)  # type: ignore
            else:
                sampler = data.SequentialSampler(dataset)  # type: ignore

        if batch_size is not None and batch_sampler is None:
            # auto_collation without custom batch_sampler
            batch_sampler = data.BatchSampler(sampler, batch_size, drop_last)

        self.batch_size = batch_size
        self.drop_last = drop_last
        self.sampler = sampler
        self.batch_sampler = batch_sampler
        self.generator = generator

        if collate_fn is None:
            if self._auto_collation:
                collate_fn = collate.default_collate
            else:
                collate_fn = collate.default_convert

        self.collate_fn = collate_fn
        self.persistent_workers = persistent_workers
        # END VENDORED CODE FROM PYTORCH

    # BEGIN VENDORED CODE FROM PYTORCH
    # https://github.com/pytorch/pytorch/blob/v1.3.1/torch/utils/data/dataloader.py#L280
    @property
    def _auto_collation(self) -> bool:
        return self.batch_sampler is not None

    # END VENDORED CODE FROM PYTORCH

    def get_data_loader(
        self, repeat: bool = False, skip: int = 0, num_replicas: int = 1, rank: int = 0
    ) -> data.DataLoader:
        batch_sampler = cast(data.BatchSampler, self.batch_sampler)
        batch_sampler = adapt_batch_sampler(
            batch_sampler, repeat=repeat, skip=skip, num_replicas=num_replicas, rank=rank
        )

        # Try to not break any torch version as old as v1.0.
        extra_kwargs = {}
        if version.parse(torch.__version__) >= version.parse("1.2.0"):
            extra_kwargs["multiprocessing_context"] = self.multiprocessing_context
        if version.parse(torch.__version__) >= version.parse("1.6.0"):
            extra_kwargs["generator"] = self.generator
        if version.parse(torch.__version__) >= version.parse("1.7.0"):
            if version.parse(torch.__version__) < version.parse("2.0.0"):
                # prefetch_factor became optional in 2.0.
                if self.prefetch_factor is None and self.num_workers == 0:
                    self.prefetch_factor = 2

            extra_kwargs["prefetch_factor"] = self.prefetch_factor
            extra_kwargs["persistent_workers"] = self.persistent_workers

        return data.DataLoader(
            self.dataset,
            batch_sampler=batch_sampler,
            num_workers=self.num_workers,
            collate_fn=self.collate_fn,
            pin_memory=self.pin_memory,
            timeout=self.timeout,
            worker_init_fn=self.worker_init_fn,
            **extra_kwargs,
        )

    def __iter__(self) -> Iterator:
        """Compatibility with the real DataLoader when using a PyTorchTrial outside Determined."""
        return iter(self.get_data_loader())

    def __len__(self) -> int:
        """Compatibility with the real DataLoader when using a PyTorchTrial outside Determined."""
        batch_sampler = cast(data.BatchSampler, self.batch_sampler)
        return len(batch_sampler)


def adapt_batch_sampler(
    batch_sampler: data.BatchSampler,
    repeat: bool = False,
    skip: int = 0,
    num_replicas: int = 1,
    rank: int = 0,
) -> data.BatchSampler:
    """
    Modify the underlying BatchSampler of a constructed DataLoader to account
    for repeating on training datasets, skipping when continuing training, and
    sharding for distributed training.
    """
    if repeat:
        batch_sampler = samplers.RepeatBatchSampler(batch_sampler)

    if num_replicas > 1:
        batch_sampler = samplers.DistributedBatchSampler(batch_sampler, num_replicas, rank)

    # SkipBatchSampler is used when we are continuing training. SkipBatchSampler must be
    # applied after DistributedBatchSampler, since the number of batches to skip is based on
    # how many global batches should be skipped, which corresponds with how many batches are emitted
    # by the DistributedBatchSampler, not the initial BatchSampler.
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
        logger.warning(f"Was not able to move data item of type '{type(data).__name__}' to device.")

    return data  # type:ignore
