from determined.common.api.authentication import Authentication, cli_auth
from determined.common.api import swagger_client as sc
import functools
from typing import Any, Callable
import argparse

configuration = sc.Configuration()
configuration.api_key_prefix['Authorization'] = 'Bearer'
experiment_api = sc.ExperimentsApi(sc.ApiClient(configuration))
job_api = sc.JobsApi(sc.ApiClient(configuration))

def auth_required(func: Callable[[argparse.Namespace], Any]) -> Callable[..., Any]:
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
        job_api = sc.JobsApi(sc.ApiClient(configuration))
        return func(namespace)

    return f


# SAMPLE_EXPERIMENT_DIR = Path.home().joinpath("projects/da/determined/e2e_tests/tests/fixtures/no_op")

# def _parse_config_file_or_exit(config_path: Path) -> Dict:
#     with open(config_path, "r") as config_file:
#         experiment_config = yaml.safe_load(config_file.read())
#         config_file.close()
#         if not experiment_config or not isinstance(experiment_config, dict):
#             print("Error: invalid experiment config file {}".format(config_file.name))
#             sys.exit(1)
#         return experiment_config


# def path_to_files(path: Path) -> List[models.V1File]:
#     files = []
#     for item in context.read_context(path)[0]:
#         content = item['content'].decode('ascii')
#         file = models.V1File(path=item['path'], type=item['type'], content=content,
#         mtime=item['mtime'], uid=item['uid'], gid=item['gid'], mode=item['mode'])
#         files.append(file)
#     return files


# def setup_experiment(experiment_path: Path, config_name: str) -> models.V1CreateExperimentRequest:
#     experiment_config = _parse_config_file_or_exit(experiment_path.joinpath(config_name))
#     model_context = path_to_files(experiment_path)
#     return models.V1CreateExperimentRequest(
#         validate_only=False,
#         config=yaml.safe_dump(experiment_config),
#         model_definition=model_context,
#         )


# def setup_tensorboard_request(config_path: Path, template_name: str, context_path: Path) -> models.V1LaunchTensorboardRequest:
#     files = path_to_files(context_path)
#     config_dict = _parse_config_file_or_exit(config_path)
#     return models.V1LaunchTensorboardRequest(experiment_ids=[1], trial_ids=None, config=config_dict, files=files)
