import pytest

from determined.common.api import authentication, bindings, certs
from determined.common.experimental import session
from tests import config as conf
from tests import experiment as exp


@pytest.mark.e2e_cpu
def test_workspace_org() -> None:
    master_url = conf.make_master_url()
    certs.cli_cert = certs.default_load(master_url)
    authentication.cli_auth = authentication.Authentication(master_url, try_reauth=True)
    sess = session.Session(master_url, "determined", authentication.cli_auth, certs.cli_cert)

    # Uncategorized workspace / project should exist already.
    r = bindings.get_GetWorkspaces(sess, name="Uncategorized")
    assert len(r.workspaces) == 1
    default_workspace = r.workspaces[0]
    r2 = bindings.get_GetWorkspaceProjects(sess, id=default_workspace.id)
    assert len(r2.projects) == 1
    default_project = r2.projects[0]
    assert default_project.name == "Uncategorized"

    """
    Until DELETE is built into API:

    DELETE FROM projects WHERE name LIKE '_Test%';
    DELETE FROM workspaces WHERE name LIKE '_Test%';
    """

    # Add a test workspace.
    r3 = bindings.post_PostWorkspace(sess, body=bindings.v1PostWorkspaceRequest(name="_TestOnly"))
    madeWorkspace = r3.workspace

    # Sort test and default workspaces.
    ####

    # Add a test project to a workspace.
    r4 = bindings.post_PostProject(
        sess,
        body=bindings.v1PostProjectRequest(name="_TestOnly", workspaceId=madeWorkspace.id),
        workspaceId=madeWorkspace.id,
    )
    madeProject = r4.project

    # Sort test and default projects.
    ####

    # Add a test note to a project.
    note = bindings.v1Note(name="Hello", contents="Hello World")
    r5 = bindings.post_AddProjectNote(
        sess,
        body=bindings.v1AddProjectNoteRequest(note),
        id=madeProject.id,
    )
    returned_notes = r5.notes
    assert len(returned_notes) == 1

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
