import contextlib
import http
import os
import random
import re
import tempfile
import uuid
from typing import Generator, List, Optional, Tuple

import pytest

from determined.common import api
from determined.common.api import bindings, errors
from tests import api_utils
from tests import config as conf
from tests import detproc
from tests import experiment as exp
from tests.cluster import test_agent_user_group


@pytest.mark.e2e_cpu
def test_workspace_org() -> None:
    sess = api_utils.user_session()
    admin_sess = api_utils.admin_session()

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
        test_exp_id = exp.run_basic_test(
            sess, conf.fixtures_path("no_op/single.yaml"), conf.fixtures_path("no_op"), 1
        )
        test_exp = bindings.get_GetExperiment(sess, experimentId=test_exp_id).experiment
        test_experiments.append(test_exp)
        exp.wait_for_experiment_state(sess, test_exp_id, bindings.experimentv1State.COMPLETED)
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
        with pytest.raises(errors.APIException) as e:
            r8 = bindings.post_PostWorkspace(
                sess, body=bindings.v1PostWorkspaceRequest(name="_TestDuplicate")
            )
            failed_duplicate_workspace = r8.workspace
            assert failed_duplicate_workspace is None
            if failed_duplicate_workspace is not None:
                test_workspaces.append(failed_duplicate_workspace)
        assert e.value.status_code == http.HTTPStatus.CONFLICT

        # Refuse to change a workspace name to an existing name
        r9 = bindings.post_PostWorkspace(
            sess, body=bindings.v1PostWorkspaceRequest(name="_TestDuplicatePatch")
        )
        duplicate_patch_workspace = r9.workspace
        assert duplicate_patch_workspace is not None
        test_workspaces.append(duplicate_patch_workspace)
        w_patch = bindings.v1PatchWorkspace.from_json(made_workspace.to_json())
        w_patch.name = "_TestDuplicate"
        with pytest.raises(errors.APIException) as e:
            bindings.patch_PatchWorkspace(sess, body=w_patch, id=made_workspace.id)
        assert e.value.status_code == http.HTTPStatus.CONFLICT

    finally:
        # Clean out workspaces and all dependencies.
        for w in test_workspaces:
            bindings.delete_DeleteWorkspace(sess, id=w.id)


@pytest.mark.e2e_cpu
@pytest.mark.parametrize("file_type", ["json", "yaml"])
def test_workspace_checkpoint_storage_file(file_type: str) -> None:
    sess = api_utils.admin_session()
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

        detproc.check_call(
            sess, ["det", "workspace", "create", w_name, "--checkpoint-storage-config-file", path]
        )

    try:
        w_id = detproc.check_json(sess, ["det", "workspace", "describe", w_name, "--json"])["id"]
        w = bindings.get_GetWorkspace(sess, id=w_id).workspace
        assert w.checkpointStorageConfig is not None
        assert w.checkpointStorageConfig["type"] == "shared_fs"
        assert w.checkpointStorageConfig["host_path"] == "/tmp/" + file_type
    finally:
        test_agent_user_group._delete_workspace_and_check(sess, w)


@pytest.mark.e2e_cpu
def test_reset_workspace_checkpoint_storage_conf() -> None:
    sess = api_utils.admin_session()

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
        test_agent_user_group._delete_workspace_and_check(sess, resp_w.workspace)


@contextlib.contextmanager
def setup_workspaces(
    session: Optional[api.Session] = None, count: int = 1
) -> Generator[List[bindings.v1Workspace], None, None]:
    session = session or api_utils.admin_session()
    assert session
    workspaces: List[bindings.v1Workspace] = []
    try:
        for _ in range(count):
            body = bindings.v1PostWorkspaceRequest(name=f"workspace_{uuid.uuid4().hex[:8]}")
            workspaces.append(bindings.post_PostWorkspace(session, body=body).workspace)

        yield workspaces

    finally:
        # kill child jobs before deletion. NTSC is handled by the workspace deletion request.
        wids = {w.id for w in workspaces}
        exps = bindings.get_GetExperiments(session).experiments
        for e in exps:
            if e.workspaceId not in wids:
                continue
            bindings.post_KillExperiment(session, id=e.id)

        for w in workspaces:
            # TODO check if it needs deleting.
            bindings.delete_DeleteWorkspace(session, id=w.id)


TERMINATING_STATES = [
    bindings.taskv1State.TERMINATED,
    bindings.taskv1State.TERMINATING,
]


# tag: no-cli
@pytest.mark.e2e_cpu
def test_workspace_delete_notebook() -> None:
    admin_session = api_utils.admin_session()

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
    api.wait_for_ntsc_state(
        admin_session,
        api.NTSC_Kind.notebook,
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
    with pytest.raises(errors.NotFoundException):
        notebooks_resp = bindings.get_GetNotebooks(
            admin_session, workspaceId=workspace_resp.workspace.id
        )


# tag: no_cli
@pytest.mark.e2e_cpu
def test_launch_in_archived() -> None:
    admin_session = api_utils.admin_session()

    with setup_workspaces(admin_session) as [workspace]:
        # archive the workspace
        bindings.post_ArchiveWorkspace(
            admin_session,
            id=workspace.id,
        )

        # create a notebook inside the workspace
        with pytest.raises(errors.NotFoundException):
            bindings.post_LaunchNotebook(
                admin_session,
                body=bindings.v1LaunchNotebookRequest(workspaceId=workspace.id),
            )


# tag: no_cli
@pytest.mark.e2e_cpu
def test_workspaceid_set() -> None:
    admin_session = api_utils.admin_session()

    with setup_workspaces(admin_session) as [workspace]:
        # create a command inside the workspace
        cmd = bindings.post_LaunchCommand(
            admin_session,
            body=bindings.v1LaunchCommandRequest(workspaceId=workspace.id),
        ).command
        assert cmd.workspaceId == workspace.id

        cmd = bindings.get_GetCommand(admin_session, commandId=cmd.id).command
        assert cmd.workspaceId == workspace.id


@pytest.mark.e2e_cpu_rbac
@pytest.mark.e2e_cpu
def test_workspace_members() -> None:
    """set up workspace with users, user-groups, and roles, and test list-member cli command"""
    test_user: List[str] = []
    test_groups: List[str] = []
    test_exp: List[str] = ["User/Group Name | User/Group | Role Name"]

    try:
        admin_sess = api_utils.admin_session()

        # Add a test workspace.
        workspace_name = api_utils.get_random_string()
        create_workspace_cmd = ["det", "workspace", "create", workspace_name]
        detproc.check_call(admin_sess, create_workspace_cmd)

        # Create test users (3) and groups (2) and assign to workspace.
        roles = ["Editor", "Viewer", "WorkspaceAdmin", "EditorRestricted", "ModelRegistryViewer"]
        count = 3
        for _ in range(count):
            user_name = api_utils.get_random_string()
            detproc.check_call(
                admin_sess, ["det", "user", "create", user_name, "--password", "Test@123"]
            )
            test_user.append(user_name)

            role_name = random.choice(roles)
            api_utils.assign_user_role(admin_sess, user_name, role_name, workspace_name)
            test_exp.append(f"{user_name} | U | {role_name}")

        count = 2
        for _ in range(count):
            group_name = api_utils.get_random_string()
            detproc.check_call(admin_sess, ["det", "user-group", "create", group_name])
            test_groups.append(group_name)

            role_name = random.choice(roles)
            api_utils.assign_group_role(admin_sess, group_name, role_name, workspace_name)
            test_exp.append(f"{group_name} | G | {role_name}")

        # List the members, and test the cli tabular output
        list_members_tab = detproc.check_output(
            admin_sess, ["det", "workspace", "list-members", workspace_name]
        )

        # Split the table output into lines
        lines = list_members_tab.strip().split("\n")
        # Process each line to remove extra whitespace and rejoin with a single space
        result_members = [" ".join(line.split()) for line in lines]

        assert all(i in result_members for i in test_exp)

    finally:
        # Clean out workspaces and all dependencies.
        for user_name in test_user:
            detproc.check_call(admin_sess, ["det", "user", "deactivate", user_name])

        for group_name in test_groups:
            detproc.check_call(admin_sess, ["det", "user-group", "delete", "--yes", group_name])

        detproc.check_call(admin_sess, ["det", "workspace", "delete", "--yes", workspace_name])


@pytest.mark.e2e_multi_k8s
@pytest.mark.e2e_single_k8s
def test_set_workspace_namespace_bindings(
    is_multirm_cluster: bool, namespaces_created: Tuple[str, str]
) -> None:
    # Create a workspace.
    sess = api_utils.admin_session()
    w_name = uuid.uuid4().hex[:8]
    detproc.check_call(sess, ["det", "w", "create", w_name])

    bound_to_namespace = "bound to namespace"
    namespace, _ = namespaces_created

    # Valid namespace name, invalid cluster name.
    nonexistent_cluster = "nonexistentrm"
    detproc.check_error(
        sess,
        [
            "det",
            "w",
            "bindings",
            "set",
            w_name,
            "--cluster-name",
            nonexistent_cluster,
            "--namespace",
            namespace,
        ],
        "no resource manager with cluster name",
    )

    w_name = uuid.uuid4().hex[:8]
    detproc.check_error(
        sess,
        [
            "det",
            "w",
            "create",
            w_name,
            "--cluster-name",
            nonexistent_cluster,
            "--namespace",
            namespace,
        ],
        "no resource manager with cluster name",
    )

    # The following test commands should fail for multirm but succeed for single kubernetes rm.
    #   * Valid namespace name, no cluster name.
    #   * Set resource quota when --auto-create-namespace-all-clusters is specified.
    if is_multirm_cluster:
        detproc.check_error(
            sess,
            ["det", "w", "create", w_name, "--namespace", namespace],
            "must specify a cluster name",
        )

        detproc.check_error(
            sess,
            [
                "det",
                "w",
                "create",
                w_name,
                "--auto-create-namespace-all-clusters",
                "--resource-quota",
                "10",
            ],
            "When using multiple resource managers, cannot set a resource quota when you request "
            + "to auto-create a namespace for all clusters.",
        )

        detproc.check_call(sess, ["det", "w", "create", w_name])

        detproc.check_error(
            sess,
            ["det", "w", "bindings", "set", w_name, "--namespace", namespace],
            "must specify a cluster name",
        )

    else:
        output = detproc.check_output(
            sess,
            ["det", "w", "create", uuid.uuid4().hex[:8], "--namespace", namespace],
        )
        assert bound_to_namespace in output

        w_name = uuid.uuid4().hex[:8]
        output = detproc.check_output(
            sess,
            [
                "det",
                "w",
                "create",
                w_name,
                "--auto-create-namespace-all-clusters",
                "--resource-quota",
                "2",
            ],
        )
        assert bound_to_namespace in output

        output = detproc.check_output(
            sess,
            ["det", "w", "bindings", "set", w_name, "--namespace", namespace],
        )
        assert bound_to_namespace in output

        detproc.check_error(
            sess,
            ["det", "w", "resource-quota", "set", w_name, "5"],
            "cannot set quota on a workspace that is not bound to an auto-created namespace",
        )

    if is_multirm_cluster:
        w_name = uuid.uuid4().hex[:8]
        detproc.check_call(sess, ["det", "w", "create", w_name])

        # MultiRM: Valid cluster name, no namespace name.
        set_binding_cmd = ["det", "w", "bindings", "set", w_name]
        create_wksp_with_binding_cmd = ["det", "w", "create", w_name]

        set_binding_cmd += ["--cluster-name", conf.DEFAULT_RM_CLUSTER_NAME]
        create_wksp_with_binding_cmd += ["--cluster-name", conf.DEFAULT_RM_CLUSTER_NAME]

        detproc.check_error(
            sess,
            set_binding_cmd,
            "must provide --namespace",
        )

    w_name = uuid.uuid4().hex[:8]
    detproc.check_error(
        sess,
        ["det", "w", "create", w_name]
        + ["--auto-create-namespace-all-clusters", "--cluster-name", "defaultrm"],
        "cannot specify a cluster name when you request to auto-create a namespace for "
        + "all clusters",
    )

    detproc.check_call(sess, ["det", "w", "create", w_name])

    # Workspace-namespace binding with no specifed namespace and no auto-create namespace.
    detproc.check_error(
        sess,
        ["det", "w", "bindings", "set", w_name],
        "must provide --namespace NAMESPACE or --auto-create-namespace, or specify "
        + "--auto-create-namespace-all-clusters",
    )

    # Auto-create namespace for all clusters and valid cluster name set.
    detproc.check_error(
        sess,
        ["det", "w", "bindings", "set", w_name]
        + ["--auto-create-namespace-all-clusters", "--cluster-name", "additionalrm"],
        "cannot specify a cluster name when you request to auto-create a namespace for "
        + "all clusters",
    )

    # MultiRM: Valid cluster name, invalid namespace name.
    # Single KubernetesRM: No cluster name, invalid namespace name.
    nonexistent_namespace = "nonexistent-namespace"

    w_name = uuid.uuid4().hex[:8]
    set_binding_cmd = ["det", "w", "bindings", "set", w_name]
    create_wksp_cmd = ["det", "w", "create", w_name]
    if is_multirm_cluster:
        set_binding_cmd += ["--cluster-name", conf.DEFAULT_RM_CLUSTER_NAME]
        create_wksp_cmd += ["--cluster-name", conf.DEFAULT_RM_CLUSTER_NAME]
    detproc.check_error(
        sess,
        create_wksp_cmd + ["--namespace", nonexistent_namespace],
        "error finding namespace",
    )
    create_wksp_cmd = create_wksp_cmd[0:4]
    detproc.check_call(sess, create_wksp_cmd)
    detproc.check_error(
        sess,
        set_binding_cmd + ["--namespace", nonexistent_namespace],
        "error finding namespace",
    )

    # Valid namespace name and valid cluster name.
    w_name = uuid.uuid4().hex[:8]
    set_binding_cmd = ["det", "w", "bindings", "set", w_name]
    create_wksp_cmd = ["det", "w", "create", w_name]
    if is_multirm_cluster:
        set_binding_cmd += ["--cluster-name", conf.DEFAULT_RM_CLUSTER_NAME]
        create_wksp_cmd += ["--cluster-name", conf.DEFAULT_RM_CLUSTER_NAME]

    output = detproc.check_output(sess, create_wksp_cmd + ["--namespace", namespace])
    assert bound_to_namespace in output

    output = detproc.check_output(sess, set_binding_cmd + ["--namespace", namespace])
    assert bound_to_namespace in output

    # MultiRM: Valid cluster name, no namespace name, auto-create namespace & resource_quota.
    if is_multirm_cluster:
        w_name = uuid.uuid4().hex[:8]
        output = detproc.check_output(
            sess,
            [
                "det",
                "w",
                "create",
                w_name,
                "--cluster-name",
                conf.DEFAULT_RM_CLUSTER_NAME,
                "--auto-create-namespace",
                "--resource-quota",
                "1",
            ],
        )
        assert bound_to_namespace in output

        output = detproc.check_output(
            sess,
            [
                "det",
                "w",
                "resource-quota",
                "set",
                w_name,
                "5",
                "--cluster-name",
                conf.DEFAULT_RM_CLUSTER_NAME,
            ],
        )
        assert re.search(r"Resource quota .* is set on workspace", output)

        detproc.check_error(
            sess,
            [
                "det",
                "w",
                "resource-quota",
                "set",
                w_name,
                "-5",
                "--cluster-name",
                conf.DEFAULT_RM_CLUSTER_NAME,
            ],
            "must be greater than or equal to 0",
        )

    # SingleRM: No cluster name, no namespace name, auto-create namespace & resource quota.
    else:
        w_name = uuid.uuid4().hex[:8]
        output = detproc.check_output(
            sess,
            [
                "det",
                "w",
                "create",
                w_name,
                "--auto-create-namespace",
                "--resource-quota",
                "1",
            ],
        )
        assert bound_to_namespace in output

        output = detproc.check_output(
            sess,
            [
                "det",
                "w",
                "resource-quota",
                "set",
                w_name,
                "5",
            ],
        )
        assert re.search(r"Resource quota .* is set on workspace", output)

    # MultiRM & SingleRM: fail to set resource quota on a workspace without a namespace binding.
    w_name = uuid.uuid4().hex[:8]
    detproc.check_error(
        sess,
        [
            "det",
            "w",
            "create",
            w_name,
            "--resource-quota",
            "1",
            "--cluster-name",
            conf.DEFAULT_RM_CLUSTER_NAME,
        ],
        "Failed to create workspace: must provide --namespace NAMESPACE or --auto-create-namespace",
    )


@pytest.mark.e2e_multi_k8s
@pytest.mark.e2e_single_k8s
def test_delete_workspace_namespace_bindings(
    is_multirm_cluster: bool, namespaces_created: Tuple[str, str]
) -> None:
    namespace, _ = namespaces_created
    success = "Successfully deleted binding."

    # Create a workspace with a binding.
    sess = api_utils.admin_session()
    w_name = uuid.uuid4().hex[:8]
    detproc.check_call(
        sess,
        [
            "det",
            "w",
            "create",
            w_name,
            "--cluster-name",
            conf.DEFAULT_RM_CLUSTER_NAME,
            "--namespace",
            namespace,
        ],
    )

    # Invalid cluster name.
    nonexistent_cluster = "nonexistentrm"
    detproc.check_error(
        sess,
        [
            "det",
            "w",
            "bindings",
            "delete",
            w_name,
            "--cluster-name",
            nonexistent_cluster,
        ],
        "no resource manager with cluster name",
    )

    # no cluster name. (Should fail for multirm but work for single kubernetes rm).
    if is_multirm_cluster:
        detproc.check_error(
            sess,
            ["det", "w", "bindings", "delete", w_name],
            "must specify a cluster name",
        )
    else:
        output = detproc.check_output(
            sess,
            ["det", "w", "bindings", "delete", w_name],
        )
        assert success in output

    # valid cluster name.

    # reset binding
    detproc.check_call(
        sess,
        [
            "det",
            "w",
            "bindings",
            "set",
            w_name,
            "--cluster-name",
            conf.DEFAULT_RM_CLUSTER_NAME,
            "--namespace",
            namespace,
        ],
    )
    output = detproc.check_output(
        sess,
        ["det", "w", "bindings", "delete", w_name, "--cluster-name", conf.DEFAULT_RM_CLUSTER_NAME],
    )
    assert success in output

    # Now that binding is deleted, try deleting default binding
    detproc.check_error(
        sess,
        ["det", "w", "bindings", "delete", w_name, "--cluster-name", conf.DEFAULT_RM_CLUSTER_NAME],
        "tried to delete default binding for cluster " + conf.DEFAULT_RM_CLUSTER_NAME,
    )


@pytest.mark.e2e_multi_k8s
@pytest.mark.e2e_single_k8s
def test_list_workspace_namespace_bindings(
    is_multirm_cluster: bool, namespaces_created: Tuple[str, str]
) -> None:
    # Create a workspace.
    w_name = uuid.uuid4().hex[:8]
    sess = api_utils.admin_session()
    detproc.check_call(sess, ["det", "w", "create", w_name])

    # List workspace-namespace bindings and ensure that they are set to the default namespace.
    cmd = ["det", "w", "bindings", "list", w_name]
    output = detproc.check_output(sess, cmd)

    cluster_namespace_pair = {
        conf.DEFAULT_RM_CLUSTER_NAME: conf.DEFAULT_KUBERNETES_NAMESPACE,
        conf.ADDITIONAL_RM_CLUSTER_NAME: conf.DEFAULT_KUBERNETES_NAMESPACE,
    }
    if not is_multirm_cluster:
        default_namespace = detproc.check_json(sess, ["det", "master", "config", "show", "--json"])[
            "resource_manager"
        ]["default_namespace"]
        cluster_namespace_pair = {"": default_namespace}

    for cluster in cluster_namespace_pair:
        assert cluster in output
        assert cluster_namespace_pair[cluster] in output

    defaultrm_namespace, additionalrm_namespace = namespaces_created

    # Modify workspace-namespace bindings for each cluster.
    set_binding_cmd = ["det", "w", "bindings", "set", w_name]
    if is_multirm_cluster:
        detproc.check_call(
            sess,
            [
                "det",
                "w",
                "bindings",
                "set",
                w_name,
                "--namespace",
                additionalrm_namespace,
                "--cluster-name",
                conf.ADDITIONAL_RM_CLUSTER_NAME,
            ],
        )
        set_binding_cmd += ["--cluster-name", conf.DEFAULT_RM_CLUSTER_NAME]

    detproc.check_call(sess, set_binding_cmd + ["--namespace", defaultrm_namespace])

    # List workspace-namespace bindings after setting a namespace other than default.
    cmd = ["det", "w", "bindings", "list", w_name]
    output = detproc.check_output(sess, cmd)

    cluster_namespace_pair = {"": defaultrm_namespace}

    if is_multirm_cluster:
        cluster_namespace_pair = {
            "defaultrm": defaultrm_namespace,
            "additionalrm": additionalrm_namespace,
        }

    for cluster in cluster_namespace_pair:
        assert cluster in output
        assert cluster_namespace_pair[cluster] in output
