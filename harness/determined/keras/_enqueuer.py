import abc
import collections
import multiprocessing
import multiprocessing.queues
import queue
import threading
from typing import Any, Deque, Dict, Iterator, Optional, Type, Union

import numpy as np
import tensorflow as tf

from determined.common import check

Queue = Union[queue.Queue, multiprocessing.Queue]
Worker = Union[threading.Thread, multiprocessing.Process]


class _Sampler:
    """
    _Sampler decides the order of batches to read from a Keras Sequence, similar to a PyTorch
    Sampler object.  In a data loader, there is one _Sampler in the main thread/process that
    decides the order of data loading for all the dataloader worker threads/processes.

    _Sampler therefore makes it possible to shuffle indices without concern for synchronous access,
    as is the case when trying to implement shuffle from inside of a Keras Sequence inside of the
    real Keras OrderedEnqueuer.
    """

    def __init__(
        self,
        length: int,
        shard_rank: int,
        num_shards: int,
        shuffle: bool,
        shuffle_seed: int,
        prior_batches_trained: int,
    ) -> None:
        self.indices = list(range(length))
        self.num_shards = num_shards
        self.shuffle = shuffle

        check.gt_eq(
            length,
            num_shards,
            "please provide a Sequence that has at least as many batches as the number of slots "
            "used for training",
        )

        # Each shard has a certain offset from which it yields data.  When the dataset length is
        # not evenly divisible by the shard size, that offset will change every epoch.
        # Example:
        #   let length=10, shard_rank=0, and num_shards=3:
        #   epoch 1: 0, 3, 6, 9
        #   epoch 2: 2, 5, 8
        #   epoch 3: 1, 4, 7
        #   epoch 4: (same as epoch 1)
        # In this example, the offset in the first three epochs is 0, then 2, then 1.
        # The initial offset is always shard_rank, and the offest is recalculated in _end_epoch().
        self.offset = shard_rank

        if self.shuffle:
            assert shuffle_seed is not None
            self.rng = np.random.RandomState(shuffle_seed)
            self.rng.shuffle(self.indices)

        # Start in the correct epoch of shuffle.
        batches_to_skip = prior_batches_trained
        while len(self._this_epoch_indices()) <= batches_to_skip:
            batches_to_skip -= len(self._this_epoch_indices())
            self._end_epoch()

        self.offset += self.num_shards * batches_to_skip

    def _this_epoch_indices(self) -> range:
        return range(self.offset, len(self.indices), self.num_shards)

    def _end_epoch(self) -> None:
        """
        The _Sampler is stateful, and _epoch_end is where it modifies it state after each epoch.
        """
        # Recalculate this shard's offset.
        self.offset = (self.offset - len(self.indices)) % self.num_shards
        # Reshuffle indices.
        if self.shuffle:
            self.rng.shuffle(self.indices)

    def yield_epoch(self) -> Iterator:
        for i in self._this_epoch_indices():
            yield self.indices[i]
        self._end_epoch()


class _Enqueuer(metaclass=abc.ABCMeta):
    """
    An _Enqueuer should pass indices from the _Sampler to dataloader workers and return the results
    in the form of python generator.
    """

    @abc.abstractmethod
    def start(self) -> None:
        """start() must be called from the main thread/process once before any calls to data()."""
        pass

    @abc.abstractmethod
    def stop(self) -> None:
        """stop() must be called from the main thread/process after any calls to data()."""
        pass

    @abc.abstractmethod
    def data(self) -> Iterator:
        """data() may be called multiple times if the provided _Sampler is finite."""
        pass

    def __enter__(self) -> "_Enqueuer":
        self.start()
        return self

    def __exit__(self, *_: Any) -> None:
        self.stop()


class _WorkerlessEnqueuer(_Enqueuer):
    """
    _WorkerlessEnqueuer queries data from a Keras Sequence directly on the main thread.  Used when
    workers=0.
    """

    def __init__(self, sequence: tf.keras.utils.Sequence, sampler: _Sampler, repeat: bool):
        self.sequence = sequence
        self.sampler = sampler
        self.repeat = repeat

    def start(self) -> None:
        pass

    def stop(self) -> None:
        pass

    def close(self) -> None:
        pass

    def data(self) -> Iterator:
        while True:
            for i in self.sampler.yield_epoch():
                yield self.sequence[i]
            self.sequence.on_epoch_end()
            if not self.repeat:
                return


def _worker(sequence: tf.keras.utils.Sequence, queries: Queue, answers: Queue) -> None:
    """
    _worker defines a data loader worker's primary loop.  This loop runs in either a
    threading.Thread or a multiprocessing.Process; the caller (a _ParallelEnqueuer) is responsible
    for passing in the appropriate type of Queue.

    Parameters:
        sequence: the user-provided Keras Sequence.
        queries: a queue of tuples of (indices, order) that need to be read from the sequence.
        answers: a queue of tuples of (data, order) that workers fill with data from the sequence.
    """

    try:
        while True:
            query = queries.get()
            if query is None:
                return
            i, order = query
            data = sequence[i]
            answers.put((data, order))
    finally:
        answers.put(None)


class _ParallelEnqueuer(_Enqueuer):
    """
    _ParallelEnqueuer defines the semantics for either a threading-based or multiprocessing-based
    enqueuer (threading and multiprocessing are intentionally designed to be API compatible).  The
    implementation-specific details are defined in the child classes.

    Generally, the strategy is:
      - in start(): start a bunch of workers with access to a pair of queues.

      - in data(): fill the query queue with tuples of (index, order) tuples.  The index comes from
        the _Sampler, and the order just defines in what order we put things into the queue.

        Workers will pop (index, order) tuples fram the query queue and put tuples of (data, order)
        in the answers queue.

        The main thread will gather the (data, order) tuples, and reorder the data elements to
        match the order they were requested.  Whenever the next-requested data is available, it
        will pass yield that data through the generator.

      - in close(): shut down all the workers and clean up resources.
    """

    def __init__(
        self,
        sequence: tf.keras.utils.Sequence,
        sampler: _Sampler,
        repeat: bool,
        workers: int,
        max_queue_size: int,
    ):
        self.sequence = sequence
        self.sampler = sampler
        self.repeat = repeat
        self.max_queue_size = max_queue_size
        check.gt(max_queue_size, 0, "max_queue_size must be greater than zero")

        # Coordination logic.
        self.order = 0
        self.requested = collections.deque()  # type: Deque[int]
        self.received = {}  # type: Dict[int, Any]
        self.started = False
        self.stopped = False
        self.index_iter = None  # type: Optional[Iterator]

        # Interthread/interprocess communications.
        self.queries = self.queue_class()()
        self.answers = self.queue_class()()

        self.workers = [
            self.worker_class()(target=_worker, args=(self.sequence, self.queries, self.answers))
            for _ in range(workers)
        ]

    def start(self) -> None:
        assert not self.started and not self.stopped, "restarting an enqueuer is not allowed"
        self.started = True
        for workers in self.workers:
            workers.start()

    def stop(self) -> None:
        if not self.started:
            self.stopped = True
        if self.stopped:
            return
        for _ in self.workers:
            self.queries.put(None)
        for workers in self.workers:
            workers.join()
        self.stopped = True

    def data(self) -> Iterator:
        while True:
            yield from self.one_epoch()
            if not self.repeat:
                return

    def fill_requests(self) -> None:
        if self.index_iter is None:
            # No data left this epoch.
            return
        while len(self.requested) < self.max_queue_size:
            try:
                i = next(self.index_iter)
            except StopIteration:
                self.index_iter = None
                return
            puttable = (i, self.order)
            self.queries.put(puttable)
            self.requested.append(self.order)
            self.order += 1

    def one_epoch(self) -> Iterator:
        self.index_iter = self.sampler.yield_epoch()
        self.fill_requests()
        while len(self.requested):
            # Block on recieving the next in-order data.
            target = self.requested.popleft()
            while target not in self.received:
                answer = self.get_answer()
                if answer is None:
                    raise ValueError("data loading worker finished unexpectedly")
                data, order = answer
                self.received[order] = data
            data = self.received.pop(target)
            self.fill_requests()
            yield data
        self.sequence.on_epoch_end()

    @abc.abstractmethod
    def queue_class(self) -> Type[Queue]:
        pass

    @abc.abstractmethod
    def worker_class(self) -> Type[Worker]:
        pass

    @abc.abstractmethod
    def get_answer(self) -> Any:
        pass


class _ThreadingEnqueuer(_ParallelEnqueuer):
    """threading.Thread-specific implementation details."""

    def queue_class(self) -> Type[Queue]:
        return queue.Queue

    def worker_class(self) -> Type[Worker]:
        return threading.Thread

    def get_answer(self) -> Any:
        return self.answers.get()


class _MultiprocessingEnqueuer(_ParallelEnqueuer):
    """multiprocessing.Process-specific implementation details."""

    def queue_class(self) -> Type[Queue]:
        return multiprocessing.Queue

    def worker_class(self) -> Type[Worker]:
        return multiprocessing.Process

    def get_answer(self) -> Any:
        """Periodically conduct a health check while waiting on workers"""
        while True:
            try:
                return self.answers.get(timeout=5)
            except multiprocessing.queues.Empty:  # type: ignore
                self.health_check()

    def health_check(self) -> None:
        for worker in self.workers:
            if not worker.is_alive():
                raise ValueError("data loading worker died unexpectedly")


def _build_enqueuer(
    sequence: tf.keras.utils.Sequence,
    workers: int,
    use_multiprocessing: bool,
    max_queue_size: int,
    shard_rank: int,
    num_shards: int,
    repeat: bool,
    shuffle: bool,
    shuffle_seed: int,
    prior_batches_trained: int,
) -> _Enqueuer:
    sampler = _Sampler(
        len(sequence),
        shard_rank,
        num_shards,
        shuffle,
        shuffle_seed,
        prior_batches_trained,
    )
    if workers < 1:
        return _WorkerlessEnqueuer(sequence, sampler, repeat)
    enqueuer_cls = _MultiprocessingEnqueuer if use_multiprocessing else _ThreadingEnqueuer
    return enqueuer_cls(sequence, sampler, repeat, workers, max_queue_size)
