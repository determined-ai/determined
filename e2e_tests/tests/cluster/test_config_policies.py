import json

import pytest
import yaml

from tests import api_utils
from tests import config as conf
from tests import detproc

VALID_EXPERIMENT_YAML = "config_policies/valid_experiment.yaml"
VALID_EXPERIMENT_JSON = "config_policies/valid_experiment.json"
VALID_NTSC_YAML = "config_policies/valid_ntsc.yaml"
VALID_NTSC_JSON = "config_policies/valid_ntsc.json"


@pytest.mark.e2e_cpu
def test_set_config_policies() -> None:
    sess = api_utils.admin_session()

    workspace_name = api_utils.get_random_string()
    create_workspace_cmd = ["det", "workspace", "create", workspace_name]
    detproc.check_call(sess, create_workspace_cmd)

    # workspace valid path with experiment type YAML
    stdout = detproc.check_output(
        sess,
        [
            "det",
            "config-policies",
            "set",
            "experiment",
            conf.fixtures_path(VALID_EXPERIMENT_YAML),
            "--workspace-name",
            workspace_name,
        ],
    )

    data = f"Set experiment config policies for workspace {workspace_name}:\n"
    with open(conf.fixtures_path(VALID_EXPERIMENT_YAML), "r") as f:
        data += f.read()
    print(stdout)
    print(data)
    assert data.rstrip() == stdout.rstrip()

    # workspace valid path with ntsc type YAML
    stdout = detproc.check_output(
        sess,
        [
            "det",
            "config-policies",
            "set",
            "tasks",
            conf.fixtures_path(VALID_NTSC_YAML),
            "--workspace-name",
            workspace_name,
        ],
    )
    data = f"Set tasks config policies for workspace {workspace_name}:\n"
    with open(conf.fixtures_path(VALID_NTSC_YAML), "r") as f:
        data += f.read()
    assert data.rstrip() == stdout.rstrip()

    # global valid path with experiment type JSON
    stdout = detproc.check_output(
        sess,
        [
            "det",
            "config-policies",
            "set",
            "experiment",
            conf.fixtures_path(VALID_EXPERIMENT_JSON),
        ],
    )

    data = "Set global experiment config policies:\n"
    with open(conf.fixtures_path(VALID_EXPERIMENT_YAML), "r") as f:
        data += f.read()
    print(stdout)
    print(data)
    assert data.rstrip() == stdout.rstrip()

    # global valid path with ntsc type JSON
    stdout = detproc.check_output(
        sess,
        [
            "det",
            "config-policies",
            "set",
            "tasks",
            conf.fixtures_path(VALID_NTSC_JSON),
        ],
    )
    data = "Set global tasks config policies:\n"
    with open(conf.fixtures_path(VALID_NTSC_YAML), "r") as f:
        data += f.read()
    assert data.rstrip() == stdout.rstrip()

    # workload type not provided
    detproc.check_error(
        sess,
        [
            "det",
            "config-policies",
            "set",
            conf.fixtures_path(VALID_NTSC_YAML),
        ],
        "argument workload_type",
    )

    # invalid path
    detproc.check_error(
        sess,
        [
            "det",
            "config-policies",
            "set",
            "experiment",
            conf.fixtures_path("config_policies/non-existent.yaml"),
            "--workspace-name",
            workspace_name,
        ],
        "No such file or directory",
    )

    # path not provided
    detproc.check_error(
        sess,
        [
            "det",
            "config-policies",
            "set",
            "experiment",
            "--workspace-name",
            workspace_name,
        ],
        "the following arguments are required: config_file",
    )


@pytest.mark.e2e_cpu
def test_describe_config_policies() -> None:
    sess = api_utils.admin_session()

    workspace_name = api_utils.get_random_string()
    create_workspace_cmd = ["det", "workspace", "create", workspace_name]
    detproc.check_call(sess, create_workspace_cmd)

    # set workspace config policies
    detproc.check_call(
        sess,
        [
            "det",
            "config-policies",
            "set",
            "experiment",
            conf.fixtures_path(VALID_EXPERIMENT_YAML),
            "--workspace-name",
            workspace_name,
        ],
    )

    # set global config policies
    detproc.check_call(
        sess,
        [
            "det",
            "config-policies",
            "set",
            "experiment",
            conf.fixtures_path(VALID_EXPERIMENT_YAML),
        ],
    )

    # workspace no specified return type
    stdout = detproc.check_output(
        sess,
        [
            "det",
            "config-policies",
            "describe",
            "experiment",
            "--workspace-name",
            workspace_name,
        ],
    )
    with open(conf.fixtures_path(VALID_EXPERIMENT_YAML), "r") as f:
        data = f.read()
    assert data.rstrip() == stdout.rstrip()

    # global no specified return type.
    stdout = detproc.check_output(
        sess,
        [
            "det",
            "config-policies",
            "describe",
            "experiment",
        ],
    )
    with open(conf.fixtures_path(VALID_EXPERIMENT_YAML), "r") as f:
        data = f.read()
    assert data.rstrip() == stdout.rstrip()

    # workspace yaml return type
    stdout = detproc.check_output(
        sess,
        [
            "det",
            "config-policies",
            "describe",
            "experiment",
            "--workspace-name",
            workspace_name,
            "--yaml",
        ],
    )
    with open(conf.fixtures_path(VALID_EXPERIMENT_YAML), "r") as f:
        data = f.read()
    assert data.rstrip() == stdout.rstrip()

    # global yaml return type
    stdout = detproc.check_output(
        sess,
        [
            "det",
            "config-policies",
            "describe",
            "experiment",
            "--yaml",
        ],
    )
    with open(conf.fixtures_path(VALID_EXPERIMENT_YAML), "r") as f:
        data = f.read()
    assert data.rstrip() == stdout.rstrip()

    # workspace json return type
    stdout = detproc.check_output(
        sess,
        [
            "det",
            "config-policies",
            "describe",
            "experiment",
            "--workspace-name",
            workspace_name,
            "--json",
        ],
    )
    with open(conf.fixtures_path(VALID_EXPERIMENT_YAML), "r") as f:
        data = f.read()
    original_data = yaml.load(data, Loader=yaml.SafeLoader)
    # check if output is a valid json containing original data
    assert original_data == json.loads(stdout)

    # global json return type
    stdout = detproc.check_output(
        sess,
        [
            "det",
            "config-policies",
            "describe",
            "experiment",
            "--json",
        ],
    )
    with open(conf.fixtures_path(VALID_EXPERIMENT_YAML), "r") as f:
        data = f.read()
    original_data = yaml.load(data, Loader=yaml.SafeLoader)
    # check if output is a valid json containing original data
    assert original_data == json.loads(stdout)


@pytest.mark.e2e_cpu
def test_delete_config_policies() -> None:
    sess = api_utils.admin_session()

    workspace_name = api_utils.get_random_string()
    create_workspace_cmd = ["det", "workspace", "create", workspace_name]
    detproc.check_call(sess, create_workspace_cmd)

    # workspace set config policies
    detproc.check_call(
        sess,
        [
            "det",
            "config-policies",
            "set",
            "experiment",
            conf.fixtures_path(VALID_EXPERIMENT_YAML),
            "--workspace-name",
            workspace_name,
        ],
    )

    # workspace delete config policies
    stdout = detproc.check_output(
        sess,
        [
            "det",
            "config-policies",
            "delete",
            "experiment",
            "--workspace-name",
            workspace_name,
        ],
    )
    assert "Successfully deleted" in stdout

    # global set config policies
    detproc.check_call(
        sess,
        [
            "det",
            "config-policies",
            "set",
            "experiment",
            conf.fixtures_path(VALID_EXPERIMENT_YAML),
        ],
    )

    # global delete config policies
    stdout = detproc.check_output(
        sess,
        [
            "det",
            "config-policies",
            "delete",
            "experiment",
        ],
    )
    assert "Successfully deleted" in stdout
