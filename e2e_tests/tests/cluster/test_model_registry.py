import pytest

from determined.experimental import Determined, ModelSortBy
from tests import config as conf
from tests import experiment as exp


@pytest.mark.e2e_cpu
def test_model_registry() -> None:
    exp_id = exp.run_basic_test(
        conf.fixtures_path("mnist_pytorch/const-pytorch11.yaml"),
        conf.tutorials_path("mnist_pytorch"),
        None,
    )

    d = Determined(conf.make_master_url())

    # Create a model and validate twiddling the metadata.
    mnist = d.create_model("mnist", "simple computer vision model")
    assert mnist.metadata == {}

    mnist.add_metadata({"testing": "metadata"})
    db_model = d.get_model("mnist")
    # Make sure the model metadata is correct and correctly saved to the db.
    assert mnist.metadata == db_model.metadata
    assert mnist.metadata == {"testing": "metadata"}

    mnist.add_metadata({"some_key": "some_value"})
    db_model = d.get_model("mnist")
    assert mnist.metadata == db_model.metadata
    assert mnist.metadata == {"testing": "metadata", "some_key": "some_value"}

    mnist.add_metadata({"testing": "override"})
    db_model = d.get_model("mnist")
    assert mnist.metadata == db_model.metadata
    assert mnist.metadata == {"testing": "override", "some_key": "some_value"}

    mnist.remove_metadata(["some_key"])
    db_model = d.get_model("mnist")
    assert mnist.metadata == db_model.metadata
    assert mnist.metadata == {"testing": "override"}

    # Register a version for the model and validate the latest.
    checkpoint = d.get_experiment(exp_id).top_checkpoint()
    model_version = mnist.register_version(checkpoint.uuid)
    assert model_version.model_version == 1

    latest_version = mnist.get_version()
    assert latest_version is not None
    assert latest_version.uuid == checkpoint.uuid

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
    assert latest_version.uuid == checkpoint.uuid

    # Ensure the correct number of versions are present.
    all_versions = mnist.get_versions()
    assert len(all_versions) == 2

    # Create some more models and validate listing models.
    d.create_model("transformer", "all you need is attention")
    d.create_model("object-detection", "a bounding box model")

    models = d.get_models(sort_by=ModelSortBy.NAME)
    assert [m.name for m in models] == ["mnist", "object-detection", "transformer"]
