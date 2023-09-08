import uuid
from typing import Any, Dict, List, Tuple

import pytest

from determined.common import api, util
from determined.common.api import bindings, errors
from determined.experimental import client
from tests import api_utils
from tests import config as conf
from tests import detproc, experiment
from tests.cluster import test_rbac, test_workspace_org


def get_random_string() -> str:
    return str(uuid.uuid4())


def all_operations(
    determined_obj: client.Determined,
    test_workspace: bindings.v1Workspace,
    checkpoint: client.Checkpoint,
) -> Tuple[client.Model, str]:
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


def view_operations(
    determined_obj: client.Determined, model: client.Model, workspace_name: str
) -> None:
    db_model = determined_obj.get_model(model.name)
    assert db_model.name == model.name
    models = determined_obj.get_models(workspace_names=[workspace_name])
    assert db_model.name in [m.name for m in models]


def user_with_view_perms_test(
    determined_obj: client.Determined, workspace_name: str, model: client.Model
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


def create_model_registry(session: api.Session, model_name: str, workspace_id: int) -> client.Model:
    resp = bindings.post_PostModel(
        session,
        body=bindings.v1PostModelRequest(name=model_name, workspaceId=workspace_id),
    )
    assert resp.model is not None
    return client.Model._from_bindings(resp.model, session)


def register_model_version(
    sess: api.Session, model_name: str, workspace_id: int
) -> Tuple[client.Model, client.ModelVersion]:
    m = None
    model_version = None

    pid = bindings.post_PostProject(
        sess,
        body=bindings.v1PostProjectRequest(name=get_random_string(), workspaceId=workspace_id),
        workspaceId=workspace_id,
    ).project.id
    m = create_model_registry(sess, model_name, workspace_id)
    experiment_id = experiment.create_experiment(
        sess,
        conf.fixtures_path("no_op/single.yaml"),
        conf.fixtures_path("no_op"),
        ["--project_id", str(pid)],
    )
    experiment.wait_for_experiment_state(sess, experiment_id, bindings.experimentv1State.COMPLETED)
    checkpoint = bindings.get_GetExperimentCheckpoints(sess, id=experiment_id).checkpoints[0]
    model_version = m.register_version(checkpoint.uuid)
    assert model_version.model_version == 1

    return m, model_version


@pytest.mark.test_model_registry_rbac
def test_model_registry_rbac() -> None:
    admin = api_utils.admin_session()
    editor, _ = api_utils.create_test_user()
    wksp_admin, _ = api_utils.create_test_user()
    viewer, _ = api_utils.create_test_user()
    noperms, _ = api_utils.create_test_user()
    model_registry_viewer, _ = api_utils.create_test_user()

    with test_workspace_org.setup_workspaces(admin) as [test_workspace]:
        for wksp in ["Uncategorized", test_workspace.name]:
            # Assign editor role.
            detproc.check_call(
                admin,
                [
                    "det",
                    "rbac",
                    "assign-role",
                    "Editor",
                    "--username-to-assign",
                    editor.username,
                    "--workspace-name",
                    wksp,
                ],
            )

            # Assign workspace admin role.
            detproc.check_call(
                admin,
                [
                    "det",
                    "rbac",
                    "assign-role",
                    "WorkspaceAdmin",
                    "--username-to-assign",
                    wksp_admin.username,
                    "--workspace-name",
                    wksp,
                ],
            )

            # Assign viewer role.
            detproc.check_call(
                admin,
                [
                    "det",
                    "rbac",
                    "assign-role",
                    "Viewer",
                    "--username-to-assign",
                    viewer.username,
                    "--workspace-name",
                    wksp,
                ],
            )

            # Assign model registry viewer role.
            detproc.check_call(
                admin,
                [
                    "det",
                    "rbac",
                    "assign-role",
                    "ModelRegistryViewer",
                    "--username-to-assign",
                    model_registry_viewer.username,
                    "--workspace-name",
                    wksp,
                ],
            )

        # Test editor user.
        d = client.Determined._from_session(editor)
        with open(conf.fixtures_path("no_op/single-one-short-step.yaml")) as f:
            config = util.yaml_safe_load(f)
        exp = d.create_experiment(config, conf.fixtures_path("no_op"))
        # wait for exp state to be completed
        assert exp.wait() == client.ExperimentState.COMPLETED
        checkpoint = d.get_experiment(exp.id).top_checkpoint()
        # need to get a new determined obj everytime a new user is logged in.
        # Same pattern is followed below.
        model_1, current_model_workspace = all_operations(
            determined_obj=d, test_workspace=test_workspace, checkpoint=checkpoint
        )

        # Test model_registry_viewer user.
        d = client.Determined._from_session(model_registry_viewer)
        user_with_view_perms_test(
            determined_obj=d, workspace_name=current_model_workspace, model=model_1
        )

        # Test viewer user.
        d = client.Determined._from_session(viewer)
        user_with_view_perms_test(
            determined_obj=d, workspace_name=current_model_workspace, model=model_1
        )

        # Test noperms user.
        d = client.Determined._from_session(noperms)
        with pytest.raises(Exception) as e:
            d.get_models()
        assert "doesn't have view permissions" in str(e.value)

        # Unassign view permissions to a certain workspace.
        # List should return models only in workspaces with permissions.
        detproc.check_call(
            admin,
            [
                "det",
                "rbac",
                "unassign-role",
                "ModelRegistryViewer",
                "--username-to-assign",
                model_registry_viewer.username,
                "--workspace-name",
                test_workspace.name,
            ],
        )

        d = client.Determined._from_session(model_registry_viewer)
        models = d.get_models()
        assert test_workspace.id not in [m.workspace_id for m in models]

        d = client.Determined._from_session(editor)
        model = d.get_model(model_1.name)
        model.delete()

        d = client.Determined._from_session(wksp_admin)
        checkpoint = d.get_experiment(exp.id).top_checkpoint()
        model_2, current_model_workspace = all_operations(
            determined_obj=d, test_workspace=test_workspace, checkpoint=checkpoint
        )

        # Remove workspace admin role for this user from test_workspace.
        detproc.check_call(
            admin,
            [
                "det",
                "rbac",
                "unassign-role",
                "WorkspaceAdmin",
                "--username-to-assign",
                wksp_admin.username,
                "--workspace-name",
                test_workspace.name,
            ],
        )

        d = client.Determined._from_session(wksp_admin)
        model = d.get_model(model_2.name)
        assert current_model_workspace == "Uncategorized"
        # move model to test_workspace should fail.
        with pytest.raises(errors.ForbiddenException) as e:
            model.move_to_workspace(workspace_name=test_workspace.name)
        assert "access denied" in str(e.value)
        model.delete()


@pytest.mark.test_model_registry_rbac
def test_model_rbac_deletes() -> None:
    with test_rbac.create_workspaces_with_users(
        [
            [
                (0, ["Editor"]),
            ],
        ]
    ) as (workspaces, creds):
        workspace_id = workspaces[0].id
        # create non-cluster admin user
        editor_session = creds[0]

        # create cluster admin user
        cluster_admin, _ = api_utils.create_test_user(
            user=bindings.v1User(username=get_random_string(), active=True, admin=False),
        )
        api_utils.assign_user_role(
            session=api_utils.admin_session(),
            user=cluster_admin.username,
            role="ClusterAdmin",
            workspace=None,
        )

        # create non-cluster admin user with OSS admin flag
        oss_admin, _ = api_utils.create_test_user(
            user=bindings.v1User(username=get_random_string(), active=True, admin=True),
        )

        model_num = 0
        try:
            # test deleting model registries
            tests: List[Dict[str, Any]] = [
                {
                    "create_session": cluster_admin,
                    "delete_session": cluster_admin,
                    "should_error": False,
                },
                {
                    "create_session": editor_session,
                    "delete_session": cluster_admin,
                    "should_error": False,
                },
                {
                    "create_session": editor_session,
                    "delete_session": editor_session,
                    "should_error": False,
                },
                {
                    "create_session": cluster_admin,
                    "delete_session": editor_session,
                    "should_error": True,
                },
                {
                    "create_session": cluster_admin,
                    "delete_session": oss_admin,
                    "should_error": True,
                },
            ]
            for t in tests:
                create_session: api.Session = t["create_session"]
                delete_session: api.Session = t["delete_session"]
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
                    "create_session": cluster_admin,
                    "delete_session": cluster_admin,
                    "should_error": False,
                },
                {
                    "create_session": editor_session,
                    "delete_session": editor_session,
                    "should_error": False,
                },
                {
                    "create_session": editor_session,
                    "delete_session": cluster_admin,
                    "should_error": False,
                },
                {
                    "create_session": cluster_admin,
                    "delete_session": editor_session,
                    "should_error": True,
                },
                {
                    "create_session": cluster_admin,
                    "delete_session": oss_admin,
                    "should_error": True,
                },
            ]

            for t in tests:
                create_session = t["create_session"]
                delete_session = t["delete_session"]
                should_error = t["should_error"]

                model_name = "model_" + str(model_num)
                model_num += 1
                m, ca_model_version = register_model_version(
                    sess=create_session, model_name=model_name, workspace_id=workspace_id
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
                            create_session,
                            modelName=model_name,
                            modelVersionNum=model_version_num,
                        )
                    assert "not found" in str(notFoundErr.value).lower()
        finally:
            admin_session = api_utils.admin_session()
            for i in range(model_num):
                try:
                    bindings.delete_DeleteModel(admin_session, modelName="model_" + str(i))
                # model is has already been cleaned up
                except errors.NotFoundException:
                    continue
