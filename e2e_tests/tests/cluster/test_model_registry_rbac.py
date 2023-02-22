import uuid
from typing import Tuple

import pytest

from determined.common import yaml
from determined.common.api import bindings, errors
from determined.common.experimental import model
from determined.experimental import Checkpoint, Determined
from determined.experimental import client as _client
from tests import api_utils
from tests import config as conf
from tests.cluster.test_users import ADMIN_CREDENTIALS, log_in_user_cli, logged_in_user

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
    with pytest.raises(errors.APIException) as e:
        # model object needs to have the same sess as det obj with logged in user.
        model = determined_obj.get_model(model.name)
        model.set_description("abcde")
    assert "access denied" in str(e.value)
    # fail create model
    with pytest.raises(errors.APIException) as e:
        determined_obj.create_model(name=get_random_string(), workspace_name=workspace_name)
    assert "access denied" in str(e.value)


@pytest.mark.test_model_registry_rbac
def test_model_registry_rbac() -> None:
    log_in_user_cli(ADMIN_CREDENTIALS)
    test_user_editor_creds = api_utils.create_test_user()
    test_user_workspace_admin_creds = api_utils.create_test_user()
    test_user_viewer_creds = api_utils.create_test_user()
    test_user_with_no_perms_creds = api_utils.create_test_user()
    test_user_model_registry_viewer_creds = api_utils.create_test_user()
    admin_session = api_utils.determined_test_session(admin=True)
    with setup_workspaces(admin_session) as [test_workspace]:
        with logged_in_user(ADMIN_CREDENTIALS):
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
                config = yaml.safe_load(f)
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
        with logged_in_user(ADMIN_CREDENTIALS):
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
        with logged_in_user(ADMIN_CREDENTIALS):
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
            with pytest.raises(errors.APIException) as e:
                model.move_to_workspace(workspace_name=test_workspace.name)
            assert "access denied" in str(e.value)
            model.delete()
