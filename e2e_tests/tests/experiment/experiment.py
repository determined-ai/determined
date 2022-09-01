import logging
import os
import re
import subprocess
import sys
import tempfile
import time
from typing import Any, Dict, List, Optional, Sequence

import pytest

from determined.common import api, yaml
from determined.common.api import authentication, bindings, certs
from determined.common.api.bindings import determinedexperimentv1State
from tests import config as conf
from tests.cluster import utils as cluster_utils


def maybe_create_experiment(
    config_file: str, model_def_file: str, create_args: Optional[List[str]] = None
) -> subprocess.CompletedProcess:
    command = [
        "det",
        "-m",
        conf.make_master_url(),
        "experiment",
        "create",
        config_file,
        model_def_file,
    ]

    if create_args is not None:
        command += create_args

    env = os.environ.copy()
    env["DET_DEBUG"] = "true"

    return subprocess.run(
        command, universal_newlines=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, env=env
    )


def create_experiment(
    config_file: str, model_def_file: str, create_args: Optional[List[str]] = None
) -> int:
    completed_process = maybe_create_experiment(config_file, model_def_file, create_args)
    assert completed_process.returncode == 0, "\nstdout:\n{} \nstderr:\n{}".format(
        completed_process.stdout, completed_process.stderr
    )
    m = re.search(r"Created experiment (\d+)\n", str(completed_process.stdout))
    assert m is not None
    return int(m.group(1))


def pause_experiment(experiment_id: int) -> None:
    command = ["det", "-m", conf.make_master_url(), "experiment", "pause", str(experiment_id)]
    subprocess.check_call(command)


def activate_experiment(experiment_id: int) -> None:
    command = ["det", "-m", conf.make_master_url(), "experiment", "activate", str(experiment_id)]
    subprocess.check_call(command)


def cancel_experiment(experiment_id: int) -> None:
    bindings.post_CancelExperiment(determined_test_session(), id=experiment_id)
    wait_for_experiment_state(experiment_id, determinedexperimentv1State.STATE_CANCELED)


def cancel_trial(trial_id: int) -> None:
    bindings.post_KillTrial(determined_test_session(), id=trial_id)
    wait_for_trial_state(trial_id, determinedexperimentv1State.STATE_CANCELED)


def wait_for_experiment_state(
    experiment_id: int,
    target_state: determinedexperimentv1State,
    max_wait_secs: int = conf.DEFAULT_MAX_WAIT_SECS,
    log_every: int = 60,
) -> None:
    for seconds_waited in range(max_wait_secs):
        try:
            state = experiment_state(experiment_id)
        # Ignore network errors while polling for experiment state to avoid a
        # single network flake to cause a test suite failure. If the master is
        # unreachable multiple times, this test will fail after max_wait_secs.
        except api.errors.MasterNotFoundException:
            logging.warning(
                "Network failure ignored when polling for state of "
                "experiment {}".format(experiment_id)
            )
            time.sleep(1)
            continue
        except api.errors.NotFoundException:
            logging.warning(
                "Experiment not yet available to check state: "
                "experiment {}".format(experiment_id)
            )
            time.sleep(0.25)
            continue

        if state == target_state:
            return

        if is_terminal_state(state):
            if state != target_state:
                report_failed_experiment(experiment_id)

            pytest.fail(
                f"Experiment {experiment_id} terminated in {state.value} state, "
                f"expected {target_state.value}"
            )

        if seconds_waited > 0 and seconds_waited % log_every == 0:
            print(
                f"Waited {seconds_waited} seconds for experiment {experiment_id} "
                f"(currently {state.value}) to reach {target_state.value}"
            )

        time.sleep(1)

    else:
        if target_state == determinedexperimentv1State.STATE_COMPLETED:
            cancel_experiment(experiment_id)
        report_failed_experiment(experiment_id)
        pytest.fail(
            "Experiment did not reach target state {} after {} seconds".format(
                target_state.value, max_wait_secs
            )
        )


def wait_for_trial_state(
    trial_id: int,
    target_state: determinedexperimentv1State,
    max_wait_secs: int = conf.DEFAULT_MAX_WAIT_SECS,
    log_every: int = 60,
) -> None:
    for seconds_waited in range(max_wait_secs):
        try:
            state = trial_state(trial_id)
        # Ignore network errors while polling for experiment state to avoid a
        # single network flake to cause a test suite failure. If the master is
        # unreachable multiple times, this test will fail after max_wait_secs.
        except api.errors.MasterNotFoundException:
            logging.warning(
                "Network failure ignored when polling for state of " "trial {}".format(trial_id)
            )
            time.sleep(1)
            continue
        except api.errors.NotFoundException:
            logging.warning("Trial not yet available to check state: " "trial {}".format(trial_id))
            time.sleep(0.25)
            continue

        if state == target_state:
            return

        if is_terminal_state(state):
            if state != target_state:
                report_failed_trial(trial_id, target_state=target_state, state=state)

            pytest.fail(
                f"Trial {trial_id} terminated in {state.value} state, "
                f"expected {target_state.value}"
            )

        if seconds_waited > 0 and seconds_waited % log_every == 0:
            print(
                f"Waited {seconds_waited} seconds for trial {trial_id} "
                f"(currently {state.value}) to reach {target_state.value}"
            )

        time.sleep(1)

    else:
        state = trial_state(trial_id)
        if target_state == determinedexperimentv1State.STATE_COMPLETED:
            cancel_trial(trial_id)
        report_failed_trial(trial_id, target_state=target_state, state=state)
        pytest.fail(
            "Trial did not reach target state {} after {} seconds".format(
                target_state.value, max_wait_secs
            )
        )


def experiment_has_active_workload(experiment_id: int) -> bool:
    certs.cli_cert = certs.default_load(conf.make_master_url())
    authentication.cli_auth = authentication.Authentication(conf.make_master_url(), try_reauth=True)
    r = api.get(conf.make_master_url(), "tasks").json()
    for task in r.values():
        if "Experiment {}".format(experiment_id) in task["name"] and len(task["resources"]) > 0:
            return True

    return False


def wait_for_experiment_active_workload(
    experiment_id: int, max_ticks: int = conf.MAX_TASK_SCHEDULED_SECS
) -> None:
    for _ in range(conf.MAX_TASK_SCHEDULED_SECS):
        if experiment_has_active_workload(experiment_id):
            return

        time.sleep(1)

    pytest.fail(
        f"The only trial cannot be scheduled within {max_ticks} seconds.",
    )


def wait_for_at_least_n_trials(
    experiment_id: int,
    n: int,
    timeout: int = 30,
) -> List["TrialPlusWorkload"]:
    """Wait for enough trials to start, then return the trials found."""
    deadline = time.time() + timeout
    while True:
        trials = experiment_trials(experiment_id)
        if len(trials) >= n:
            return trials
        if time.time() > deadline:
            raise TimeoutError(f"did not see {n} trials running in {timeout}s; trials={trials}")


def wait_for_experiment_workload_progress(
    experiment_id: int, max_ticks: int = conf.MAX_TRIAL_BUILD_SECS
) -> None:
    for _ in range(conf.MAX_TRIAL_BUILD_SECS):
        trials = experiment_trials(experiment_id)
        if len(trials) > 0:
            only_trial = trials[0]
            if len(only_trial.workloads) > 1:
                return
        time.sleep(1)

    pytest.fail(
        f"Trial cannot finish first workload within {max_ticks} seconds.",
    )


def experiment_has_completed_workload(experiment_id: int) -> bool:
    certs.cli_cert = certs.default_load(conf.make_master_url())
    authentication.cli_auth = authentication.Authentication(conf.make_master_url(), try_reauth=True)
    trials = experiment_trials(experiment_id)

    if not any(trials):
        return False

    for t in trials:
        for s in t.workloads:
            if (
                s.training is not None
                and s.training.state == determinedexperimentv1State.STATE_COMPLETED
            ) or (
                s.validation is not None
                and s.validation.state == determinedexperimentv1State.STATE_COMPLETED
            ):
                return True
    return False


def experiment_first_trial(exp_id: int) -> int:
    session = determined_test_session()
    trials = bindings.get_GetExperimentTrials(session, experimentId=exp_id).trials

    assert len(trials) > 0
    trial = trials[0]
    trial_id = trial.id
    return trial_id


def determined_test_session() -> api.Session:
    murl = conf.make_master_url()
    certs.cli_cert = certs.default_load(murl)
    authentication.cli_auth = authentication.Authentication(murl, try_reauth=True)
    return api.Session(murl, "determined", authentication.cli_auth, certs.cli_cert)


def experiment_config_json(experiment_id: int) -> Dict[str, Any]:
    r = bindings.get_GetExperiment(determined_test_session(), experimentId=experiment_id)
    assert r.experiment and r.experiment.config
    return r.experiment.config


def experiment_state(experiment_id: int) -> determinedexperimentv1State:
    r = bindings.get_GetExperiment(determined_test_session(), experimentId=experiment_id)
    return r.experiment.state


def trial_state(trial_id: int) -> determinedexperimentv1State:
    r = bindings.get_GetTrial(determined_test_session(), trialId=trial_id)
    return r.trial.state


class TrialPlusWorkload:
    def __init__(
        self, trial: bindings.trialv1Trial, workloads: Sequence[bindings.v1WorkloadContainer]
    ):
        self.trial = trial
        self.workloads = workloads


def experiment_trials(experiment_id: int) -> List[TrialPlusWorkload]:
    sess = determined_test_session()
    r1 = bindings.get_GetExperimentTrials(sess, experimentId=experiment_id)
    src_trials = r1.trials
    trials = []
    for trial in src_trials:
        r2 = bindings.get_GetTrial(sess, trialId=trial.id)
        r3 = bindings.get_GetTrialWorkloads(sess, trialId=trial.id, limit=1000)
        trials.append(TrialPlusWorkload(r2.trial, r3.workloads))
    return trials


def cancel_single(experiment_id: int, should_have_trial: bool = False) -> None:
    cancel_experiment(experiment_id)

    if should_have_trial:
        trials = experiment_trials(experiment_id)
        assert len(trials) == 1, len(trials)

        trial = trials[0].trial
        assert trial.state == determinedexperimentv1State.STATE_CANCELED


def is_terminal_state(state: determinedexperimentv1State) -> bool:
    return state in (
        determinedexperimentv1State.STATE_CANCELED,
        determinedexperimentv1State.STATE_COMPLETED,
        determinedexperimentv1State.STATE_ERROR,
    )


def trial_metrics(trial_id: int) -> Dict[str, Any]:
    certs.cli_cert = certs.default_load(conf.make_master_url())
    authentication.cli_auth = authentication.Authentication(conf.make_master_url(), try_reauth=True)
    r = api.get(conf.make_master_url(), "trials/{}/metrics".format(trial_id))
    json = r.json()  # type: Dict[str, Any]
    return json


def num_trials(experiment_id: int) -> int:
    return len(experiment_trials(experiment_id))


def num_active_trials(experiment_id: int) -> int:
    return sum(
        1 if t.trial.state == determinedexperimentv1State.STATE_ACTIVE else 0
        for t in experiment_trials(experiment_id)
    )


def num_completed_trials(experiment_id: int) -> int:
    return sum(
        1 if t.trial.state == determinedexperimentv1State.STATE_COMPLETED else 0
        for t in experiment_trials(experiment_id)
    )


def num_error_trials(experiment_id: int) -> int:
    return sum(
        1 if t.trial.state == determinedexperimentv1State.STATE_ERROR else 0
        for t in experiment_trials(experiment_id)
    )


def trial_logs(trial_id: int, follow: bool = False) -> List[str]:
    certs.cli_cert = certs.default_load(conf.make_master_url())
    authentication.cli_auth = authentication.Authentication(conf.make_master_url(), try_reauth=True)
    return [tl["message"] for tl in api.trial_logs(conf.make_master_url(), trial_id, follow=follow)]


def workloads_with_training(
    workloads: Sequence[bindings.v1WorkloadContainer],
) -> List[bindings.v1MetricsWorkload]:
    ret: List[bindings.v1MetricsWorkload] = []
    for w in workloads:
        if w.training:
            ret.append(w.training)
    return ret


def workloads_with_validation(
    workloads: Sequence[bindings.v1WorkloadContainer],
) -> List[bindings.v1MetricsWorkload]:
    ret: List[bindings.v1MetricsWorkload] = []
    for w in workloads:
        if w.validation:
            ret.append(w.validation)
    return ret


def workloads_with_checkpoint(
    workloads: Sequence[bindings.v1WorkloadContainer],
) -> List[bindings.v1CheckpointWorkload]:
    ret: List[bindings.v1CheckpointWorkload] = []
    for w in workloads:
        if w.checkpoint:
            ret.append(w.checkpoint)
    return ret


def check_if_string_present_in_trial_logs(trial_id: int, target_string: str) -> bool:
    logs = trial_logs(trial_id, follow=True)
    for log_line in logs:
        if target_string in log_line:
            return True
    print(f"{target_string} not found in trial {trial_id} logs:\n{''.join(logs)}", file=sys.stderr)
    return False


def assert_patterns_in_trial_logs(trial_id: int, patterns: List[str]) -> None:
    """Match each regex pattern in the list to the logs, one-at-a-time, in order."""
    assert patterns, "must provide at least one pattern"
    patterns_iter = iter(patterns)
    p = re.compile(next(patterns_iter))
    logs = trial_logs(trial_id, follow=True)
    for log_line in logs:
        if p.search(log_line) is None:
            continue
        # Matched a pattern.
        try:
            p = re.compile(next(patterns_iter))
        except StopIteration:
            # All patterns have been matched.
            return
    # Some patterns were not found.
    text = '"\n  "'.join([p.pattern, *patterns_iter])
    raise ValueError(
        f'the following patterns:\n  "{text}"\nwere not found in the trial logs:\n\n{"".join(logs)}'
    )


def assert_equivalent_trials(A: int, B: int, validation_metrics: List[str]) -> None:
    full_trial_metrics1 = trial_metrics(A)
    full_trial_metrics2 = trial_metrics(B)

    assert len(full_trial_metrics1["workloads"]) == len(full_trial_metrics2["workloads"])
    for step1, step2 in zip(full_trial_metrics1["workloads"], full_trial_metrics2["workloads"]):
        metric1 = step1["metrics"]["batch_metrics"]
        metric2 = step2["metrics"]["batch_metrics"]
        for batch1, batch2 in zip(metric1, metric2):
            assert len(batch1) == len(batch2) == 2
            assert batch1["loss"] == pytest.approx(batch2["loss"])

        if step1["validation"] is not None or step2["validation"] is not None:
            assert step1["validation"] is not None
            assert step2["validation"] is not None

            for metric in validation_metrics:
                val1 = step1.get("validation").get("metrics").get("validation_metrics").get(metric)
                val2 = step2.get("validation").get("metrics").get("validation_metrics").get(metric)
                assert val1 == pytest.approx(val2)


def assert_performed_initial_validation(exp_id: int) -> None:
    trials = experiment_trials(exp_id)

    assert len(trials) > 0
    workloads = trials[0].workloads

    assert len(workloads) > 0
    zeroth_step = workloads_with_validation(workloads)[0]

    assert zeroth_step.totalBatches == 0
    assert zeroth_step.state == determinedexperimentv1State.STATE_COMPLETED


def last_workload_matches_last_checkpoint(
    workloads: Sequence[bindings.v1WorkloadContainer],
) -> None:
    assert len(workloads) > 0

    checkpoint_workloads = workloads_with_checkpoint(workloads)
    assert len(checkpoint_workloads) > 0
    last_checkpoint = checkpoint_workloads[-1]
    assert last_checkpoint.state == bindings.determinedcheckpointv1State.STATE_COMPLETED

    last_workload = workloads[-1]
    if last_workload.training or last_workload.validation:
        last_workload_detail = last_workload.training or last_workload.validation
        assert last_workload_detail is not None
        assert last_workload_detail.totalBatches == last_checkpoint.totalBatches
        assert last_workload_detail.state == determinedexperimentv1State.STATE_COMPLETED
    elif last_workload.checkpoint:
        last_checkpoint_detail = last_workload.checkpoint
        assert last_checkpoint_detail is not None
        assert last_checkpoint_detail.totalBatches == last_checkpoint.totalBatches
        assert last_checkpoint_detail.state == bindings.determinedcheckpointv1State.STATE_COMPLETED


def assert_performed_final_checkpoint(exp_id: int) -> None:
    trials = experiment_trials(exp_id)
    assert len(trials) > 0
    last_workload_matches_last_checkpoint(trials[0].workloads)


def run_describe_cli_tests(experiment_id: int) -> None:
    """
    Runs `det experiment describe` CLI command on a finished
    experiment. Will raise an exception if `det experiment describe`
    encounters a traceback failure.
    """
    # "det experiment describe" without metrics.
    with tempfile.TemporaryDirectory() as tmpdir:
        subprocess.check_call(
            [
                "det",
                "-m",
                conf.make_master_url(),
                "experiment",
                "describe",
                str(experiment_id),
                "--outdir",
                tmpdir,
            ]
        )

        assert os.path.exists(os.path.join(tmpdir, "experiments.csv"))
        assert os.path.exists(os.path.join(tmpdir, "workloads.csv"))
        assert os.path.exists(os.path.join(tmpdir, "trials.csv"))

    # "det experiment describe" with metrics.
    with tempfile.TemporaryDirectory() as tmpdir:
        subprocess.check_call(
            [
                "det",
                "-m",
                conf.make_master_url(),
                "experiment",
                "describe",
                str(experiment_id),
                "--metrics",
                "--outdir",
                tmpdir,
            ]
        )

        assert os.path.exists(os.path.join(tmpdir, "experiments.csv"))
        assert os.path.exists(os.path.join(tmpdir, "workloads.csv"))
        assert os.path.exists(os.path.join(tmpdir, "trials.csv"))


def run_list_cli_tests(experiment_id: int) -> None:
    """
    Runs list-related CLI commands on a finished experiment. Will raise an
    exception if the CLI command encounters a traceback failure.
    """

    subprocess.check_call(
        ["det", "-m", conf.make_master_url(), "experiment", "list-trials", str(experiment_id)]
    )

    subprocess.check_call(
        ["det", "-m", conf.make_master_url(), "experiment", "list-checkpoints", str(experiment_id)]
    )
    subprocess.check_call(
        [
            "det",
            "-m",
            conf.make_master_url(),
            "experiment",
            "list-checkpoints",
            "--best",
            str(1),
            str(experiment_id),
        ]
    )


def report_failed_experiment(experiment_id: int) -> None:
    trials = experiment_trials(experiment_id)
    active = sum(1 for t in trials if t.trial.state == determinedexperimentv1State.STATE_ACTIVE)
    paused = sum(1 for t in trials if t.trial.state == determinedexperimentv1State.STATE_PAUSED)
    stopping_completed = sum(
        1 for t in trials if t.trial.state == determinedexperimentv1State.STATE_STOPPING_COMPLETED
    )
    stopping_canceled = sum(
        1 for t in trials if t.trial.state == determinedexperimentv1State.STATE_STOPPING_CANCELED
    )
    stopping_error = sum(
        1 for t in trials if t.trial.state == determinedexperimentv1State.STATE_STOPPING_ERROR
    )
    completed = sum(
        1 for t in trials if t.trial.state == determinedexperimentv1State.STATE_COMPLETED
    )
    canceled = sum(1 for t in trials if t.trial.state == determinedexperimentv1State.STATE_CANCELED)
    errored = sum(1 for t in trials if t.trial.state == determinedexperimentv1State.STATE_ERROR)
    stopping_killed = sum(
        1 for t in trials if t.trial.state == determinedexperimentv1State.STATE_STOPPING_KILLED
    )

    print(
        f"Experiment {experiment_id}: {len(trials)} trials, {completed} completed, "
        f"{active} active, {paused} paused, {stopping_completed} stopping-completed, "
        f"{stopping_canceled} stopping-canceled, {stopping_error} stopping-error, "
        f"{stopping_killed} stopping-killed, {canceled} canceled, {errored} errored",
        file=sys.stderr,
    )

    for trial in trials:
        print_trial_logs(trial.trial.id)


def report_failed_trial(
    trial_id: int, target_state: determinedexperimentv1State, state: determinedexperimentv1State
) -> None:
    print(f"Trial {trial_id} was not {target_state.value} but {state.value}", file=sys.stderr)
    print_trial_logs(trial_id)


def print_trial_logs(trial_id: int) -> None:
    print("******** Start of logs for trial {} ********".format(trial_id), file=sys.stderr)
    print("".join(trial_logs(trial_id)), file=sys.stderr)
    print("******** End of logs for trial {} ********".format(trial_id), file=sys.stderr)


def run_basic_test(
    config_file: str,
    model_def_file: str,
    expected_trials: Optional[int],
    create_args: Optional[List[str]] = None,
    max_wait_secs: int = conf.DEFAULT_MAX_WAIT_SECS,
    expect_workloads: bool = True,
    expect_checkpoints: bool = True,
    priority: int = -1,
) -> int:
    assert os.path.isdir(model_def_file)
    experiment_id = create_experiment(config_file, model_def_file, create_args)
    if priority != -1:
        set_priority(experiment_id=experiment_id, priority=priority)

    wait_for_experiment_state(
        experiment_id,
        determinedexperimentv1State.STATE_COMPLETED,
        max_wait_secs=max_wait_secs,
    )
    assert num_active_trials(experiment_id) == 0

    verify_completed_experiment_metadata(
        experiment_id, expected_trials, expect_workloads, expect_checkpoints
    )

    return experiment_id


def set_priority(experiment_id: int, priority: int) -> None:
    command = [
        "det",
        "-m",
        conf.make_master_url(),
        "experiment",
        "set",
        "priority",
        str(experiment_id),
        str(priority),
    ]

    completed_process = subprocess.run(
        command, universal_newlines=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE
    )

    assert completed_process.returncode == 0, "\nstdout:\n{} \nstderr:\n{}".format(
        completed_process.stdout, completed_process.stderr
    )


def verify_completed_experiment_metadata(
    experiment_id: int,
    num_expected_trials: Optional[int],
    expect_workloads: bool = True,
    expect_checkpoints: bool = True,
) -> None:
    # If `expected_trials` is None, the expected number of trials is
    # non-deterministic.
    if num_expected_trials is not None:
        assert num_trials(experiment_id) == num_expected_trials
        assert num_completed_trials(experiment_id) == num_expected_trials

    # Check that every trial and step is COMPLETED.
    trials = experiment_trials(experiment_id)
    assert len(trials) > 0

    for t in trials:
        trial = t.trial
        if trial.state != determinedexperimentv1State.STATE_COMPLETED:
            report_failed_trial(
                trial.id,
                target_state=determinedexperimentv1State.STATE_COMPLETED,
                state=trial.state,
            )
            pytest.fail(f"Trial {trial.id} was not STATE_COMPLETED but {trial.state.value}")

        if not expect_workloads:
            continue

        if len(t.workloads) == 0:
            print_trial_logs(trial.id)
            raise AssertionError(
                f"trial {trial.id} is in {trial.state.value} state but has 0 steps/workloads"
            )

        # Check that batches appear in increasing order.
        batch_ids = []
        for s in t.workloads:
            if s.training:
                batch_ids.append(s.training.totalBatches)
                assert s.training.state == determinedexperimentv1State.STATE_COMPLETED
            if s.validation:
                batch_ids.append(s.validation.totalBatches)
                assert s.validation.state == determinedexperimentv1State.STATE_COMPLETED
            if s.checkpoint:
                batch_ids.append(s.checkpoint.totalBatches)
                assert s.checkpoint.state in {
                    bindings.determinedcheckpointv1State.STATE_COMPLETED,
                    bindings.determinedcheckpointv1State.STATE_DELETED,
                }
        assert all(x <= y for x, y in zip(batch_ids, batch_ids[1:]))

    # The last step of every trial should be the same batch number as the last checkpoint.
    if expect_checkpoints:
        for t in trials:
            last_workload_matches_last_checkpoint(t.workloads)

    # When the experiment completes, all slots should now be free. This
    # requires terminating the experiment's last container, which might
    # take some time.
    max_secs_to_free_slots = 30
    for _ in range(max_secs_to_free_slots):
        if cluster_utils.num_free_slots() == cluster_utils.num_slots():
            break
        time.sleep(1)
    else:
        raise AssertionError("Slots failed to free after experiment {}".format(experiment_id))

    # Run a series of CLI tests on the finished experiment, to sanity check
    # that basic CLI commands don't raise errors.
    run_describe_cli_tests(experiment_id)
    run_list_cli_tests(experiment_id)


# Use Determined to run an experiment that we expect to fail.
def run_failure_test(config_file: str, model_def_file: str, error_str: Optional[str] = None) -> int:
    experiment_id = create_experiment(config_file, model_def_file)

    wait_for_experiment_state(experiment_id, determinedexperimentv1State.STATE_ERROR)

    # The searcher is configured with a `max_trials` of 8. Since the
    # first step of each trial results in an error, there should be no
    # completed trials.
    #
    # Most of the trials should result in ERROR, but depending on that
    # seems fragile: if we support task preemption in the future, we
    # might start a trial but cancel it before we hit the error in the
    # model definition.

    assert num_active_trials(experiment_id) == 0
    assert num_completed_trials(experiment_id) == 0
    assert num_error_trials(experiment_id) >= 1

    # For each failed trial, check for the expected error in the logs.
    trials = experiment_trials(experiment_id)
    for t in trials:
        trial = t.trial
        if trial.state != determinedexperimentv1State.STATE_ERROR:
            continue

        logs = trial_logs(trial.id)
        if error_str is not None:
            assert any(error_str in line for line in logs)

    return experiment_id


def run_basic_test_with_temp_config(
    config: Dict[Any, Any],
    model_def_path: str,
    expected_trials: Optional[int],
    create_args: Optional[List[str]] = None,
    max_wait_secs: int = conf.DEFAULT_MAX_WAIT_SECS,
) -> int:
    with tempfile.NamedTemporaryFile() as tf:
        with open(tf.name, "w") as f:
            yaml.dump(config, f)
        experiment_id = run_basic_test(
            tf.name,
            model_def_path,
            expected_trials,
            create_args,
            max_wait_secs=max_wait_secs,
        )
    return experiment_id


def run_failure_test_with_temp_config(
    config: Dict[Any, Any],
    model_def_path: str,
    error_str: Optional[str] = None,
) -> int:
    with tempfile.NamedTemporaryFile() as tf:
        with open(tf.name, "w") as f:
            yaml.dump(config, f)
        return run_failure_test(tf.name, model_def_path, error_str=error_str)


def shared_fs_checkpoint_config() -> Dict[str, str]:
    return {
        "type": "shared_fs",
        "host_path": "/tmp",
        "storage_path": "determined-integration-checkpoints",
    }


def s3_checkpoint_config(secrets: Dict[str, str], prefix: Optional[str] = None) -> Dict[str, str]:
    config_dict = {
        "type": "s3",
        "access_key": secrets["INTEGRATIONS_S3_ACCESS_KEY"],
        "secret_key": secrets["INTEGRATIONS_S3_SECRET_KEY"],
        "bucket": secrets["INTEGRATIONS_S3_BUCKET"],
    }
    if prefix is not None:
        config_dict["prefix"] = prefix

    return config_dict


def s3_checkpoint_config_no_creds() -> Dict[str, str]:
    return {"type": "s3", "bucket": "determined-ai-examples"}


def root_user_home_bind_mount() -> Dict[str, str]:
    return {"host_path": "/tmp", "container_path": "/root"}
