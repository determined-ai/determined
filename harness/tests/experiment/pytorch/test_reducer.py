import itertools
import logging
import threading
import traceback
from collections import namedtuple
from typing import Any, Callable, List

import numpy as np
import pytest

from determined import core
from determined.pytorch import Reducer, _PyTorchReducerContext, _simple_reduce_metrics

logger = logging.getLogger(__name__)


def test_reducer() -> None:
    metrics = np.array([0.25, 0.5, 0.75, 1, 25.5, 1.9])
    assert np.around(_simple_reduce_metrics(Reducer.AVG, metrics), decimals=2) == 4.98
    assert _simple_reduce_metrics(Reducer.SUM, metrics) == 29.9
    assert _simple_reduce_metrics(Reducer.MIN, metrics) == 0.25
    assert _simple_reduce_metrics(Reducer.MAX, metrics) == 25.5

    batches_per_process = [1, 2, 5, 4, 5, 6]
    assert (
        np.around(_simple_reduce_metrics(Reducer.AVG, metrics, batches_per_process), decimals=2)
        == 6.43
    )


DummyDistributedReducerContext = namedtuple(
    "DummyDistributedReducerContext", "distributed_context reducer_context wrapped_reducer"
)


def dummy_reducer(values: List) -> Any:
    logger.debug(f"reducing {values}")
    flat = [v for sublist in values for v in sublist]
    return {"values": flat, "sum": sum(flat)}


@pytest.mark.parametrize("cross_size", [1, 3])
@pytest.mark.parametrize("local_size", [1, 3])
def test_custom_reducer_slot_order(cross_size: int, local_size: int) -> None:
    size = cross_size * local_size
    dataset_size = 47

    def do_parallel(fn: Callable) -> List:
        """
        Run the same function on one-thread-per-rank, assert there were no exceptions, and return
        the results from each rank.
        """
        results = [None] * size  # type: List
        errors = [None] * size  # type: List
        threads = []

        for cross_rank, local_rank in itertools.product(range(cross_size), range(local_size)):
            rank = cross_rank * local_size + local_rank

            def _fn(rank: int, cross_rank: int, local_rank: int) -> None:
                try:
                    results[rank] = fn(rank, cross_rank, local_rank)
                except Exception:
                    errors[rank] = traceback.format_exc()
                    raise

            threads.append(threading.Thread(target=_fn, args=(rank, cross_rank, local_rank)))

        # encourage allgather to occur in not-the-correct order to test the reordering
        for thread in reversed(threads):
            thread.start()

        for thread in threads:
            thread.join()

        assert errors == [None] * size, "not all threads exited without error"

        return results

    def make_reducer_context(
        rank: int, cross_rank: int, local_rank: int
    ) -> DummyDistributedReducerContext:
        distributed_context = core.DistributedContext(
            rank=cross_rank * local_size + local_rank,
            size=cross_size * local_size,
            local_rank=local_rank,
            local_size=local_size,
            cross_rank=cross_rank,
            cross_size=cross_size,
            chief_ip="localhost",
            force_tcp=False,
        )
        reducer_context = _PyTorchReducerContext(distributed_context.allgather)
        # reducer_context.wrap_reducer(lambda x: x, "dummy")
        wrapped_reducer = reducer_context.wrap_reducer(dummy_reducer)
        return DummyDistributedReducerContext(distributed_context, reducer_context, wrapped_reducer)

    trials = do_parallel(make_reducer_context)

    def get_batch_list(
        rank: int, batch_size: int, num_workers: int, seq: List[int]
    ) -> List[List[int]]:
        total_batches = (len(seq) + (batch_size - 1)) // batch_size
        my_batch_indices = [i for i in range(total_batches) if i % num_workers == rank]
        all_batches = [
            seq[batch_size * k : min(batch_size * k + batch_size, len(seq))]
            for k in range(total_batches)
        ]
        return [b for i, b in enumerate(all_batches) if i in my_batch_indices]

    observations = list(range(dataset_size))
    for rank, trial in enumerate(trials):
        for batch in get_batch_list(rank, 2, len(trials), observations):
            trial.wrapped_reducer.update(batch)

    results = do_parallel(lambda rank, _, __: trials[rank].reducer_context.reduce_metrics(False))
    logger.debug(results)

    # Close all distributed contexts
    for trial in trials:
        trial.distributed_context.close()

    for i, result in enumerate(results):
        assert result["sum"] == dataset_size * (dataset_size - 1) // 2
        assert all(
            i == v for i, v in enumerate(result["values"])
        ), f"result[{i}]={result} is not in original order"
