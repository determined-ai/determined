import logging
import pathlib
import subprocess
import tempfile
import time
from typing import List

import pytest
from urllib3 import connectionpool

from determined import searcher
from determined.common.api import bindings
from determined.experimental import client
from tests import api_utils
from tests import config as conf
from tests import experiment as exp
from tests.experiment import test_custom_searcher
from tests.fixtures.custom_searcher import searchers

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
    config = conf.load_config(conf.fixtures_path("custom_searcher/core_api_searcher_asha.yaml"))
    config["entrypoint"] += " --exp-name " + exp_name
    config["entrypoint"] += " --config-name " + config_name
    if len(exception_points) > 0:
        config["entrypoint"] += " --exception-points " + " ".join(exception_points)
    config["max_restarts"] = len(exception_points)

    experiment_id = exp.run_basic_test_with_temp_config(
        config, conf.fixtures_path("custom_searcher"), 1
    )
    session = api_utils.determined_test_session()

    # searcher experiment
    searcher_exp = bindings.get_GetExperiment(session, experimentId=experiment_id).experiment
    assert searcher_exp.state == bindings.experimentv1State.COMPLETED

    # actual experiment
    response = bindings.get_GetExperiments(session, name=exp_name)
    experiments = response.experiments
    assert len(experiments) == 1

    experiment = experiments[0]
    assert experiment.numTrials == 16

    response_trials = bindings.get_GetExperimentTrials(session, experimentId=experiment.id).trials

    # 16 trials in rung 1 (#batches = 150)
    assert sum(t.totalBatchesProcessed >= 150 for t in response_trials) == 16
    # at least 4 trials in rung 2 (#batches = 600)
    assert sum(t.totalBatchesProcessed >= 600 for t in response_trials) >= 4
    # at least 1 trial in rung 3 (#batches = 2400)
    assert sum(t.totalBatchesProcessed == 2400 for t in response_trials) >= 1

    for trial in response_trials:
        assert trial.state == bindings.trialv1State.COMPLETED

    # check logs to ensure failures actually happened
    logs = str(
        subprocess.check_output(
            ["det", "-m", conf.make_master_url(), "experiment", "logs", str(experiment_id)]
        )
    )
    failures = logs.count("Max retries exceeded with url: http://dummyurl (Caused by None)")
    assert failures == len(exception_points)

    # check for resubmitting operations
    resubmissions = logs.count("determined.searcher: Resubmitting operations for event.id=")
    assert resubmissions == sum([x == "after_save" for x in exception_points])


@pytest.mark.nightly
def test_run_asha_batches_exp(tmp_path: pathlib.Path, client_login: None) -> None:
    config = conf.load_config(conf.fixtures_path("no_op/adaptive.yaml"))
    config["searcher"] = {
        "name": "custom",
        "metric": "validation_error",
        "smaller_is_better": True,
        "unit": "batches",
    }
    config["name"] = "asha"
    config["description"] = "custom searcher"

    max_length = 2000
    max_trials = 16
    num_rungs = 3
    divisor = 4

    search_method = searchers.ASHASearchMethod(
        max_length, max_trials, num_rungs, divisor, test_type="noop"
    )
    search_runner = searcher.LocalSearchRunner(search_method, tmp_path)
    experiment_id = search_runner.run(config, model_dir=conf.fixtures_path("no_op"))

    assert client._determined is not None
    session = client._determined._session
    response = bindings.get_GetExperiment(session, experimentId=experiment_id)

    assert response.experiment.numTrials == 16
    assert search_method.asha_search_state.pending_trials == 0
    assert search_method.asha_search_state.completed_trials == 16
    assert len(search_runner.state.trials_closed) == len(
        search_method.asha_search_state.closed_trials
    )

    response_trials = bindings.get_GetExperimentTrials(session, experimentId=experiment_id).trials

    # 16 trials in rung 1 (#batches = 125)
    assert sum(t.totalBatchesProcessed >= 125 for t in response_trials) == 16
    # at least 4 trials in rung 2 (#batches = 500)
    assert sum(t.totalBatchesProcessed >= 500 for t in response_trials) >= 4
    # at least 1 trial in rung 3 (#batches = 2000)
    assert sum(t.totalBatchesProcessed == 2000 for t in response_trials) >= 1

    ok = True
    for trial in response_trials:
        ok = ok and test_custom_searcher.check_trial_state(trial, bindings.trialv1State.COMPLETED)
    assert ok, "some trials failed"


@pytest.mark.nightly
@pytest.mark.parametrize(
    "exceptions",
    [
        [
            "initial_operations_start",  # fail before sending initial operations
            "after_save",  # fail on save - should not send initial operations again
            "save_method_state",
            "save_method_state",
            "after_save",
            "on_trial_created",
            "_get_close_rungs_ops",
        ],
        [  # searcher state and search method state are restored to last saved state
            "on_validation_completed",
            "on_validation_completed",
            "save_method_state",
            "save_method_state",
            "after_save",
            "after_save",
            "load_method_state",
            "on_validation_completed",
            "shutdown",
        ],
    ],
)
def test_resume_asha_batches_exp(exceptions: List[str], client_login: None) -> None:
    config = conf.load_config(conf.fixtures_path("no_op/adaptive.yaml"))
    config["searcher"] = {
        "name": "custom",
        "metric": "validation_error",
        "smaller_is_better": True,
        "unit": "batches",
    }
    config["name"] = "asha"
    config["description"] = ";".join(exceptions) if exceptions else "custom searcher"

    max_length = 2000
    max_trials = 16
    num_rungs = 3
    divisor = 4
    failures_expected = len(exceptions)

    with tempfile.TemporaryDirectory() as searcher_dir:
        logging.info(f"searcher_dir type = {type(searcher_dir)}")
        failures = 0
        while failures < failures_expected:
            try:
                exception_point = exceptions.pop(0)
                search_method = searchers.ASHASearchMethod(
                    max_length,
                    max_trials,
                    num_rungs,
                    divisor,
                    test_type="noop",
                    exception_points=[exception_point],
                )
                search_runner_mock = test_custom_searcher.FallibleSearchRunner(
                    exception_point, search_method, pathlib.Path(searcher_dir)
                )
                search_runner_mock.run(config, model_dir=conf.fixtures_path("no_op"))
                pytest.fail("Expected an exception")
            except connectionpool.MaxRetryError:
                failures += 1

        assert failures == failures_expected

        search_method = searchers.ASHASearchMethod(
            max_length, max_trials, num_rungs, divisor, test_type="noop"
        )
        search_runner = searcher.LocalSearchRunner(search_method, pathlib.Path(searcher_dir))
        experiment_id = search_runner.run(config, model_dir=conf.fixtures_path("no_op"))

    assert search_runner.state.experiment_completed is True
    assert client._determined is not None
    session = client._determined._session
    response = bindings.get_GetExperiment(session, experimentId=experiment_id)

    assert response.experiment.numTrials == 16
    # asha search method state
    assert search_method.asha_search_state.pending_trials == 0
    assert search_method.asha_search_state.completed_trials == 16
    # searcher state
    assert len(search_runner.state.trials_created) == 16
    assert len(search_runner.state.trials_closed) == 16

    assert len(search_runner.state.trials_closed) == len(
        search_method.asha_search_state.closed_trials
    )

    response_trials = bindings.get_GetExperimentTrials(session, experimentId=experiment_id).trials

    # 16 trials in rung 1 (#batches = 125)
    assert sum(t.totalBatchesProcessed >= 125 for t in response_trials) == 16
    # at least 4 trials in rung 2 (#batches = 500)
    assert sum(t.totalBatchesProcessed >= 500 for t in response_trials) >= 4
    # at least 1 trial in rung 3 (#batches = 2000)
    assert sum(t.totalBatchesProcessed == 2000 for t in response_trials) >= 1

    for trial in response_trials:
        assert trial.state == bindings.trialv1State.COMPLETED

    assert search_method.progress(search_runner.state) == pytest.approx(1.0)
