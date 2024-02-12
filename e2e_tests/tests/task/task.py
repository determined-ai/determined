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
        print(resp.task.taskState)
        time.sleep(0.1)
    return False
