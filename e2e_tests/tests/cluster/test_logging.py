import re
from typing import Any, Dict

import pytest

from determined.cli import command
from determined.common import api
from determined.common.api import authentication, bindings, certs
from tests import config as conf
from tests import experiment as exp


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
    authentication.cli_auth = authentication.Authentication(conf.make_master_url(), try_reauth=True)

    experiment_id = exp.run_basic_test(
        conf.fixtures_path("no_op/single.yaml"), conf.fixtures_path("no_op"), 1
    )
    trial = exp.experiment_trials(experiment_id)[0].trial
    trial_id = trial.id
    task_id = trial.taskId
    assert task_id != ""

    log_regex = re.compile("^.*New trial runner.*$")
    # Trial-specific APIs should work just fine.
    check_logs(master_url, trial_id, log_regex, api.trial_logs, api.trial_log_fields)
    # And so should new task log APIs.
    check_logs(master_url, task_id, log_regex, api.task_logs, api.task_log_fields)


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
    # TODO: refactor tests to not use cli singleton auth.
    master_url = conf.make_master_url()
    certs.cli_cert = certs.default_load(conf.make_master_url())
    authentication.cli_auth = authentication.Authentication(conf.make_master_url(), try_reauth=True)

    rps = bindings.get_GetResourcePools(
        api.Session(master_url, "determined", authentication.cli_auth, certs.cli_cert)
    )
    assert rps.resourcePools and len(rps.resourcePools) > 0, "missing resource pool"

    if (
        rps.resourcePools[0].type == bindings.v1ResourcePoolType.RESOURCE_POOL_TYPE_K8S
        and task_type == command.TaskTypeCommand
    ):
        # TODO(DET-6712): Investigate intermittent slowness with K8s command logs.
        return

    body = {}
    if task_type == command.TaskTypeTensorBoard:
        exp_id = exp.run_basic_test(
            conf.fixtures_path("no_op/single.yaml"),
            conf.fixtures_path("no_op"),
            1,
        )
        body.update({"experiment_ids": [exp_id]})

    resp = command.launch_command(
        master_url,
        f"api/v1/{command.RemoteTaskNewAPIs[task_type]}",
        task_config,
        "",
        data={},
        default_body=body,
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
    for log in log_fn(master_url, entity_id, follow=True):
        if log_regex.match(log["message"]):
            break
    else:
        dump_logs_stdout(master_url, entity_id, log_fn)
        pytest.fail("ran out of logs without a match")

    # Just make sure these calls 200 and return some logs.
    assert any(log_fn(master_url, entity_id, tail=10)), "tail returned no logs"
    assert any(log_fn(master_url, entity_id, head=10)), "head returned no logs"

    # Task log fields should work, follow or no follow.
    assert any(log_fields_fn(master_url, entity_id, follow=True)), "log fields returned nothing"
    fields_list = list(log_fields_fn(master_url, entity_id))
    assert any(fields_list), "no task log fields were returned"

    # Convert fields to log_fn filters and check all are valid.
    fields = fields_list[0]
    assert any(fields.values()), "no filter values were returned"

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


def dump_logs_stdout(
    master_url: str,
    entity_id: Any,
    log_fn: Any,
) -> None:
    for log in log_fn(master_url, entity_id, follow=True):
        print(log)


def to_snake_case(camel_case: str) -> str:
    return re.sub(r"(?<!^)(?=[A-Z])", "_", camel_case).lower()
