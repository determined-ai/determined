import functools
import re
import socket
from typing import Any, Callable, Dict, Iterable, Optional, Union

import pytest

from determined.cli import command
from determined.common import api
from determined.common.api import authentication, bindings, certs
from tests import config as conf
from tests import experiment as exp

Log = Union[bindings.v1TaskLogsResponse, bindings.v1TrialLogsResponse]


LogFields = Union[bindings.v1TaskLogsFieldsResponse, bindings.v1TrialLogsFieldsResponse]


@pytest.mark.e2e_cpu
@pytest.mark.e2e_cpu_elastic
@pytest.mark.e2e_cpu_postgres
@pytest.mark.e2e_cpu_cross_version
@pytest.mark.e2e_gpu
@pytest.mark.timeout(10 * 60)
def test_trial_logs() -> None:
    # TODO: refactor tests to not use cli singleton auth.
    master_url = conf.make_master_url()
    certs.cli_cert = certs.default_load(conf.make_master_url())
    authentication.cli_auth = authentication.Authentication(conf.make_master_url())
    session = api.Session(master_url, "determined", authentication.cli_auth, certs.cli_cert)

    experiment_id = exp.run_basic_test(
        conf.fixtures_path("no_op/single.yaml"), conf.fixtures_path("no_op"), 1
    )
    trial = exp.experiment_trials(experiment_id)[0].trial
    trial_id = trial.id
    task_id = trial.taskId
    assert task_id != ""

    log_regex = re.compile("^.*New trial runner.*$")

    # Trial-specific APIs should work just fine.
    check_logs(
        log_regex,
        functools.partial(api.trial_logs, session, trial_id),
        functools.partial(bindings.get_TrialLogsFields, session, trialId=trial_id),
    )

    # And so should new task log APIs.
    check_logs(
        log_regex,
        functools.partial(api.task_logs, session, task_id),
        functools.partial(bindings.get_TaskLogsFields, session, taskId=task_id),
    )


@pytest.mark.e2e_cpu
@pytest.mark.e2e_cpu_elastic
@pytest.mark.e2e_cpu_cross_version
@pytest.mark.e2e_gpu  # Note, e2e_gpu and not gpu_required hits k8s cpu tests.
@pytest.mark.timeout(5 * 60)
@pytest.mark.parametrize(
    "task_type,task_config,log_regex",
    [
        (command.TaskTypeCommand, {"entrypoint": ["echo", "hello"]}, re.compile("^.*hello.*$")),
        (command.TaskTypeNotebook, {}, re.compile("^.*Jupyter Server .* is running.*$")),
        (command.TaskTypeShell, {}, re.compile("^.*Server listening on.*$")),
        (command.TaskTypeTensorBoard, {}, re.compile("^.*TensorBoard .* at .*$")),
    ],
)
def test_task_logs(task_type: str, task_config: Dict[str, Any], log_regex: Any) -> None:
    master_url = conf.make_master_url()
    certs.cli_cert = certs.default_load(conf.make_master_url())
    authentication.cli_auth = authentication.Authentication(conf.make_master_url())
    session = api.Session(master_url, "determined", authentication.cli_auth, certs.cli_cert)

    rps = bindings.get_GetResourcePools(session)
    assert rps.resourcePools and len(rps.resourcePools) > 0, "missing resource pool"

    if task_type == command.TaskTypeTensorBoard:
        exp_id = exp.run_basic_test(
            conf.fixtures_path("no_op/single.yaml"),
            conf.fixtures_path("no_op"),
            1,
        )
        treq = bindings.v1LaunchTensorboardRequest(config=task_config, experimentIds=[exp_id])
        task_id = bindings.post_LaunchTensorboard(session, body=treq).tensorboard.id
    elif task_type == command.TaskTypeNotebook:
        nreq = bindings.v1LaunchNotebookRequest(config=task_config)
        task_id = bindings.post_LaunchNotebook(session, body=nreq).notebook.id
    elif task_type == command.TaskTypeCommand:
        creq = bindings.v1LaunchCommandRequest(config=task_config)
        task_id = bindings.post_LaunchCommand(session, body=creq).command.id
    elif task_type == command.TaskTypeShell:
        sreq = bindings.v1LaunchShellRequest(config=task_config)
        task_id = bindings.post_LaunchShell(session, body=sreq).shell.id
    else:
        raise ValueError("unknown task type: {task_type}")

    def task_logs(**kwargs: Any) -> Iterable[Log]:
        return api.task_logs(session, task_id, **kwargs)

    def task_log_fields(follow: Optional[bool] = None) -> Iterable[LogFields]:
        return bindings.get_TaskLogsFields(session, taskId=task_id, follow=follow)

    try:
        check_logs(
            log_regex,
            functools.partial(api.task_logs, session, task_id),
            functools.partial(bindings.get_TaskLogsFields, session, taskId=task_id),
        )
    except socket.timeout:
        raise TimeoutError(f"timed out waiting for {task_type} with id {task_id}")

    finally:
        command._kill(master_url, task_type, task_id)


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
        for log in log_fn(follow=True):
            print(log.message)
        pytest.fail("ran out of logs without a match")

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
