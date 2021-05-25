import pathlib
from typing import Any, Dict, List, Optional, Union

from determined._swagger.client.api.experiments_api import ExperimentsApi
from determined._swagger.client.api.internal_api import InternalApi
from determined._swagger.client.api.trials_api import TrialsApi
from determined._swagger.client.api_client import ApiClient
from determined._swagger.client.configuration import Configuration
from determined._swagger.client.models.v1_create_experiment_request import V1CreateExperimentRequest
from determined._swagger.client.models.v1_file import V1File
from determined.common import api, check, context, yaml
from determined.common.experimental import checkpoint, experiment, model, session, trial


def _path_to_files(path: pathlib.Path) -> List[V1File]:
    files = []
    for item in context.read_context(path)[0]:
        content = item["content"].decode("utf-8")
        file = V1File(
            path=item["path"],
            type=item["type"],
            content=content,
            mtime=item["mtime"],
            uid=item["uid"],
            gid=item["gid"],
            mode=item["mode"],
        )
        files.append(file)
    return files


class Determined:
    """
    Determined gives access to Determined API objects.

    Arguments:
        master (string, optional): The URL of the Determined master. If
            this argument is not specified, the environment variables ``DET_MASTER`` and
            ``DET_MASTER_ADDR`` will be checked for the master URL in that order.
        user (string, optional): The Determined username used for
            authentication. (default: ``determined``)
    """

    def __init__(
        self,
        master: Optional[str] = None,
        user: Optional[str] = None,
    ):
        self._session = session.Session(master, user)

        userauth = api.authentication.Authentication.instance()

        configuration = Configuration()
        configuration.host = self._session._master.rstrip("/")
        configuration.username = userauth.token_store.get_active_user()
        configuration.api_key_prefix["Authorization"] = "Bearer"
        configuration.api_key["Authorization"] = userauth.get_session_token()

        self._experiments = ExperimentsApi(ApiClient(configuration))
        self._internal = InternalApi(ApiClient(configuration))
        self._trials = TrialsApi(ApiClient(configuration))

    def create_experiment(
        self,
        config: Union[str, pathlib.Path, Dict],
        model_dir: str,
    ) -> experiment.ExperimentReference:
        """
        Create an experiment with config parameters and model direcotry. The function
        returns :class:`~determined.experimental.ExperimentReference` of the experiment.

        Arguments:
            config(string, pathlib.Path, dictionary): experiment config filename (.yaml)
                or a dict.
            model_dir(string): directory containing model definition.
        """
        check.is_instance(
            config, (str, pathlib.Path, dict), "config parameter must be dictionary or path"
        )
        if isinstance(config, str):
            with open(config) as f:
                experiment_config = yaml.safe_load(f)
        elif isinstance(config, pathlib.Path):
            with config.open() as f:
                experiment_config = yaml.safe_load(f)
        elif isinstance(config, Dict):
            experiment_config = config

        model_context = _path_to_files(pathlib.Path(model_dir))

        experiment_request = V1CreateExperimentRequest(
            model_definition=model_context,
            config=yaml.safe_dump(experiment_config),
        )
        experiment_response = self._internal.determined_create_experiment(experiment_request)
        return experiment.ExperimentReference(
            experiment_response.experiment.id,
            self._session._master,
            self._experiments,
        )

    def get_experiment(self, experiment_id: int) -> experiment.ExperimentReference:
        """
        Get the :class:`~determined.experimental.ExperimentReference` representing the
        experiment with the provided experiment ID.
        """
        return experiment.ExperimentReference(
            experiment_id,
            self._session._master,
            self._experiments,
        )

    def get_trial(self, trial_id: int) -> trial.TrialReference:
        """
        Get the :class:`~determined.experimental.TrialReference` representing the
        trial with the provided trial ID.
        """
        return trial.TrialReference(trial_id, self._session._master, self._trials)

    def get_checkpoint(self, uuid: str) -> checkpoint.Checkpoint:
        """
        Get the :class:`~determined.experimental.Checkpoint` representing the
        checkpoint with the provided UUID.
        """
        r = api.get(self._session._master, "/api/v1/checkpoints/{}".format(uuid)).json()
        return checkpoint.Checkpoint.from_json(r["checkpoint"], master=self._session._master)

    def create_model(
        self, name: str, description: Optional[str] = "", metadata: Optional[Dict[str, Any]] = None
    ) -> model.Model:
        """
        Add a model to the model registry.

        Arguments:
            name (string): The name of the model. This name must be unique.
            description (string, optional): A description of the model.
            metadata (dict, optional): Dictionary of metadata to add to the model.
        """
        r = api.post(
            self._session._master,
            "/api/v1/models/{}".format(name),
            body={"description": description, "metadata": metadata},
        )

        return model.Model.from_json(r.json().get("model"), self._session._master)

    def get_model(self, name: str) -> model.Model:
        """
        Get the :class:`~determined.experimental.Model` from the model registry
        with the provided name. If no model with that name is found in the registry,
        an exception is raised.
        """
        r = api.get(self._session._master, "/api/v1/models/{}".format(name))
        return model.Model.from_json(r.json().get("model"), self._session._master)

    def get_models(
        self,
        sort_by: model.ModelSortBy = model.ModelSortBy.NAME,
        order_by: model.ModelOrderBy = model.ModelOrderBy.ASCENDING,
        name: str = "",
        description: str = "",
    ) -> List[model.Model]:
        """
        Get a list of all models in the model registry.

        Arguments:
            sort_by: Which field to sort by. See :class:`~determined.experimental.ModelSortBy`.
            order_by: Whether to sort in ascending or descending order. See
                :class:`~determined.experimental.ModelOrderBy`.
            name: If this parameter is set, models will be filtered to only
                include models with names matching this parameter.
            description: If this parameter is set, models will be filtered to
                only include models with descriptions matching this parameter.
        """
        r = api.get(
            self._session._master,
            "/api/v1/models/",
            params={
                "sort_by": sort_by.value,
                "order_by": order_by.value,
                "name": name,
                "description": description,
            },
        )

        models = r.json().get("models")
        return [model.Model.from_json(m, self._session._master) for m in models]
