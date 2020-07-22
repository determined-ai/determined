from typing import Any, Dict, List, Optional

import pytest
from tensorflow.python.training.tracking.tracking import AutoTrackable

from determined.experimental import Determined
from tests import cluster
from tests import config as conf
from tests import experiment as exp

# The loss and gradient update for this model is deterministic and can
# be computed by hand if you are patient enough. See
#   tests/fixtures/estimator_dataset/model.py
# for how these values were computed.
DATASET_EXPERIMENT_EXPECTED_LOSSES = [14, 4536, 50648544, 3364532256768]


@pytest.mark.e2e_gpu  # type: ignore
def test_mnist_estimator_load() -> None:
    config = conf.load_config(conf.fixtures_path("mnist_estimator/single.yaml"))
    config = conf.set_tf1_image(config)
    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.official_examples_path("trial/mnist_estimator"), 1
    )

    trials = exp.experiment_trials(experiment_id)
    model = Determined(conf.make_master_url()).get_trial(trials[0]["id"]).top_checkpoint().load()
    assert isinstance(model, AutoTrackable)


@pytest.mark.parallel  # type: ignore
@pytest.mark.parametrize("native_parallel", [True, False])  # type: ignore
@pytest.mark.parametrize("tf2", [False, True])  # type: ignore
def test_mnist_estimmator_const_parallel(native_parallel: bool, tf2: bool) -> None:
    if tf2 and native_parallel:
        pytest.skip("TF2 native parallel training is not currently supported.")

    config = conf.load_config(conf.fixtures_path("mnist_estimator/single-multi-slot.yaml"))
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_native_parallel(config, native_parallel)
    config = conf.set_max_steps(config, 2)
    config = conf.set_tf2_image(config) if tf2 else conf.set_tf1_image(config)

    exp.run_basic_test_with_temp_config(
        config, conf.official_examples_path("trial/mnist_estimator"), 1
    )


@pytest.mark.parametrize(  # type: ignore
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
        config, conf.official_examples_path("trial/mnist_estimator"), 1
    )

    trials = exp.experiment_trials(experiment_id1)
    assert len(trials) == 1

    first_trial = trials[0]
    first_trial_id = first_trial["id"]

    assert len(first_trial["steps"]) == 1
    first_checkpoint_id = first_trial["steps"][0]["checkpoint"]["id"]

    config_obj = conf.load_config(conf.fixtures_path("mnist_estimator/single.yaml"))

    config_obj["searcher"]["source_trial_id"] = first_trial_id
    config_obj = conf.set_tf2_image(config_obj) if tf2 else conf.set_tf1_image(config_obj)

    experiment_id2 = exp.run_basic_test_with_temp_config(
        config_obj, conf.official_examples_path("trial/mnist_estimator"), 1
    )

    trials = exp.experiment_trials(experiment_id2)
    assert len(trials) == 1
    assert trials[0]["warm_start_checkpoint_id"] == first_checkpoint_id


def run_dataset_experiment(
    searcher_max_steps: int,
    batches_per_step: int,
    secrets: Dict[str, str],
    tf2: bool,
    slots_per_trial: int = 1,
    source_trial_id: Optional[str] = None,
) -> List[Dict[str, Any]]:
    config = conf.load_config(conf.fixtures_path("estimator_dataset/const.yaml"))
    config.setdefault("searcher", {})
    config["searcher"]["max_steps"] = searcher_max_steps
    config["batches_per_step"] = batches_per_step
    config = conf.set_tf2_image(config) if tf2 else conf.set_tf1_image(config)

    if source_trial_id is not None:
        config["searcher"]["source_trial_id"] = source_trial_id

    config.setdefault("resources", {})
    config["resources"]["slots_per_trial"] = slots_per_trial

    if cluster.num_agents() > 1:
        config["checkpoint_storage"] = exp.s3_checkpoint_config(secrets)

    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.fixtures_path("estimator_dataset"), 1
    )
    return exp.experiment_trials(experiment_id)


@pytest.mark.e2e_gpu  # type: ignore
@pytest.mark.parametrize("tf2", [False])  # type: ignore
def test_dataset_restore(secrets: Dict[str, str], tf2: bool) -> None:
    for searcher_max_steps, batches_per_step in [(4, 1), (2, 2), (1, 4)]:
        trials = run_dataset_experiment(searcher_max_steps, batches_per_step, secrets, tf2)
        losses = exp.get_flat_metrics(trials[0]["id"], "loss")
        assert losses == DATASET_EXPERIMENT_EXPECTED_LOSSES

    trials = run_dataset_experiment(1, 1, secrets, tf2)
    next_trials = run_dataset_experiment(3, 1, secrets, tf2, source_trial_id=trials[0]["id"])
    losses = exp.get_flat_metrics(trials[0]["id"], "loss") + exp.get_flat_metrics(
        next_trials[0]["id"], "loss"
    )

    # TODO(DET-834): Separate step ID from from data loader state.
    #
    # To match the behavior of other Trials, when we warm start, reset the
    # data loader state. Thus, we expect warm started trials to behave
    # differently from non-warm started ones.
    #
    # Below are the adjusted losses for the dataset experiment if we start from
    # the beginning of the dataset after warm starting.
    modified_losses = [14, 504, 163296, 1823347456]
    assert modified_losses != DATASET_EXPERIMENT_EXPECTED_LOSSES
    assert losses == modified_losses


@pytest.mark.parametrize(  # type: ignore
    "tf2",
    [
        pytest.param(True, marks=pytest.mark.tensorflow2_cpu),
        pytest.param(False, marks=pytest.mark.tensorflow1_cpu),
    ],
)
def test_mnist_estimator_data_layer_lfs(tf2: bool) -> None:
    run_mnist_estimator_data_layer_test(tf2, "lfs")


@pytest.mark.parallel  # type: ignore
@pytest.mark.parametrize("tf2", [True, False])  # type: ignore
def test_custom_reducer_distributed(secrets: Dict[str, str], tf2: bool) -> None:
    config = conf.load_config(conf.fixtures_path("estimator_dataset/distributed.yaml"))
    # Run with multiple steps to verify we are resetting reducers right.
    config = conf.set_max_steps(config, 2)
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_tf2_image(config) if tf2 else conf.set_tf1_image(config)

    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.fixtures_path("estimator_dataset"), 1
    )

    trial = exp.experiment_trials(experiment_id)[0]
    last_validation = trial["steps"][len(trial["steps"]) - 1]["validation"]
    metrics = last_validation["metrics"]["validation_metrics"]
    label_sum = 2 * sum(range(16))
    assert metrics["label_sum_fn"] == label_sum
    assert metrics["label_sum_cls"] == label_sum


@pytest.mark.e2e_gpu  # type: ignore
@pytest.mark.parametrize("tf2", [True, False])  # type: ignore
@pytest.mark.parametrize("storage_type", ["s3"])  # type: ignore
def test_mnist_estimator_data_layer_s3(tf2: bool, storage_type: str) -> None:
    run_mnist_estimator_data_layer_test(tf2, storage_type)


def run_mnist_estimator_data_layer_test(tf2: bool, storage_type: str) -> None:
    config = conf.load_config(conf.experimental_path("trial/data_layer_mnist_estimator/const.yaml"))
    config = conf.set_max_steps(config, 2)
    config = conf.set_tf2_image(config) if tf2 else conf.set_tf1_image(config)
    if storage_type == "lfs":
        config = conf.set_shared_fs_data_layer(config)
    else:
        config = conf.set_s3_data_layer(config)

    exp.run_basic_test_with_temp_config(
        config, conf.experimental_path("trial/data_layer_mnist_estimator"), 1
    )


@pytest.mark.parallel  # type: ignore
@pytest.mark.parametrize("storage_type", ["lfs", "s3"])  # type: ignore
def test_mnist_estimator_data_layer_parallel(storage_type: str) -> None:
    config = conf.load_config(conf.experimental_path("trial/data_layer_mnist_estimator/const.yaml"))
    config = conf.set_max_steps(config, 2)
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_tf1_image(config)
    if storage_type == "lfs":
        config = conf.set_shared_fs_data_layer(config)
    else:
        config = conf.set_s3_data_layer(config)

    exp.run_basic_test_with_temp_config(
        config, conf.experimental_path("trial/data_layer_mnist_estimator"), 1
    )


@pytest.mark.e2e_gpu  # type: ignore
def test_mnist_estimator_adaptive_with_data_layer() -> None:
    config = conf.load_config(conf.fixtures_path("mnist_estimator/adaptive.yaml"))
    config = conf.set_tf2_image(config)
    config = conf.set_shared_fs_data_layer(config)

    exp.run_basic_test_with_temp_config(
        config, conf.experimental_path("trial/data_layer_mnist_estimator"), None
    )
