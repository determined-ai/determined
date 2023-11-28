import json
from pathlib import Path

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
        context_directory = context.read_v1_context(Path("e2e_tests"), [])
        req = bindings.v1CreateGenericTaskRequest(
            config=config_text, contextDirectory=context_directory, projectId=None, forkedFrom=None
        )
        task_resp = bindings.post_CreateGenericTask(sess, body=req)
        assert len(task_resp.taskId) > 0
        config_file.close()

        config_resp = bindings.get_GetTaskConfig(sess, taskId=task_resp.taskId)
        result = json.loads(config_resp.config)
        expected = json.loads(
            """
        {
            "bind_mounts": [
                {
                    "container_path": "./master",
                    "host_path": "/Users/aaronamanuel/workspace/determined/master",
                    "propagation": "rprivate",
                    "read_only":false
                }],
                "debug":false,
                "entrypoint": ["python3","run.py"],
                "environment": {
                    "add_capabilities": null,
                    "drop_capabilities": null,
                    "environment_variables":{},
                    "force_pull_image": false,
                    "image":{
                        "cpu": "determinedai/environments:py-3.8-pytorch-1.12-tf-2.11-cpu-2b7e2a1",
                        "cuda": "determinedai/environments:cuda-11.3-pytorch-1.12-tf-2.11-gpu-2b7e2a1",
                        "rocm": "determinedai/environments:rocm-5.0-pytorch-1.10-tf-2.7-rocm-2b7e2a1"
                        },
                    "pod_spec": null,
                    "ports": null,
                    "proxy_ports":null
                    },
                "pbs":{},
                "resources": {
                    "devices":null,
                    "is_single_node":true,
                    "max_slots":null,
                    "priority":null,
                    "resource_pool":"default",
                    "shm_size":null,
                    "slots_per_task":1,
                    "slots_per_trial":null,
                    "weight":null
                    },
                "slurm":{},
                "work_dir":null
        }
        """
        )
        assert sorted(expected.items()) == sorted(result.items())


@pytest.mark.e2e_cpu
def test_task_fork() -> None:
    sess = api_utils.determined_test_session()
    with open(conf.fixtures_path("configuration/test_config.yaml"), "r") as config_file:
        config = command.parse_config(config_file, None, [], [])
        config_text = util.yaml_safe_dump(config)
        context_directory = context.read_v1_context(Path("e2e_tests"), [])
        req = bindings.v1CreateGenericTaskRequest(
            config=config_text, contextDirectory=context_directory, projectId=None, forkedFrom=None
        )
        task_resp = bindings.post_CreateGenericTask(sess, body=req)
        assert len(task_resp.taskId) > 0
        config_file.close()

        with open(conf.fixtures_path("configuration/test_config_fork.yaml"), "r") as config_file:
            config = command.parse_config(config_file, None, [], [])
            config_text = util.yaml_safe_dump(config)
            context_directory = context.read_v1_context(Path("e2e_tests"), [])
            req = bindings.v1CreateGenericTaskRequest(
                config=config_text,
                contextDirectory=context_directory,
                projectId=None,
                forkedFrom=task_resp.taskId,
            )
            fork_task_resp = bindings.post_CreateGenericTask(sess, body=req)
            assert len(fork_task_resp.taskId) > 0
            config_file.close()
