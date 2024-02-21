import time
from typing import List

import pytest

from determined.common.api import bindings
from tests import api_utils
from tests import config as conf
from tests import detproc
from tests import experiment as exp

TIMESTAMP = int(time.time())


@pytest.mark.e2e_cpu_2a
@pytest.mark.parametrize(
    "config_name,exp_name,exception_points",
    [
        ("core_api_model.yaml", f"custom-searcher-asha-test-{TIMESTAMP}", []),
        (  # test fail on initialization
            # test single resubmit of operations
            # test resumption on fail before saving
            "core_api_model.yaml",
            f"custom-searcher-asha-test-fail1-{TIMESTAMP}",
            [
                "initial_operations_start",
                "after_save",
                "on_validation_completed",
            ],
        ),
        (  # test resubmitting operations multiple times
            # test fail on shutdown
            "core_api_model.yaml",
            f"custom-searcher-asha-test-fail2-{TIMESTAMP}",
            [
                "on_validation_completed",
                "after_save",
                "after_save",
                "after_save",
                "shutdown",
            ],
        ),
    ],
)
def test_run_asha_searcher_exp_core_api(
    config_name: str, exp_name: str, exception_points: List[str]
) -> None:
    sess = api_utils.user_session()
    config = conf.load_config(conf.fixtures_path("custom_searcher/core_api_searcher_asha.yaml"))
    config["entrypoint"] += " --exp-name " + exp_name
    config["entrypoint"] += " --config-name " + config_name
    if len(exception_points) > 0:
        config["entrypoint"] += " --exception-points " + " ".join(exception_points)
    config["max_restarts"] = len(exception_points)

    experiment_id = exp.run_basic_test_with_temp_config(
        sess, config, conf.fixtures_path("custom_searcher"), 1
    )

    # searcher experiment
    searcher_exp = bindings.get_GetExperiment(sess, experimentId=experiment_id).experiment
    assert searcher_exp.state == bindings.experimentv1State.COMPLETED

    # actual experiment
    response = bindings.get_GetExperiments(sess, name=exp_name)
    experiments = response.experiments
    assert len(experiments) == 1

    experiment = experiments[0]
    assert experiment.numTrials == 16

    response_trials = bindings.get_GetExperimentTrials(sess, experimentId=experiment.id).trials

    # 16 trials in rung 1 (#batches = 150)
    assert sum(t.totalBatchesProcessed >= 150 for t in response_trials) == 16
    # at least 4 trials in rung 2 (#batches = 600)
    assert sum(t.totalBatchesProcessed >= 600 for t in response_trials) >= 4
    # at least 1 trial in rung 3 (#batches = 2400)
    assert sum(t.totalBatchesProcessed == 2400 for t in response_trials) >= 1

    for trial in response_trials:
        assert trial.state == bindings.trialv1State.COMPLETED

    # check logs to ensure failures actually happened
    logs = detproc.check_output(sess, ["det", "experiment", "logs", str(experiment_id)])
    failures = logs.count("Max retries exceeded with url: http://dummyurl (Caused by None)")
    assert failures == len(exception_points)

    # check for resubmitting operations
    resubmissions = logs.count("determined.searcher: Resubmitting operations for event.id=")
    assert resubmissions == sum([x == "after_save" for x in exception_points])
