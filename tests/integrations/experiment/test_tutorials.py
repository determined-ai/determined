import pytest

from tests.integrations import config as conf
from tests.integrations.experiment import create_native_experiment, experiment


@pytest.mark.e2e_gpu  # type: ignore
def test_tutorial() -> None:
    exp_id1 = create_native_experiment(
        conf.tutorials_path("native-tf-keras"), ["python", "tf_keras_native.py"]
    )
    experiment.wait_for_experiment_state(
        exp_id1, "COMPLETED", max_wait_secs=conf.DEFAULT_MAX_WAIT_SECS
    )
    exp_id2 = create_native_experiment(
        conf.tutorials_path("native-tf-keras"), ["python", "tf_keras_native_hparam_search.py"]
    )
    experiment.wait_for_experiment_state(
        exp_id2, "COMPLETED", max_wait_secs=conf.DEFAULT_MAX_WAIT_SECS
    )


@pytest.mark.parallel  # type: ignore
def test_tutorial_dtrain() -> None:
    create_native_experiment(
        conf.tutorials_path("native-tf-keras"), ["python", "tf_keras_native_dtrain.py"]
    )
