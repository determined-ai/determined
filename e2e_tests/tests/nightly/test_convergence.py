from typing import Any, Dict, List

import pytest

from determined.experimental import client
from tests import api_utils
from tests import config as conf
from tests import experiment as exp


def _get_validation_metrics(detobj: client.Determined, trial_id: int) -> List[Dict[str, Any]]:
    return [m.metrics for m in detobj.stream_trials_validation_metrics([trial_id])]


@pytest.mark.nightly
def test_mnist_pytorch_accuracy() -> None:
    sess = api_utils.user_session()
    config = conf.load_config(conf.tutorials_path("mnist_pytorch/const.yaml"))
    experiment_id = exp.run_basic_test_with_temp_config(
        sess, config, conf.tutorials_path("mnist_pytorch"), 1
    )

    trials = exp.experiment_trials(sess, experiment_id)
    detobj = client.Determined._from_session(sess)
    validations = _get_validation_metrics(detobj, trials[0].trial.id)
    validation_accuracies = [v["accuracy"] for v in validations]

    target_accuracy = 0.97
    assert max(validation_accuracies) > target_accuracy, (
        f"mnist_pytorch did not reach minimum target accuracy {target_accuracy}."
        f" full validation accuracy history: {validation_accuracies}"
    )


@pytest.mark.nightly
def test_hf_trainer_api_accuracy() -> None:
    sess = api_utils.user_session()
    test_dir = "hf_image_classification"
    config = conf.load_config(conf.hf_trainer_examples_path(f"{test_dir}/const.yaml"))
    experiment_id = exp.run_basic_test_with_temp_config(
        sess, config, conf.hf_trainer_examples_path(test_dir), 1
    )

    trials = exp.experiment_trials(sess, experiment_id)
    detobj = client.Determined._from_session(sess)
    validations = _get_validation_metrics(detobj, trials[0].trial.id)
    validation_accuracies = [v["eval_accuracy"] for v in validations]

    target_accuracy = 0.82
    assert max(validation_accuracies) > target_accuracy, (
        f"hf_trainer_api did not reach minimum target accuracy {target_accuracy}."
        f" full validation accuracy history: {validation_accuracies}"
    )
