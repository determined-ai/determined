import threading
import time
from typing import Any, Dict, Tuple
from unittest import mock

import pytest

from determined import core
from tests import parallel


class MockPreemptState:
    """
    MockPreemptState uses threading to simulate the behavior of the long-
    polling requests call.
    """

    def __init__(self) -> None:
        # mock is the main Session Mock
        self.mock_session = mock.MagicMock()
        self.mock_session.get.side_effect = self.session_get
        self._state = False
        self._cond = threading.Condition()

    def preempt(self) -> None:
        """preempt() is called from the test code."""
        with self._cond:
            self._state = True
            self._cond.notify()

    def session_get(self, path: str, params: Dict[str, Any], timeout: float) -> mock.MagicMock:
        """session_get() is called from a background thread."""
        # We only mock one GET endpoint.
        assert path.endswith("signals/preemption"), path
        assert "timeout_seconds" in params, params

        timeout_seconds = float(params["timeout_seconds"])
        deadline = time.time() + timeout_seconds

        def wait_for_preemption() -> None:
            if timeout_seconds == 0:
                return
            with self._cond:
                while self._state is False:
                    now = time.time()
                    if now >= deadline:
                        return
                    try:
                        self._cond.wait(timeout=deadline - now)
                    except TimeoutError:
                        return

        # Wait for the preemption signal or until the deadline is reached.
        wait_for_preemption()

        response = mock.MagicMock()
        response.json.return_value = {"preempt": self._state}
        return response


def make_test_preempt_context(
    dist: core.DistributedContext,
    mode: core.PreemptMode,
) -> Tuple[MockPreemptState, core.PreemptContext]:

    state = MockPreemptState()
    context = core.PreemptContext(state.mock_session, "allocation_id", dist, mode)
    return state, context


def wait_on_watcher(preempt_context: core.PreemptContext) -> None:
    # It's racy as to when the watcher will actually see the signal.
    watcher = preempt_context._watcher
    assert watcher is not None
    for i in range(5):
        if watcher._should_preempt:
            break
        time.sleep((i / 10) ** 2)
    assert watcher._should_preempt is True, watcher._should_preempt


@pytest.mark.parametrize("auto_ack", [False, True], ids=lambda x: f"auto_ack:{x}")
@pytest.mark.parametrize("dummy", [False, True], ids=lambda x: f"dummy:{x}")
def test_preempt_workers_ask_chief(dummy: bool, auto_ack: bool) -> None:
    with parallel.Execution(2) as pex:

        @pex.run
        def do_test() -> None:
            if not dummy:
                state, context = make_test_preempt_context(
                    pex.distributed, core.PreemptMode.WorkersAskChief
                )
            else:
                context = core.DummyPreemptContext(
                    pex.distributed, core.PreemptMode.WorkersAskChief
                )

            with context:
                if pex.rank == 0:
                    # Check preemption.
                    assert context.should_preempt() is False
                    # Make sure the worker is receiving broadcasts.
                    _ = pex.distributed.broadcast(False)
                    if not dummy:
                        # No ack preemption calls yet.
                        state.mock_session.post.assert_not_called()
                        # Send the preemption signal.
                        state.preempt()
                        wait_on_watcher(context)
                        assert context.should_preempt(auto_ack=auto_ack) is True
                        # Call again, to make sure we only ack once.
                        assert context.should_preempt(auto_ack=auto_ack) is True
                        if auto_ack:
                            state.mock_session.post.assert_called_once()
                        else:
                            state.mock_session.post.assert_not_called()
                else:
                    # Intercept the broadcast from the chief to make sure it's happening.
                    out = pex.distributed.broadcast(None)
                    assert out is False, out
                    # Try receving from the chief.
                    assert context.should_preempt() is False
                    if not dummy:
                        # The chief should send a True now.
                        assert context.should_preempt() is True
                        # Only the chief acknowledges the preemption signal.
                        state.mock_session.post.assert_not_called()


@pytest.mark.parametrize("auto_ack", [False, True], ids=lambda x: f"auto_ack:{x}")
@pytest.mark.parametrize("dummy", [False, True], ids=lambda x: f"dummy:{x}")
def test_preempt_chief_only(dummy: bool, auto_ack: bool) -> None:
    with parallel.Execution(2) as pex:

        # Steal the automatically-created pex.distributed contexts, then test chief/worker serially
        # so we know they're not using distributed comms.
        @pex.run
        def distributed_contexts() -> core.DistributedContext:
            return pex.distributed

        # Test chief.
        if not dummy:
            state, context = make_test_preempt_context(
                distributed_contexts[0], core.PreemptMode.ChiefOnly
            )
        else:
            context = core.DummyPreemptContext(distributed_contexts[0], core.PreemptMode.ChiefOnly)
        with context:
            assert context.should_preempt() is False
            if not dummy:
                # No ack preemption calls yet.
                state.mock_session.post.assert_not_called()
                # Send the preemption signal.
                state.preempt()
                wait_on_watcher(context)
                assert context.should_preempt(auto_ack=auto_ack) is True
                # Call again, to make sure we only ack once.
                assert context.should_preempt(auto_ack=auto_ack) is True
                if auto_ack:
                    state.mock_session.post.assert_called_once()
                else:
                    state.mock_session.post.assert_not_called()

        # Test worker.
        if not dummy:
            state, context = make_test_preempt_context(
                distributed_contexts[1], core.PreemptMode.ChiefOnly
            )
        else:
            context = core.DummyPreemptContext(distributed_contexts[1], core.PreemptMode.ChiefOnly)
        with context:
            with pytest.raises(RuntimeError, match="should_preempt.*called from non-chief"):
                context.should_preempt()


@pytest.mark.parametrize("auto_ack", [False, True], ids=lambda x: f"auto_ack:{x}")
@pytest.mark.parametrize("dummy", [False, True], ids=lambda x: f"dummy:{x}")
def test_preempt_workers_ask_master(dummy: bool, auto_ack: bool) -> None:
    with parallel.Execution(2) as pex:

        # Steal the automatically-created pex.distributed contexts, then test chief/worker serially
        # so we know they're not using distributed comms.
        @pex.run
        def distributed_contexts() -> core.DistributedContext:
            return pex.distributed

        # Test steps are identical for chief and worker.
        for dist in distributed_contexts:
            if not dummy:
                state, context = make_test_preempt_context(dist, core.PreemptMode.WorkersAskMaster)
            else:
                context = core.DummyPreemptContext(dist, core.PreemptMode.WorkersAskMaster)
            with context:
                assert context.should_preempt() is False
                if not dummy:
                    # No ack preemption calls yet.
                    state.mock_session.post.assert_not_called()
                    # Send the preemption signal.
                    state.preempt()
                    wait_on_watcher(context)
                    # Call again, to make sure we only ack once.
                    assert context.should_preempt(auto_ack=auto_ack) is True
                    if auto_ack:
                        state.mock_session.post.assert_called_once()
                    else:
                        state.mock_session.post.assert_not_called()


@pytest.mark.parametrize("dummy", [False, True])
def test_check_started(dummy: bool) -> None:
    with parallel.Execution(2) as pex:

        @pex.run
        def do_test() -> None:
            if not dummy:
                state, context = make_test_preempt_context(
                    pex.distributed, core.PreemptMode.WorkersAskChief
                )
            else:
                context = core.DummyPreemptContext(
                    pex.distributed, core.PreemptMode.WorkersAskChief
                )

            with pytest.raises(RuntimeError, match="cannot call.*should_preempt.*before.*start"):
                context.should_preempt()
            with context:
                assert context.should_preempt() is False
