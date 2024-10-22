import pytest

from tests import api_utils
from tests import config as conf
from tests import experiment as exp


@pytest.mark.parallel
@pytest.mark.parametrize("aggregation_frequency", [1, 4])
def test_tf_keras_parallel(aggregation_frequency: int) -> None:
    sess = api_utils.user_session()
    config = conf.load_config(conf.cv_examples_path("iris_tf_keras/const.yaml"))
    assert "--epochs" not in config["entrypoint"], "please update test"
    config["entrypoint"] += " --epochs 1"
    config = conf.set_aggregation_frequency(config, aggregation_frequency)
    config = conf.set_tf2_image(config)
    config = conf.set_profiling_enabled(config)

    experiment_id = exp.run_basic_test_with_temp_config(
        sess, config, conf.cv_examples_path("iris_tf_keras"), 1
    )
    trials = exp.experiment_trials(sess, experiment_id)
    assert len(trials) == 1

    # Check on record/batch counts we emitted in logs.
    validation_size = 30
    num_workers = config.get("resources", {}).get("slots_per_trial", 1)
    global_batch_size = config["hyperparameters"]["global_batch_size"]
    scheduling_unit = config.get("scheduling_unit", 100)
    per_slot_batch_size = global_batch_size // num_workers
    exp_val_batches = (validation_size + (per_slot_batch_size - 1)) // per_slot_batch_size
    patterns = [
        # Expect two copies of matching training reports.
        f"trained: {scheduling_unit * global_batch_size} records.*in {scheduling_unit} batches",
        f"trained: {scheduling_unit * global_batch_size} records.*in {scheduling_unit} batches",
        f"validated: {validation_size} records.*in {exp_val_batches} batches",
    ]
    exp.assert_patterns_in_trial_logs(sess, trials[0].trial.id, patterns)
