import datetime
import logging
import multiprocessing
import os
import re
import subprocess
import sys
import tempfile
import time
from typing import Any, Dict, List, Optional

import dateutil.parser
import pytest
import requests

from determined import experimental
from determined.common import api, yaml
from determined.common.api import authentication, certs
from tests import config as conf
from tests.cluster import utils as cluster_utils


def maybe_create_native_experiment(context_dir: str, command: List[str]) -> Optional[int]:
    target_env = os.environ.copy()
    target_env["DET_MASTER"] = conf.make_master_url()

    with subprocess.Popen(
        command, stdin=subprocess.PIPE, stdout=subprocess.PIPE, cwd=context_dir, env=target_env
    ) as p:
        assert p.stdout is not None
        for line in p.stdout:
            m = re.search(r"Created experiment (\d+)\n", line.decode())
            if m is not None:
                return int(m.group(1))

    return None


def create_native_experiment(context_dir: str, command: List[str]) -> int:
    experiment_id = maybe_create_native_experiment(context_dir, command)
    if experiment_id is None:
        pytest.fail(f"Failed to create experiment in {context_dir}: {command}")

    return experiment_id


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


def change_experiment_state(experiment_id: int, new_state: str) -> None:
    # TODO(DET-5678): refactor tests to not use cli singleton auth.
    certs.cli_cert = certs.default_load(conf.make_master_url())
    authentication.cli_auth = authentication.Authentication(conf.make_master_url(), try_reauth=True)
    r = api.patch(
        conf.make_master_url(),
        "experiments/{}".format(experiment_id),
        headers={"Content-Type": "application/merge-patch+json"},
        json={"state": new_state},
    )
    assert r.status_code == requests.codes.no_content, r.text


def cancel_experiment(experiment_id: int) -> None:
    change_experiment_state(experiment_id, "STOPPING_CANCELED")
    # We may never observe the STOPPING_CANCELED state.
    wait_for_experiment_state(experiment_id, "CANCELED")


def cancel_experiment_v1(experiment_id: int) -> None:
    certs.cli_cert = certs.default_load(conf.make_master_url())
    authentication.cli_auth = authentication.Authentication(conf.make_master_url(), try_reauth=True)
    r = api.post(conf.make_master_url(), "/api/v1/experiments/{}/cancel".format(experiment_id))
    r.raise_for_status()
    wait_for_experiment_state(experiment_id, "CANCELED")


def wait_for_experiment_state(
    experiment_id: int,
    target_state: str,
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

        if state == target_state:
            return

        if is_terminal_state(state):
            if state != target_state:
                report_failed_experiment(experiment_id)

            pytest.fail(
                f"Experiment {experiment_id} terminated in {state} state, expected {target_state}"
            )

        if seconds_waited > 0 and seconds_waited % log_every == 0:
            print(
                f"Waited {seconds_waited} seconds for experiment {experiment_id} "
                f"(currently {state}) to reach {target_state}"
            )

        time.sleep(1)

    else:
        if target_state == "COMPLETED":
            cancel_experiment(experiment_id)
        report_failed_experiment(experiment_id)
        pytest.fail(
            "Experiment did not reach target state {} after {} seconds".format(
                target_state, max_wait_secs
            )
        )


def experiment_has_active_workload(experiment_id: int) -> bool:
    certs.cli_cert = certs.default_load(conf.make_master_url())
    authentication.cli_auth = authentication.Authentication(conf.make_master_url(), try_reauth=True)
    r = api.get(conf.make_master_url(), "tasks").json()
    for task in r.values():
        if "Experiment {}".format(experiment_id) in task["name"] and len(task["containers"]) > 0:
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


def wait_for_experiment_workload_progress(
    experiment_id: int, max_ticks: int = conf.MAX_TRIAL_BUILD_SECS
) -> None:
    for _ in range(conf.MAX_TRIAL_BUILD_SECS):
        trials = experiment_trials(experiment_id)
        if len(trials) > 0:
            only_trial = trials[0]
            if len(only_trial["steps"]) > 1:
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

    return any(any(s["state"] == "COMPLETED" for s in t["steps"]) for t in trials)


def experiment_json(experiment_id: int) -> Dict[str, Any]:
    certs.cli_cert = certs.default_load(conf.make_master_url())
    authentication.cli_auth = authentication.Authentication(conf.make_master_url(), try_reauth=True)
    r = api.get(conf.make_master_url(), "experiments/{}".format(experiment_id))
    assert r.status_code == requests.codes.ok, r.text
    json = r.json()  # type: Dict[str, Any]
    return json


def experiment_state(experiment_id: int) -> str:
    state = experiment_json(experiment_id)["state"]  # type: str
    return state


def experiment_trials(experiment_id: int) -> List[Dict[str, Any]]:
    trials = experiment_json(experiment_id)["trials"]  # type: List[Dict[str, Any]]
    return trials


def num_experiments() -> int:
    certs.cli_cert = certs.default_load(conf.make_master_url())
    authentication.cli_auth = authentication.Authentication(conf.make_master_url(), try_reauth=True)
    r = api.get(conf.make_master_url(), "experiments")
    assert r.status_code == requests.codes.ok, r.text
    return len(r.json())


def cancel_single(experiment_id: int, should_have_trial: bool = False) -> None:
    cancel_experiment(experiment_id)

    trials = experiment_trials(experiment_id)
    if should_have_trial or len(trials) > 0:
        assert len(trials) == 1

        trial = trials[0]
        assert trial["state"] == "CANCELED"


def cancel_single_v1(experiment_id: int, should_have_trial: bool = False) -> None:
    cancel_experiment_v1(experiment_id)

    trials = experiment_trials(experiment_id)
    if should_have_trial or len(trials) > 0:
        assert len(trials) == 1

        trial = trials[0]
        assert trial["state"] == "CANCELED"


def is_terminal_state(state: str) -> bool:
    return state in ("CANCELED", "COMPLETED", "ERROR")


def trial_metrics(trial_id: int) -> Dict[str, Any]:
    certs.cli_cert = certs.default_load(conf.make_master_url())
    authentication.cli_auth = authentication.Authentication(conf.make_master_url(), try_reauth=True)
    r = api.get(conf.make_master_url(), "trials/{}/metrics".format(trial_id))
    assert r.status_code == requests.codes.ok, r.text
    json = r.json()  # type: Dict[str, Any]
    return json


def get_flat_metrics(trial_id: int, metric: str) -> List:
    full_trial_metrics = trial_metrics(trial_id)
    metrics = [m for step in full_trial_metrics["steps"] for m in step["metrics"]["batch_metrics"]]
    return [v[metric] for v in metrics]


def num_trials(experiment_id: int) -> int:
    return len(experiment_trials(experiment_id))


def num_active_trials(experiment_id: int) -> int:
    return sum(1 if t["state"] == "ACTIVE" else 0 for t in experiment_trials(experiment_id))


def num_completed_trials(experiment_id: int) -> int:
    return sum(1 if t["state"] == "COMPLETED" else 0 for t in experiment_trials(experiment_id))


def num_error_trials(experiment_id: int) -> int:
    return sum(1 if t["state"] == "ERROR" else 0 for t in experiment_trials(experiment_id))


def trial_logs(trial_id: int) -> List[str]:
    certs.cli_cert = certs.default_load(conf.make_master_url())
    authentication.cli_auth = authentication.Authentication(conf.make_master_url(), try_reauth=True)
    return [tl["message"] for tl in api.trial_logs(conf.make_master_url(), trial_id)]


def check_if_string_present_in_trial_logs(trial_id: int, target_string: str) -> bool:
    logs = trial_logs(trial_id)
    for log_line in logs:
        if target_string in log_line:
            return True
    return False


def assert_equivalent_trials(A: int, B: int, validation_metrics: List[str]) -> None:
    full_trial_metrics1 = trial_metrics(A)
    full_trial_metrics2 = trial_metrics(B)

    assert len(full_trial_metrics1["steps"]) == len(full_trial_metrics2["steps"])
    for step1, step2 in zip(full_trial_metrics1["steps"], full_trial_metrics2["steps"]):
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
    steps = trials[0]["steps"]

    assert len(steps) > 0
    zeroth_step = steps[0]

    assert zeroth_step["validation"] is not None
    assert zeroth_step["validation"]["total_batches"] == 0
    assert zeroth_step["validation"]["state"] == "COMPLETED"


def assert_performed_final_checkpoint(exp_id: int) -> None:
    trials = experiment_trials(exp_id)

    assert len(trials) > 0
    steps = trials[0]["steps"]

    assert len(steps) > 0
    last_step = steps[-1]

    assert last_step["checkpoint"] is not None
    assert last_step["checkpoint"]["state"] == "COMPLETED"


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
    active = sum(1 for t in trials if t["state"] == "ACTIVE")
    paused = sum(1 for t in trials if t["state"] == "PAUSED")
    stopping_completed = sum(1 for t in trials if t["state"] == "STOPPING_COMPLETED")
    stopping_canceled = sum(1 for t in trials if t["state"] == "STOPPING_CANCELED")
    stopping_error = sum(1 for t in trials if t["state"] == "STOPPING_ERROR")
    completed = sum(1 for t in trials if t["state"] == "COMPLETED")
    canceled = sum(1 for t in trials if t["state"] == "CANCELED")
    errored = sum(1 for t in trials if t["state"] == "ERROR")
    stopping_killed = sum(1 for t in trials if t["state"] == "STOPPING_KILLED")

    print(
        f"Experiment {experiment_id}: {len(trials)} trials, {completed} completed, "
        f"{active} active, {paused} paused, {stopping_completed} stopping-completed, "
        f"{stopping_canceled} stopping-canceled, {stopping_error} stopping-error, "
        f"{stopping_killed} stopping-killed, {canceled} canceled, {errored} errored",
        file=sys.stderr,
    )

    for trial in trials:
        print_trial_logs(trial["id"])


def report_failed_trial(trial_id: int, state: str) -> None:
    print(f"Trial {trial_id} was not COMPLETED but {state}", file=sys.stderr)
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
) -> int:
    assert os.path.isdir(model_def_file)
    experiment_id = create_experiment(config_file, model_def_file, create_args)
    wait_for_experiment_state(experiment_id, "COMPLETED", max_wait_secs=max_wait_secs)
    assert num_active_trials(experiment_id) == 0

    verify_completed_experiment_metadata(experiment_id, expected_trials)

    return experiment_id


def verify_completed_experiment_metadata(
    experiment_id: int, num_expected_trials: Optional[int]
) -> None:
    # If `expected_trials` is None, the expected number of trials is
    # non-deterministic.
    if num_expected_trials is not None:
        assert num_trials(experiment_id) == num_expected_trials
        assert num_completed_trials(experiment_id) == num_expected_trials

    # Check that every trial and step is COMPLETED.
    trials = experiment_trials(experiment_id)
    assert len(trials) > 0

    for trial in trials:
        if trial["state"] != "COMPLETED":
            report_failed_trial(trial["id"], trial["state"])
            pytest.fail(f"Trial {trial['id']} was not COMPLETED but {trial['state']}")

        assert len(trial["steps"]) > 0

        # Check that batches appear in increasing order.
        batch_ids = [s["total_batches"] for s in trial["steps"]]
        assert all(x <= y for x, y in zip(batch_ids, batch_ids[1:]))

        for step in trial["steps"]:
            assert step["state"] == "COMPLETED"

            if step["validation"]:
                validation = step["validation"]
                assert validation["state"] == "COMPLETED"

            if step["checkpoint"]:
                checkpoint = step["checkpoint"]
                assert checkpoint["state"] in {"COMPLETED", "DELETED"}

    # The last step of every trial should have a checkpoint.
    for trial in trials:
        last_step = trial["steps"][-1]
        assert last_step["checkpoint"]

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
def run_failure_test(
    config_file: str, model_def_file: str, error_str: Optional[str] = None
) -> None:
    experiment_id = create_experiment(config_file, model_def_file)

    wait_for_experiment_state(experiment_id, "ERROR")

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
        if t["state"] != "ERROR":
            continue

        trial_id = t["id"]
        logs = trial_logs(trial_id)
        if error_str is not None:
            assert any(error_str in line for line in logs)


def get_validation_metric_from_last_step(
    experiment_id: int, trial_id: int, validation_metric_name: str
) -> float:
    trial = experiment_trials(experiment_id)[trial_id]
    last_validation = trial["steps"][len(trial["steps"]) - 1]["validation"]
    return last_validation["metrics"]["validation_metrics"][validation_metric_name]  # type: ignore


class ExperimentDurations:
    def __init__(
        self,
        experiment_duration: datetime.timedelta,
        training_duration: datetime.timedelta,
        validation_duration: datetime.timedelta,
        checkpoint_duration: datetime.timedelta,
    ):
        self.experiment_duration = experiment_duration
        self.training_duration = training_duration
        self.validation_duration = validation_duration
        self.checkpoint_duration = checkpoint_duration

    def __str__(self) -> str:
        duration_strs = []
        duration_strs.append(f"experiment duration: {self.experiment_duration}")
        duration_strs.append(f"training duration: {self.training_duration}")
        duration_strs.append(f"validation duration: {self.validation_duration}")
        duration_strs.append(f"checkpoint duration: {self.checkpoint_duration}")
        return "\n".join(duration_strs)


def get_experiment_durations(experiment_id: int, trial_idx: int) -> ExperimentDurations:
    experiment_metadata = experiment_json(experiment_id)
    end_time = dateutil.parser.parse(experiment_metadata["end_time"])
    start_time = dateutil.parser.parse(experiment_metadata["start_time"])
    experiment_duration = end_time - start_time

    training_duration = datetime.timedelta(seconds=0)
    validation_duration = datetime.timedelta(seconds=0)
    checkpoint_duration = datetime.timedelta(seconds=0)
    for step in experiment_metadata["trials"][trial_idx]["steps"]:
        end_time = dateutil.parser.parse(step["end_time"])
        start_time = dateutil.parser.parse(step["start_time"])
        training_duration += end_time - start_time
        if "validation" in step and step["validation"]:
            end_time = dateutil.parser.parse(step["validation"]["end_time"])
            start_time = dateutil.parser.parse(step["validation"]["start_time"])
            validation_duration += end_time - start_time
        if "checkpoint" in step and step["checkpoint"]:
            end_time = dateutil.parser.parse(step["checkpoint"]["end_time"])
            start_time = dateutil.parser.parse(step["checkpoint"]["start_time"])
            checkpoint_duration += end_time - start_time
    return ExperimentDurations(
        experiment_duration, training_duration, validation_duration, checkpoint_duration
    )


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
) -> None:
    with tempfile.NamedTemporaryFile() as tf:
        with open(tf.name, "w") as f:
            yaml.dump(config, f)
        run_failure_test(tf.name, model_def_path, error_str=error_str)


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


def _export_and_load_model(experiment_id: int, master_url: str) -> None:
    experimental.Determined(master_url).get_experiment(experiment_id).top_checkpoint().load()


def export_and_load_model(experiment_id: int) -> None:
    # We run this in a subprocess to avoid module name collisions
    # when performing checkpoint export of different models.
    ctx = multiprocessing.get_context("spawn")
    p = ctx.Process(
        target=_export_and_load_model,
        args=(
            experiment_id,
            conf.make_master_url(),
        ),
    )
    p.start()
    p.join()
    assert p.exitcode == 0, p.exitcode
