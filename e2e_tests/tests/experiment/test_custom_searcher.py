import logging
import pathlib
import subprocess
import tempfile
import time
from typing import Iterator, List, Optional

import pytest
from urllib3 import connectionpool

from determined import searcher
from determined.common import yaml
from determined.common.api import bindings
from determined.experimental import client
from tests import api_utils
from tests import config as conf
from tests import experiment as exp
from tests.fixtures.custom_searcher import searchers

TIMESTAMP = int(time.time())


def check_trial_state(trial: bindings.trialv1Trial, expect: bindings.trialv1State) -> bool:
    """If the trial is in an unexpected state, dump logs and return False."""
    if trial.state == expect:
        return True
    exp.print_trial_logs(trial.id)
    return False


@pytest.fixture
def client_login() -> Iterator[None]:
    client.login(master=conf.make_master_url())
    yield
    client._determined = None


@pytest.mark.e2e_cpu
def test_run_custom_searcher_experiment(tmp_path: pathlib.Path) -> None:
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
    search_method = searchers.SingleSearchMethod(config, 500)
    search_runner = searcher.LocalSearchRunner(search_method, tmp_path)
    experiment_id = search_runner.run(config, model_dir=conf.fixtures_path("no_op"))

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
        search_method = searchers.RandomSearchMethod(
            max_trials, max_concurrent_trials, max_length, test_type="noop"
        )
        search_runner = searcher.LocalSearchRunner(search_method, pathlib.Path(searcher_dir))
        experiment_id = search_runner.run(config, model_dir=conf.fixtures_path("no_op"))

    assert client._determined is not None
    session = client._determined._session
    response = bindings.get_GetExperiment(session, experimentId=experiment_id)
    assert response.experiment.numTrials == 5
    assert search_method.created_trials == 5
    assert search_method.pending_trials == 0
    assert search_method.closed_trials == 5
    assert len(search_runner.state.trials_created) == search_method.created_trials
    assert len(search_runner.state.trials_closed) == search_method.closed_trials


@pytest.mark.e2e_cpu_2a
@pytest.mark.parametrize(
    "config_name,exp_name,exception_points,metric_as_dict",
    [
        ("core_api_model.yaml", f"custom-searcher-random-test-{TIMESTAMP}", [], True),
        (
            "core_api_model.yaml",
            f"custom-searcher-random-test-fail1-{TIMESTAMP}",
            ["initial_operations_start", "progress_middle", "on_trial_closed_shutdown"],
            False,
        ),
        (
            "core_api_model.yaml",
            f"custom-searcher-random-test-fail2-{TIMESTAMP}",
            ["on_validation_completed", "on_trial_closed_end", "on_trial_created_5"],
            False,
        ),
        (
            "core_api_model.yaml",
            f"custom-searcher-random-test-fail3-{TIMESTAMP}",
            ["on_trial_created", "after_save"],
            False,
        ),
        (
            "core_api_model.yaml",
            f"custom-searcher-random-test-fail5-{TIMESTAMP}",
            [
                "on_trial_created",
                "after_save",
                "after_save",
                "on_validation_completed",
                "after_save",
            ],
            False,
        ),
    ],
)
def test_run_random_searcher_exp_core_api(
    config_name: str,
    exp_name: str,
    exception_points: List[str],
    metric_as_dict: bool,
) -> None:
    config = conf.load_config(conf.fixtures_path("custom_searcher/core_api_searcher_random.yaml"))
    config["entrypoint"] += " --exp-name " + exp_name
    config["entrypoint"] += " --config-name " + config_name
    if len(exception_points) > 0:
        config["entrypoint"] += " --exception-points " + " ".join(exception_points)
    if metric_as_dict:
        config["entrypoint"] += " --metric-as-dict"
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
    assert experiment.numTrials == 5

    trials = bindings.get_GetExperimentTrials(session, experimentId=experiment.id).trials

    ok = True
    for trial in trials:
        ok = ok and check_trial_state(trial, bindings.trialv1State.COMPLETED)
    assert ok, "some trials failed"

    for trial in trials:
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
    resubmissions = logs.count("determined.searcher: Resubmitting operations for event.id=")
    assert resubmissions == sum([x == "after_save" for x in exception_points])


@pytest.mark.e2e_cpu_2a
def test_pause_multi_trial_random_searcher_core_api() -> None:
    config = conf.load_config(conf.fixtures_path("custom_searcher/core_api_searcher_random.yaml"))
    exp_name = f"random-pause-{TIMESTAMP}"
    config["entrypoint"] += " --exp-name " + exp_name
    config["entrypoint"] += " --config-name core_api_model.yaml"

    model_def_path = conf.fixtures_path("custom_searcher")

    with tempfile.NamedTemporaryFile() as tf:
        with open(tf.name, "w") as f:
            yaml.dump(config, f)

        searcher_exp_id = exp.create_experiment(tf.name, model_def_path, None)
        exp.wait_for_experiment_state(
            searcher_exp_id,
            bindings.experimentv1State.RUNNING,
        )
    # make sure both experiments have started by checking
    # that multi-trial experiment has at least 1 running trials
    multi_trial_exp_id = exp.wait_for_experiment_by_name_is_active(exp_name, 1)

    # pause multi-trial experiment
    exp.pause_experiment(multi_trial_exp_id)
    exp.wait_for_experiment_state(multi_trial_exp_id, bindings.experimentv1State.PAUSED)

    # activate multi-trial experiment
    exp.activate_experiment(multi_trial_exp_id)

    # wait for searcher to complete
    exp.wait_for_experiment_state(searcher_exp_id, bindings.experimentv1State.COMPLETED)

    # searcher experiment
    searcher_exp = bindings.get_GetExperiment(
        api_utils.determined_test_session(), experimentId=searcher_exp_id
    ).experiment
    assert searcher_exp.state == bindings.experimentv1State.COMPLETED

    # actual experiment
    experiment = bindings.get_GetExperiment(
        api_utils.determined_test_session(), experimentId=multi_trial_exp_id
    ).experiment
    assert experiment.numTrials == 5

    trials = bindings.get_GetExperimentTrials(
        api_utils.determined_test_session(), experimentId=experiment.id
    ).trials

    ok = True
    for trial in trials:
        ok = ok and check_trial_state(trial, bindings.trialv1State.COMPLETED)
    assert ok, "some trials failed"

    for trial in trials:
        assert trial.totalBatchesProcessed == 500


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
                search_method = searchers.RandomSearchMethod(
                    max_trials,
                    max_concurrent_trials,
                    max_length,
                    test_type="noop",
                    exception_points=[exception_point],
                )
                search_runner_mock = FallibleSearchRunner(
                    exception_point, search_method, pathlib.Path(searcher_dir)
                )
                search_runner_mock.run(config, model_dir=conf.fixtures_path("no_op"))
                pytest.fail("Expected an exception")
            except connectionpool.MaxRetryError:
                failures += 1

        assert failures == failures_expected

        search_method = searchers.RandomSearchMethod(
            max_trials, max_concurrent_trials, max_length, test_type="noop"
        )
        search_runner = searcher.LocalSearchRunner(search_method, pathlib.Path(searcher_dir))
        experiment_id = search_runner.run(config, model_dir=conf.fixtures_path("no_op"))

    assert search_runner.state.last_event_id == 41
    assert search_runner.state.experiment_completed is True
    assert client._determined is not None
    session = client._determined._session
    response = bindings.get_GetExperiment(session, experimentId=experiment_id)
    assert response.experiment.numTrials == 5
    assert search_method.created_trials == 5
    assert search_method.pending_trials == 0
    assert search_method.closed_trials == 5
    assert len(search_runner.state.trials_created) == search_method.created_trials
    assert len(search_runner.state.trials_closed) == search_method.closed_trials

    assert search_method.progress(search_runner.state) == pytest.approx(1.0)


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
        ok = ok and check_trial_state(trial, bindings.trialv1State.COMPLETED)
    assert ok, "some trials failed"


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
                search_runner_mock = FallibleSearchRunner(
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


class FallibleSearchRunner(searcher.LocalSearchRunner):
    def __init__(
        self,
        exception_point: str,
        search_method: searcher.SearchMethod,
        searcher_dir: Optional[pathlib.Path] = None,
    ):
        super(FallibleSearchRunner, self).__init__(search_method, searcher_dir)
        self.fail_on_save = False
        if exception_point == "after_save":
            self.fail_on_save = True

    def save_state(self, experiment_id: int, operations: List[searcher.Operation]) -> None:
        super(FallibleSearchRunner, self).save_state(experiment_id, operations)
        if self.fail_on_save:
            logging.info(
                "Raising exception in after saving the state and before posting operations"
            )
            ex = connectionpool.MaxRetryError(
                connectionpool.HTTPConnectionPool(host="dummyhost", port=8080), "http://dummyurl"
            )
            raise ex
