import os
import pathlib
import random
import time

import pytest

from determined.common.api import bindings, errors
from determined.experimental import client
from tests import api_utils
from tests import config as conf
from tests.experiment import noop


@pytest.mark.e2e_cpu
def test_completed_experiment_and_checkpoint_apis(tmp_path: pathlib.Path) -> None:
    sess = api_utils.user_session()
    detobj = client.Determined._from_session(sess)

    config = noop.generate_config(
        [
            noop.Report({"x": 3, "z": 1}, group="training"),
            noop.Report({"x": 3, "z": 1}, group="validation"),
            noop.Checkpoint(),
            noop.Report({"x": 2, "z": 2}, group="training"),
            noop.Report({"x": 2, "z": 2}, group="validation"),
            noop.Checkpoint(),
            noop.Report({"x": 1, "z": 3}, group="training"),
            noop.Report({"x": 1, "z": 3}, group="validation"),
            noop.Checkpoint(),
        ]
    )
    # Test creation of experiment without a model definition.
    exp = detobj.create_experiment(config)
    exp.kill()
    # Test the use of the includes parameter, by feeding the model definition file via includes.
    emptydir = tmp_path
    model_def = conf.fixtures_path("noop/train.py")
    exp = detobj.create_experiment(config, emptydir, includes=[model_def])
    exp.kill()
    exp = detobj.create_experiment(config, conf.fixtures_path("noop"))

    # Await first trial is safe to call before a trial has started.
    trial = exp.await_first_trial()

    # .logs(follow=True) block until the trial completes.
    all_logs = list(trial.logs(follow=True))

    assert exp.wait(interval=0.01) == client.ExperimentState.COMPLETED

    assert all_logs == list(trial.logs())
    assert list(trial.logs(head=10)) == all_logs[:10]
    assert list(trial.logs(tail=10)) == all_logs[-10:]

    trials = exp.get_trials()
    assert len(trials) == 1, trials
    assert detobj.get_trial(trial.id).id == trial.id

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
        trial.select_checkpoint(best=True, sort_by="z", smaller_is_better=False).uuid == ckpt.uuid
    )
    assert len(trial.get_checkpoints()) == 3
    assert trial.get_checkpoints()[0].uuid == ckpt.uuid

    assert exp.top_checkpoint().uuid == ckpt.uuid
    assert ckpt.uuid in (c.uuid for c in exp.top_n_checkpoints(100))
    assert detobj.get_checkpoint(ckpt.uuid).uuid == ckpt.uuid

    # Adding checkpoint metadata.
    ckpt.add_metadata({"newkey": "newvalue"})
    # Cache should be updated.
    assert ckpt.metadata
    assert ckpt.metadata["newkey"] == "newvalue"
    # Database should be updated.
    ckpt = detobj.get_checkpoint(ckpt.uuid)
    assert ckpt.metadata
    assert ckpt.metadata["newkey"] == "newvalue"

    # Removing checkpoint metadata
    ckpt.remove_metadata(["newkey"])
    assert "newkey" not in ckpt.metadata
    ckpt = detobj.get_checkpoint(ckpt.uuid)
    assert ckpt.metadata
    assert "newkey" not in ckpt.metadata


@pytest.mark.e2e_cpu
def test_checkpoint_apis(tmp_path: pathlib.Path) -> None:
    sess = api_utils.user_session()
    detobj = client.Determined._from_session(sess)
    # Create and keep 10 checkpoitns.
    config = {"checkpoint_storage": {"save_trial_best": 10}}
    config = noop.generate_config(noop.traininglike_steps(10, metric_scale=0.9), config=config)
    exp = detobj.create_experiment(config, conf.fixtures_path("noop"))

    # Await first trial is safe to call before a trial has started.
    trial = exp.await_first_trial()

    assert exp.wait(interval=0.01) == client.ExperimentState.COMPLETED
    trials = exp.get_trials()
    assert len(trials) == 1, trials

    checkpoints = trial.get_checkpoints()
    assert len(checkpoints) == 10

    # Validate end (report) time sorting.
    checkpoints = trial.get_checkpoints(
        sort_by=client.CheckpointSortBy.END_TIME, order_by=client.OrderBy.DESC
    )
    end_times = [checkpoint.report_time for checkpoint in checkpoints]
    assert all(x >= y for x, y in zip(end_times, end_times[1:]))  # type: ignore

    # Validate state sorting.
    checkpoints = trial.get_checkpoints(
        sort_by=client.CheckpointSortBy.STATE, order_by=client.OrderBy.ASC
    )
    states = []
    for checkpoint in checkpoints:
        assert checkpoint.state
        states.append(checkpoint.state.value)
    assert all(x <= y for x, y in zip(states, states[1:]))

    # Validate UUID sorting.
    checkpoints = trial.get_checkpoints(
        sort_by=client.CheckpointSortBy.UUID, order_by=client.OrderBy.ASC
    )
    uuids = [checkpoint.uuid for checkpoint in checkpoints]
    assert all(x <= y for x, y in zip(uuids, uuids[1:]))

    # Validate batch number sorting.
    checkpoints = trial.get_checkpoints(
        sort_by=client.CheckpointSortBy.BATCH_NUMBER, order_by=client.OrderBy.DESC
    )
    batch_numbers = []
    for checkpoint in checkpoints:
        assert checkpoint.metadata
        batch_numbers.append(checkpoint.metadata["steps_completed"])
    assert all(x >= y for x, y in zip(batch_numbers, batch_numbers[1:]))

    # Validate metric sorting.
    checkpoints = trial.get_checkpoints(sort_by="x", order_by=client.OrderBy.ASC)
    validation_metrics = [
        checkpoint.training.validation_metrics["avgMetrics"]["x"]  # type: ignore
        for checkpoint in checkpoints
    ]
    assert all(x <= y for x, y in zip(validation_metrics, validation_metrics[1:]))

    # Expect 10 completed checkpoints.
    checkpoints = [c for c in checkpoints if c.state == client.CheckpointState.COMPLETED]
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
            if checkpoint.state == client.CheckpointState.DELETED
        ]
        if deleted_checkpoints:
            break
        assert time.time() < deadline, "checkpoint deletion took too long"
        time.sleep(0.1)

    assert len(deleted_checkpoints) == 1
    assert deleted_checkpoints[0].uuid == deleted_checkpoint.uuid

    # Partially delete next checkpoint.
    partially_deleted_checkpoint = checkpoints[1]
    partially_deleted_checkpoint.remove_files(["state"])

    # Wait for status to be PARTIALLY_DELETED
    partially_deleted_checkpoints = []
    start = time.time()
    deadline = start + 30
    while True:
        checkpoints = trial.get_checkpoints()
        partially_deleted_checkpoints = [
            checkpoint
            for checkpoint in checkpoints
            if checkpoint.state == client.CheckpointState.PARTIALLY_DELETED
        ]
        if partially_deleted_checkpoints:
            break
        assert time.time() < deadline, "checkpoint took too long to partially delete"
        time.sleep(0.1)
    assert len(partially_deleted_checkpoints) == 1
    assert partially_deleted_checkpoints[0].uuid == partially_deleted_checkpoint.uuid
    assert partially_deleted_checkpoints[0].resources
    assert "state" not in partially_deleted_checkpoints[0].resources

    # Ensure we can download the partially deleted checkpoint.
    downloaded_path = partially_deleted_checkpoints[0].download(path=os.path.join(tmp_path, "c"))
    files = os.listdir(downloaded_path)
    assert "metadata.json" in files
    assert "state" not in files

    # Ensure we can delete a partially deleted checkpoint.
    partially_deleted_checkpoints[0].delete()
    start = time.time()
    deadline = start + 30
    while True:
        checkpoints = trial.get_checkpoints()
        deleted_checkpoints = [
            checkpoint
            for checkpoint in checkpoints
            if checkpoint.state == client.CheckpointState.DELETED
            and checkpoint.uuid == partially_deleted_checkpoint.uuid
        ]
        if deleted_checkpoints:
            break
        assert time.time() < deadline, "partially deleted checkpoint took too long to delete"
        time.sleep(0.1)


@pytest.mark.e2e_cpu
def test_experiment_manipulation() -> None:
    sess = api_utils.user_session()
    detobj = client.Determined._from_session(sess)

    def make_live_experiment() -> client.Experiment:
        # Create an experiment that takes a long time to run.
        actions = [noop.Sleep(1) for _ in range(100)]
        exp = noop.create_experiment(sess, actions)
        # Wait for a trial to actually start
        exp.await_first_trial()
        return exp

    exp = make_live_experiment()
    exp.pause()
    with pytest.raises(ValueError, match="Make sure the experiment is active"):
        # Wait throws an error if the experiment is paused by a user.
        exp.wait(interval=0.1)

    exp.activate()

    exp.cancel()
    assert exp.wait() == client.ExperimentState.CANCELED

    assert isinstance(exp.config, dict)

    # Delete this experiment, but continue the test while it's deleting.
    exp.delete()
    deleting_exp = exp

    # Create another experiment and kill it.
    exp = make_live_experiment()
    exp.kill()
    assert exp.wait() == client.ExperimentState.CANCELED

    # Test remaining methods
    exp.archive()
    assert bindings.get_GetExperiment(sess, experimentId=exp.id).experiment.archived

    exp.unarchive()
    assert not bindings.get_GetExperiment(sess, experimentId=exp.id).experiment.archived

    # Create another experiment and kill its trial.
    exp = make_live_experiment()
    trial = exp.get_trials()[0]
    trial.kill()
    assert exp.wait() == client.ExperimentState.CANCELED

    # Make sure that the experiment we deleted earlier does actually delete.
    with pytest.raises(errors.NotFoundException):
        for _ in range(300):
            detobj.get_experiment(deleting_exp.id).get_trials()
            time.sleep(0.1)


@pytest.mark.e2e_cpu
def test_models() -> None:
    sess = api_utils.user_session()
    detobj = client.Determined._from_session(sess)
    suffix = [random.sample("abcdefghijklmnopqrstuvwxyz", 1) for _ in range(10)]
    model_name = f"test-model-{suffix}"
    model = detobj.create_model(model_name)
    try:
        assert model_name in (m.name for m in detobj.get_models())

        model.archive()
        model.unarchive()

        labels = ["test-model-label-0", "test-model-label-1"]
        model.set_labels(labels)
        model.add_metadata({"a": 1, "b": 2, "c": 3})
        model.set_description("modeldescr")

        # Check cached values
        assert set(detobj.get_model_labels()) == set(labels)
        assert model.metadata == {"a": 1, "b": 2, "c": 3}, model.metadata
        assert model.description == "modeldescr", model.description

        # avoid false-positives due to caching on the model object itself
        model = detobj.get_model(model_name)
        assert model.labels
        assert set(model.labels) == set(labels)
        assert model.metadata == {"a": 1, "b": 2, "c": 3}, model.metadata
        assert model.description == "modeldescr", model.description

        model.set_labels([])
        model.remove_metadata(["a", "b"])

        # break the cache again, testing get_model_by_id.
        assert model.model_id is not None, "model_id was populated by create_model"
        model = detobj.get_model_by_id(model.model_id)
        assert model.labels == []
        assert model.metadata == {"c": 3}, model.metadata

    finally:
        model.delete()

    with pytest.raises(errors.NotFoundException):
        detobj.get_model(model_name)


@pytest.mark.e2e_cpu
def test_stream_metrics() -> None:
    sess = api_utils.user_session()
    detobj = client.Determined._from_session(sess)
    exp = noop.create_experiment(sess, noop.traininglike_steps(10, metric_scale=0.5))
    assert exp.wait(interval=0.01) == client.ExperimentState.COMPLETED

    trial = exp.get_trials()[0]

    for metrics in [
        list(trial.stream_metrics("training")),
        list(detobj.stream_trials_metrics([trial.id], "training")),
    ]:
        assert len(metrics) == 10
        for i, actual in enumerate(metrics):
            assert actual == client.TrialMetrics(
                trial_id=trial.id,
                trial_run_id=1,
                steps_completed=i + 1,
                end_time=actual.end_time,
                metrics={"x": 0.5**i},
                group="training",
            )

    for val_metrics in [
        list(trial.stream_metrics("validation")),
        list(detobj.stream_trials_metrics([trial.id], "validation")),
    ]:
        assert len(val_metrics) == 10
        for i, actual in enumerate(val_metrics):
            assert actual == client.TrialMetrics(
                trial_id=trial.id,
                trial_run_id=1,
                steps_completed=i + 1,
                end_time=actual.end_time,
                metrics={"x": 0.5**i},
                group="validation",
            )


@pytest.mark.e2e_cpu
def test_model_versions() -> None:
    sess = api_utils.user_session()
    detobj = client.Determined._from_session(sess)
    exp = noop.create_experiment(sess, [noop.Checkpoint()])
    assert exp.wait() == client.ExperimentState.COMPLETED
    ckpt = exp.top_checkpoint()

    suffix = [random.sample("abcdefghijklmnopqrstuvwxyz", 1) for _ in range(10)]
    model_name = f"test-model-{suffix}"
    model = detobj.create_model(model_name)
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

        with pytest.raises(errors.NotFoundException):
            model.get_version(ver.model_version)

        # Test get_version without an arg, when no version exists.
        assert model.get_version() is None

    finally:
        model.delete()


@pytest.mark.e2e_cpu
def test_rp_workspace_mapping() -> None:
    sess = api_utils.user_session()
    workspace_names = ["Workspace A", "Workspace B"]
    overwrite_workspace_names = ["Workspace C", "Workspace D"]
    rp_names = ["default"]  # TODO: not sure how to add more rp
    workspace_ids = []

    for wn in workspace_names + overwrite_workspace_names:
        req = bindings.v1PostWorkspaceRequest(name=wn)
        workspace_ids.append(bindings.post_PostWorkspace(sess, body=req).workspace.id)

    try:
        with pytest.raises(
            errors.APIException,
            match="default resource pool default cannot be bound to any workspace",
        ):
            rp = client.ResourcePool(sess, rp_names[0])
            rp.add_bindings(workspace_names)
    finally:
        for wid in workspace_ids:
            bindings.delete_DeleteWorkspace(session=sess, id=wid)


@pytest.mark.e2e_cpu
def test_create_experiment_w_template(tmp_path: pathlib.Path) -> None:
    # Create a minimal experiment with a simple template
    # Wait until a trial has started to ensure experiment creation has no errors
    # Verify that the content in template is indeed applied
    sess = api_utils.user_session()
    template_name = "test_template"
    template_config_key = "description"
    template_config_value = "test_sdk_template"
    try:
        # create template
        template_config = conf.load_config(conf.fixtures_path("templates/template.yaml"))
        template_config[template_config_key] = template_config_value
        tpl = bindings.v1Template(
            name=template_name,
            config=template_config,
            workspaceId=1,
        )
        tpl_resp = bindings.post_PostTemplate(sess, body=tpl, template_name=tpl.name)
        exp_ref = noop.create_paused_experiment(sess, template=tpl_resp.template.name)
        assert exp_ref.config is not None
        assert exp_ref.config[template_config_key] == template_config_value, exp_ref.config
        exp_ref.kill()

    finally:
        bindings.delete_DeleteTemplate(sess, templateName=template_name)
