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
        assert r.workspaces and len(r.workspaces) == 1
        default_workspace = r.workspaces[0]
        assert default_workspace.immutable
        r2 = bindings.get_GetWorkspaceProjects(sess, id=default_workspace.id)
        assert r2.projects and len(r2.projects) == 1
        default_project = r2.projects[0]
        assert default_project.name == "Uncategorized"
        assert default_project.immutable

        # Add a test workspace.
        r3 = bindings.post_PostWorkspace(
            sess, body=bindings.v1PostWorkspaceRequest(name="_TestOnly")
        )
        made_workspace = r3.workspace
        assert made_workspace is not None
        test_workspaces.append(made_workspace)
        get_workspace = bindings.get_GetWorkspace(sess, id=made_workspace.id).workspace
        assert get_workspace and get_workspace.name == "_TestOnly"
        assert not madeWorkspace.immutable and not get_workspace.immutable

        # Patch the workspace
        w_patch = bindings.v1PatchWorkspace.from_json(made_workspace.to_json())
        w_patch.name = "_TestPatched"
        bindings.patch_PatchWorkspace(sess, body=w_patch, id=made_workspace.id)
        get_workspace = bindings.get_GetWorkspace(sess, id=made_workspace.id).workspace
        assert get_workspace.name == "_TestPatched"

        # Sort test and default workspaces.
        ww = bindings.post_PostWorkspace(sess, body=bindings.v1PostWorkspaceRequest(name="_TestWS"))
        assert ww.workspace is not None
        test_workspaces.append(ww.workspace)
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

        # Add a test project to a workspace.
        r4 = bindings.post_PostProject(
            sess,
            body=bindings.v1PostProjectRequest(name="_TestOnly", workspaceId=made_workspace.id),
            workspaceId=made_workspace.id,
        )
        made_project = r4.project
        assert made_project is not None
        test_projects.append(made_project)
        get_project = bindings.get_GetProject(sess, id=made_project.id).project
        assert get_project and get_project.name == "_TestOnly"
        assert not madeProject.immutable and not get_project.immutable

        # Patch the project
        p_patch = bindings.v1PatchProject.from_json(made_project.to_json())
        p_patch.name = "_TestPatchedProject"
        bindings.patch_PatchProject(sess, body=p_patch, id=made_project.id)
        get_project = bindings.get_GetProject(sess, id=made_project.id).project
        assert get_project.name == "_TestPatchedProject"

        # Refuse to patch the default project
        with pytest.raises(errors.APIException):
            bindings.patch_PatchProject(sess, body=p_patch, id=default_project.id)

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
        assert p1 and p2
        test_projects += [p1, p2]
        list_test_4 = bindings.get_GetWorkspaceProjects(sess, id=made_workspace.id).projects
        assert list_test_4 is not None
        assert ["_TestPatchedProject", "_TestPRJ", "_TestEarly"] == list(
            map(lambda w: w.name, list_test_4)
        )
        list_test_5 = bindings.get_GetWorkspaceProjects(
            sess, id=made_workspace.id, orderBy=bindings.v1OrderBy.ORDER_BY_DESC
        ).projects
        assert list_test_5 is not None
        assert ["_TestEarly", "_TestPRJ", "_TestPatchedProject"] == list(
            map(lambda w: w.name, list_test_5)
        )
        list_test_6 = bindings.get_GetWorkspaceProjects(
            sess,
            id=made_workspace.id,
            sortBy=bindings.v1GetWorkspaceProjectsRequestSortBy.SORT_BY_NAME,
        ).projects
        assert list_test_6 is not None
        assert ["_TestEarly", "_TestPatchedProject", "_TestPRJ"] == list(
            map(lambda w: w.name, list_test_6)
        )

        # Move a project to another workspace
        bindings.post_MoveProject(
            sess,
            projectId=made_project.id,
            body=ww.workspace.id,
        )
        get_project = bindings.get_GetProject(sess, id=made_project.id).project
        assert get_project.workspaceId == ww.workspace.id

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
        assert returned_notes and len(returned_notes) == 2

        # Create an experiment in the default project and move to a test project.
        test_exp_id = run_basic_test(
            conf.fixtures_path("no_op/single.yaml"), conf.fixtures_path("no_op"), 1
        )
        test_exp = bindings.get_GetExperiment(sess, experimentId=test_exp_id).experiment
        test_experiments.append(test_exp)
        wait_for_experiment_state(test_exp_id, bindings.determinedexperimentv1State.STATE_COMPLETED)
        assert test_exp.projectId == default_project.id
        bindings.post_MoveExperiment(sess, experimentId=test_exp_id, body=made_project.id)
        modified_exp = bindings.get_GetExperiment(sess, experimentId=test_exp_id).experiment
        assert modified_exp.projectId == made_project.id
        bindings.post_MoveExperiment(sess, experimentId=test_exp_id, body=default_project.id)

    finally:
        # Clean out experiments, projects, workspaces.
        # In dependency order:
        for e in test_experiments:
            bindings.delete_DeleteExperiment(sess, experimentId=e.id)
        for p in test_projects:
            bindings.delete_DeleteProject(sess, id=p.id)
        for w in test_workspaces:
            bindings.delete_DeleteWorkspace(sess, id=w.id)
