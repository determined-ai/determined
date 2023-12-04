import uuid
from typing import Any, Dict, List, Tuple

import pytest

from determined.common import util
from determined.common.api import Session, authentication, bindings, errors
from determined.common.api.bindings import experimentv1State
from determined.common.experimental import model
from determined.experimental import Checkpoint, Determined
from determined.experimental import client as _client
from tests import api_utils
from tests import config as conf
from tests import experiment
from tests.cluster.test_rbac import create_workspaces_with_users
from tests.cluster.test_users import log_in_user_cli, logged_in_user

from .test_groups import det_cmd
from .test_workspace_org import setup_workspaces


def get_random_string() -> str:
    return str(uuid.uuid4())


def all_operations(
    determined_obj: Determined,
    test_workspace: bindings.v1Workspace,
    checkpoint: Checkpoint,
) -> Tuple[model.Model, str]:
    test_model_name = get_random_string()

    determined_obj.create_model(name=test_model_name, workspace_name=test_workspace.name)
    model_obj = determined_obj.get_model(identifier=test_model_name)

    model_obj.set_description("abcde")
    model_obj = determined_obj.get_model(model_obj.name)
    assert model_obj.description == "abcde"

    # Register a version for the model and validate the latest.

    model_version = model_obj.register_version(checkpoint.uuid)
    assert model_version.model_version == 1

    latest_version = model_obj.get_version()
    assert latest_version is not None
    assert latest_version.checkpoint is not None
    assert latest_version.checkpoint.uuid == checkpoint.uuid

    # Get checkpoint (ensure you can access this through model).
    c = determined_obj.get_checkpoint(checkpoint.uuid)
    assert c.uuid == checkpoint.uuid

    latest_version.set_name("Test 2021")
    db_version = model_obj.get_version()
    assert db_version is not None
    assert db_version.name == "Test 2021"

    model_obj.move_to_workspace(workspace_name="Uncategorized")
    models = determined_obj.get_models(workspace_names=["Uncategorized"])
    assert model_obj.name in [m.name for m in models]
    return model_obj, "Uncategorized"


def view_operations(determined_obj: Determined, model: model.Model, workspace_name: str) -> None:
    db_model = determined_obj.get_model(model.name)
    assert db_model.name == model.name
    models = determined_obj.get_models(workspace_names=[workspace_name])
    assert db_model.name in [m.name for m in models]


def user_with_view_perms_test(
    determined_obj: Determined, workspace_name: str, model: model.Model
) -> None:
    view_operations(determined_obj=determined_obj, model=model, workspace_name=workspace_name)
    # fail edit model
    with pytest.raises(errors.ForbiddenException) as e:
        # model object needs to have the same sess as det obj with logged in user.
        model = determined_obj.get_model(model.name)
        model.set_description("abcde")
    assert "access denied" in str(e.value)
    # fail create model
    with pytest.raises(errors.ForbiddenException) as e:
        determined_obj.create_model(name=get_random_string(), workspace_name=workspace_name)
    assert "access denied" in str(e.value)


def create_model_registry(session: Session, model_name: str, workspace_id: int) -> model.Model:
    resp = bindings.post_PostModel(
        session,
        body=bindings.v1PostModelRequest(name=model_name, workspaceId=workspace_id),
    )
    assert resp.model is not None
    return model.Model._from_bindings(resp.model, session)


def register_model_version(
    creds: authentication.Credentials, model_name: str, workspace_id: int
) -> Tuple[model.Model, model.ModelVersion]:
    m = None
    model_version = None
    session = api_utils.determined_test_session(creds)
    with logged_in_user(creds):
        pid = bindings.post_PostProject(
            session,
            body=bindings.v1PostProjectRequest(name=get_random_string(), workspaceId=workspace_id),
            workspaceId=workspace_id,
        ).project.id
        m = create_model_registry(session, model_name, workspace_id)
        experiment_id = experiment.create_experiment(
            conf.fixtures_path("no_op/single.yaml"),
            conf.fixtures_path("no_op"),
            ["--project_id", str(pid)],
        )
        experiment.wait_for_experiment_state(
            experiment_id, experimentv1State.COMPLETED, credentials=creds
        )
        checkpoint = bindings.get_GetExperimentCheckpoints(
            id=experiment_id, session=session
        ).checkpoints[0]
        model_version = m.register_version(checkpoint.uuid)
        assert model_version.model_version == 1
    return m, model_version


@pytest.mark.test_model_registry_rbac
def test_model_registry_rbac() -> None:
    log_in_user_cli(conf.ADMIN_CREDENTIALS)
    test_user_editor_creds = api_utils.create_test_user()
    test_user_workspace_admin_creds = api_utils.create_test_user()
    test_user_viewer_creds = api_utils.create_test_user()
    test_user_with_no_perms_creds = api_utils.create_test_user()
    test_user_model_registry_viewer_creds = api_utils.create_test_user()
    admin_session = api_utils.determined_test_session(admin=True)
    with setup_workspaces(admin_session) as [test_workspace]:
        with logged_in_user(conf.ADMIN_CREDENTIALS):
            # Assign editor role to user in Uncategorized and test_workspace.
            det_cmd(
                [
                    "rbac",
                    "assign-role",
                    "Editor",
                    "--username-to-assign",
                    test_user_editor_creds.username,
                    "--workspace-name",
                    "Uncategorized",
                ],
                check=True,
            )

            det_cmd(
                [
                    "rbac",
                    "assign-role",
                    "Editor",
                    "--username-to-assign",
                    test_user_editor_creds.username,
                    "--workspace-name",
                    test_workspace.name,
                ],
                check=True,
            )

            # Assign workspace admin to user in Uncategorized and test_workspace.
            det_cmd(
                [
                    "rbac",
                    "assign-role",
                    "WorkspaceAdmin",
                    "--username-to-assign",
                    test_user_workspace_admin_creds.username,
                    "--workspace-name",
                    "Uncategorized",
                ],
                check=True,
            )
            det_cmd(
                [
                    "rbac",
                    "assign-role",
                    "WorkspaceAdmin",
                    "--username-to-assign",
                    test_user_workspace_admin_creds.username,
                    "--workspace-name",
                    test_workspace.name,
                ],
                check=True,
            )

            # Assign viewer to user in Uncategorized and test_workspace.
            det_cmd(
                [
                    "rbac",
                    "assign-role",
                    "Viewer",
                    "--username-to-assign",
                    test_user_viewer_creds.username,
                    "--workspace-name",
                    "Uncategorized",
                ],
                check=True,
            )
            det_cmd(
                [
                    "rbac",
                    "assign-role",
                    "Viewer",
                    "--username-to-assign",
                    test_user_viewer_creds.username,
                    "--workspace-name",
                    test_workspace.name,
                ],
                check=True,
            )

            # Assign model registry viewer to user in Uncategorized and test_workspace.
            det_cmd(
                [
                    "rbac",
                    "assign-role",
                    "ModelRegistryViewer",
                    "--username-to-assign",
                    test_user_model_registry_viewer_creds.username,
                    "--workspace-name",
                    "Uncategorized",
                ],
                check=True,
            )
            det_cmd(
                [
                    "rbac",
                    "assign-role",
                    "ModelRegistryViewer",
                    "--username-to-assign",
                    test_user_model_registry_viewer_creds.username,
                    "--workspace-name",
                    test_workspace.name,
                ],
                check=True,
            )
        master_url = conf.make_master_url()

        with logged_in_user(test_user_editor_creds):
            # need to get a new determined obj everytime a new user is logged in.
            # Same pattern is followed below.
            d = Determined(master_url)
            with open(conf.fixtures_path("no_op/single-one-short-step.yaml")) as f:
                config = util.yaml_safe_load(f)
            exp = d.create_experiment(config, conf.fixtures_path("no_op"))
            # wait for exp state to be completed
            assert exp.wait() == _client.ExperimentState.COMPLETED
            checkpoint = d.get_experiment(exp.id).top_checkpoint()
            # need to get a new determined obj everytime a new user is logged in.
            # Same pattern is followed below.
            model_1, current_model_workspace = all_operations(
                determined_obj=d, test_workspace=test_workspace, checkpoint=checkpoint
            )

        with logged_in_user(test_user_model_registry_viewer_creds):
            d = Determined(master_url)
            user_with_view_perms_test(
                determined_obj=d, workspace_name=current_model_workspace, model=model_1
            )

        with logged_in_user(test_user_viewer_creds):
            d = Determined(master_url)
            user_with_view_perms_test(
                determined_obj=d, workspace_name=current_model_workspace, model=model_1
            )

        with logged_in_user(test_user_with_no_perms_creds):
            d = Determined(master_url)
            with pytest.raises(Exception) as e:
                d.get_models()
            assert "doesn't have view permissions" in str(e.value)

        # Unassign view permissions to a certain workspace.
        # List should return models only in workspaces with permissions.
        with logged_in_user(conf.ADMIN_CREDENTIALS):
            det_cmd(
                [
                    "rbac",
                    "unassign-role",
                    "ModelRegistryViewer",
                    "--username-to-assign",
                    test_user_model_registry_viewer_creds.username,
                    "--workspace-name",
                    test_workspace.name,
                ],
                check=True,
            )
        with logged_in_user(test_user_model_registry_viewer_creds):
            d = Determined(master_url)
            models = d.get_models()
            assert test_workspace.id not in [m.workspace_id for m in models]

        with logged_in_user(test_user_editor_creds):
            d = Determined(master_url)
            model = d.get_model(model_1.name)
            model.delete()

        with logged_in_user(test_user_workspace_admin_creds):
            d = Determined(master_url)
            checkpoint = d.get_experiment(exp.id).top_checkpoint()
            model_2, current_model_workspace = all_operations(
                determined_obj=d, test_workspace=test_workspace, checkpoint=checkpoint
            )

        # Remove workspace admin role for this user from test_workspace.
        with logged_in_user(conf.ADMIN_CREDENTIALS):
            det_cmd(
                [
                    "rbac",
                    "unassign-role",
                    "WorkspaceAdmin",
                    "--username-to-assign",
                    test_user_workspace_admin_creds.username,
                    "--workspace-name",
                    test_workspace.name,
                ],
                check=True,
            )

        with logged_in_user(test_user_workspace_admin_creds):
            d = Determined(master_url)
            model = d.get_model(model_2.name)
            assert current_model_workspace == "Uncategorized"
            # move model to test_workspace should fail.
            with pytest.raises(errors.ForbiddenException) as e:
                model.move_to_workspace(workspace_name=test_workspace.name)
            assert "access denied" in str(e.value)
            model.delete()


@pytest.mark.test_model_registry_rbac
def test_model_rbac_deletes() -> None:
    with create_workspaces_with_users(
        [
            [
                (0, ["Editor"]),
            ],
        ]
    ) as (workspaces, creds):
        workspace_id = workspaces[0].id
        # create non-cluster admin user
        editor_creds = creds[0]
        editor_session = api_utils.determined_test_session(editor_creds)

        # create cluster admin user
        cluster_admin_creds = api_utils.create_test_user(
            add_password=True,
            user=bindings.v1User(username=get_random_string(), active=True, admin=False),
        )
        api_utils.assign_user_role(
            session=api_utils.determined_test_session(conf.ADMIN_CREDENTIALS),
            user=cluster_admin_creds.username,
            role="ClusterAdmin",
            workspace=None,
        )
        cluster_admin_session = api_utils.determined_test_session(cluster_admin_creds)

        # create non-cluster admin user with OSS admin flag
        oss_admin_creds = api_utils.create_test_user(
            add_password=True,
            user=bindings.v1User(username=get_random_string(), active=True, admin=True),
        )
        oss_admin_session = api_utils.determined_test_session(oss_admin_creds)

        model_num = 0
        try:
            # test deleting model registries
            tests: List[Dict[str, Any]] = [
                {
                    "create_session": cluster_admin_session,
                    "delete_session": cluster_admin_session,
                    "should_error": False,
                },
                {
                    "create_session": editor_session,
                    "delete_session": cluster_admin_session,
                    "should_error": False,
                },
                {
                    "create_session": editor_session,
                    "delete_session": editor_session,
                    "should_error": False,
                },
                {
                    "create_session": cluster_admin_session,
                    "delete_session": editor_session,
                    "should_error": True,
                },
                {
                    "create_session": cluster_admin_session,
                    "delete_session": oss_admin_session,
                    "should_error": True,
                },
            ]
            for t in tests:
                create_session: Session = t["create_session"]
                delete_session: Session = t["delete_session"]
                should_error: bool = t["should_error"]

                model_name = "model_" + str(model_num)
                model_num += 1
                create_model_registry(create_session, model_name, workspace_id)

                if should_error:
                    with pytest.raises(errors.ForbiddenException) as permErr:
                        bindings.delete_DeleteModel(delete_session, modelName=model_name)
                    assert "access denied" in str(permErr.value)
                else:
                    bindings.delete_DeleteModel(delete_session, modelName=model_name)
                    with pytest.raises(errors.NotFoundException) as notFoundErr:
                        bindings.get_GetModel(create_session, modelName=model_name)
                    assert "not found" in str(notFoundErr.value).lower()

            # test deleting model versions
            tests = [
                {
                    "create_creds": cluster_admin_creds,
                    "delete_session": cluster_admin_session,
                    "should_error": False,
                },
                {
                    "create_creds": editor_creds,
                    "delete_session": editor_session,
                    "should_error": False,
                },
                {
                    "create_creds": editor_creds,
                    "delete_session": cluster_admin_session,
                    "should_error": False,
                },
                {
                    "create_creds": cluster_admin_creds,
                    "delete_session": editor_session,
                    "should_error": True,
                },
                {
                    "create_creds": cluster_admin_creds,
                    "delete_session": oss_admin_session,
                    "should_error": True,
                },
            ]

            for t in tests:
                create_creds: authentication.Credentials = t["create_creds"]
                delete_session = t["delete_session"]
                should_error = t["should_error"]

                model_name = "model_" + str(model_num)
                model_num += 1
                m, ca_model_version = register_model_version(
                    creds=create_creds, model_name=model_name, workspace_id=workspace_id
                )
                model_version_num = ca_model_version.model_version

                if should_error:
                    with pytest.raises(errors.ForbiddenException) as permErr:
                        bindings.delete_DeleteModelVersion(
                            delete_session,
                            modelName=model_name,
                            modelVersionNum=model_version_num,
                        )
                    assert "access denied" in str(permErr.value)
                else:
                    bindings.delete_DeleteModelVersion(
                        delete_session, modelName=model_name, modelVersionNum=model_version_num
                    )
                    with pytest.raises(errors.NotFoundException) as notFoundErr:
                        bindings.get_GetModelVersion(
                            api_utils.determined_test_session(create_creds),
                            modelName=model_name,
                            modelVersionNum=model_version_num,
                        )
                    assert "not found" in str(notFoundErr.value).lower()
        finally:
            for i in range(model_num):
                admin_session = api_utils.determined_test_session(conf.ADMIN_CREDENTIALS)
                try:
                    bindings.delete_DeleteModel(admin_session, modelName="model_" + str(i))
                # model is has already been cleaned up
                except errors.NotFoundException:
                    continue
