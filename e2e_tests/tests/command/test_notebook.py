import re
from typing import List

import pytest

import tests.config as conf
from determined.common import api
from determined.common.api import NTSC_Kind, bindings, get_ntsc_details, wait_for_ntsc_state
from tests import command as cmd
from tests.api_utils import determined_test_session, kill_ntsc, launch_ntsc
from tests.cluster.test_users import ADMIN_CREDENTIALS


@pytest.mark.slow
@pytest.mark.e2e_cpu
def test_basic_notebook_start_and_kill() -> None:
    lines = []  # type: List[str]
    with cmd.interactive_command("notebook", "start") as notebook:
        for line in notebook.stdout:
            if re.search("Jupyter Notebook .*is running at", line) is not None:
                return
            lines.append(line)

    raise ValueError(lines)


@pytest.mark.e2e_cpu
def test_notebook_proxy() -> None:
    session = determined_test_session(ADMIN_CREDENTIALS)

    def get_proxy(session: api.Session, task_id: str):
        url = conf.make_master_url(f"proxy/{task_id}/")
        print(f"getting {url}")
        session.get(url)

    typ = NTSC_Kind.notebook
    created_id = launch_ntsc(session, 1, typ)
    print(f"created {typ} {created_id}")
    wait_for_ntsc_state(
        session,
        NTSC_Kind(typ),
        created_id,
        lambda s: s == bindings.taskv1State.RUNNING,
        timeout=300,
    )
    deets = get_ntsc_details(session, typ, created_id)
    assert deets.state == bindings.taskv1State.RUNNING, f"{typ} should be running"
    err = api.task_is_ready(determined_test_session(ADMIN_CREDENTIALS), created_id)
    assert err is None, f"{typ} should be ready {err}"
    print(deets)
    try:
        get_proxy(session, created_id)
    finally:
        kill_ntsc(session, typ, created_id)
