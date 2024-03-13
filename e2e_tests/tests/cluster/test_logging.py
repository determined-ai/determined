import functools
import re
import sys
import threading
from typing import Any, Callable, Dict, Iterable, Optional, Union

import pytest

from determined.common import api
from determined.common.api import bindings
from tests import api_utils
from tests import config as conf
from tests import experiment as exp

Log = Union[bindings.v1TaskLogsResponse, bindings.v1TrialLogsResponse]


LogFields = Union[bindings.v1TaskLogsFieldsResponse, bindings.v1TrialLogsFieldsResponse]


@pytest.mark.e2e_cpu
@pytest.mark.e2e_cpu_elastic
@pytest.mark.e2e_cpu_postgres
@pytest.mark.e2e_cpu_cross_version
@pytest.mark.e2e_gpu
@pytest.mark.e2e_slurm
@pytest.mark.e2e_pbs
@pytest.mark.timeout(10 * 60)
def test_trial_logs() -> None:
    sess = api_utils.user_session()

    experiment_id = exp.run_basic_test(
        sess, conf.fixtures_path("no_op/single.yaml"), conf.fixtures_path("no_op"), 1
    )
    trial = exp.experiment_trials(sess, experiment_id)[0].trial
    trial_id = trial.id
    task_id = trial.taskId
    assert task_id != ""

    log_regex = re.compile("^.*New trial runner.*$")

    # Trial-specific APIs should work just fine.
    check_logs(
        log_regex,
        functools.partial(api.trial_logs, sess, trial_id),
        functools.partial(bindings.get_TrialLogsFields, sess, trialId=trial_id),
    )

    # And so should new task log APIs.
    check_logs(
        log_regex,
        functools.partial(api.task_logs, sess, task_id),
        functools.partial(bindings.get_TaskLogsFields, sess, taskId=task_id),
    )


@pytest.mark.e2e_cpu
@pytest.mark.e2e_cpu_elastic
@pytest.mark.e2e_cpu_cross_version
@pytest.mark.e2e_gpu  # Note, e2e_gpu and not gpu_required hits k8s cpu tests.
@pytest.mark.e2e_slurm
@pytest.mark.e2e_pbs
@pytest.mark.parametrize(
    "task_type,task_config,log_regex",
    [
        ("command", {"entrypoint": ["echo", "hello"]}, re.compile("^.*hello.*$")),
        ("notebook", {}, re.compile("^.*Jupyter Server .* is running.*$")),
        ("shell", {}, re.compile("^.*Server listening on.*$")),
        ("tensorboard", {}, re.compile("^.*TensorBoard .* at .*$")),
    ],
)
def test_task_logs(task_type: str, task_config: Dict[str, Any], log_regex: Any) -> None:
    sess = api_utils.user_session()

    rps = bindings.get_GetResourcePools(sess)
    assert rps.resourcePools and len(rps.resourcePools) > 0, "missing resource pool"

    if task_type == "tensorboard":
        exp_id = exp.run_basic_test(
            sess,
            conf.fixtures_path("no_op/single.yaml"),
            conf.fixtures_path("no_op"),
            1,
        )
        treq = bindings.v1LaunchTensorboardRequest(config=task_config, experimentIds=[exp_id])
        task_id = bindings.post_LaunchTensorboard(sess, body=treq).tensorboard.id
    elif task_type == "notebook":
        nreq = bindings.v1LaunchNotebookRequest(config=task_config)
        task_id = bindings.post_LaunchNotebook(sess, body=nreq).notebook.id
    elif task_type == "command":
        creq = bindings.v1LaunchCommandRequest(config=task_config)
        task_id = bindings.post_LaunchCommand(sess, body=creq).command.id
    elif task_type == "shell":
        sreq = bindings.v1LaunchShellRequest(config=task_config)
        task_id = bindings.post_LaunchShell(sess, body=sreq).shell.id
    else:
        raise ValueError("unknown task type: {task_type}")

    def task_logs(**kwargs: Any) -> Iterable[Log]:
        return api.task_logs(sess, task_id, **kwargs)

    def task_log_fields(follow: Optional[bool] = None) -> Iterable[LogFields]:
        return bindings.get_TaskLogsFields(sess, taskId=task_id, follow=follow)

    try:
        result: Optional[Exception] = None

        def do_check_logs() -> None:
            nonlocal result
            try:
                check_logs(
                    log_regex,
                    functools.partial(api.task_logs, sess, task_id),
                    functools.partial(bindings.get_TaskLogsFields, sess, taskId=task_id),
                )
            except Exception as e:
                result = e

        thread = threading.Thread(target=do_check_logs, daemon=True)
        thread.start()
        thread.join(timeout=5 * 60)
        if thread.is_alive():
            # The thread did not exit
            raise ValueError("do_check_logs timed out")
        elif isinstance(result, Exception):
            # There was a failure on the thread.
            raise result
    except Exception:
        print("============= test_task_logs_failed, logs from task =============")
        for log in task_logs():
            print(log.log, end="", file=sys.stderr)
        print("============= end of task logs =============")
        raise

    finally:
        if task_type == "tensorboard":
            bindings.post_KillTensorboard(sess, tensorboardId=task_id)
        elif task_type == "notebook":
            bindings.post_KillNotebook(sess, notebookId=task_id)
        elif task_type == "command":
            bindings.post_KillCommand(sess, commandId=task_id)
        elif task_type == "shell":
            bindings.post_KillShell(sess, shellId=task_id)


def check_logs(
    log_regex: Any,
    log_fn: Callable[..., Iterable[Log]],
    log_fields_fn: Callable[..., Iterable[LogFields]],
) -> None:
    # This is also testing that follow terminates. If we timeout here, that's it.
    for log in log_fn(follow=True):
        if log_regex.match(log.message):
            break
    else:
        raise ValueError("ran out of logs without a match")

    # Just make sure these calls 200 and return some logs.
    assert any(log_fn(tail=10)), "tail returned no logs"
    assert any(log_fn(head=10)), "head returned no logs"

    # Task log fields should work, follow or no follow.
    assert any(log_fields_fn(follow=True)), "log fields returned nothing"
    fields_list = list(log_fields_fn())
    assert any(fields_list), "no task log fields were returned"

    # Convert fields to log_fn filters and check all are valid.
    fields = fields_list[0].to_json()
    assert any(fields.values()), "no filter values were returned"

    for k, v in fields.items():
        if not any(v):
            continue

        # Make sure each filter returns some logs (it should or else it shouldn't be a filter).
        assert any(
            log_fn(
                **{
                    to_snake_case(k): v[0],
                },
            )
        ), "good filter returned no logs"

    # Changed -1 to represent no-rank filter.
    assert any(log_fn(rank_ids=[-1])), "-1 rank returns logs with no rank"

    # Check other negative ranks are nonsense.
    assert not any(log_fn(rank_ids=[-2])), "bad filter returned logs"


def to_snake_case(camel_case: str) -> str:
    return re.sub(r"(?<!^)(?=[A-Z])", "_", camel_case).lower()
