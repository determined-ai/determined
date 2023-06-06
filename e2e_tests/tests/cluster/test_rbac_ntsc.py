import contextlib
from typing import Generator, List, Optional, Sequence, Tuple

import pytest

import tests.config as conf
from determined import cli
from determined.common import api
from determined.common.api import Session, authentication, bindings, errors
from determined.common.api._util import (
    NTSC_Kind,
    all_ntsc,
    get_ntsc_details,
    proxied_ntsc,
    wait_for_ntsc_state,
)
from tests import experiment as exp
from tests import utils
from tests.api_utils import (
    configure_token_store,
    create_test_user,
    determined_test_session,
    kill_ntsc,
    launch_ntsc,
    list_ntsc,
    set_prio_ntsc,
)
from tests.cluster.test_rbac import create_workspaces_with_users, rbac_disabled
from tests.cluster.test_workspace_org import setup_workspaces

from .test_groups import det_cmd_json
from .test_users import ADMIN_CREDENTIALS, logged_in_user

DEFAULT_WID = 1  # default workspace ID


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

    with setup_workspaces(count=2) as workspaces:
        cli.rbac.assign_role(
            utils.CliArgsMock(
                username_to_assign=u_viewer_ws0.username,
                workspace_name=workspaces[0].name,
                role_name="Viewer",
            )
        )
        cli.rbac.assign_role(
            utils.CliArgsMock(
                username_to_assign=u_editor_ws1.username,
                workspace_name=workspaces[1].name,
                role_name="Editor",
            )
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
            assert len(r.notebooks) == 1
            r = bindings.get_GetNotebooks(
                determined_test_session(u_viewer_ws0), workspaceId=workspaces[0].id
            )
            assert len(r.notebooks) == 1
            with pytest.raises(errors.APIException) as e:
                r = bindings.get_GetNotebooks(
                    determined_test_session(u_viewer_ws0), workspaceId=workspaces[1].id
                )
                assert e.value.status_code == 404

            # User with only view role on first workspace
            with logged_in_user(u_viewer_ws0):
                json_out = det_cmd_json(["notebook", "ls", "--all", "--json"])
                assert len(json_out) == 1


tensorboard_wait_time = 300


def only_tensorboard_can_launch(
    session: Session, workspace: int, typ: NTSC_Kind, exp_id: Optional[int] = None
) -> None:
    """
    Tensorboard requires the 'view experiment' permission rather than the 'create NSC' permission
    and so can be launched in some workspaces other NSCs can't.
    """
    if typ == NTSC_Kind.tensorboard:
        launch_ntsc(session, workspace, typ, exp_id)
        return

    with pytest.raises(errors.ForbiddenException):
        launch_ntsc(session, workspace, typ, exp_id)


@pytest.mark.e2e_cpu_rbac
@pytest.mark.skipif(rbac_disabled(), reason="ee rbac is required for this test")
def test_ntsc_iface_access() -> None:
    def can_access_logs(creds: authentication.Credentials, ntsc_id: str) -> bool:
        session = determined_test_session(creds)
        try:
            list(bindings.get_TaskLogs(session, taskId=ntsc_id))
            return True
        except errors.APIException as e:
            if e.status_code != 404 and "not found" not in e.message:
                # FIXME: the endpoint should respond with 404.
                raise e
            return False

    with create_workspaces_with_users(
        [
            [
                (0, ["Viewer", "Editor"]),
                (1, ["Viewer"]),
            ],
            [
                (2, ["Viewer"]),
                (0, ["Viewer"]),
            ],
            [
                (0, ["Editor"]),
            ],
        ]
    ) as (workspaces, creds):
        # launch one of each ntsc in the first workspace
        for typ in all_ntsc:
            experiment_id = None
            if typ == NTSC_Kind.tensorboard:
                pid = bindings.post_PostProject(
                    determined_test_session(creds[0]),
                    body=bindings.v1PostProjectRequest(name="test", workspaceId=workspaces[0].id),
                    workspaceId=workspaces[0].id,
                ).project.id

                with logged_in_user(creds[0]):
                    # experiment for tensorboard
                    experiment_id = exp.create_experiment(
                        conf.fixtures_path("no_op/single.yaml"),
                        conf.fixtures_path("no_op"),
                        ["--project_id", str(pid)],
                    )

            created_id = launch_ntsc(
                determined_test_session(creds[0]), workspaces[0].id, typ, experiment_id
            )

            # user 0
            assert can_access_logs(
                creds[0], created_id
            ), f"user 0 should be able to access {typ} logs"
            session = determined_test_session(creds[0])
            get_ntsc_details(session, typ, created_id)  # user 0 should be able to get details.
            kill_ntsc(session, typ, created_id)  # user 0 should be able to kill.
            set_prio_ntsc(session, typ, created_id, 1)  # user 0 should be able to set priority.
            launch_ntsc(
                session, workspaces[0].id, typ, experiment_id
            )  # user 0 should be able to launch in workspace 0.

            # user 0 should be able to launch tensorboards and not NSCs in workspace 1.
            only_tensorboard_can_launch(session, workspaces[1].id, typ, experiment_id)

            # user 1
            assert can_access_logs(
                creds[1], created_id
            ), f"user 1 should be able to access {typ} logs"
            session = determined_test_session(creds[1])
            get_ntsc_details(session, typ, created_id)  # user 1 should be able to get details.
            with pytest.raises(errors.ForbiddenException) as fe:
                kill_ntsc(session, typ, created_id)  # user 1 should not be able to kill.
            assert "access denied" in fe.value.message
            with pytest.raises(errors.ForbiddenException) as fe:
                set_prio_ntsc(
                    session, typ, created_id, 1
                )  # user 1 should not be able to set priority.
            assert "access denied" in fe.value.message
            # user 1 should be able to launch tensorboards but not NSCs in workspace 0.
            only_tensorboard_can_launch(session, workspaces[0].id, typ, experiment_id)
            # tensorboard requires workspace access so returns workspace not found if
            # the user does not have access to the workspace.
            if typ == NTSC_Kind.tensorboard:
                with pytest.raises(errors.NotFoundException):
                    launch_ntsc(session, workspaces[1].id, typ, experiment_id)
            else:
                with pytest.raises(errors.ForbiddenException):
                    launch_ntsc(session, workspaces[1].id, typ, experiment_id)

            # user 2
            assert not can_access_logs(
                creds[2], created_id
            ), f"user 2 should not be able to access {typ} logs"
            session = determined_test_session(creds[2])
            with pytest.raises(errors.APIException) as e:
                get_ntsc_details(
                    session, typ, created_id
                )  # user 2 should not be able to get details.
            assert e.value.status_code == 404, f"user 2 should not be able to get details for {typ}"
            with pytest.raises(errors.APIException) as e:
                kill_ntsc(
                    session, typ, created_id
                )  # user 2 should not be able to kill or know it exists.
            assert e.value.status_code == 404, f"user 2 should not be able to kill {typ}"
            with pytest.raises(errors.APIException) as e:
                set_prio_ntsc(
                    session, typ, created_id, 1
                )  # user 2 should not be able to set priority or know it exists.
            assert e.value.status_code == 404, f"user 2 should not be able to set priority {typ}"
            if typ == NTSC_Kind.tensorboard:
                with pytest.raises(errors.NotFoundException):
                    launch_ntsc(session, workspaces[0].id, typ, experiment_id)
            else:
                with pytest.raises(errors.ForbiddenException):
                    launch_ntsc(session, workspaces[0].id, typ, experiment_id)
            # user 2 has view access to workspace 1 so gets forbidden instead of not found.
            with pytest.raises(errors.ForbiddenException):
                launch_ntsc(session, workspaces[1].id, typ, experiment_id)

            # test visibility
            created_id2 = launch_ntsc(
                determined_test_session(creds[0]), workspaces[2].id, typ, experiment_id
            )

            # none of the users should be able to get details
            for cred in [creds[1], creds[2]]:
                session = determined_test_session(cred)
                # exception for creds[1], who can access the experiment and tensorboard
                if typ != NTSC_Kind.tensorboard and cred == creds[2]:
                    with pytest.raises(errors.APIException) as e:
                        get_ntsc_details(session, typ, created_id2)
                assert e.value.status_code == 404
                results = list_ntsc(session, typ)
                for r in results:
                    if r.id == created_id2:
                        pytest.fail(f"should not be able to see {typ} {r.id} in the list results")
                with pytest.raises(errors.APIException) as e:
                    list_ntsc(session, typ, workspace_id=workspaces[2].id)
                # FIXME only notebooks return the correct 404.
                assert e.value.status_code == 404, f"{typ} should fail with 404"
                with pytest.raises(errors.APIException) as e:
                    list_ntsc(session, typ, workspace_id=12532459)
                assert e.value.status_code == 404, f"{typ} should fail with 404"

            # kill the ntsc
            kill_ntsc(determined_test_session(creds[0]), typ, created_id)


@pytest.mark.e2e_cpu_rbac
@pytest.mark.skipif(rbac_disabled(), reason="ee rbac is required for this test")
def test_ntsc_proxy() -> None:
    def get_proxy(creds: authentication.Credentials, task_id: str) -> Optional[errors.APIException]:
        session = determined_test_session(creds)
        url = conf.make_master_url(f"proxy/{task_id}/")
        try:
            session.get(url)
            return None
        except errors.APIException as e:
            return e

    with create_workspaces_with_users(
        [
            [
                (0, ["Viewer", "Editor"]),
                (1, ["Viewer"]),
            ],
            [
                (2, ["Viewer"]),
            ],
        ]
    ) as (workspaces, creds):
        # launch one of each ntsc in the first workspace
        for typ in proxied_ntsc:
            experiment_id = None
            if typ == NTSC_Kind.tensorboard:
                pid = bindings.post_PostProject(
                    determined_test_session(creds[0]),
                    body=bindings.v1PostProjectRequest(name="test", workspaceId=workspaces[0].id),
                    workspaceId=workspaces[0].id,
                ).project.id

                with logged_in_user(creds[0]):
                    # experiment for tensorboard
                    experiment_id = exp.create_experiment(
                        conf.fixtures_path("no_op/single.yaml"),
                        conf.fixtures_path("no_op"),
                        ["--project_id", str(pid)],
                    )

            created_id = launch_ntsc(
                determined_test_session(creds[0]), workspaces[0].id, typ, experiment_id
            )

            print(f"created {typ} {created_id}")
            wait_for_ntsc_state(
                determined_test_session(creds[0]),
                NTSC_Kind(typ),
                created_id,
                lambda s: s == bindings.taskv1State.RUNNING,
                timeout=300,
            )
            deets = get_ntsc_details(determined_test_session(creds[0]), typ, created_id)
            assert deets.state == bindings.taskv1State.RUNNING, f"{typ} should be running"
            err = api.task_is_ready(determined_test_session(ADMIN_CREDENTIALS), created_id)
            assert err is None, f"{typ} should be ready {err}"
            assert (
                get_proxy(creds[0], created_id) is None
            ), f"user 0 should be able to access {typ} through proxy"
            assert (
                get_proxy(creds[1], created_id) is None
            ), f"user 1 should be able to access {typ} through proxy"
            view_err = get_proxy(creds[2], created_id)
            assert view_err is not None, f"user 2 should not be able to access {typ} through proxy"
            assert view_err.status_code == 404, f"user 2 should error out with not found{typ}"

            # kill the ntsc
            kill_ntsc(determined_test_session(creds[0]), typ, created_id)


@pytest.mark.e2e_cpu_rbac
@pytest.mark.skipif(rbac_disabled(), reason="ee rbac is required for this test")
def test_tsb_listed() -> None:
    with create_workspaces_with_users(
        [
            [
                (0, ["Editor"]),
                (1, ["Viewer"]),
            ],
        ]
    ) as ([workspace], creds):
        pid = bindings.post_PostProject(
            determined_test_session(creds[0]),
            body=bindings.v1PostProjectRequest(name="test", workspaceId=workspace.id),
            workspaceId=workspace.id,
        ).project.id

        session = determined_test_session(creds[0])

        with logged_in_user(creds[0]):
            # experiment for tensorboard
            experiment_id = exp.create_experiment(
                conf.fixtures_path("no_op/single.yaml"),
                conf.fixtures_path("no_op"),
                ["--project_id", str(pid)],
            )

            created_id = launch_ntsc(session, workspace.id, NTSC_Kind.tensorboard, experiment_id)

            # list tensorboards and make sure it's included in the response.
            tsbs = bindings.get_GetTensorboards(session, workspaceId=workspace.id).tensorboards
            assert len(tsbs) == 1, "should be one tensorboard"
            assert tsbs[0].id == created_id, "should be the tensorboard we created"

            tsbs = bindings.get_GetTensorboards(
                determined_test_session(credentials=creds[1]), workspaceId=workspace.id
            ).tensorboards
            assert len(tsbs) == 1, "should be one tensorboard"
            assert tsbs[0].id == created_id, "should be the tensorboard we created"


@pytest.mark.e2e_cpu_rbac
@pytest.mark.skipif(rbac_disabled(), reason="ee rbac is required for this test")
def test_tsb_launch_on_trials() -> None:
    with create_workspaces_with_users(
        [
            [
                (0, ["Editor"]),
            ],
        ]
    ) as ([workspace], creds):
        session = determined_test_session(creds[0])
        pid = bindings.post_PostProject(
            session,
            body=bindings.v1PostProjectRequest(name="test", workspaceId=workspace.id),
            workspaceId=workspace.id,
        ).project.id
        with logged_in_user(ADMIN_CREDENTIALS):
            experiment_id = exp.create_experiment(
                conf.fixtures_path("no_op/single.yaml"),
                conf.fixtures_path("no_op"),
                ["--project_id", str(pid)],
            )

        def get_exp_trials() -> Tuple[bool, List[int]]:
            trial_ids: List[int] = []
            try:
                trials = bindings.get_GetExperimentTrials(
                    session,
                    experimentId=experiment_id,
                ).trials
                trial_ids = [t.id for t in trials]
            except errors.APIException:
                return False, []
            if len(trial_ids) > 0:
                return True, trial_ids
            return False, []

        trial_ids: List[int] = utils.wait_for(get_exp_trials, conf.DEFAULT_MAX_WAIT_SECS)
        assert len(trial_ids) == 1, f"we should get 1 trials {trial_ids}"

        bindings.post_LaunchTensorboard(
            session,
            body=bindings.v1LaunchTensorboardRequest(workspaceId=workspace.id, trialIds=trial_ids),
        ).tensorboard.id
