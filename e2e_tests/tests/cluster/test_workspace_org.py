from time import sleep

import pytest

from determined.common.api import authentication, bindings, certs
from determined.common.experimental import session
from tests import config as conf


@pytest.mark.e2e_cpu
def test_workspace_org() -> None:
    master_url = conf.make_master_url()
    certs.cli_cert = certs.default_load(master_url)
    authentication.cli_auth = authentication.Authentication(master_url, try_reauth=True)
    sess = session.Session(master_url, "determined", authentication.cli_auth, certs.cli_cert)

    # Uncategorized workspace / project should exist already.
    r = bindings.get_GetWorkspaces(sess, name="Uncategorized")
    assert r.workspaces and len(r.workspaces) == 1
    default_workspace = r.workspaces[0]
    r2 = bindings.get_GetWorkspaceProjects(sess, id=default_workspace.id)
    assert r2.projects and len(r2.projects) == 1
    default_project = r2.projects[0]
    assert default_project.name == "Uncategorized"

    """
    Until DELETE is built into API:

    DELETE FROM projects WHERE name LIKE '_Test%';
    DELETE FROM workspaces WHERE name LIKE '_Test%';
    """

    # Add test workspaces.
    r3 = bindings.post_PostWorkspace(sess, body=bindings.v1PostWorkspaceRequest(name="_TestOnly"))
    madeWorkspace = r3.workspace
    assert madeWorkspace is not None
    bindings.post_PostWorkspace(sess, body=bindings.v1PostWorkspaceRequest(name="_TestWS"))
    sleep(0.1)
    get_workspace = bindings.get_GetWorkspace(sess, id=madeWorkspace.id).workspace
    assert get_workspace and get_workspace.name == "_TestOnly"

    # Sort test and default workspaces.
    list_test_1 = bindings.get_GetWorkspaces(sess).workspaces
    assert ["Uncategorized", "_TestOnly", "_TestWS"] == list(map(lambda w: w.name, list_test_1))
    list_test_2 = bindings.get_GetWorkspaces(
        sess, orderBy=bindings.v1OrderBy.ORDER_BY_DESC
    ).workspaces
    assert ["_TestWS", "_TestOnly", "Uncategorized"] == list(map(lambda w: w.name, list_test_2))
    list_test_3 = bindings.get_GetWorkspaces(
        sess, sortBy=bindings.v1GetWorkspacesRequestSortBy.SORT_BY_NAME
    ).workspaces
    assert ["_TestOnly", "_TestWS", "Uncategorized"] == list(map(lambda w: w.name, list_test_3))

    # Add test projects to a workspace.
    r4 = bindings.post_PostProject(
        sess,
        body=bindings.v1PostProjectRequest(name="_TestOnly", workspaceId=madeWorkspace.id),
        workspaceId=madeWorkspace.id,
    )
    madeProject = r4.project
    assert madeProject is not None
    bindings.post_PostProject(
        sess,
        body=bindings.v1PostProjectRequest(name="_TestPRJ", workspaceId=madeWorkspace.id),
        workspaceId=madeWorkspace.id,
    )
    bindings.post_PostProject(
        sess,
        body=bindings.v1PostProjectRequest(name="_TestEarly", workspaceId=madeWorkspace.id),
        workspaceId=madeWorkspace.id,
    )
    sleep(0.1)
    get_project = bindings.get_GetProject(sess, id=madeProject.id).project
    assert get_project and get_project.name == "_TestOnly"

    # Sort workspaces' projects.
    list_test_4 = bindings.get_GetWorkspaceProjects(sess, id=madeWorkspace.id).projects
    assert list_test_4 is not None
    assert ["_TestOnly", "_TestPRJ", "_TestEarly"] == list(map(lambda w: w.name, list_test_4))
    list_test_5 = bindings.get_GetWorkspaceProjects(
        sess, id=madeWorkspace.id, orderBy=bindings.v1OrderBy.ORDER_BY_DESC
    ).projects
    assert list_test_5 is not None
    assert ["_TestEarly", "_TestPRJ", "_TestOnly"] == list(map(lambda w: w.name, list_test_5))
    list_test_6 = bindings.get_GetWorkspaceProjects(
        sess, id=madeWorkspace.id, sortBy=bindings.v1GetWorkspaceProjectsRequestSortBy.SORT_BY_NAME
    ).projects
    assert list_test_6 is not None
    assert ["_TestEarly", "_TestOnly", "_TestPRJ"] == list(map(lambda w: w.name, list_test_6))

    # Add a test note to a project.
    note = bindings.v1Note(name="Hello", contents="Hello World")
    note2 = bindings.v1Note(name="Hello 2", contents="Hello World")
    bindings.post_AddProjectNote(
        sess,
        body=note,
        projectId=madeProject.id,
    )
    sleep(0.1)
    r5 = bindings.post_AddProjectNote(
        sess,
        body=note2,
        projectId=madeProject.id,
    )
    returned_notes = r5.notes
    assert returned_notes and len(returned_notes) == 2

    # TODO: add a test experiment to a project.

    """
    test_workspaces = []
    try:
        # Run on test workspaces only
    finally:
        # Clean out test workspaces and projects
        for w in test_workspaces:
            w.delete()
    """
