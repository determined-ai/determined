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
