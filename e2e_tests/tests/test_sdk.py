import os
import random
import shutil
import tempfile
import time

import pytest

from determined.common import yaml
from determined.common.api import bindings, errors
from determined.common.experimental import resource_pool
from determined.common.experimental.metrics import TrialMetrics
from determined.experimental import client as _client
from tests import config as conf


@pytest.mark.e2e_cpu
def test_completed_experiment_and_checkpoint_apis(client: _client.Determined) -> None:
    with open(conf.fixtures_path("no_op/single-one-short-step.yaml")) as f:
        config = yaml.safe_load(f)
    config["hyperparameters"]["num_validation_metrics"] = 2
    # Test the use of the includes parameter, by feeding the model definition file via includes.
    emptydir = tempfile.mkdtemp()
    try:
        model_def = conf.fixtures_path("no_op/model_def.py")
        exp = client.create_experiment(config, emptydir, includes=[model_def])
    finally:
        os.rmdir(emptydir)
    exp = client.create_experiment(config, conf.fixtures_path("no_op"))

    # Await first trial is safe to call before a trial has started.
    trial = exp.await_first_trial()

    # .logs(follow=True) block until the trial completes.
    all_logs = list(trial.logs(follow=True))

    assert exp.wait() == _client.ExperimentState.COMPLETED

    assert all_logs == list(trial.logs())
    assert list(trial.logs(head=10)) == all_logs[:10]
    assert list(trial.logs(tail=10)) == all_logs[-10:]

    trials = exp.get_trials()
    assert len(trials) == 1, trials
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
    assert len(trial.get_checkpoints()) == 1
    assert trial.get_checkpoints()[0].uuid == ckpt.uuid

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


@pytest.mark.e2e_cpu
def test_checkpoint_apis(client: _client.Determined) -> None:
    with open(conf.fixtures_path("no_op/single-default-ckpt.yaml")) as f:
        config = yaml.safe_load(f)

    # Test for 100 batches/checkpoint every 10 = 10 checkpoints.
    config["min_checkpoint_period"]["batches"] = 10
    config["min_validation_period"]["batches"] = 10
    config["checkpoint_storage"] = {}
    config["checkpoint_storage"]["save_trial_best"] = 10

    exp = client.create_experiment(config, conf.fixtures_path("no_op"))

    # Await first trial is safe to call before a trial has started.
    trial = exp.await_first_trial()

    assert exp.wait() == _client.ExperimentState.COMPLETED
    trials = exp.get_trials()
    assert len(trials) == 1, trials

    checkpoints = trial.get_checkpoints()
    assert len(checkpoints) == 10

    # Validate end (report) time sorting.
    checkpoints = trial.get_checkpoints(
        sort_by=_client.CheckpointSortBy.END_TIME, order_by=_client.CheckpointOrderBy.DESC
    )
    end_times = [checkpoint.report_time for checkpoint in checkpoints]
    assert all(x >= y for x, y in zip(end_times, end_times[1:]))  # type: ignore

    # Validate state sorting.
    checkpoints = trial.get_checkpoints(
        sort_by=_client.CheckpointSortBy.STATE, order_by=_client.CheckpointOrderBy.ASC
    )
    states = [checkpoint.state.value for checkpoint in checkpoints]
    assert all(x <= y for x, y in zip(states, states[1:]))

    # Validate UUID sorting.
    checkpoints = trial.get_checkpoints(
        sort_by=_client.CheckpointSortBy.UUID, order_by=_client.CheckpointOrderBy.ASC
    )
    uuids = [checkpoint.uuid for checkpoint in checkpoints]
    assert all(x <= y for x, y in zip(uuids, uuids[1:]))

    # Validate batch number sorting.
    checkpoints = trial.get_checkpoints(
        sort_by=_client.CheckpointSortBy.BATCH_NUMBER, order_by=_client.CheckpointOrderBy.DESC
    )
    batch_numbers = [checkpoint.metadata["steps_completed"] for checkpoint in checkpoints]
    assert all(x >= y for x, y in zip(batch_numbers, batch_numbers[1:]))

    # Validate metric sorting.
    checkpoints = trial.get_checkpoints(
        sort_by="validation_error", order_by=_client.CheckpointOrderBy.ASC
    )
    validation_metrics = [
        checkpoint.training.validation_metrics["avgMetrics"]["validation_error"]  # type: ignore
        for checkpoint in checkpoints
    ]
    assert all(x <= y for x, y in zip(validation_metrics, validation_metrics[1:]))

    # Expect 10 completed checkpoints.
    checkpoints = [
        checkpoint
        for checkpoint in checkpoints
        if checkpoint.state == _client.CheckpointState.COMPLETED
    ]
    assert len(checkpoints) == 10

    # Delete first checkpoint.
    deleted_checkpoint = checkpoints[0]
    checkpoints[0].delete()

    # Wait for status to be DELETED.
    start = time.time()
    deadline = start + 30
    while True:
        checkpoints = trial.get_checkpoints()
        deleted_checkpoints = [
            checkpoint
            for checkpoint in checkpoints
            if checkpoint.state == _client.CheckpointState.DELETED
        ]
        if deleted_checkpoints:
            break
        assert time.time() < deadline, "experiment took too long to start trials"
        time.sleep(0.1)

    assert len(deleted_checkpoints) == 1
    assert deleted_checkpoints[0].uuid == deleted_checkpoint.uuid

    # Partially delete first checkpoint.
    partially_deleted_checkpoint = checkpoints[1]
    partially_deleted_checkpoint.remove_files(["*.pkl"])

    # Wait for status to be PARTIALLY_DELETED
    partially_deleted_checkpoints = []
    start = time.time()
    deadline = start + 30
    while True:
        checkpoints = trial.get_checkpoints()
        partially_deleted_checkpoints = [
            checkpoint
            for checkpoint in checkpoints
            if checkpoint.state == _client.CheckpointState.PARTIALLY_DELETED
        ]
        if partially_deleted_checkpoints:
            break
        assert time.time() < deadline, "checkpoint took too long to partially delete"
        time.sleep(0.1)
    assert len(partially_deleted_checkpoints) == 1
    assert partially_deleted_checkpoints[0].uuid == partially_deleted_checkpoint.uuid
    assert "workload_sequencer.pkl" not in partially_deleted_checkpoints[0].resources

    # Ensure we can download the partially deleted checkpoint.
    temp_dir = tempfile.mkdtemp()
    try:
        downloaded_path = partially_deleted_checkpoints[0].download(
            path=os.path.join(temp_dir, "c")
        )
        files = os.listdir(downloaded_path)
        assert "no_op_checkpoint" in files
        assert "workload_sequencer.pkl" not in files
    finally:
        shutil.rmtree(temp_dir, ignore_errors=False)

    # Ensure we can delete a partially deleted checkpoint.
    partially_deleted_checkpoints[0].delete()
    start = time.time()
    deadline = start + 30
    while True:
        checkpoints = trial.get_checkpoints()
        deleted_checkpoints = [
            checkpoint
            for checkpoint in checkpoints
            if checkpoint.state == _client.CheckpointState.DELETED
            and checkpoint.uuid == partially_deleted_checkpoint.uuid
        ]
        if deleted_checkpoints:
            break
        assert time.time() < deadline, "partially deleted checkpoint took too long to delete"
        time.sleep(0.1)


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
def test_stream_metrics(client: _client.Determined) -> None:
    with open(conf.fixtures_path("no_op/single-one-short-step.yaml")) as f:
        config = yaml.safe_load(f)
    config["hyperparameters"]["num_validation_metrics"] = 2
    exp = client.create_experiment(config, conf.fixtures_path("no_op"))
    assert exp.wait() == _client.ExperimentState.COMPLETED

    trials = exp.get_trials()
    assert len(trials) == 1
    trial = trials[0]

    for metrics in [
        list(trial.stream_training_metrics()),
        list(client.stream_trials_training_metrics([trial.id])),
    ]:
        assert len(metrics) == config["searcher"]["max_length"]["batches"]
        for i, actual in enumerate(metrics):
            # assert actual == TrainingMetrics(
            assert actual == TrialMetrics(
                trial_id=trial.id,
                trial_run_id=1,
                steps_completed=i + 1,
                end_time=actual.end_time,
                metrics={"loss": config["hyperparameters"]["metrics_base"] ** (i + 1)},
                batch_metrics=[{"loss": config["hyperparameters"]["metrics_base"] ** (i + 1)}],
                group="training",
            )

    for val_metrics in [
        list(trial.stream_validation_metrics()),
        list(client.stream_trials_validation_metrics([trial.id])),
    ]:
        assert len(val_metrics) == 1
        # assert val_metrics[0] == ValidationMetrics(
        assert val_metrics[0] == TrialMetrics(
            trial_id=trial.id,
            trial_run_id=1,
            steps_completed=100,
            end_time=val_metrics[0].end_time,
            metrics={
                "validation_error": config["hyperparameters"]["metrics_base"] ** 100,
                "validation_metric_1": config["hyperparameters"]["metrics_base"] ** 100,
            },
            group="validation",
        )


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
        ver2 = model.get_version(ver.model_version)
        assert ver2 is not None
        assert ver2.name == "vername", ver2.name
        assert ver2.notes == "vernotes", ver2.notes

        # Test get_version without an arg, while a version exists.
        ver3 = model.get_version()
        assert ver3
        assert ver3.model_version == ver.model_version

        ver2.delete()

        with pytest.raises(errors.APIException):
            model.get_version(ver.model_version)

        # Test get_version without an arg, when no version exists.
        assert model.get_version() is None

    finally:
        model.delete()


@pytest.mark.e2e_cpu
def test_rp_workspace_mapping(client: _client.Determined) -> None:
    workspace_names = ["Workspace A", "Workspace B"]
    overwrite_workspace_names = ["Workspace C", "Workspace D"]
    rp_names = ["default"]  # TODO: not sure how to add more rp
    workspace_ids = []

    for wn in workspace_names + overwrite_workspace_names:
        req = bindings.v1PostWorkspaceRequest(name=wn)
        workspace_ids.append(
            bindings.post_PostWorkspace(session=client._session, body=req).workspace.id
        )

    try:
        with pytest.raises(
            errors.APIException,
            match="default resource pool default cannot be bound to any workspace",
        ):
            rp = resource_pool.ResourcePool(client._session, rp_names[0])
            rp.add_bindings(workspace_names)
    finally:
        for wid in workspace_ids:
            bindings.delete_DeleteWorkspace(session=client._session, id=wid)
