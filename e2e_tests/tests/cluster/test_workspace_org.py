import os
import tempfile
import uuid
from http import HTTPStatus
from typing import List

import pytest

from determined.common import api
from determined.common.api import authentication, bindings, errors
from determined.common.api._util import NTSC_Kind, wait_for_ntsc_state
from determined.common.api.errors import APIException
from tests import api_utils
from tests import config as conf
from tests.api_utils import ADMIN_CREDENTIALS
from tests.cluster.test_users import change_user_password, logged_in_user
from tests.cluster.utils import setup_workspaces
from tests.experiment import run_basic_test, wait_for_experiment_state
from tests.utils import det_cmd, det_cmd_json

from .test_agent_user_group import _delete_workspace_and_check


@pytest.mark.e2e_cpu
def test_workspace_org() -> None:
    with logged_in_user(ADMIN_CREDENTIALS):
        change_user_password("determined", "")
    master_url = conf.make_master_url()
    authentication.cli_auth = authentication.Authentication(master_url)
    sess = api.Session(master_url, None, None, None)
    admin_auth = authentication.Authentication(
        master_url, ADMIN_CREDENTIALS.username, ADMIN_CREDENTIALS.password
    )
    admin_sess = api.Session(master_url, ADMIN_CREDENTIALS.username, admin_auth, None)

    test_experiments: List[bindings.v1Experiment] = []
    test_projects: List[bindings.v1Project] = []
    test_workspaces: List[bindings.v1Workspace] = []

    try:
        # Uncategorized workspace / project should exist already.
        r = bindings.get_GetWorkspaces(sess, name="Uncategorized")
        assert len(r.workspaces) == 1
        default_workspace = r.workspaces[0]
        assert default_workspace.immutable
        r2 = bindings.get_GetWorkspaceProjects(sess, id=default_workspace.id)
        assert len(r2.projects) == 1
        default_project = r2.projects[0]
        assert default_project.name == "Uncategorized"
        assert default_project.immutable

        # Add a test workspace.
        r3 = bindings.post_PostWorkspace(
            sess, body=bindings.v1PostWorkspaceRequest(name="_TestOnly")
        )
        made_workspace = r3.workspace
        test_workspaces.append(made_workspace)
        get_workspace = bindings.get_GetWorkspace(sess, id=made_workspace.id).workspace
        assert get_workspace.name == made_workspace.name
        assert not made_workspace.immutable and not get_workspace.immutable

        # Patch the workspace
        w_patch = bindings.v1PatchWorkspace.from_json(made_workspace.to_json())
        w_patch.name = "_TestPatched"
        bindings.patch_PatchWorkspace(sess, body=w_patch, id=made_workspace.id)
        get_workspace = bindings.get_GetWorkspace(sess, id=made_workspace.id).workspace
        assert get_workspace.name == "_TestPatched"

        # Archive the workspace
        assert not made_workspace.archived
        bindings.post_ArchiveWorkspace(sess, id=made_workspace.id)
        get_workspace_2 = bindings.get_GetWorkspace(sess, id=made_workspace.id).workspace
        assert get_workspace_2.archived
        with pytest.raises(errors.APIException):
            # Cannot patch archived workspace
            bindings.patch_PatchWorkspace(sess, body=w_patch, id=made_workspace.id)
        with pytest.raises(errors.APIException):
            # Cannot create project inside archived workspace
            bindings.post_PostProject(
                sess,
                body=bindings.v1PostProjectRequest(name="Nope2", workspaceId=made_workspace.id),
                workspaceId=made_workspace.id,
            )
        bindings.post_UnarchiveWorkspace(sess, id=made_workspace.id)
        get_workspace_3 = bindings.get_GetWorkspace(sess, id=made_workspace.id).workspace
        assert not get_workspace_3.archived

        # Refuse to patch, archive, unarchive, or delete the default workspace
        with pytest.raises(errors.APIException):
            bindings.patch_PatchWorkspace(sess, body=w_patch, id=default_workspace.id)
        with pytest.raises(errors.APIException):
            bindings.post_ArchiveWorkspace(admin_sess, id=default_workspace.id)
        with pytest.raises(errors.APIException):
            bindings.post_UnarchiveWorkspace(admin_sess, id=default_workspace.id)
        with pytest.raises(errors.APIException):
            bindings.delete_DeleteWorkspace(admin_sess, id=default_workspace.id)

        # A non admin user gets forbidden trying to modify the default workspace.
        with pytest.raises(errors.ForbiddenException):
            bindings.post_ArchiveWorkspace(sess, id=default_workspace.id)
        with pytest.raises(errors.ForbiddenException):
            bindings.post_UnarchiveWorkspace(sess, id=default_workspace.id)
        with pytest.raises(errors.ForbiddenException):
            bindings.delete_DeleteWorkspace(sess, id=default_workspace.id)

        # Sort test and default workspaces.
        workspace2 = bindings.post_PostWorkspace(
            sess, body=bindings.v1PostWorkspaceRequest(name="_TestWS")
        ).workspace
        test_workspaces.append(workspace2)
        list_test_1 = bindings.get_GetWorkspaces(sess).workspaces
        assert ["Uncategorized", "_TestPatched", "_TestWS"] == [w.name for w in list_test_1]
        list_test_2 = bindings.get_GetWorkspaces(sess, orderBy=bindings.v1OrderBy.DESC).workspaces
        assert ["_TestWS", "_TestPatched", "Uncategorized"] == [w.name for w in list_test_2]
        list_test_3 = bindings.get_GetWorkspaces(
            sess, sortBy=bindings.v1GetWorkspacesRequestSortBy.NAME
        ).workspaces
        assert ["_TestPatched", "_TestWS", "Uncategorized"] == [w.name for w in list_test_3]

        # Test pinned workspaces.
        pinned = bindings.get_GetWorkspaces(
            sess,
            pinned=True,
        ).workspaces
        assert len(pinned) == 2
        bindings.post_UnpinWorkspace(sess, id=made_workspace.id)
        pinned = bindings.get_GetWorkspaces(
            sess,
            pinned=True,
        ).workspaces
        assert len(pinned) == 1
        bindings.post_PinWorkspace(sess, id=made_workspace.id)
        pinned = bindings.get_GetWorkspaces(
            sess,
            pinned=True,
        ).workspaces
        assert len(pinned) == 2

        # Add a test project to a workspace.
        r4 = bindings.post_PostProject(
            sess,
            body=bindings.v1PostProjectRequest(name="_TestOnly", workspaceId=made_workspace.id),
            workspaceId=made_workspace.id,
        )
        made_project = r4.project
        test_projects.append(made_project)
        get_project = bindings.get_GetProject(sess, id=made_project.id).project
        assert get_project.name == made_project.name
        assert not made_project.immutable and not get_project.immutable

        # Project cannot be created in the default workspace.
        with pytest.raises(errors.APIException):
            bindings.post_PostProject(
                sess,
                body=bindings.v1PostProjectRequest(name="Nope", workspaceId=default_workspace.id),
                workspaceId=default_workspace.id,
            )

        # Patch the project
        p_patch = bindings.v1PatchProject.from_json(made_project.to_json())
        p_patch.name = "_TestPatchedProject"
        bindings.patch_PatchProject(sess, body=p_patch, id=made_project.id)
        get_project = bindings.get_GetProject(sess, id=made_project.id).project
        assert get_project.name == "_TestPatchedProject"

        # Archive the project
        assert not made_project.archived
        bindings.post_ArchiveProject(sess, id=made_project.id)
        get_project_2 = bindings.get_GetProject(sess, id=made_project.id).project
        assert get_project_2.archived

        # Cannot patch or move an archived project
        with pytest.raises(errors.APIException):
            bindings.patch_PatchProject(sess, body=p_patch, id=made_project.id)
        with pytest.raises(errors.APIException):
            bindings.post_MoveProject(
                sess,
                projectId=made_project.id,
                body=bindings.v1MoveProjectRequest(
                    destinationWorkspaceId=workspace2.id,
                    projectId=made_project.id,
                ),
            )

        # Unarchive the project
        bindings.post_UnarchiveProject(sess, id=made_project.id)
        get_project_3 = bindings.get_GetProject(sess, id=made_project.id).project
        assert not get_project_3.archived

        # Can't archive, un-archive, or move while parent workspace is archived
        bindings.post_ArchiveWorkspace(sess, id=made_workspace.id)
        get_project_4 = bindings.get_GetProject(sess, id=made_project.id).project
        assert get_project_4.archived
        with pytest.raises(errors.APIException):
            bindings.post_ArchiveProject(sess, id=made_project.id)
        with pytest.raises(errors.APIException):
            bindings.post_UnarchiveProject(sess, id=made_project.id)
        with pytest.raises(errors.APIException):
            bindings.post_MoveProject(
                sess,
                projectId=made_project.id,
                body=bindings.v1MoveProjectRequest(
                    destinationWorkspaceId=workspace2.id,
                    projectId=made_project.id,
                ),
            )
        bindings.post_UnarchiveWorkspace(sess, id=made_workspace.id)

        # Refuse to patch, archive, unarchive, or delete the default project
        with pytest.raises(errors.APIException):
            bindings.patch_PatchProject(sess, body=p_patch, id=default_project.id)
        with pytest.raises(errors.APIException):
            bindings.post_ArchiveProject(admin_sess, id=default_project.id)
        with pytest.raises(errors.APIException):
            bindings.post_UnarchiveProject(admin_sess, id=default_project.id)
        with pytest.raises(errors.APIException):
            bindings.delete_DeleteProject(admin_sess, id=default_project.id)

        # A non admin user gets forbidden trying to modify the default project.
        with pytest.raises(errors.ForbiddenException):
            bindings.post_ArchiveProject(sess, id=default_project.id)
        with pytest.raises(errors.ForbiddenException):
            bindings.post_UnarchiveProject(sess, id=default_project.id)
        with pytest.raises(errors.ForbiddenException):
            bindings.delete_DeleteProject(sess, id=default_project.id)

        # Sort workspaces' projects.
        p1 = bindings.post_PostProject(
            sess,
            body=bindings.v1PostProjectRequest(name="_TestPRJ", workspaceId=made_workspace.id),
            workspaceId=made_workspace.id,
        ).project
        p2 = bindings.post_PostProject(
            sess,
            body=bindings.v1PostProjectRequest(name="_TestEarly", workspaceId=made_workspace.id),
            workspaceId=made_workspace.id,
        ).project
        test_projects += [p1, p2]
        list_test_4 = bindings.get_GetWorkspaceProjects(sess, id=made_workspace.id).projects
        assert ["_TestPatchedProject", "_TestPRJ", "_TestEarly"] == [p.name for p in list_test_4]
        list_test_5 = bindings.get_GetWorkspaceProjects(
            sess, id=made_workspace.id, orderBy=bindings.v1OrderBy.DESC
        ).projects
        assert ["_TestEarly", "_TestPRJ", "_TestPatchedProject"] == [p.name for p in list_test_5]
        list_test_6 = bindings.get_GetWorkspaceProjects(
            sess,
            id=made_workspace.id,
            sortBy=bindings.v1GetWorkspaceProjectsRequestSortBy.NAME,
        ).projects
        assert ["_TestEarly", "_TestPatchedProject", "_TestPRJ"] == [p.name for p in list_test_6]

        # Move a project to another workspace
        bindings.post_MoveProject(
            sess,
            projectId=made_project.id,
            body=bindings.v1MoveProjectRequest(
                destinationWorkspaceId=workspace2.id,
                projectId=made_project.id,
            ),
        )
        get_project = bindings.get_GetProject(sess, id=made_project.id).project
        assert get_project.workspaceId == workspace2.id

        # Default project cannot be moved.
        with pytest.raises(errors.APIException):
            bindings.post_MoveProject(
                admin_sess,
                projectId=default_project.id,
                body=bindings.v1MoveProjectRequest(
                    destinationWorkspaceId=workspace2.id,
                    projectId=default_project.id,
                ),
            )

        # Project cannot be moved into the default workspace.
        with pytest.raises(errors.APIException):
            bindings.post_MoveProject(
                sess,
                projectId=made_project.id,
                body=bindings.v1MoveProjectRequest(
                    destinationWorkspaceId=default_workspace.id,
                    projectId=made_project.id,
                ),
            )

        # Project cannot be moved into an archived workspace.
        bindings.post_ArchiveWorkspace(sess, id=made_workspace.id)
        with pytest.raises(errors.APIException):
            bindings.post_MoveProject(
                sess,
                projectId=made_project.id,
                body=bindings.v1MoveProjectRequest(
                    destinationWorkspaceId=made_workspace.id,
                    projectId=made_project.id,
                ),
            )
        bindings.post_UnarchiveWorkspace(sess, id=made_workspace.id)

        # Add a test note to a project.
        note = bindings.v1Note(name="Hello", contents="Hello World")
        note2 = bindings.v1Note(name="Hello 2", contents="Hello World")
        bindings.post_AddProjectNote(
            sess,
            body=note,
            projectId=made_project.id,
        )
        r5 = bindings.post_AddProjectNote(
            sess,
            body=note2,
            projectId=made_project.id,
        )
        returned_notes = r5.notes
        assert len(returned_notes) == 2

        # Put notes
        r6 = bindings.put_PutProjectNotes(
            sess,
            body=bindings.v1PutProjectNotesRequest(notes=[note], projectId=made_project.id),
            projectId=made_project.id,
        )
        returned_notes = r6.notes
        assert len(returned_notes) == 1

        # Create an experiment in the default project.
        test_exp_id = run_basic_test(
            conf.fixtures_path("no_op/single.yaml"), conf.fixtures_path("no_op"), 1
        )
        test_exp = bindings.get_GetExperiment(sess, experimentId=test_exp_id).experiment
        test_experiments.append(test_exp)
        wait_for_experiment_state(test_exp_id, bindings.experimentv1State.COMPLETED)
        assert test_exp.projectId == default_project.id

        # Move the test experiment into a user-made project
        dproj_exp = bindings.get_GetExperiments(sess, projectId=default_project.id).experiments
        exp_count = len(bindings.get_GetExperiments(sess, projectId=made_project.id).experiments)
        assert exp_count == 0
        mbody = bindings.v1MoveExperimentRequest(
            destinationProjectId=made_project.id, experimentId=test_exp_id
        )
        bindings.post_MoveExperiment(sess, experimentId=test_exp_id, body=mbody)
        modified_exp = bindings.get_GetExperiment(sess, experimentId=test_exp_id).experiment
        assert modified_exp.projectId == made_project.id

        # Confirm the test experiment is in the new project, no longer in old project.
        exp_count = len(bindings.get_GetExperiments(sess, projectId=made_project.id).experiments)
        assert exp_count == 1
        dproj_exp2 = bindings.get_GetExperiments(sess, projectId=default_project.id).experiments
        assert len(dproj_exp2) == len(dproj_exp) - 1

        # Cannot move an experiment out of an archived project
        bindings.post_ArchiveProject(sess, id=made_project.id)
        mbody2 = bindings.v1MoveExperimentRequest(
            destinationProjectId=default_project.id, experimentId=test_exp_id
        )
        with pytest.raises(errors.APIException):
            bindings.post_MoveExperiment(sess, experimentId=test_exp_id, body=mbody2)
        bindings.post_UnarchiveProject(sess, id=made_project.id)

        # Moving an experiment into default project
        bindings.post_MoveExperiment(sess, experimentId=test_exp_id, body=mbody2)

        # Cannot move an experiment into an archived project
        bindings.post_ArchiveProject(sess, id=made_project.id)
        with pytest.raises(errors.APIException):
            bindings.post_MoveExperiment(sess, experimentId=test_exp_id, body=mbody)

        # Refuse to create a workspace with a duplicate name
        r7 = bindings.post_PostWorkspace(
            sess, body=bindings.v1PostWorkspaceRequest(name="_TestDuplicate")
        )
        duplicate_workspace = r7.workspace
        assert duplicate_workspace is not None
        test_workspaces.append(duplicate_workspace)
        with pytest.raises(APIException) as e:
            r8 = bindings.post_PostWorkspace(
                sess, body=bindings.v1PostWorkspaceRequest(name="_TestDuplicate")
            )
            failed_duplicate_workspace = r8.workspace
            assert failed_duplicate_workspace is None
            if failed_duplicate_workspace is not None:
                test_workspaces.append(failed_duplicate_workspace)
        assert e.value.status_code == HTTPStatus.CONFLICT

        # Refuse to change a workspace name to an existing name
        r9 = bindings.post_PostWorkspace(
            sess, body=bindings.v1PostWorkspaceRequest(name="_TestDuplicatePatch")
        )
        duplicate_patch_workspace = r9.workspace
        assert duplicate_patch_workspace is not None
        test_workspaces.append(duplicate_patch_workspace)
        w_patch = bindings.v1PatchWorkspace.from_json(made_workspace.to_json())
        w_patch.name = "_TestDuplicate"
        with pytest.raises(APIException) as e:
            bindings.patch_PatchWorkspace(sess, body=w_patch, id=made_workspace.id)
        assert e.value.status_code == HTTPStatus.CONFLICT

    finally:
        # Clean out workspaces and all dependencies.
        for w in test_workspaces:
            bindings.delete_DeleteWorkspace(sess, id=w.id)


@pytest.mark.e2e_cpu
@pytest.mark.parametrize("file_type", ["json", "yaml"])
def test_workspace_checkpoint_storage_file(file_type: str) -> None:
    sess = api_utils.determined_test_session(admin=True)
    w_name = uuid.uuid4().hex[:8]
    with tempfile.TemporaryDirectory() as tmpdir:
        path = os.path.join(tmpdir, "config")
        with open(path, "w") as f:
            if file_type == "json":
                f.write('{"type":"shared_fs","host_path":"/tmp/json"}')
            else:
                f.write(
                    """
type: shared_fs
host_path: /tmp/yaml"""
                )

        det_cmd(
            ["workspace", "create", w_name, "--checkpoint-storage-config-file", path], check=True
        )

    try:
        w_id = det_cmd_json(["workspace", "describe", w_name, "--json"])["id"]
        w = bindings.get_GetWorkspace(sess, id=w_id).workspace
        assert w.checkpointStorageConfig is not None
        assert w.checkpointStorageConfig["type"] == "shared_fs"
        assert w.checkpointStorageConfig["host_path"] == "/tmp/" + file_type
    finally:
        _delete_workspace_and_check(sess, w)


@pytest.mark.e2e_cpu
def test_reset_workspace_checkpoint_storage_conf() -> None:
    sess = api_utils.determined_test_session(admin=True)

    # Make project with checkpoint storage config.
    resp_w = bindings.post_PostWorkspace(
        sess,
        body=bindings.v1PostWorkspaceRequest(
            name=f"workspace_aug_{uuid.uuid4().hex[:8]}",
            checkpointStorageConfig={"type": "shared_fs", "host_path": "/tmp"},
        ),
    )

    try:
        assert resp_w.workspace.checkpointStorageConfig is not None
        assert resp_w.workspace.checkpointStorageConfig["type"] == "shared_fs"
        assert resp_w.workspace.checkpointStorageConfig["host_path"] == "/tmp"

        # Reset storage config.
        resp_patch = bindings.patch_PatchWorkspace(
            sess,
            body=bindings.v1PatchWorkspace(
                checkpointStorageConfig={},
            ),
            id=resp_w.workspace.id,
        )
        assert resp_patch.workspace.checkpointStorageConfig is None
    finally:
        _delete_workspace_and_check(sess, resp_w.workspace)


TERMINATING_STATES = [
    bindings.taskv1State.TERMINATED,
    bindings.taskv1State.TERMINATING,
]


# tag: no-cli
@pytest.mark.e2e_cpu
def test_workspace_delete_notebook() -> None:
    admin_session = api_utils.determined_test_session(admin=True)

    # create a workspace using bindings

    workspace_resp = bindings.post_PostWorkspace(
        admin_session,
        body=bindings.v1PostWorkspaceRequest(
            name=f"workspace_{uuid.uuid4().hex[:8]}",
        ),
    )

    # create a notebook inside the workspace
    created_resp = bindings.post_LaunchNotebook(
        admin_session,
        body=bindings.v1LaunchNotebookRequest(workspaceId=workspace_resp.workspace.id),
    )

    # check that the notebook exists
    notebook_resp = bindings.get_GetNotebook(admin_session, notebookId=created_resp.notebook.id)
    assert notebook_resp.notebook.state not in TERMINATING_STATES

    # check that the notebook is returned in the list of notebooks
    notebooks_resp = bindings.get_GetNotebooks(
        admin_session, workspaceId=workspace_resp.workspace.id
    )
    nb = next((nb for nb in notebooks_resp.notebooks if nb.id == created_resp.notebook.id), None)
    assert nb is not None

    with setup_workspaces(admin_session) as [workspace2]:
        # create a notebook inside another workspace
        outside_notebook = bindings.post_LaunchNotebook(
            admin_session,
            body=bindings.v1LaunchNotebookRequest(workspaceId=workspace2.id),
        ).notebook

        # delete the workspace
        bindings.delete_DeleteWorkspace(admin_session, id=workspace_resp.workspace.id)

        # check that the other notebook is not terminated
        outside_notebook = bindings.get_GetNotebook(
            admin_session, notebookId=outside_notebook.id
        ).notebook
        assert outside_notebook.state not in TERMINATING_STATES

    # check that notebook is terminated or terminating.
    wait_for_ntsc_state(
        admin_session,
        NTSC_Kind.notebook,
        ntsc_id=created_resp.notebook.id,
        predicate=lambda state: state in TERMINATING_STATES,
    )
    notebook_resp = bindings.get_GetNotebook(admin_session, notebookId=created_resp.notebook.id)
    assert notebook_resp.notebook.state in TERMINATING_STATES

    # check that the notebook is not returned in the list of notebooks by default.
    notebooks_resp = bindings.get_GetNotebooks(admin_session)
    nb = next((nb for nb in notebooks_resp.notebooks if nb.id == created_resp.notebook.id), None)
    assert nb is None

    # the api returns a 404
    with pytest.raises(errors.APIException):
        notebooks_resp = bindings.get_GetNotebooks(
            admin_session, workspaceId=workspace_resp.workspace.id
        )


# tag: no_cli
@pytest.mark.e2e_cpu
def test_launch_in_archived() -> None:
    admin_session = api_utils.determined_test_session(admin=True)

    with setup_workspaces(admin_session) as [workspace]:
        # archive the workspace
        bindings.post_ArchiveWorkspace(
            admin_session,
            id=workspace.id,
        )

        # create a notebook inside the workspace
        with pytest.raises(errors.APIException) as e:
            bindings.post_LaunchNotebook(
                admin_session,
                body=bindings.v1LaunchNotebookRequest(workspaceId=workspace.id),
            )
        assert e.value.status_code == 404


# tag: no_cli
@pytest.mark.e2e_cpu
def test_workspaceid_set() -> None:
    admin_session = api_utils.determined_test_session(admin=True)

    with setup_workspaces(admin_session) as [workspace]:
        # create a command inside the workspace
        cmd = bindings.post_LaunchCommand(
            admin_session,
            body=bindings.v1LaunchCommandRequest(workspaceId=workspace.id),
        ).command
        assert cmd.workspaceId == workspace.id

        cmd = bindings.get_GetCommand(admin_session, commandId=cmd.id).command
        assert cmd.workspaceId == workspace.id
