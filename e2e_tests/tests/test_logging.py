import re
from typing import Any, Dict

import pytest

from determined.cli import command
from determined.common import api
from determined.common.api import authentication, certs
from tests import config as conf
from tests import experiment as exp


@pytest.mark.e2e_cpu
@pytest.mark.e2e_cpu_elastic
@pytest.mark.e2e_gpu
@pytest.mark.timeout(300)
def test_trial_logs() -> None:
    # TODO: refactor tests to not use cli singleton auth.
    master_url = conf.make_master_url()
    certs.cli_cert = certs.default_load(conf.make_master_url())
    authentication.cli_auth = authentication.Authentication(conf.make_master_url(), try_reauth=True)

    experiment_id = exp.run_basic_test(
        conf.fixtures_path("no_op/single.yaml"), conf.fixtures_path("no_op"), 1
    )
    trial_id = exp.experiment_trials(experiment_id)[0]["id"]
    task_id = exp.experiment_trials(experiment_id)[0]["task_id"]

    log_regex = re.compile("^.*New trial runner.*$")
    # Trial-specific APIs should work just fine.
    check_logs(master_url, trial_id, log_regex, api.trial_logs, api.trial_log_fields)
    # And so should new task log APIs.
    check_logs(master_url, task_id, log_regex, api.task_logs, api.task_log_fields)


@pytest.mark.e2e_cpu
@pytest.mark.e2e_cpu_elastic
@pytest.mark.e2e_gpu
@pytest.mark.timeout(300)
@pytest.mark.parametrize(
    "task_type,task_config,task_extras,log_regex",
    [
        (command.TaskTypeCommand, {"entrypoint": ["echo", "hello"]}, {}, re.compile("^.*hello.*$")),
        (command.TaskTypeNotebook, {}, {}, re.compile("^.*Jupyter Server .* is running.*$")),
        (command.TaskTypeShell, {}, {}, re.compile("^.*Server listening on.*$")),
        (
            command.TaskTypeTensorboard,
            {},
            {"experiment_ids": [1]},
            re.compile("^.*TensorBoard .* at .*$"),
        ),
    ],
)
def test_task_logs(
    task_type: str, task_config: Dict[str, Any], task_extras: Dict[str, Any], log_regex: Any
) -> None:
    # TODO: refactor tests to not use cli singleton auth.
    master_url = conf.make_master_url()
    certs.cli_cert = certs.default_load(conf.make_master_url())
    authentication.cli_auth = authentication.Authentication(conf.make_master_url(), try_reauth=True)

    # Ensure tensorboard tests have an experiment to work with.
    if (
        task_type == command.TaskTypeTensorboard
        and not api.get(master_url, "/api/v1/experiments").json()["experiments"]
    ):
        exp.run_basic_test(conf.fixtures_path("no_op/single.yaml"), conf.fixtures_path("no_op"), 1)

    resp = command.launch_command(
        master_url,
        f"api/v1/{command.RemoteTaskNewAPIs[task_type]}",
        task_config,
        "",
        default_body=task_extras,
    )
    task_id = resp[command.RemoteTaskName[task_type]]["id"]
    try:
        check_logs(master_url, task_id, log_regex, api.task_logs, api.task_log_fields)
    finally:
        command._kill(master_url, task_type, task_id)


def check_logs(
    master_url: str,
    entity_id: Any,
    log_regex: Any,
    log_fn: Any,
    log_fields_fn: Any,
) -> None:
    # This is also testing that follow terminates. If we timeout here, that's it.
    match = False
    for log in log_fn(master_url, entity_id, follow=True):
        if log_regex.match(log["message"]):
            match = True
            break
    assert match, "ran out of logs without a match"

    # Just make sure these calls 200 and return some logs.
    assert any(log_fn(master_url, entity_id, tail=10)), "tail returned no logs"
    assert any(log_fn(master_url, entity_id, head=10)), "head returned no logs"

    # Task log fields should work, follow or no follow.
    assert any(log_fields_fn(master_url, entity_id, follow=True))
    fields_list = list(log_fields_fn(master_url, entity_id))
    assert any(fields_list), "no task log fields were returned"

    # Convert fields to log_fn filters and check all are valid.
    fields = fields_list[0]
    for k, v in fields.items():
        if not any(v):
            continue

        # Make sure each filter returns some logs (it should or else it shouldn't be a filter).
        assert any(
            log_fn(
                master_url,
                entity_id,
                **{
                    to_snake_case(k): v[0],
                },
            )
        ), "good filter returned no logs"

    # Check nonsense is nonsense.
    assert not any(log_fn(master_url, entity_id, rank_ids=[-1])), "bad filter returned logs"


def to_snake_case(camel_case: str) -> str:
    return re.sub(r"(?<!^)(?=[A-Z])", "_", camel_case).lower()
