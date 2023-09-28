import re
import uuid
from time import sleep

import pytest

from determined.common.api import Session, bindings, errors
from determined.common.api.bindings import experimentv1State
from tests import api_utils, command
from tests import config as conf
from tests import experiment as exp

GID, GROUPNAME = 1234, "group1234"


# TODO(ilia): Add this utility to Python SDK.
def _delete_workspace_and_check(
    sess: Session, w: bindings.v1Workspace, max_ticks: int = 60
) -> None:
    resp = bindings.delete_DeleteWorkspace(sess, id=w.id)
    if resp.completed:
        return

    for _ in range(max_ticks):
        sleep(1)
        try:
            w = bindings.get_GetWorkspace(sess, id=w.id).workspace
            if w.state == bindings.v1WorkspaceState.DELETE_FAILED:
                raise errors.DeleteFailedException(w.errorMessage)
            elif w.state == bindings.v1WorkspaceState.DELETING:
                continue
        except errors.NotFoundException:
            break


def _check_test_command(workspace_name: str) -> None:
    with command.interactive_command(
        "cmd", "run", "-w", workspace_name, "bash", "-c", "echo $(id -g -n):$(id -g)"
    ) as cmd:
        for line in cmd.stdout:
            if f"{GROUPNAME}:{GID}" in line:
                break
        else:
            raise AssertionError(f"Did not find {GROUPNAME}:{GID} in output")


def _check_test_experiment(project_id: int) -> None:
    # Create an experiment in that project.
    test_exp_id = exp.create_experiment(
        conf.fixtures_path("core_api/whoami.yaml"),
        conf.fixtures_path("core_api"),
        ["--project_id", str(project_id)],
    )
    exp.wait_for_experiment_state(
        test_exp_id,
        experimentv1State.COMPLETED,
    )

    trials = exp.experiment_trials(test_exp_id)
    trial_id = trials[0].trial.id
    trial_logs = exp.trial_logs(trial_id)

    marker = "id output: "
    for line in trial_logs:
        if marker in line:
            id_output = line[line.index(marker) + len(marker) :]
            match = re.match(r"uid=(\d+)\((\w+)\).+?gid=(\d+)\((\w+)\)", id_output)
            if match is None:
                pytest.fail("failed to parse id output")

            uid, username, gid, groupname = match.groups()
            assert int(gid) == GID
            assert groupname == GROUPNAME
            break
    else:
        pytest.fail("failed to find id output")


@pytest.mark.e2e_cpu
def test_workspace_post_gid() -> None:
    sess = api_utils.determined_test_session(admin=True)

    # Make project with workspace.
    resp_w = bindings.post_PostWorkspace(
        sess,
        body=bindings.v1PostWorkspaceRequest(
            name=f"workspace_aug_{uuid.uuid4().hex[:8]}",
            agentUserGroup=bindings.v1AgentUserGroup(agentGid=GID, agentGroup=GROUPNAME),
        ),
    )
    w = resp_w.workspace

    try:
        resp_p = bindings.post_PostProject(
            sess,
            body=bindings.v1PostProjectRequest(
                name="workspace_aug_1_project_1",
                workspaceId=w.id,
            ),
            workspaceId=w.id,
        )
        p = resp_p.project

        _check_test_experiment(p.id)
        _check_test_command(w.name)
    finally:
        _delete_workspace_and_check(sess, w)


@pytest.mark.e2e_cpu
def test_workspace_patch_gid() -> None:
    sess = api_utils.determined_test_session(admin=True)

    # Make project with workspace.
    resp_w = bindings.post_PostWorkspace(
        sess, body=bindings.v1PostWorkspaceRequest(name=f"workspace_aug_{uuid.uuid4().hex[:8]}")
    )
    w = resp_w.workspace

    try:
        bindings.patch_PatchWorkspace(
            sess,
            body=bindings.v1PatchWorkspace(
                name=w.name,
                agentUserGroup=bindings.v1AgentUserGroup(
                    agentGid=GID,
                    agentGroup=GROUPNAME,
                ),
            ),
            id=w.id,
        )

        resp_p = bindings.post_PostProject(
            sess,
            body=bindings.v1PostProjectRequest(
                name="workspace_aug_1_project_1",
                workspaceId=w.id,
            ),
            workspaceId=w.id,
        )
        p = resp_p.project

        _check_test_experiment(p.id)
        _check_test_command(w.name)
    finally:
        _delete_workspace_and_check(sess, w)


@pytest.mark.e2e_cpu
def test_workspace_partial_patch() -> None:
    # TODO(ilia): Implement better partial patch with fieldmasks.
    # This may need a changes to the way python bindings generate json payloads.
    sess = api_utils.determined_test_session(admin=True)

    # Make project with workspace.
    resp_w = bindings.post_PostWorkspace(
        sess,
        body=bindings.v1PostWorkspaceRequest(
            name=f"workspace_aug_{uuid.uuid4().hex[:8]}",
            agentUserGroup=bindings.v1AgentUserGroup(agentGid=GID, agentGroup=GROUPNAME),
        ),
    )

    w = resp_w.workspace
    new_name = w.name + " but new"

    try:
        # Does not reset AUG.
        resp_patch = bindings.patch_PatchWorkspace(
            sess,
            body=bindings.v1PatchWorkspace(
                name=new_name,
            ),
            id=w.id,
        )
        assert resp_patch.workspace.name == new_name
        assert resp_patch.workspace.agentUserGroup
        assert resp_patch.workspace.agentUserGroup.agentGid == GID
    finally:
        _delete_workspace_and_check(sess, w)
