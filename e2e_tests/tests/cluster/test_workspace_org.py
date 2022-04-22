from typing import List

import pytest

from determined.common.api import authentication, bindings, certs, errors
from determined.common.experimental import session
from tests import config as conf
from tests.experiment import run_basic_test, wait_for_experiment_state


@pytest.mark.e2e_cpu
def test_workspace_org() -> None:
    master_url = conf.make_master_url()
    certs.cli_cert = certs.default_load(master_url)
    authentication.cli_auth = authentication.Authentication(master_url, try_reauth=True)
    sess = session.Session(master_url, "determined", authentication.cli_auth, certs.cli_cert)

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
            bindings.post_ArchiveWorkspace(sess, id=default_workspace.id)
        with pytest.raises(errors.APIException):
            bindings.post_UnarchiveWorkspace(sess, id=default_workspace.id)
        with pytest.raises(errors.APIException):
            bindings.delete_DeleteWorkspace(sess, id=default_workspace.id)

        # Sort test and default workspaces.
        workspace2 = bindings.post_PostWorkspace(
            sess, body=bindings.v1PostWorkspaceRequest(name="_TestWS")
        ).workspace
        test_workspaces.append(workspace2)
        list_test_1 = bindings.get_GetWorkspaces(sess).workspaces
        assert ["Uncategorized", "_TestPatched", "_TestWS"] == list(
            map(lambda w: w.name, list_test_1)
        )
        list_test_2 = bindings.get_GetWorkspaces(
            sess, orderBy=bindings.v1OrderBy.ORDER_BY_DESC
        ).workspaces
        assert ["_TestWS", "_TestPatched", "Uncategorized"] == list(
            map(lambda w: w.name, list_test_2)
        )
        list_test_3 = bindings.get_GetWorkspaces(
            sess, sortBy=bindings.v1GetWorkspacesRequestSortBy.SORT_BY_NAME
        ).workspaces
        assert ["_TestPatched", "_TestWS", "Uncategorized"] == list(
            map(lambda w: w.name, list_test_3)
        )

        # Test pinned workspaces.
        bindings.post_PinWorkspace(sess, id=made_workspace.id)
        pinned = bindings.get_GetWorkspaces(
            sess,
            pinned=True,
        ).workspaces
        assert len(pinned) == 1 and pinned[0].id == made_workspace.id
        bindings.post_UnpinWorkspace(sess, id=made_workspace.id)
        pinned = bindings.get_GetWorkspaces(
            sess,
            pinned=True,
        ).workspaces
        assert len(pinned) == 0

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
            bindings.post_ArchiveProject(sess, id=default_project.id)
        with pytest.raises(errors.APIException):
            bindings.post_UnarchiveProject(sess, id=default_project.id)
        with pytest.raises(errors.APIException):
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
        assert ["_TestPatchedProject", "_TestPRJ", "_TestEarly"] == list(
            map(lambda w: w.name, list_test_4)
        )
        list_test_5 = bindings.get_GetWorkspaceProjects(
            sess, id=made_workspace.id, orderBy=bindings.v1OrderBy.ORDER_BY_DESC
        ).projects
        assert ["_TestEarly", "_TestPRJ", "_TestPatchedProject"] == list(
            map(lambda w: w.name, list_test_5)
        )
        list_test_6 = bindings.get_GetWorkspaceProjects(
            sess,
            id=made_workspace.id,
            sortBy=bindings.v1GetWorkspaceProjectsRequestSortBy.SORT_BY_NAME,
        ).projects
        assert ["_TestEarly", "_TestPatchedProject", "_TestPRJ"] == list(
            map(lambda w: w.name, list_test_6)
        )

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
                sess,
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

        # Create an experiment in the default project and move to a test project.
        test_exp_id = run_basic_test(
            conf.fixtures_path("no_op/single.yaml"), conf.fixtures_path("no_op"), 1
        )
        test_exp = bindings.get_GetExperiment(sess, experimentId=test_exp_id).experiment
        test_experiments.append(test_exp)
        wait_for_experiment_state(test_exp_id, bindings.determinedexperimentv1State.STATE_COMPLETED)
        assert test_exp.projectId == default_project.id

        # Moving an experiment out of the default project
        mbody = bindings.v1MoveExperimentRequest(
            destinationProjectId=made_project.id, experimentId=test_exp_id
        )
        bindings.post_MoveExperiment(sess, experimentId=test_exp_id, body=mbody)
        modified_exp = bindings.get_GetExperiment(sess, experimentId=test_exp_id).experiment
        assert modified_exp.projectId == made_project.id

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

    finally:
        # Clean out experiments, projects, workspaces.
        # In dependency order:
        for e in test_experiments:
            bindings.delete_DeleteExperiment(sess, experimentId=e.id)
        for p in test_projects:
            bindings.delete_DeleteProject(sess, id=p.id)
        for w in test_workspaces:
            bindings.delete_DeleteWorkspace(sess, id=w.id)
