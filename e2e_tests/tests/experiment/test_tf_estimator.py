from typing import Dict

import pytest

from tests import config as conf
from tests import experiment as exp


@pytest.mark.parallel
@pytest.mark.tensorflow2
@pytest.mark.parametrize("tf2", [False, True])
def test_mnist_estimator_const_parallel(tf2: bool) -> None:
    config = conf.load_config(conf.fixtures_path("mnist_estimator/single-multi-slot.yaml"))
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_max_length(config, {"batches": 200})
    config = conf.set_tf2_image(config) if tf2 else conf.set_tf1_image(config)
    config = conf.set_perform_initial_validation(config, True)

    exp_id = exp.run_basic_test_with_temp_config(config, conf.fixtures_path("mnist_estimator"), 1)
    exp.assert_performed_initial_validation(exp_id)


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
