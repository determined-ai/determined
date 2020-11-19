from typing import Any, Dict, List, Optional

import determined_client

from determined_common import api
from determined_common.experimental.checkpoint import Checkpoint
from determined_common.experimental.experiment import Experiment
from determined_common.experimental.model import Model, ModelOrderBy, ModelSortBy
from determined_common.experimental.session import Session
from determined_common.experimental.trial import Trial


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
        master: Optional[str] = "http://localhost:8080",
        user: Optional[str] = "determined",
        password: Optional[str] = "",
    ):
        # migrate to login API - Old replace with code below
        # self._session = Session(master, user)

        # This is where swagger auth will go
        self.configuration = determined_client.Configuration()
        self.configuration.host = master
        self.configuration.api_key_prefix["Authorization"] = "Bearer"

        self.api_client = determined_client.ApiClient(self.configuration)

        # Login
        auth_api = determined_client.AuthenticationApi(self.api_client)
        api_response = auth_api.determined_login(
            determined_client.models.V1LoginRequest(user, password)
        )
        # Set auth token
        self.configuration.api_key["Authorization"] = api_response.token

    def create_experiment(self, config, context_dir=None, local=False, test=False):
        experiment = Experiment.create_experiment(self.api_client, config, context_dir, local, test)
        experiment.activate()
        return experiment

    def get_experiment(self, experiment_id: int) -> Experiment:
        """
        Get the :class:`~determined.experimental.ExperimentReference` representing the
        experiment with the provided experiment ID.
        """
        return Experiment.get_experiment(self.api_client, experiment_id)

    def get_trial(self, trial_id: int) -> Trial:
        """
        Get the :class:`~determined.experimental.TrialReference` representing the
        trial with the provided trial ID.
        """
        # return Trial(trial_id, self._session._master)
        return Trial.get_trial(self.api_client, trial_id)

    def get_checkpoint(self, uuid: str) -> Checkpoint:
        """
        Get the :class:`~determined.experimental.Checkpoint` representing the
        checkpoint with the provided UUID.
        """
        return Checkpoint.get_checkpoint(self.api_client, uuid)

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
        return Model.create_model(self.api_client, name, description, metadata)

    def get_model(self, name: str) -> Model:
        """
        Get the :class:`~determined.experimental.Model` from the model registry
        with the provided name. If no model with that name is found in the registry,
        an exception is raised.
        """
        # r = api.get(self._session._master, "/api/v1/models/{}".format(name))
        # return Model.from_json(r.json().get("model"), self._session._master)
        return Model.get_model(self.api_client, name)

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

        return Model.get_models(self.api_client)
