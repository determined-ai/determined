import random
import time

import pytest

from determined.common import yaml
from determined.common.api import bindings, errors
from determined.experimental import client as _client
from tests import config as conf


@pytest.fixture(scope="session")
def client() -> _client.Determined:
    """
    Reduce logins by having one session-level fixture do the login.
    """
    return _client.Determined(conf.make_master_url())


@pytest.mark.e2e_cpu
def test_completed_experiment_and_checkpoint_apis(client: _client.Determined) -> None:
    with open(conf.fixtures_path("no_op/single-one-short-step.yaml")) as f:
        config = yaml.safe_load(f)
    config["hyperparameters"]["num_validation_metrics"] = 2
    exp = client.create_experiment(config, conf.fixtures_path("no_op"))
    assert exp.wait() == _client.ExperimentState.COMPLETED

    trials = exp.get_trials()
    assert len(trials) == 1, trials
    trial = trials[0]
    assert client.get_trial(trial.id).id == trial.id

    ckpt = trial.top_checkpoint()

    # Training checkpoints should have training metadata.
    assert ckpt.training is not None
    assert ckpt.training.trial_id == trial.id
    assert ckpt.training.experiment_id == exp.id

    # Various ways to look up the trial.
    assert trial.select_checkpoint(uuid=ckpt.uuid).uuid == ckpt.uuid
    assert trial.select_checkpoint(latest=True).uuid == ckpt.uuid
    assert trial.select_checkpoint(best=True).uuid == ckpt.uuid
    assert (
        trial.select_checkpoint(
            best=True, sort_by="validation_metric_1", smaller_is_better=True
        ).uuid
        == ckpt.uuid
    )
    assert exp.top_checkpoint().uuid == ckpt.uuid
    assert ckpt.uuid in (c.uuid for c in exp.top_n_checkpoints(100))
    assert client.get_checkpoint(ckpt.uuid).uuid == ckpt.uuid

    # Adding checkpoint metadata.
    ckpt.add_metadata({"newkey": "newvalue"})
    # Cache should be updated.
    assert ckpt.metadata["newkey"] == "newvalue"
    # Database should be updated.
    assert client.get_checkpoint(ckpt.uuid).metadata["newkey"] == "newvalue"

    # Removing checkpoint metadata
    ckpt.remove_metadata(["newkey"])
    assert "newkey" not in ckpt.metadata
    assert "newkey" not in client.get_checkpoint(ckpt.uuid).metadata


def _make_live_experiment(client: _client.Determined) -> _client.ExperimentReference:
    # Create an experiment that takes a long time to run
    with open(conf.fixtures_path("no_op/single-very-many-long-steps.yaml")) as f:
        config = yaml.safe_load(f)

    exp = client.create_experiment(config, conf.fixtures_path("no_op"))
    # Wait for a trial to actually start.
    start = time.time()
    deadline = start + 90
    while True:
        trials = exp.get_trials()
        if trials:
            break
        assert time.time() < deadline, "experiment took too long to start trials"
        time.sleep(0.1)

    return exp


@pytest.mark.e2e_cpu
def test_experiment_manipulation(client: _client.Determined) -> None:
    exp = _make_live_experiment(client)

    exp.pause()
    with pytest.raises(ValueError, match="Make sure the experiment is active"):
        # Wait throws an error if the experiment is paused by a user.
        exp.wait(interval=0.1)

    exp.activate()

    exp.cancel()
    assert exp.wait() == _client.ExperimentState.CANCELED

    assert isinstance(exp.get_config(), dict)

    # Delete this experiment, but continue the test while it's deleting.
    exp.delete()
    deleting_exp = exp

    # Create another experiment and kill it.
    exp = _make_live_experiment(client)
    exp.kill()
    assert exp.wait() == _client.ExperimentState.CANCELED

    # Test remaining methods
    exp.archive()
    assert bindings.get_GetExperiment(client._session, experimentId=exp.id).experiment.archived

    exp.unarchive()
    assert not bindings.get_GetExperiment(client._session, experimentId=exp.id).experiment.archived

    # Create another experiment and kill its trial.
    exp = _make_live_experiment(client)
    trial = exp.get_trials()[0]
    trial.kill()
    assert exp.wait() == _client.ExperimentState.CANCELED

    # Make sure that the experiment we deleted earlier does actually delete.
    with pytest.raises(errors.APIException):
        for _ in range(300):
            client.get_experiment(deleting_exp.id).get_trials()
            time.sleep(0.1)


@pytest.mark.e2e_cpu
def test_models(client: _client.Determined) -> None:
    suffix = [random.sample("abcdefghijklmnopqrstuvwxyz", 1) for _ in range(10)]
    model_name = f"test-model-{suffix}"
    model = client.create_model(model_name)
    try:
        assert model_name in (m.name for m in client.get_models())

        model.archive()
        model.unarchive()

        labels = ["test-model-label-0", "test-model-label-1"]
        model.set_labels(labels)
        model.add_metadata({"a": 1, "b": 2, "c": 3})
        model.set_description("modeldescr")

        # Check cached values
        assert set(client.get_model_labels()) == set(labels)
        assert model.metadata == {"a": 1, "b": 2, "c": 3}, model.metadata
        assert model.description == "modeldescr", model.description

        # avoid false-positives due to caching on the model object itself
        model = client.get_model(model_name)
        assert set(model.labels) == set(labels)
        assert model.metadata == {"a": 1, "b": 2, "c": 3}, model.metadata
        assert model.description == "modeldescr", model.description

        model.set_labels([])
        model.remove_metadata(["a", "b"])

        # break the cache again, testing get_model_by_id.
        model = client.get_model_by_id(model.model_id)
        assert model.labels == []
        assert model.metadata == {"c": 3}, model.metadata

    finally:
        model.delete()

    with pytest.raises(errors.APIException):
        client.get_model(model_name)


@pytest.mark.e2e_cpu
def test_model_versions(client: _client.Determined) -> None:
    with open(conf.fixtures_path("no_op/single-one-short-step.yaml")) as f:
        config = yaml.safe_load(f)
    exp = client.create_experiment(config, conf.fixtures_path("no_op"))
    assert exp.wait() == _client.ExperimentState.COMPLETED
    ckpt = exp.top_checkpoint()

    suffix = [random.sample("abcdefghijklmnopqrstuvwxyz", 1) for _ in range(10)]
    model_name = f"test-model-{suffix}"
    model = client.create_model(model_name)
    try:
        ver = model.register_version(ckpt.uuid)

        assert ver.model_version in (v.model_version for v in model.get_versions())

        ver.set_name("vername")
        ver.set_notes("vernotes")

        # Check the cache.
        assert ver.name == "vername", ver.name
        assert ver.notes == "vernotes", ver.notes

        # Break the cache.
        ver2 = model.get_version(ver.model_version_id)
        assert ver2 is not None
        assert ver2.name == "vername", ver2.name
        assert ver2.notes == "vernotes", ver2.notes

        # Test get_version without an arg, while a version exists.
        ver3 = model.get_version()
        assert ver3
        assert ver3.model_version_id == ver.model_version_id

        ver2.delete()

        with pytest.raises(errors.APIException):
            model.get_version(ver.model_version_id)

        # Test get_version without an arg, when no version exists.
        assert model.get_version() is None

    finally:
        model.delete()
