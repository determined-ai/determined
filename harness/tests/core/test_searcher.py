from typing import Any
from unittest import mock

import pytest

from determined import core
from tests import parallel


def make_test_searcher(max_length: int, dist: core.DistributedContext) -> core.SearcherContext:
    session = mock.MagicMock()
    searcher = core.SearcherContext(
        session=session,
        dist=dist,
        trial_id=1,
        max_length=max_length,
    )
    return searcher


@pytest.mark.parametrize("dummy", [False, True])
def test_searcher_workers_ask_chief(dummy: bool) -> None:
    with parallel.Execution(2) as pex:

        @pex.run
        def searchers() -> core.SearcherContext:
            if not dummy:
                searcher = make_test_searcher(5, pex.distributed)
            else:
                searcher = core.DummySearcherContext(dist=pex.distributed)
            epochs_trained = 0
            # Iterate through ops.
            for op in searcher.operations():
                assert pex.distributed.allgather(op.length) == [op.length] * pex.size
                while epochs_trained < op.length:
                    epochs_trained += 1
                    expect = [epochs_trained] * pex.size
                    assert pex.distributed.allgather(epochs_trained) == expect
                    with parallel.raises_when(
                        pex.rank != 0, RuntimeError, match="op.report_progress.*chief"
                    ):
                        op.report_progress(epochs_trained)
                with parallel.raises_when(
                    pex.rank != 0, RuntimeError, match="op.report_completed.*chief"
                ):
                    op.report_completed(0.0)

            return searcher

        if not dummy:
            # Expect calls from chief: 5x progress
            chief = searchers[0]
            post_mock: Any = chief._session.post
            assert post_mock.call_count == 5, post_mock.call_args_list

            # The workers must not make any REST API calls at all.
            worker = searchers[1]
            post_mock = worker._session.post
            post_mock.assert_not_called()


def test_completion_check() -> None:
    with parallel.Execution(2) as pex:

        @pex.run
        def do_test() -> None:
            searcher = make_test_searcher(5, pex.distributed)

            ops = iter(searcher.operations())
            next(ops)
            # Don't complete the op.
            with parallel.raises_when(
                pex.rank == 0, RuntimeError, match="must call op.report_completed"
            ):
                next(ops)
            # Wake up worker manually; it is hung waiting for the now-failed chief.
            if pex.rank == 0:
                pex.distributed.broadcast(10)


@pytest.mark.parametrize("dummy", [False, True])
def test_searcher_chief_only(dummy: bool) -> None:
    with parallel.Execution(2) as pex:

        @pex.run
        def do_test() -> None:
            if not dummy:
                searcher = make_test_searcher(1, pex.distributed)
            else:
                searcher = core.DummySearcherContext(dist=pex.distributed)

            with parallel.raises_when(
                pex.rank != 0, RuntimeError, match="searcher.operations.*chief"
            ):
                next(iter(searcher.operations(core.SearcherMode.ChiefOnly)))
