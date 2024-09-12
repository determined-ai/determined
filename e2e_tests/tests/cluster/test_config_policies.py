import json

import pytest
import yaml

from tests import api_utils
from tests import config as conf
from tests import detproc


@pytest.mark.e2e_cpu
def test_set_config_policies() -> None:
    sess = api_utils.user_session()

    workspace_name = api_utils.get_random_string()
    create_workspace_cmd = ["det", "workspace", "create", workspace_name]
    detproc.check_call(sess, create_workspace_cmd)

    # valid path with experiment type
    stdout = detproc.check_output(
        sess,
        [
            "det",
            "config-policies",
            "set",
            "experiment",
            "--workspace",
            workspace_name,
            "--config-file",
            conf.fixtures_path("config_policies/valid.yaml"),
        ],
    )

    with open(conf.fixtures_path("config_policies/valid.yaml"), "r") as f:
        data = f.read()
    print(stdout)
    print(data)
    assert data.rstrip() == stdout.rstrip()

    # valid path with ntsc type
    stdout = detproc.check_output(
        sess,
        [
            "det",
            "config-policies",
            "set",
            "ntsc",
            "--workspace",
            workspace_name,
            "--config-file",
            conf.fixtures_path("config_policies/valid.yaml"),
        ],
    )

    with open(conf.fixtures_path("config_policies/valid.yaml"), "r") as f:
        data = f.read()
    assert data.rstrip() == stdout.rstrip()

    # invalid path
    detproc.check_error(
        sess,
        [
            "det",
            "config-policies",
            "set",
            "experiment",
            "--workspace",
            workspace_name,
            "--config-file",
            conf.fixtures_path("config_policies/non-existent.yaml"),
        ],
        "No such file or directory",
    )

    # workspace name not provided
    detproc.check_error(
        sess,
        [
            "det",
            "config-policies",
            "set",
            "ntsc",
            "--config-file",
            conf.fixtures_path("config_policies/valid.yaml"),
        ],
        "the following arguments are required: --workspace",
    )

    # path not provided
    detproc.check_error(
        sess,
        [
            "det",
            "config-policies",
            "set",
            "ntsc",
            "--workspace",
            workspace_name,
        ],
        "the following arguments are required: --config-file",
    )


@pytest.mark.e2e_cpu
def test_describe_config_policies() -> None:
    sess = api_utils.user_session()

    workspace_name = api_utils.get_random_string()
    create_workspace_cmd = ["det", "workspace", "create", workspace_name]
    detproc.check_call(sess, create_workspace_cmd)

    # set config policies
    detproc.check_call(
        sess,
        [
            "det",
            "config-policies",
            "set",
            "ntsc",
            "--workspace",
            workspace_name,
            "--config-file",
            conf.fixtures_path("config_policies/valid.yaml"),
        ],
    )

    # no specified return type
    stdout = detproc.check_output(
        sess,
        [
            "det",
            "config-policies",
            "describe",
            "ntsc",
            "--workspace",
            workspace_name,
        ],
    )
    with open(conf.fixtures_path("config_policies/valid.yaml"), "r") as f:
        data = f.read()
    assert data.rstrip() == stdout.rstrip()

    # yaml return type
    stdout = detproc.check_output(
        sess,
        [
            "det",
            "config-policies",
            "describe",
            "ntsc",
            "--workspace",
            workspace_name,
            "--yaml",
        ],
    )
    with open(conf.fixtures_path("config_policies/valid.yaml"), "r") as f:
        data = f.read()
    assert data.rstrip() == stdout.rstrip()

    # json return type
    stdout = detproc.check_output(
        sess,
        [
            "det",
            "config-policies",
            "describe",
            "ntsc",
            "--workspace",
            workspace_name,
            "--json",
        ],
    )
    with open(conf.fixtures_path("config_policies/valid.yaml"), "r") as f:
        data = f.read()
    original_data = yaml.load(data, Loader=yaml.SafeLoader)
    # check if output is a valid json containing original data
    assert original_data == json.loads(stdout)

    # workspace name not provided
    detproc.check_error(
        sess,
        [
            "det",
            "config-policies",
            "describe",
            "ntsc",
        ],
        "the following arguments are required: --workspace",
    )


@pytest.mark.e2e_cpu
def test_delete_config_policies() -> None:
    sess = api_utils.user_session()

    workspace_name = api_utils.get_random_string()
    create_workspace_cmd = ["det", "workspace", "create", workspace_name]
    detproc.check_call(sess, create_workspace_cmd)

    # set config policies
    detproc.check_call(
        sess,
        [
            "det",
            "config-policies",
            "set",
            "ntsc",
            "--workspace",
            workspace_name,
            "--config-file",
            conf.fixtures_path("config_policies/valid.yaml"),
        ],
    )

    stdout = detproc.check_output(
        sess,
        [
            "det",
            "config-policies",
            "delete",
            "ntsc",
            "--workspace",
            workspace_name,
        ],
    )
    assert "Successfully deleted" in stdout

    # workspace name not provided
    detproc.check_error(
        sess,
        [
            "det",
            "config-policies",
            "delete",
            "ntsc",
        ],
        "the following arguments are required: --workspace",
    )
