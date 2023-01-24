import contextlib
from typing import Generator, List, Sequence

import pytest

from determined.common.api import Session, bindings, errors
from tests.api_utils import configure_token_store, create_test_user, determined_test_session

from .test_groups import det_cmd, det_cmd_json
from .test_users import ADMIN_CREDENTIALS, get_random_string, logged_in_user

DEFAULT_WID = 1  # default workspace ID


def rbac_disabled() -> bool:
    try:
        return not bindings.get_GetMaster(determined_test_session()).rbacEnabled
    except (errors.APIException, errors.MasterNotFoundException):
        return True


@contextlib.contextmanager
def setup_workspaces(names: List[str]) -> Generator[List[bindings.v1Workspace], None, None]:
    session = determined_test_session(ADMIN_CREDENTIALS)
    workspaces: Sequence[bindings.v1Workspace] = []
    try:
        for name in names:
            body = bindings.v1PostWorkspaceRequest(name=name)
            bindings.post_PostWorkspace(session, body=body)

        # create two workspaces
        workspaces = bindings.get_GetWorkspaces(session).workspaces
        workspaces = list(
            filter(lambda w: w.name in names, workspaces)
        )  # throw out the default one
        assert len(workspaces) == 2
        yield workspaces

    finally:
        for w in workspaces:
            if w.name in names:
                bindings.delete_DeleteWorkspace(session, id=w.id)


@contextlib.contextmanager
def setup_notebooks(
    session: Session, notebooks: List[bindings.v1LaunchNotebookRequest]
) -> Generator[List[bindings.v1Notebook], None, None]:
    created: List[bindings.v1Notebook] = []
    try:
        for nb_req in notebooks:
            r = bindings.post_LaunchNotebook(session, body=nb_req)
            created.append(r.notebook)
        yield created

    finally:
        for nb in created:
            bindings.post_KillNotebook(session, notebookId=nb.id)


def filter_out_ntsc(
    base: Sequence[bindings.v1Notebook], target: Sequence[bindings.v1Notebook]
) -> List[bindings.v1Notebook]:
    """throw out target notebooks that are not in the base list"""
    accepted_ids = {nb.id for nb in base}
    return list(filter(lambda nb: nb.id in accepted_ids, target))


@pytest.mark.e2e_cpu_rbac
@pytest.mark.skipif(rbac_disabled(), reason="ee rbac is required for this test")
def test_notebook() -> None:
    configure_token_store(ADMIN_CREDENTIALS)
    u_viewer_ws0 = create_test_user(add_password=True)
    u_editor_ws1 = create_test_user(add_password=True)
    admin_session = determined_test_session(ADMIN_CREDENTIALS)

    with setup_workspaces([get_random_string() for _ in range(2)]) as workspaces:
        det_cmd(
            [
                "rbac",
                "assign-role",
                "Viewer",
                "--username-to-assign",
                u_viewer_ws0.username,
                "--workspace-name",
                workspaces[0].name,
            ],
            check=True,
        )

        det_cmd(
            [
                "rbac",
                "assign-role",
                "Editor",
                "--username-to-assign",
                u_editor_ws1.username,
                "--workspace-name",
                workspaces[1].name,
            ],
            check=True,
        )

        nb_reqs = [
            bindings.v1LaunchNotebookRequest(workspaceId=workspaces[0].id),
            bindings.v1LaunchNotebookRequest(workspaceId=workspaces[1].id),
            bindings.v1LaunchNotebookRequest(),
        ]
        with setup_notebooks(admin_session, nb_reqs) as notebooks:
            r = bindings.get_GetNotebooks(admin_session)
            assert len(filter_out_ntsc(notebooks, r.notebooks)) == 3
            r = bindings.get_GetNotebooks(admin_session, workspaceId=workspaces[0].id)
            assert len(filter_out_ntsc(notebooks, r.notebooks)) == 1
            r = bindings.get_GetNotebooks(admin_session, workspaceId=workspaces[1].id)
            assert len(filter_out_ntsc(notebooks, r.notebooks)) == 1
            r = bindings.get_GetNotebooks(admin_session, workspaceId=DEFAULT_WID)
            assert len(filter_out_ntsc(notebooks, r.notebooks)) == 1

            r = bindings.get_GetNotebooks(determined_test_session(u_viewer_ws0))
            assert len(r.notebooks) == 3  # TODO(rbac impl): 1
            r = bindings.get_GetNotebooks(
                determined_test_session(u_viewer_ws0), workspaceId=workspaces[0].id
            )
            assert len(r.notebooks) == 1

            # User with only view role on first workspace
            with logged_in_user(u_viewer_ws0):
                json_out = det_cmd_json(["notebook", "ls", "--all", "--json"])
                assert len(json_out) == 3  # TODO(rbac impl): 1
