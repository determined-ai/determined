from __future__ import print_function
from typing import Any, Dict, List, Optional, Set, Tuple, Callable
import time
import functools
import base64
import io
import argparse
from argparse import Namespace
from determined.common.declarative_argparse import Arg, Cmd
import yaml
import sys
import json
from determined.common.api import swagger_client as sc
from determined.common.api.swagger_client import models
from determined.common.api.authentication import Authentication, cli_auth
from pathlib import Path
from determined.common.api.swagger_client.rest import ApiException
from pprint import pprint
from determined.common import context, yaml, constants


# SERVER_ADDRESS = "http://localhost:8080"
SAMPLE_EXPERIMENT_DIR = Path.home().joinpath("projects/da/determined/e2e_tests/tests/fixtures/no_op")
# DET_USER = "determined"
# DET_PASS = ""


def _parse_config_file_or_exit(config_path: Path) -> Dict:
    with open(config_path, "r") as config_file:
        experiment_config = yaml.safe_load(config_file.read())
        config_file.close()
        if not experiment_config or not isinstance(experiment_config, dict):
            print("Error: invalid experiment config file {}".format(config_file.name))
            sys.exit(1)
        return experiment_config


def path_to_files(path: Path) -> List[models.V1File]:
    files = []
    for item in context.read_context(path)[0]:
        content = item['content'].decode('ascii')
        file = models.V1File(path=item['path'], type=item['type'], content=content,
        mtime=item['mtime'], uid=item['uid'], gid=item['gid'], mode=item['mode'])
        files.append(file)
    return files


def setup_experiment(experiment_path: Path, config_name: str) -> models.V1CreateExperimentRequest:
    experiment_config = _parse_config_file_or_exit(experiment_path.joinpath(config_name))
    model_context = path_to_files(experiment_path)
    return models.V1CreateExperimentRequest(
        validate_only=False,
        config=yaml.safe_dump(experiment_config),
        model_definition=model_context,
        )


def setup_tensorboard_request(config_path: Path, template_name: str, context_path: Path) -> models.V1LaunchTensorboardRequest:
    files = path_to_files(context_path)
    config_dict = _parse_config_file_or_exit(config_path)
    return models.V1LaunchTensorboardRequest(experiment_ids=[1], trial_ids=None, config=config_dict, files=files)

configuration = sc.Configuration()
configuration.api_key_prefix['Authorization'] = 'Bearer'
experiment_api = sc.ExperimentsApi(sc.ApiClient(configuration))

def swgAuthRequired(func: Callable[[argparse.Namespace], Any]) -> Callable[..., Any]:
    """
    A decorator for cli functions.
    """

    @functools.wraps(func)
    def f(namespace: argparse.Namespace) -> Any:
        global cli_auth, configuration, experiment_api
        configuration.host = namespace.master
        cli_auth = Authentication(namespace.master, namespace.user, try_reauth=True)
        token = cli_auth.get_session_token()
        configuration.api_key['Authorization'] = token

        # TODO avoid global?
        # tensorboard_api = sc.TensorboardsApi(sc.ApiClient(configuration))
        experiment_api = sc.ExperimentsApi(sc.ApiClient(configuration))
        return func(namespace)

    return f

if __name__ == '__main__':
    try:
        # configuration.host = SERVER_ADDRESS
        # Login
        # auth_api = sc.AuthenticationApi(sc.ApiClient(configuration))
        # api_response = auth_api.determined_login(models.V1LoginRequest(DET_USER, DET_PASS)) # TODO hash the password
        # Set auth token
        # configuration.api_key['Authorization'] = api_response.token

        # tensorboard_api = sc.TensorboardsApi(sc.ApiClient(configuration))
        experiment_api = sc.ExperimentsApi(sc.ApiClient(configuration))

        # Create an experiment
        # api_response = experiment_api.determined_create_experiment(setup_experiment(SAMPLE_EXPERIMENT_DIR, "single.yaml"))
        # pprint(api_response)

        # Change an experiment description
        # api_response = experiment_api.determined_patch_experiment(1, {"description": "my new description"})
        # pprint(api_response)

        # Activate an experiment description
        # api_response = experiment_api.determined_activate_experiment(1)
        # pprint(api_response)

        # Launch a tensorboard
        # tensorboard_request = setup_tensorboard_request(
        #     Path.cwd().joinpath("cmd-config.yaml"),
        #     None,
        #     Path.cwd().joinpath(".swagger-codegen")
        # )
        # api_response = tensorboard_api.determined_launch_tensorboard(tensorboard_request)
        # print(api_response)

        # Get an experiment
        api_response = experiment_api.determined_get_experiment(1)
        print(api_response)

    except ApiException as e:
        print("exception", e)

@swgAuthRequired
def list(args: Namespace) -> None:
    api_response = experiment_api.determined_get_experiment(1)
    print(api_response)
    pass

args_description = [
    Cmd("j|ob", None, "manage job", [
        Cmd("list", list, "list jobs", [
            Arg("-o", "--output", type=str, default="yaml",
                help="Output format, one of json|yaml")
        ]),
    ])
]  # type: List[Any]

# fmt: on
