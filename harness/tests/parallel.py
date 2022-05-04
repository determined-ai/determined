import contextlib
import itertools
import sys
import threading
import traceback
from typing import Any, Callable, Dict, Iterator, List, Tuple, TypeVar

import pytest

from determined import core

T = TypeVar("T")


class Execution:
    """
    parallel.Execution is a tool for writing easy threading-based parallel tests.

    Execution.run is the main helper function, but there are a few magic getters that return
    correct thread-specific values when called from within an Execution.run-wrapped function.

    Example usage:

        SIZE = 10
        with parallel.Execution(SIZE) as pex:
            @pex.run
            def all_ranks():
                return pex.rank

            assert all_ranks == list(range(SIZE))

    """

    def __init__(
        self, size: int, local_size: int = 1, make_distributed_context: bool = True
    ) -> None:
        assert size % local_size == 0, f"size%local_size must be 0 ({size}%{local_size})"
        self.size = size
        self.local_size = local_size
        self.cross_size = size // local_size
        # We keep some thread-specific info to implement the magic getters.
        self._info: Dict[int, Tuple[int, int, int]] = {}

        self._dist = None
        if make_distributed_context:

            def _make_distributed_context() -> core.DistributedContext:
                return core.DistributedContext(
                    rank=self.rank,
                    size=self.size,
                    local_rank=self.local_rank,
                    local_size=self.local_size,
                    cross_rank=self.cross_rank,
                    cross_size=self.cross_size,
                    chief_ip="localhost",
                )

            self._dist = self.run(_make_distributed_context)

    def __enter__(self) -> "Execution":
        return self

    def __exit__(self, *arg: Any) -> None:
        if not self._dist:
            return
        for dist in self._dist:
            dist.close()

    def run(self, fn: Callable[[], T]) -> List[T]:
        """
        Run the same function on one-thread-per-rank, assert there were no exceptions, and return
        the results from each rank.

        run can be used as a decorator or called directly.
        """
        results = [None] * self.size  # type: List
        errors = [None] * self.size  # type: List
        threads = []

        for cross_rank, local_rank in itertools.product(
            range(self.cross_size), range(self.local_size)
        ):
            rank = cross_rank * self.local_size + local_rank

            def _fn(rank: int, cross_rank: int, local_rank: int) -> None:
                thread_id = threading.get_ident()
                self._info[thread_id] = (rank, cross_rank, local_rank)
                try:
                    results[rank] = fn()
                # Catch anything, including a pytest.Fail (so we can preserve it).
                except BaseException as e:
                    # Print the error to stderr immediately, in case it results in a hang.
                    traceback.print_exc()
                    errors[rank] = (rank, e, sys.exc_info())
                finally:
                    del self._info[thread_id]

            threads.append(threading.Thread(target=_fn, args=(rank, cross_rank, local_rank)))

        for thread in threads:
            thread.start()

        for thread in threads:
            thread.join()

        # Filter empty errors.
        errors = [e for e in errors if e is not None]
        if len(errors) > 1:
            # In multi-errors situations, print all of them
            for rank, _, exc_info in errors:
                print(f"\nERROR ON RANK={rank}:", file=sys.stderr)
                traceback.print_exception(*exc_info)
            print(file=sys.stderr)
        if errors:
            # Reraise just the first exception.
            _, e, _ = errors[0]
            raise e

        return results

    @property
    def rank(self) -> int:
        """
        Only callable within an @Execution.run-wrapped function.

        Use the thread identifier to figure out what the rank is for the caller, and return the
        rank of that caller.

        This is syntactic sugar to avoid having to write a large number of functions that take
        parameters of (rank, cross_rank, local_rank).
        """
        thread_id = threading.get_ident()
        assert thread_id in self._info, "must be called from within an @Execute-decorated function"
        return self._info[thread_id][0]

    @property
    def cross_rank(self) -> int:
        thread_id = threading.get_ident()
        assert thread_id in self._info, "must be called from within an @Execute-decorated function"
        return self._info[thread_id][1]

    @property
    def local_rank(self) -> int:
        thread_id = threading.get_ident()
        assert thread_id in self._info, "must be called from within an @Execute-decorated function"
        return self._info[thread_id][2]

    @property
    def distributed(self) -> core.DistributedContext:
        assert self._dist is not None, "Execute was configured make_distributed_context=False"
        thread_id = threading.get_ident()
        assert thread_id in self._info, "must be called from within an @Execute-decorated function"
        return self._dist[self.rank]


@contextlib.contextmanager
def raises_when(pred: bool, *args: Any, **kwargs: Any) -> Iterator[None]:
    """
    A wrapper around pytest.raises that has a handy predicate argument.

    Useful in @parallel.Execution.run-wrapped functions.

    Example usage:

        with parallel.Execution(2) as pex:
            @pex.run
            def all_workers_fn():
                with raises_when(pex.rank!=0, AssertionError):
                    assert pex.rank == 0

    """
    if not pred:
        yield
        return

    with pytest.raises(*args, **kwargs):
        yield
