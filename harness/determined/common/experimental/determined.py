import io
from pathlib import Path
from typing import Any, Dict, List, Optional

from determined.common import api, context, yaml
from determined.common.api import authentication as auth
from determined.common.experimental.checkpoint import Checkpoint
from determined.common.experimental.experiment import ExperimentReference
from determined.common.experimental.model import Model, ModelOrderBy, ModelSortBy
from determined.common.experimental.session import Session
from determined.common.experimental.trial import TrialReference

import determined.client
from determined.client import V1File as V1File
from determined.client import V1CreateExperimentRequest as CreateExperimentRequest

def _path_to_files(path):
    files = []
    for item in context.read_context(path)[0]:
        content = item["content"].decode('utf-8')
        file = V1File(
            path = item["path"],
            type = item["type"],
            content = content,
            mtime = item["mtime"],
            uid = item["uid"],
            gid = item["gid"],
            mode = item["mode"],
        )
        files.append(file)
    return files

def _parse_config_file(config_file: io.FileIO) -> Dict:
    experiment_config = yaml.safe_load(config_file.read())
    config_file.close()
    return experiment_config

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
        self._session = Session(master, user)
        self._auth = auth.Authentication.instance()
        self._configuration = determined.client.Configuration()

        # Remove trailing '/' character for Swagger
        if (self._session._master[-1] == "/"):
            self._configuration.host = self._session._master[:-1]
        else:
            self._configuration.host = self._session._master
        self._configuration.username = self._auth.token_store.get_active_user()
        self._configuration.api_key_prefix["Authorization"] = "Bearer"
        self._configuration.api_key["Authorization"] = self._auth.get_session_token()

        self._experiments = determined.client.ExperimentsApi(determined.client.ApiClient(self._configuration))
        self._internal = determined.client.InternalApi(determined.client.ApiClient(self._configuration))
        self._trials = determined.client.TrialsApi(determined.client.ApiClient(self._configuration))

    def create_experiment(
        self,
        model_dir: str,
        exp_config: object = None,
    ) -> ExperimentReference:
        if isinstance(exp_config, str):
            f = open(exp_config)
            experiment_config = _parse_config_file(f)
        elif isinstance(exp_config, Dict):
            experiment_config = exp_config
        else:
            raise ValueError("Invalid experiment config")

        model_context = _path_to_files(Path(model_dir))

        experiment_request = CreateExperimentRequest(
            model_definition = model_context,
            config = yaml.safe_dump(experiment_config),
        )
        experiment_response = self._internal.determined_create_experiment(experiment_request)
        return ExperimentReference(experiment_response.experiment.id,
                                    self._session._master,
                                    self._experiments,
                                    experiment_response.config)

    def get_experiment(self, experiment_id: int) -> ExperimentReference:
        """
        Get the :class:`~determined.experimental.ExperimentReference` representing the
        experiment with the provided experiment ID.
        """
        return ExperimentReference(experiment_id, self._session._master, self._experiments)

    def get_trial(self, trial_id: int) -> TrialReference:
        """
        Get the :class:`~determined.experimental.TrialReference` representing the
        trial with the provided trial ID.
        """
        return TrialReference(trial_id, self._session._master, self._trials)

    def get_checkpoint(self, uuid: str) -> Checkpoint:
        """
        Get the :class:`~determined.experimental.Checkpoint` representing the
        checkpoint with the provided UUID.
        """
        r = api.get(self._session._master, "/api/v1/checkpoints/{}".format(uuid)).json()
        return Checkpoint.from_json(r["checkpoint"], master=self._session._master)

    def create_model(
        self, name: str, description: Optional[str] = "", metadata: Optional[Dict[str, Any]] = None
    ) -> Model:
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

        return Model.from_json(r.json().get("model"), self._session._master)

    def get_model(self, name: str) -> Model:
        """
        Get the :class:`~determined.experimental.Model` from the model registry
        with the provided name. If no model with that name is found in the registry,
        an exception is raised.
        """
        r = api.get(self._session._master, "/api/v1/models/{}".format(name))
        return Model.from_json(r.json().get("model"), self._session._master)

    def get_models(
        self,
        sort_by: ModelSortBy = ModelSortBy.NAME,
        order_by: ModelOrderBy = ModelOrderBy.ASCENDING,
        name: str = "",
        description: str = "",
    ) -> List[Model]:
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
        return [Model.from_json(m, self._session._master) for m in models]
