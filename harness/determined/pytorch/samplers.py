"""
Guidelines for Reproducible Datasets
====================================

.. note::

   Normally, using ``det.pytorch.DataLoader`` is required and handles all of the below details
   without any special effort on your part (see :ref:`pytorch-data-loading`).  When
   ``det.pytorch.DataLoader()`` is not suitable (especially in the case of ``IterableDatasets``),
   you may disable this requirement by calling
   :meth:`context.experimental.disable_dataset_reproducibility_checks()
   <determined.pytorch.PyTorchExperimentalContext.disable_dataset_reproducibility_checks>`
   in your Trial's ``__init__()`` method.  Then you may choose to follow the below guidelines for
   ensuring dataset reproducibility on your own.

Achieving a reproducible dataset that is able to pause and continue (sometimes called "incremental
training") is easy if you follow a few rules.

- Even if you are going to ultimately return an IterableDataset, it is best to use PyTorch's Sampler
  class as the basis for choosing the order of records.  Operations on Samplers are quick and cheap,
  while operations on data afterwards are expensive.  For more details, see the discussion of random
  vs sequential access `here <https://yogadl.readthedocs.io>`_.  If you don't have a custom sampler,
  start with a simple one:

  .. code::python

     sampler = torch.utils.data.SequentialSampler(my_dataset)

- **Shuffle first**: Always use a reproducible shuffle when you shuffle.  Determined provides two
  shuffling samplers for this purpose; the ``ReproducibleShuffleSampler`` for operating on records
  and the ``ReproducibleShuffleBatchSampler`` for operating on batches.  You should prefer to
  shuffle on records (use the ``ReproducibleShuffleSampler``) whenever possible, to achieve the
  highest-quality shuffle.

- **Repeat when training**: In Determined, you always repeat your training dataset and you
  never repeat your validation datasets.  Determined provides a RepeatSampler and a
  RepeatBatchSampler to wrap your sampler or batch_sampler.  For your training dataset, make sure
  that you always repeat AFTER you shuffle, otherwise your shuffle will hang.

- **Always shard, and not before a repeat**: Use Determined's DistributedSampler or
  DistributedBatchSampler to provide a unique shard of data to each worker based on your sampler or
  batch_sampler.  It is best to always shard your data, and even when you are not doing distributed
  training, because in non-distributed-training settings, the sharding is nearly zero-cost, and it
  makes distributed training seamless if you ever want to use it in the future.

  It is generally important to shard after you repeat, unless you can guarantee that each shard
  of the dataset will have the same length.  Otherwise, differences between the epoch boundaries for
  each worker can grow over time, especially on small datasets.  If you shard after you repeat, you
  can change the number of workers arbitrarily without issue.

- **Skip when training, and always last**: In Determined, training datasets should always be able to
  start from an arbitrary point in the dataset.  This allows for advanced hyperparameter searches
  and responsive preemption for training on spot instances in the cloud.  The easiest way to do
  this, which is also very efficient, is to apply a skip to the sampler.

  Determined provides a SkipBatchSampler that you can apply to your batch_sampler for this purpose.
  There is also a SkipSampler that you can apply to your sampler, but you should prefer to skip on
  batches unless you are confident that your dataset always yields identical size batches, where the
  number of records to skip can be reliably calculatd from the number of batches already trained.

  Always skip AFTER your repeat, so that the skip only happens once, and not on every epoch.

  Always skip AFTER your shuffle, to preserve the reproducibility of the shuffle.

Here is some example code that follows each of these rules that you can use as a starting point if
you find that the built-in context.DataLoader() does not support your use case.

  .. code::python

    def make_batch_sampler(
        sampler_or_dataset,
        mode,  # mode="training" or mode="validation"
        shuffle_seed,
        num_workers,
        rank,
        batch_size,
        skip,
    ):
        if isinstance(sampler_or_dataset, torch.utils.data.Sampler):
            sampler = sampler_or_dataset
        else:
            # Create a SequentialSampler if we started with a Dataset.
            sampler = torch.utils.data.SequentialSampler(sampler_or_dataset)

        if mode == "training":
            # Shuffle first.
            sampler = samplers.ReproducibleShuffleSampler(sampler, shuffle_seed)

            # Repeat when training.
            sampler = samplers.RepeatSampler(sampler)

        # Always shard, and not before a repeat.
        sampler = samplers.DistributedSampler(sampler, num_workers=num_workers, rank=rank)

        # Batch before skip, because Determined counts batches, not records.
        batch_sampler = torch.utils.data.BatchSampler(sampler, batch_size, drop_last=False)

        if mode == "training":
            # Skip when training, and always last.
            batch_sampler = samplers.SkipBatchSampler(batch_sampler, skip)

        return batch_sampler


    class MyPyTorchTrial(det.pytorch.PyTorchTrial):
        def __init__(self, context):
            context.experimental.disable_dataset_reproducibility_checks()

        def build_training_data_loader(self):
            my_dataset = ...

            batch_sampler = make_batch_sampler(
                dataset=my_dataset,
                mode="training",
                seed=self.context.get_trial_seed(),
                num_workers=self.context.distributed.get_size(),
                rank=self.distributed.get_rank(),
                batch_size=self.context.get_per_slot_batch_size(),
                skip=self.context.get_initial_batch(),
            )

            return torch.utils.data.DataLoader(my_dataset, batch_sampler=batch_sampler)
"""
from typing import Iterator

import numpy as np
import torch

from determined.common import check


class RepeatSampler(torch.utils.data.Sampler):
    """
    RepeatSampler yields infinite batches indices by repeatedly iterating
    through the batches of another Sampler. __len__ is just the length of
    the underlying Sampler.
    """

    def __init__(self, sampler: torch.utils.data.Sampler) -> None:
        self._sampler = sampler

    def __len__(self) -> int:
        return len(self._sampler)  # type: ignore

    def __iter__(self) -> Iterator:
        while True:
            yield from self._sampler


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

    def __iter__(self) -> Iterator:
        while True:
            yield from self.batch_sampler


class DistributedSampler(torch.utils.data.Sampler):
    """
    DistributedSampler will iterate through an underlying sampler and return samples which
    belong to this shard.

    DistributedSampler is different than the PyTorch built-in torch.utils.data.DistributedSampler
    because theirs is meant to be a standalone sampler.  Theirs does shuffling and assumes a
    constant size dataset as an input.  Ours is meant to be used a building block in a chain of
    samplers, so it accepts a sampler as input that may or may not be constant-size.
    """

    def __init__(self, sampler: torch.utils.data.Sampler, num_workers: int, rank: int) -> None:
        self._sampler = sampler
        self._num_workers = num_workers
        self._rank = rank

    def __len__(self) -> int:
        sampler_len = len(self._sampler)  # type: ignore
        all_workers_get_samples = sampler_len // self._num_workers
        worker_gets_extra_sample = int(sampler_len % self._num_workers > self._rank)
        return all_workers_get_samples + worker_gets_extra_sample

    def __iter__(self) -> Iterator:
        if self._num_workers == 1:
            yield from self._sampler
        else:
            for i, batch in enumerate(self._sampler):
                if i % self._num_workers == self._rank:
                    yield batch


class DistributedBatchSampler(torch.utils.data.BatchSampler):
    """
    DistributedBatchSampler will iterate through an underlying batch sampler and return batches
    which belong to this shard.

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
        self, batch_sampler: torch.utils.data.BatchSampler, num_workers: int, rank: int
    ) -> None:
        check.gt(rank, -1, "rank must be non-negative")
        check.gt(num_workers, 0, "num_workers must be positive")
        check.lt(rank, num_workers, "rank must be less than num_workers")

        self.batch_sampler = batch_sampler
        self.num_workers = num_workers
        self.rank = rank

    def __len__(self) -> int:
        full_global_batches = len(self.batch_sampler) // self.num_workers
        worker_gets_partial_batch = int(len(self.batch_sampler) % self.num_workers > self.rank)
        return full_global_batches + worker_gets_partial_batch

    def __iter__(self) -> Iterator:
        if self.num_workers == 1:
            yield from self.batch_sampler
        else:
            for i, batch in enumerate(self.batch_sampler):
                if i % self.num_workers == self.rank:
                    yield batch


class SkipSampler(torch.utils.data.BatchSampler):
    """
    SkipSampler skips some records from an underlying Sampler, and yields the rest.

    Always skip before you repeat when you are continuing training, or you will apply the skip on
    every epoch.

    .. warning::

       When trying to achieve reproducibility after pausing and restarting, you should never prefer
       this SkipSampler over the SkipBatchSampler, unless you are sure that your dataset will always
       yield identically sized batches.  This is due to how Determined counts batches trained but
       does not count records trained.  Reproducibility when skipping records is only possible if
       the records to skip can be reliably calculated based on batch size and batches trained.

    Because the SkipSampler is only meant to be used on a training dataset (we never checkpoint
    during evaluation), and because the training dataset should always be repeated before applying
    the skip (so you only skip once rather than many times), the length reported is always the
    length of the underlying sampler, regardless of the size of the skip.
    """

    def __init__(self, sampler: torch.utils.data.BatchSampler, skip: int) -> None:
        self._sampler = sampler
        self._skip = skip

    def __len__(self) -> int:
        return len(self._sampler)

    def __iter__(self) -> Iterator:
        iterator = iter(self._sampler)
        for _ in range(self._skip):
            try:
                next(iterator)
            except StopIteration:
                return
        yield from iterator


class SkipBatchSampler(torch.utils.data.BatchSampler):
    """
    SkipBatchSampler skips some batches from an underlying BatchSampler, and yield the rest.

    Always skip before you repeat when you are continuing training, or you will apply the skip on
    every epoch.

    Because the SkipBatchSampler is only meant to be used on a training dataset (we never
    checkpoint during evaluation), and because the training dataset should always be repeated
    before applying the skip (so you only skip once rather than many times), the length reported
    is always the length of the underlying sampler, regardless of the size of the skip.
    """

    def __init__(self, batch_sampler: torch.utils.data.BatchSampler, skip: int) -> None:
        self.batch_sampler = batch_sampler
        self.skip = skip
        self.length = len(batch_sampler)

    def __len__(self) -> int:
        return len(self.batch_sampler)

    def __iter__(self) -> Iterator:
        iterator = iter(self.batch_sampler)
        for _ in range(self.skip):
            try:
                next(iterator)
            except StopIteration:
                return
        yield from iterator


class ReproducibleShuffleSampler(torch.utils.data.Sampler):
    """
    ReproducibleShuffleSampler will apply a deterministic shuffle based on a seed.

    .. warning::

       Always shuffle before skipping and before repeating.  Skip-before-shuffle would break
       the reproducibility of the shuffle, and repeat-before-shuffle would cause the shuffle
       to hang as it iterates through an infinite sampler.
    """

    def __init__(self, sampler: torch.utils.data.Sampler, seed: int) -> None:
        self._sampler = sampler
        self._rng = np.random.RandomState(seed)

    def __iter__(self) -> Iterator:
        indices = list(self._sampler)
        self._rng.shuffle(indices)
        return iter(indices)

    def __len__(self) -> int:
        # Check the original sampler in case its length changes every epoch.
        # TODO: that would likely cause reproducibility issues.
        return len(self._sampler)  # type: ignore


class ReproducibleShuffleBatchSampler(torch.utils.data.Sampler):
    """
    ReproducibleShuffleBatchSampler will apply a deterministic shuffle based on a seed.

    .. warning::

       Always shuffle before skipping and before repeating.  Skip-before-shuffle would break
       the reproducibility of the shuffle, and repeat-before-shuffle would cause the shuffle
       to hang as it iterates through an infinite sampler.

    .. warning::

       Always prefer ReproducibleShuffleSampler over this class when possible.  The reason is that
       shuffling at the batch level results in a superior shuffle, where the contents of each
       batch are varied between epochs, rather than just the order of batches.
    """

    def __init__(self, batch_sampler: torch.utils.data.BatchSampler, seed: int) -> None:
        self._batch_sampler = batch_sampler
        self._rng = np.random.RandomState(seed)

    def __iter__(self) -> Iterator:
        indices = list(self._batch_sampler)
        self._rng.shuffle(indices)
        return iter(indices)

    def __len__(self) -> int:
        # Check the original batch_sampler in case its length changes every epoch.
        # TODO: that would likely cause reproducibility issues.
        return len(self._batch_sampler)
