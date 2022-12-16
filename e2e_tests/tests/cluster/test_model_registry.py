import subprocess
import uuid

import pytest

from determined.common import api
from determined.common.api import authentication, bindings
from determined.experimental import Determined, ModelSortBy
from tests import config as conf
from tests import experiment as exp
from tests.cluster.test_users import ADMIN_CREDENTIALS, log_in_user, log_out_user


@pytest.mark.e2e_cpu
def test_model_registry() -> None:
    exp_id = exp.run_basic_test(
        conf.fixtures_path("mnist_pytorch/const-pytorch11.yaml"),
        conf.tutorials_path("mnist_pytorch"),
        None,
    )

    log_out_user()  # Ensure that we use determined credentials.

    d = Determined(conf.make_master_url())
    mnist = None
    objectdetect = None
    tform = None

    existing_models = [m.name for m in d.get_models(sort_by=ModelSortBy.NAME)]

    try:
        # Create a model and validate twiddling the metadata.
        mnist = d.create_model("mnist", "simple computer vision model", labels=["a", "b"])
        assert mnist.metadata == {}

        mnist.add_metadata({"testing": "metadata"})
        db_model = d.get_model(mnist.name)
        # Make sure the model metadata is correct and correctly saved to the db.
        assert mnist.metadata == db_model.metadata
        assert mnist.metadata == {"testing": "metadata"}

        # Confirm we can look up a model by its ID
        db_model = d.get_model_by_id(mnist.model_id)
        assert db_model.name == "mnist"
        db_model = d.get_model(mnist.model_id)
        assert db_model.name == "mnist"

        # Confirm DB assigned username
        assert db_model.username == "determined"

        mnist.add_metadata({"some_key": "some_value"})
        db_model = d.get_model(mnist.name)
        assert mnist.metadata == db_model.metadata
        assert mnist.metadata == {"testing": "metadata", "some_key": "some_value"}

        mnist.add_metadata({"testing": "override"})
        db_model = d.get_model(mnist.name)
        assert mnist.metadata == db_model.metadata
        assert mnist.metadata == {"testing": "override", "some_key": "some_value"}

        mnist.remove_metadata(["some_key"])
        db_model = d.get_model(mnist.name)
        assert mnist.metadata == db_model.metadata
        assert mnist.metadata == {"testing": "override"}

        mnist.set_labels(["hello", "world"])
        db_model = d.get_model(mnist.name)
        assert mnist.labels == db_model.labels
        assert db_model.labels == ["hello", "world"]

        # confirm patch does not overwrite other fields
        mnist.set_description("abcde")
        db_model = d.get_model(mnist.name)
        assert db_model.metadata == {"testing": "override"}
        assert db_model.labels == ["hello", "world"]

        # overwrite labels to empty list
        mnist.set_labels([])
        db_model = d.get_model(mnist.name)
        assert db_model.labels == []

        # archive and unarchive
        assert mnist.archived is False
        mnist.archive()
        db_model = d.get_model(mnist.name)
        assert db_model.archived is True
        mnist.unarchive()
        db_model = d.get_model(mnist.name)
        assert db_model.archived is False

        # Register a version for the model and validate the latest.
        checkpoint = d.get_experiment(exp_id).top_checkpoint()
        model_version = mnist.register_version(checkpoint.uuid)
        assert model_version.model_version == 1

        latest_version = mnist.get_version()
        assert latest_version is not None
        assert latest_version.checkpoint.uuid == checkpoint.uuid

        latest_version.set_name("Test 2021")
        db_version = mnist.get_version()
        assert db_version is not None
        assert db_version.name == "Test 2021"

        latest_version.set_notes("# Hello Markdown")
        db_version = mnist.get_version()
        assert db_version is not None
        assert db_version.notes == "# Hello Markdown"

        # Run another basic test and register its checkpoint as a version as well.
        # Validate the latest has been updated.
        exp_id = exp.run_basic_test(
            conf.fixtures_path("mnist_pytorch/const-pytorch11.yaml"),
            conf.tutorials_path("mnist_pytorch"),
            None,
        )
        checkpoint = d.get_experiment(exp_id).top_checkpoint()
        model_version = mnist.register_version(checkpoint.uuid)
        assert model_version.model_version == 2

        latest_version = mnist.get_version()
        assert latest_version is not None
        assert latest_version.checkpoint.uuid == checkpoint.uuid

        # Ensure the correct number of versions are present.
        all_versions = mnist.get_versions()
        assert len(all_versions) == 2

        for v in all_versions:
            mv = mnist.get_version(v.model_version)
            assert mv is not None
            assert mv.model_version == v.model_version

        # Test deletion of model version
        latest_version.delete()
        all_versions = mnist.get_versions()
        assert len(all_versions) == 1

        # Create some more models and validate listing models.
        tform = d.create_model("transformer", "all you need is attention")
        objectdetect = d.create_model("ac - Dc", "a test name model")

        models = d.get_models(sort_by=ModelSortBy.NAME)
        model_names = [m.name for m in models if m.name not in existing_models]
        assert model_names == ["ac - Dc", "mnist", "transformer"]

        # Test model labels combined
        mnist.set_labels(["hello", "world"])
        tform.set_labels(["world", "test", "zebra"])
        labels = d.get_model_labels()
        assert labels == ["world", "hello", "test", "zebra"]

        # Test deletion of model
        tform.delete()
        tform = None
        models = d.get_models(sort_by=ModelSortBy.NAME)
        model_names = [m.name for m in models if m.name not in existing_models]
        assert model_names == ["ac - Dc", "mnist"]
    finally:
        # Clean model registry of test models
        for model in [mnist, objectdetect, tform]:
            if model is not None:
                model.delete()


def get_random_string() -> str:
    return str(uuid.uuid4())


@pytest.mark.e2e_cpu
def test_model_cli() -> None:
    test_model_1_name = get_random_string()
    master_url = conf.make_master_url()
    command = ["det", "-m", master_url, "model", "create", test_model_1_name]
    subprocess.run(command, check=True)
    d = Determined(master_url)
    model_1 = d.get_model(identifier=test_model_1_name)
    assert model_1.workspace_id == 1
    # Test det model list and det model describe
    command = ["det", "-m", master_url, "model", "list"]
    output = str(subprocess.check_output(command))
    assert "Workspace ID" in output and "1" in output

    command = ["det", "-m", master_url, "model", "describe", test_model_1_name]
    output = str(subprocess.check_output(command))
    assert "Workspace ID" in output and "1" in output

    # add a test workspace.
    log_in_user(ADMIN_CREDENTIALS)
    admin_auth = authentication.Authentication(
        master_url, ADMIN_CREDENTIALS.username, ADMIN_CREDENTIALS.password
    )
    admin_sess = api.Session(master_url, ADMIN_CREDENTIALS.username, admin_auth, None)
    test_workspace_name = get_random_string()
    test_workspace = bindings.post_PostWorkspace(
        admin_sess, body=bindings.v1PostWorkspaceRequest(name=test_workspace_name)
    ).workspace

    # create model in test_workspace
    test_model_2_name = get_random_string()
    command = [
        "det",
        "-m",
        master_url,
        "model",
        "create",
        test_model_2_name,
        "-w",
        test_workspace_name,
    ]
    subprocess.run(command, check=True)
    model_2 = d.get_model(identifier=test_model_2_name)
    assert model_2.workspace_id == test_workspace.id

    # Test det model list -w workspace_name and det model describe
    command = ["det", "-m", master_url, "model", "list", "-w", test_workspace.name]
    output = str(subprocess.check_output(command))
    assert (
        "Workspace ID" in output
        and str(test_workspace.id) in output
        and test_model_2_name in output
        and test_model_1_name not in output
    )  # should only output models in given workspace

    # move test_model_1 to test_workspace
    command = [
        "det",
        "-m",
        master_url,
        "model",
        "move",
        test_model_1_name,
        "-w",
        test_workspace_name,
    ]
    subprocess.run(command, check=True)
    model_1 = d.get_model(test_model_1_name)
    assert model_1.workspace_id == test_workspace.id

    # Delete test models and workspace
    model_1.delete()
    model_2.delete()
    bindings.delete_DeleteWorkspace(session=admin_sess, id=test_workspace.id)
