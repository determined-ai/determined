from typing import Any, Dict, List, Optional

import determined_client
import yaml

from determined_common import api
from determined_common.experimental.checkpoint import Checkpoint
from determined_common.experimental.experiment import ExperimentReference
from determined_common.experimental.model import Model, ModelOrderBy, ModelSortBy
from determined_common.experimental.session import Session
from determined_common.experimental.trial import TrialReference


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
        print("Creating Experiment")
        experiment_api = determined_client.ExperimentsApi(self.api_client)
        model_definition = ExperimentReference.path_to_files(context_dir)
        create_experiment_request = determined_client.models.V1CreateExperimentRequest(
            config=yaml.safe_dump(config), validate_only=False, model_definition=model_definition
        )

        response = experiment_api.determined_create_experiment(create_experiment_request)
        config = response.config
        experiment = response.experiment
        experiment_obj = {}
        for attribute in experiment.attribute_map:
            experiment_obj[attribute] = getattr(
                experiment, attribute, getattr(experiment, attribute)
            )
        experiment = ExperimentReference(
            self.api_client, self.api_client.configuration.host, config, experiment_obj
        )

        experiment.activate()
        return experiment

    def get_experiment(self, experiment_id: int) -> ExperimentReference:
        """
        Get the :class:`~determined.experimental.ExperimentReference` representing the
        experiment with the provided experiment ID.
        """
        experiment_api = determined_client.ExperimentsApi(self.api_client)
        response = experiment_api.determined_get_experiment(experiment_id)

        config = response.config
        experiment = response.experiment

        experiment_obj = {}
        for attribute in experiment.attribute_map:
            experiment_obj[attribute] = getattr(
                experiment, attribute, getattr(experiment, attribute)
            )

        return ExperimentReference(
            self.api_client, self.api_client.configuration.host, config, experiment_obj
        )

    def get_trial(self, trial_id: int) -> TrialReference:
        """
        Get the :class:`~determined.experimental.TrialReference` representing the
        trial with the provided trial ID.
        """
        trial_api = determined_client.api.TrialsApi(self.api_client)
        trial_response = trial_api.determined_get_trial(trial_id)
        return TrialReference.from_spec(self.api_client, trial_response.trial)

    def get_checkpoint(self, uuid: str) -> Checkpoint:
        """
        Get the :class:`~determined.experimental.Checkpoint` representing the
        checkpoint with the provided UUID.
        """
        checkpoint_api = determined_client.CheckpointsApi(self.api_client)
        response = checkpoint_api.determined_get_checkpoint(uuid)

        return Checkpoint.from_spec(self.api_client, response.checkpoint)

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
        models_api = determined_client.ModelsApi(self.api_client)

        if not description:
            description = ""
        if not metadata:
            metadata = {}

        model_body = determined_client.models.v1_model.V1Model(
            name=name, description=description, metadata=metadata
        )

        model_response = models_api.determined_post_model(model_name=name, body=model_body)
        return Model.from_spec(model_response.model, self.api_client)

    def get_model(self, name: str) -> Model:
        """
        Get the :class:`~determined.experimental.Model` from the model registry
        with the provided name. If no model with that name is found in the registry,
        an exception is raised.
        """
        # r = api.get(self._session._master, "/api/v1/models/{}".format(name))
        # return Model.from_json(r.json().get("model"), self._session._master)
        model_api = determined_client.ModelsApi(self.api_client)
        response = model_api.determined_get_model(name)
        return Model.from_spec(response.model, self.api_client)

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
        model_api = determined_client.api.ModelsApi(self.api_client)
        models_response = model_api.determined_get_models()

        return [Model.from_spec(model, self.api_client) for model in models_response.models]
