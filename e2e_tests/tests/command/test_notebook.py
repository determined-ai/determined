import re
from typing import List

import pytest

from determined.common import api
from determined.common.api import bindings
from tests import api_utils
from tests import command as cmd


@pytest.mark.slow
@pytest.mark.e2e_cpu
@pytest.mark.e2e_slurm
@pytest.mark.e2e_pbs
def test_basic_notebook_start_and_kill() -> None:
    sess = api_utils.user_session()
    lines = []  # type: List[str]
    with cmd.interactive_command(sess, ["notebook", "start"]) as notebook:
        for line in notebook.stdout:
            if re.search("Jupyter Notebook .*is running at", line) is not None:
                return
            lines.append(line)

    raise ValueError(lines)


@pytest.mark.e2e_cpu
def test_notebook_proxy() -> None:
    session = api_utils.user_session()

    def get_proxy(session: api.Session, task_id: str) -> None:
        session.get(f"proxy/{task_id}/")

    typ = api.NTSC_Kind.notebook
    created_id = api_utils.launch_ntsc(session, 1, typ).id
    print(f"created {typ} {created_id}")
    api.wait_for_ntsc_state(
        session,
        api.NTSC_Kind(typ),
        created_id,
        lambda s: s == bindings.taskv1State.RUNNING,
        timeout=300,
    )
    deets = api.get_ntsc_details(session, typ, created_id)
    assert deets.state == bindings.taskv1State.RUNNING, f"{typ} should be running"
    err = api.wait_for_task_ready(session, created_id)
    assert err is None, f"{typ} should be ready {err}"
    print(deets)
    try:
        get_proxy(session, created_id)
    finally:
        api_utils.kill_ntsc(session, typ, created_id)
