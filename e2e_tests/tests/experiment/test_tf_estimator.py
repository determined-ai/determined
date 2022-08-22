from typing import Dict

import pytest
from tensorflow.python.training.tracking.tracking import AutoTrackable

from determined.experimental import Determined
from tests import config as conf
from tests import experiment as exp


@pytest.mark.e2e_gpu
@pytest.mark.tensorflow2
def test_mnist_estimator_load() -> None:
    config = conf.load_config(conf.fixtures_path("mnist_estimator/single.yaml"))
    config = conf.set_tf1_image(config)
    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.cv_examples_path("mnist_estimator"), 1
    )

    trials = exp.experiment_trials(experiment_id)
    model = Determined(conf.make_master_url()).get_trial(trials[0].trial.id).top_checkpoint().load()
    assert isinstance(model, AutoTrackable)


@pytest.mark.parallel
@pytest.mark.tensorflow2
@pytest.mark.parametrize("tf2", [False, True])
def test_mnist_estimator_const_parallel(tf2: bool) -> None:
    config = conf.load_config(conf.fixtures_path("mnist_estimator/single-multi-slot.yaml"))
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_max_length(config, {"batches": 200})
    config = conf.set_tf2_image(config) if tf2 else conf.set_tf1_image(config)
    config = conf.set_perform_initial_validation(config, True)

    exp_id = exp.run_basic_test_with_temp_config(
        config, conf.cv_examples_path("mnist_estimator"), 1
    )
    exp.assert_performed_initial_validation(exp_id)


@pytest.mark.tensorflow2
@pytest.mark.parametrize(
    "tf2",
    [
        pytest.param(True, marks=pytest.mark.tensorflow2_cpu),
        pytest.param(False, marks=pytest.mark.tensorflow1_cpu),
    ],
)
def test_mnist_estimator_warm_start(tf2: bool) -> None:
    config = conf.load_config(conf.fixtures_path("mnist_estimator/single.yaml"))
    config = conf.set_tf2_image(config) if tf2 else conf.set_tf1_image(config)
    experiment_id1 = exp.run_basic_test_with_temp_config(
        config, conf.cv_examples_path("mnist_estimator"), 1
    )

    trials = exp.experiment_trials(experiment_id1)
    assert len(trials) == 1

    first_trial = trials[0]
    first_trial_id = first_trial.trial.id

    assert len(first_trial.workloads) == 3
    checkpoint_workloads = exp.workloads_with_checkpoint(first_trial.workloads)
    first_checkpoint_uuid = checkpoint_workloads[0].uuid

    config_obj = conf.load_config(conf.fixtures_path("mnist_estimator/single.yaml"))

    config_obj["searcher"]["source_trial_id"] = first_trial_id
    config_obj = conf.set_tf2_image(config_obj) if tf2 else conf.set_tf1_image(config_obj)

    experiment_id2 = exp.run_basic_test_with_temp_config(
        config_obj, conf.cv_examples_path("mnist_estimator"), 1
    )

    trials = exp.experiment_trials(experiment_id2)
    assert len(trials) == 1
    assert trials[0].trial.warmStartCheckpointUuid == first_checkpoint_uuid


@pytest.mark.tensorflow2
@pytest.mark.parametrize(
    "tf2",
    [
        pytest.param(True, marks=pytest.mark.tensorflow2_cpu),
        pytest.param(False, marks=pytest.mark.tensorflow1_cpu),
    ],
)
def test_mnist_estimator_data_layer_lfs(tf2: bool) -> None:
    run_mnist_estimator_data_layer_test(tf2, "lfs")


@pytest.mark.parallel
@pytest.mark.tensorflow2
@pytest.mark.parametrize("tf2", [True, False])
def test_custom_reducer_distributed(secrets: Dict[str, str], tf2: bool) -> None:
    config = conf.load_config(conf.fixtures_path("estimator_dataset/distributed.yaml"))
    # Run with multiple steps to verify we are resetting reducers right.
    config = conf.set_max_length(config, {"batches": 2})
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_tf2_image(config) if tf2 else conf.set_tf1_image(config)

    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.fixtures_path("estimator_dataset"), 1
    )

    trial = exp.experiment_trials(experiment_id)[0]
    last_validation = exp.workloads_with_validation(trial.workloads)[-1]
    metrics = last_validation.metrics.avgMetrics
    label_sum = 2 * sum(range(16))
    assert metrics["label_sum_fn"] == label_sum
    assert metrics["label_sum_cls"] == label_sum


@pytest.mark.e2e_gpu
@pytest.mark.tensorflow2
@pytest.mark.parametrize("tf2", [True, False])
@pytest.mark.parametrize("storage_type", ["s3"])
def test_mnist_estimator_data_layer_s3(
    tf2: bool, storage_type: str, secrets: Dict[str, str]
) -> None:
    run_mnist_estimator_data_layer_test(tf2, storage_type)


def run_mnist_estimator_data_layer_test(tf2: bool, storage_type: str) -> None:
    config = conf.load_config(conf.fixtures_path("data_layer_estimator/const.yaml"))
    config = conf.set_max_length(config, {"batches": 200})
    config = conf.set_tf2_image(config) if tf2 else conf.set_tf1_image(config)
    if storage_type == "lfs":
        config = conf.set_shared_fs_data_layer(config)
    else:
        config = conf.set_s3_data_layer(config)

    exp.run_basic_test_with_temp_config(config, conf.fixtures_path("data_layer_estimator"), 1)


@pytest.mark.parallel
@pytest.mark.tensorflow2
@pytest.mark.parametrize("storage_type", ["lfs", "s3"])
def test_mnist_estimator_data_layer_parallel(storage_type: str, secrets: Dict[str, str]) -> None:
    config = conf.load_config(conf.fixtures_path("data_layer_estimator/const.yaml"))
    config = conf.set_max_length(config, {"batches": 200})
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_tf1_image(config)
    if storage_type == "lfs":
        config = conf.set_shared_fs_data_layer(config)
    else:
        config = conf.set_s3_data_layer(config)

    exp.run_basic_test_with_temp_config(config, conf.fixtures_path("data_layer_estimator"), 1)


@pytest.mark.e2e_gpu
@pytest.mark.tensorflow2
def test_mnist_estimator_adaptive_with_data_layer() -> None:
    config = conf.load_config(conf.fixtures_path("mnist_estimator/adaptive.yaml"))
    config = conf.set_tf2_image(config)
    config = conf.set_shared_fs_data_layer(config)

    exp.run_basic_test_with_temp_config(config, conf.fixtures_path("data_layer_estimator"), None)


@pytest.mark.parallel
@pytest.mark.tensorflow2
def test_on_trial_close_callback() -> None:
    config = conf.load_config(conf.fixtures_path("estimator_no_op/const.yaml"))
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_max_length(config, {"batches": 3})

    exp_id = exp.run_basic_test_with_temp_config(config, conf.fixtures_path("estimator_no_op"), 1)

    assert exp.check_if_string_present_in_trial_logs(
        exp.experiment_trials(exp_id)[0].trial.id, "rank 0 has completed on_trial_close"
    )
