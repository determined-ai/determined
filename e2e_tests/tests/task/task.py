import time

import pytest

from determined.common import api


def wait_for_task_state(
    test_session: api.Session,
    task_id: str,
    expected_state: api.bindings.v1GenericTaskState,
    timeout: int = 30,
) -> None:
    deadline = time.time() + timeout
    while time.time() < deadline:
        resp = api.bindings.get_GetTask(test_session, taskId=task_id)
        if expected_state == resp.task.taskState:
            return
        time.sleep(0.1)
    pytest.fail(f"task failed to complete after {timeout} seconds")


def wait_for_task_start(
    test_session: api.Session,
    task_id: str,
    timeout: int = 30,
    start_or_run: bool = False,
) -> None:
    deadline = time.time() + timeout
    while time.time() < deadline:
        resp = api.bindings.get_GetTask(test_session, taskId=task_id)
        state = resp.task.allocations[0].state
        if start_or_run:
            if (
                state == api.bindings.taskv1State.STARTING
                or state == api.bindings.taskv1State.RUNNING
            ):
                return
        elif state == api.bindings.taskv1State.RUNNING:
            return
        time.sleep(0.1)
    pytest.fail(f"task failed to run after {timeout} seconds")
