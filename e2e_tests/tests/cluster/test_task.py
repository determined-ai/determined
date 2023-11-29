import json
import pathlib

import pytest

from determined.cli import command
from determined.common import context, util
from determined.common.api import bindings
from tests import api_utils
from tests import config as conf


@pytest.mark.e2e_cpu
def test_task_get_config() -> None:
    # create generic task
    sess = api_utils.determined_test_session(admin=True)
    with open(conf.fixtures_path("configuration/test_config.yaml"), "r") as config_file:
        config = command.parse_config(config_file, None, [], [])
        config_text = util.yaml_safe_dump(config)
        context_directory = context.read_v1_context(pathlib.Path("e2e_tests"), [])
        req = bindings.v1CreateGenericTaskRequest(
            config=config_text,
            contextDirectory=context_directory,
            projectId=None,
        )
        task_resp = bindings.post_CreateGenericTask(sess, body=req)
        assert len(task_resp.taskId) > 0
        config_file.close()

        config_resp = bindings.get_GetTaskConfig(sess, taskId=task_resp.taskId)
        result = json.loads(config_resp.config)
        expected = {"entrypoint": ["python3", "run.py"]}
        assert expected["entrypoint"] == result["entrypoint"]
