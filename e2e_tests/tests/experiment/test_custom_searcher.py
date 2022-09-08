import logging
import subprocess
import tempfile
from pathlib import Path
from typing import List, Optional

import pytest
from urllib3.connectionpool import HTTPConnectionPool, MaxRetryError

from determined.common.api import bindings
from determined.experimental import client
from determined.searcher.search_method import Operation, SearchMethod
from determined.searcher.search_runner import LocalSearchRunner
from tests import config as conf
from tests import experiment as exp
from tests.fixtures.core_api.searchers import (
    ASHASearchMethod,
    RandomSearchMethod,
    SingleSearchMethod,
)


@pytest.mark.e2e_cpu
def test_run_custom_searcher_experiment(tmp_path: Path) -> None:
    # example searcher script
    config = conf.load_config(conf.fixtures_path("no_op/single.yaml"))
    config["searcher"] = {
        "name": "custom",
        "metric": "validation_error",
        "smaller_is_better": True,
        "unit": "batches",
    }
    config["name"] = "single"
    config["description"] = "custom searcher"
    search_method = SingleSearchMethod(config, 500)
    search_runner = LocalSearchRunner(search_method, tmp_path)
    experiment_id = search_runner.run(config, context_dir=conf.fixtures_path("no_op"))

    assert client._determined is not None
    session = client._determined._session
    response = bindings.get_GetExperiment(session, experimentId=experiment_id)
    assert response.experiment.numTrials == 1


@pytest.mark.e2e_cpu_2a
def test_run_random_searcher_exp() -> None:
    config = conf.load_config(conf.fixtures_path("no_op/single.yaml"))
    config["searcher"] = {
        "name": "custom",
        "metric": "validation_error",
        "smaller_is_better": True,
        "unit": "batches",
    }
    config["name"] = "random"
    config["description"] = "custom searcher"

    max_trials = 5
    max_concurrent_trials = 2
    max_length = 500

    with tempfile.TemporaryDirectory() as searcher_dir:
        search_method = RandomSearchMethod(
            max_trials, max_concurrent_trials, max_length, test_type="noop"
        )
        search_runner = LocalSearchRunner(search_method, Path(searcher_dir))
        experiment_id = search_runner.run(config, context_dir=conf.fixtures_path("no_op"))

    assert client._determined is not None
    session = client._determined._session
    response = bindings.get_GetExperiment(session, experimentId=experiment_id)
    assert response.experiment.numTrials == 5
    assert search_method.created_trials == 5
    assert search_method.pending_trials == 0
    assert search_method.closed_trials == 5
    assert len(search_method.searcher_state.trials_created) == search_method.created_trials
    assert len(search_method.searcher_state.trials_closed) == search_method.closed_trials


@pytest.mark.e2e_cpu_2a
@pytest.mark.parametrize(
    "exp_name,exception_points",
    [
        ("custom-searcher-random-test", []),
        (
            "custom-searcher-random-test-fail1",
            ["initial_operations_start", "progress_middle", "on_trial_closed_shutdown"],
        ),
        (
            "custom-searcher-random-test-fail2",
            ["on_validation_completed", "on_trial_closed_end", "on_trial_created_5"],
        ),
        (
            "custom-searcher-random-test-fail3",
            ["on_trial_created", "after_save"],
        ),
        (
            "custom-searcher-random-test-fail5",
            [
                "on_trial_created",
                "after_save",
                "after_save",
                "on_validation_completed",
                "after_save",
            ],
        ),
    ],
)
def test_run_random_searcher_exp_core_api(exp_name: str, exception_points: List[str]) -> None:
    config = conf.load_config(conf.fixtures_path("core_api/core_api_searcher_random.yaml"))
    config["entrypoint"] += " --exp-name " + exp_name
    if len(exception_points) > 0:
        config["entrypoint"] += " --exception-points " + " ".join(exception_points)
    config["max_restarts"] = len(exception_points)

    experiment_id = exp.run_basic_test_with_temp_config(config, conf.fixtures_path("core_api"), 1)

    session = exp.determined_test_session()

    # searcher experiment
    searcher_exp = bindings.get_GetExperiment(session, experimentId=experiment_id).experiment
    assert searcher_exp.state == bindings.determinedexperimentv1State.STATE_COMPLETED

    # actual experiment
    response = bindings.get_GetExperiments(session, name=exp_name)
    experiments = response.experiments
    assert len(experiments) == 1

    experiment = experiments[0]
    assert experiment.numTrials == 5

    trials = bindings.get_GetExperimentTrials(session, experimentId=experiment.id).trials
    for trial in trials:
        assert trial.state == bindings.determinedexperimentv1State.STATE_COMPLETED
        assert trial.totalBatchesProcessed == 500

    # check logs to ensure failures actually happened
    logs = str(
        subprocess.check_output(
            ["det", "-m", conf.make_master_url(), "experiment", "logs", str(experiment_id)]
        )
    )
    failures = logs.count("Max retries exceeded with url: http://dummyurl (Caused by None)")
    assert failures == len(exception_points)

    # check for resubmitting operations
    resubmissions = logs.count("root: Resubmitting operations for event.id=")
    assert resubmissions == sum([x == "after_save" for x in exception_points])


@pytest.mark.e2e_cpu_2a
@pytest.mark.parametrize(
    "exceptions",
    [
        ["initial_operations_start", "progress_middle", "on_trial_closed_shutdown"],
        ["on_validation_completed", "on_trial_closed_end", "on_trial_created_5"],
        ["on_trial_created", "save_method_state", "after_save"],
        [
            "on_trial_created",
            "save_method_state",
            "load_method_state",
            "after_save",
            "after_save",
            "on_validation_completed",
            "after_save",
            "save_method_state",
        ],
    ],
)
def test_resume_random_searcher_exp(exceptions: List[str]) -> None:
    config = conf.load_config(conf.fixtures_path("no_op/single.yaml"))
    config["searcher"] = {
        "name": "custom",
        "metric": "validation_error",
        "smaller_is_better": True,
        "unit": "batches",
    }
    config["description"] = ";".join(exceptions) if exceptions else "custom searcher"

    max_trials = 5
    max_concurrent_trials = 2
    max_length = 500
    failures_expected = len(exceptions)
    logging.info(f"expected_failures={failures_expected}")

    # do not use pytest tmp_path to experience LocalSearchRunner in the wild
    with tempfile.TemporaryDirectory() as searcher_dir:
        failures = 0
        while failures < failures_expected:
            try:
                exception_point = exceptions.pop(0)
                # re-create RandomSearchMethod and LocalSearchRunner after every fail
                # to simulate python process crash
                search_method = RandomSearchMethod(
                    max_trials,
                    max_concurrent_trials,
                    max_length,
                    test_type="noop",
                    exception_points=[exception_point],
                )
                search_runner_mock = FallibleSearchRunner(
                    exception_point, search_method, Path(searcher_dir)
                )
                search_runner_mock.run(config, context_dir=conf.fixtures_path("no_op"))
                pytest.fail("Expected an exception")
            except MaxRetryError:
                failures += 1

        assert failures == failures_expected

        search_method = RandomSearchMethod(
            max_trials, max_concurrent_trials, max_length, test_type="noop"
        )
        search_runner = LocalSearchRunner(search_method, Path(searcher_dir))
        experiment_id = search_runner.run(config, context_dir=conf.fixtures_path("no_op"))

    assert search_method.searcher_state.last_event_id == 41
    assert search_method.searcher_state.experiment_completed is True
    assert client._determined is not None
    session = client._determined._session
    response = bindings.get_GetExperiment(session, experimentId=experiment_id)
    assert response.experiment.numTrials == 5
    assert search_method.created_trials == 5
    assert search_method.pending_trials == 0
    assert search_method.closed_trials == 5
    assert len(search_method.searcher_state.trials_created) == search_method.created_trials
    assert len(search_method.searcher_state.trials_closed) == search_method.closed_trials

    assert search_method.progress() == pytest.approx(1.0)


@pytest.mark.e2e_cpu
def test_run_asha_batches_exp(tmp_path: Path) -> None:
    config = conf.load_config(conf.fixtures_path("no_op/adaptive.yaml"))
    config["searcher"] = {
        "name": "custom",
        "metric": "validation_error",
        "smaller_is_better": True,
        "unit": "batches",
    }
    config["name"] = "asha"
    config["description"] = "custom searcher"

    max_length = 3000
    max_trials = 16
    num_rungs = 3
    divisor = 4

    search_method = ASHASearchMethod(max_length, max_trials, num_rungs, divisor, test_type="noop")
    search_runner = LocalSearchRunner(search_method, tmp_path)
    experiment_id = search_runner.run(config, context_dir=conf.fixtures_path("no_op"))

    assert client._determined is not None
    session = client._determined._session
    response = bindings.get_GetExperiment(session, experimentId=experiment_id)

    assert response.experiment.numTrials == 16
    assert search_method.asha_search_state.pending_trials == 0
    assert search_method.asha_search_state.completed_trials == 16
    assert len(search_method.searcher_state.trials_closed) == len(
        search_method.asha_search_state.closed_trials
    )

    response_trials = bindings.get_GetExperimentTrials(session, experimentId=experiment_id).trials

    # 16 trials in rung 1 (#batches = 187)
    assert sum(t.totalBatchesProcessed >= 187 for t in response_trials) == 16
    # at least 4 trials in rung 2 (#batches = 750)
    assert sum(t.totalBatchesProcessed >= 750 for t in response_trials) >= 4
    # at least 1 trial in rung 3 (#batches = 3000)
    assert sum(t.totalBatchesProcessed == 3000 for t in response_trials) >= 1

    for trial in response_trials:
        assert trial.state == bindings.determinedexperimentv1State.STATE_COMPLETED


@pytest.mark.e2e_cpu_2a
@pytest.mark.parametrize(
    "exp_name,exception_points",
    [
        ("custom-searcher-asha-test", []),
        (
            "custom-searcher-asha-test-fail1",
            [
                "initial_operations_start",  # fail before sending initial operations
                "after_save",  # fail on save - should not send initial operations again
                "on_validation_completed",
                "after_save",
                "_get_close_rungs_ops",
            ],
        ),
        (
            "custom-searcher-asha-test-fail2",
            [
                "on_validation_completed",
                "after_save",
                "after_save",
                "after_save",
                "on_validation_completed",
                "shutdown",
            ],
        ),
    ],
)
def test_run_asha_searcher_exp_core_api(exp_name: str, exception_points: List[str]) -> None:
    config = conf.load_config(conf.fixtures_path("core_api/core_api_searcher_asha.yaml"))
    config["entrypoint"] += " --exp-name " + exp_name
    if len(exception_points) > 0:
        config["entrypoint"] += " --exception-points " + " ".join(exception_points)
    config["max_restarts"] = len(exception_points)

    experiment_id = exp.run_basic_test_with_temp_config(config, conf.fixtures_path("core_api"), 1)
    session = exp.determined_test_session()

    # searcher experiment
    searcher_exp = bindings.get_GetExperiment(session, experimentId=experiment_id).experiment
    assert searcher_exp.state == bindings.determinedexperimentv1State.STATE_COMPLETED

    # actual experiment
    response = bindings.get_GetExperiments(session, name=exp_name)
    experiments = response.experiments
    assert len(experiments) == 1

    experiment = experiments[0]
    assert experiment.numTrials == 16

    response_trials = bindings.get_GetExperimentTrials(session, experimentId=experiment.id).trials

    # 16 trials in rung 1 (#batches = 187)
    assert sum(t.totalBatchesProcessed >= 187 for t in response_trials) == 16
    # at least 4 trials in rung 2 (#batches = 750)
    assert sum(t.totalBatchesProcessed >= 750 for t in response_trials) >= 4
    # at least 1 trial in rung 3 (#batches = 3000)
    assert sum(t.totalBatchesProcessed == 3000 for t in response_trials) >= 1

    for trial in response_trials:
        assert trial.state == bindings.determinedexperimentv1State.STATE_COMPLETED

    # check logs to ensure failures actually happened
    logs = str(
        subprocess.check_output(
            ["det", "-m", conf.make_master_url(), "experiment", "logs", str(experiment_id)]
        )
    )
    failures = logs.count("Max retries exceeded with url: http://dummyurl (Caused by None)")
    assert failures == len(exception_points)

    # check for resubmitting operations
    resubmissions = logs.count("root: Resubmitting operations for event.id=")
    assert resubmissions == sum([x == "after_save" for x in exception_points])


@pytest.mark.e2e_cpu
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
def test_resume_asha_batches_exp(exceptions: List[str]) -> None:
    config = conf.load_config(conf.fixtures_path("no_op/adaptive.yaml"))
    config["searcher"] = {
        "name": "custom",
        "metric": "validation_error",
        "smaller_is_better": True,
        "unit": "batches",
    }
    config["name"] = "asha"
    config["description"] = ";".join(exceptions) if exceptions else "custom searcher"

    max_length = 3000
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
                search_method = ASHASearchMethod(
                    max_length,
                    max_trials,
                    num_rungs,
                    divisor,
                    test_type="noop",
                    exception_points=[exception_point],
                )
                search_runner_mock = FallibleSearchRunner(
                    exception_point, search_method, Path(searcher_dir)
                )
                search_runner_mock.run(config, context_dir=conf.fixtures_path("no_op"))
                pytest.fail("Expected an exception")
            except MaxRetryError:
                failures += 1

        assert failures == failures_expected

        search_method = ASHASearchMethod(
            max_length, max_trials, num_rungs, divisor, test_type="noop"
        )
        search_runner = LocalSearchRunner(search_method, Path(searcher_dir))
        experiment_id = search_runner.run(config, context_dir=conf.fixtures_path("no_op"))

    assert search_method.searcher_state.experiment_completed is True
    assert client._determined is not None
    session = client._determined._session
    response = bindings.get_GetExperiment(session, experimentId=experiment_id)

    assert response.experiment.numTrials == 16
    # asha search method state
    assert search_method.asha_search_state.pending_trials == 0
    assert search_method.asha_search_state.completed_trials == 16
    # searcher state
    assert len(search_method.searcher_state.trials_created) == 16
    assert len(search_method.searcher_state.trials_closed) == 16

    assert len(search_method.searcher_state.trials_closed) == len(
        search_method.asha_search_state.closed_trials
    )

    response_trials = bindings.get_GetExperimentTrials(session, experimentId=experiment_id).trials

    # 16 trials in rung 1 (#batches = 187)
    assert sum(t.totalBatchesProcessed >= 187 for t in response_trials) == 16
    # at least 4 trials in rung 2 (#batches = 750)
    assert sum(t.totalBatchesProcessed >= 750 for t in response_trials) >= 4
    # at least 1 trial in rung 3 (#batches = 3000)
    assert sum(t.totalBatchesProcessed == 3000 for t in response_trials) >= 1

    for trial in response_trials:
        assert trial.state == bindings.determinedexperimentv1State.STATE_COMPLETED

    assert search_method.progress() == pytest.approx(1.0)


class FallibleSearchRunner(LocalSearchRunner):
    def __init__(
        self,
        exception_point: str,
        search_method: SearchMethod,
        searcher_dir: Optional[Path] = None,
    ):
        super(FallibleSearchRunner, self).__init__(search_method, searcher_dir)
        self.fail_on_save = False
        if exception_point == "after_save":
            self.fail_on_save = True

    def save_state(self, experiment_id: int, operations: List[Operation]) -> None:
        super(FallibleSearchRunner, self).save_state(experiment_id, operations)
        if self.fail_on_save:
            logging.info(
                "Raising exception in after saving the state and before posting operations"
            )
            ex = MaxRetryError(HTTPConnectionPool(host="dummyhost", port=8080), "http://dummyurl")
            raise ex
