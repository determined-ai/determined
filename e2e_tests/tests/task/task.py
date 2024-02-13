import time

from determined.common import api


def wait_for_task_state(
    test_session: api.Session,
    task_id: str,
    expected_state: api.bindings.v1GenericTaskState,
    timeout: int,
) -> bool:
    deadline = time.time() + timeout
    while time.time() < deadline:
        resp = api.bindings.get_GetTask(test_session, taskId=task_id)
        if expected_state == resp.task.taskState:
            return True
        time.sleep(0.1)
    return False


def wait_for_task_start(
    test_session: api.Session,
    task_id: str,
    timeout: int,
) -> bool:
    deadline = time.time() + timeout
    while time.time() < deadline:
        resp = api.bindings.get_GetTask(test_session, taskId=task_id)
        if resp.task.allocations[0].state == api.bindings.taskv1State.RUNNING:
            return True
        time.sleep(0.1)
    return False
