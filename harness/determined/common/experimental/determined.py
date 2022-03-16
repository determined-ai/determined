import pathlib
import warnings
from typing import Any, Dict, List, Optional, Union, cast

from determined.common import check, context, util, yaml
from determined.common.api import authentication, certs
from determined.common.experimental import checkpoint, experiment, model, session, trial


class _CreateExperimentResponse:
    def __init__(self, raw: Any):
        if not isinstance(raw, dict):
            raise ValueError(f"CreateExperimentResponse must be a dict; got {raw}")

        if "experiment" not in raw:
            raise ValueError(f"CreateExperimentResponse must have an experiment field; got {raw}")
        exp = raw["experiment"]
        if not isinstance(exp, dict):
            raise ValueError(f'CreateExperimentResponse["experiment"] must be a dict; got {exp}')
        if "id" not in exp:
            raise ValueError(f'CreateExperimentResponse["experiment"] must have an id; got {exp}')
        exp_id = exp["id"]
        if not isinstance(exp_id, int):
            raise ValueError(
                f'CreateExperimentResponse["experiment"]["id"] must be a int; got {exp_id}'
            )
        self.id = exp_id


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
        password: Optional[str] = None,
        cert_path: Optional[str] = None,
        cert_name: Optional[str] = None,
        noverify: bool = False,
    ):
        master = master or util.get_default_master_address()

        cert = certs.default_load(
            master_url=master,
            explicit_path=cert_path,
            explicit_cert_name=cert_name,
            explicit_noverify=noverify,
        )

        # TODO: This should probably be try_reauth=False, but it appears that would break the case
        # where the default credentials are available from the master and could be discovered by
        # a REST API call against the master.
        auth = authentication.Authentication(master, user, password, try_reauth=True, cert=cert)

        self._session = session.Session(master, user, auth, cert)

    def create_experiment(
        self,
        config: Union[str, pathlib.Path, Dict],
        model_dir: Union[str, pathlib.Path],
    ) -> experiment.ExperimentReference:
        """
        Create an experiment with config parameters and model directory. The function
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
                experiment_config = util.safe_load_yaml_with_exceptions(f)
        elif isinstance(config, pathlib.Path):
            with config.open() as f:
                experiment_config = util.safe_load_yaml_with_exceptions(f)
        elif isinstance(config, Dict):
            experiment_config = config

        if isinstance(model_dir, str):
            model_dir = pathlib.Path(model_dir)

        model_context, _ = context.read_context(model_dir)

        resp = self._session.post(
            "/api/v1/experiments",
            json={
                "config": yaml.safe_dump(experiment_config),
                "model_definition": model_context,
            },
        )

        exp_id = _CreateExperimentResponse(resp.json()).id
        exp = experiment.ExperimentReference(exp_id, self._session)
        exp.activate()

        return exp

    def get_experiment(self, experiment_id: int) -> experiment.ExperimentReference:
        """
        Get the :class:`~determined.experimental.ExperimentReference` representing the
        experiment with the provided experiment ID.
        """
        return experiment.ExperimentReference(
            experiment_id,
            self._session,
        )

    def get_trial(self, trial_id: int) -> trial.TrialReference:
        """
        Get the :class:`~determined.experimental.TrialReference` representing the
        trial with the provided trial ID.
        """
        return trial.TrialReference(trial_id, self._session)

    def get_checkpoint(self, uuid: str) -> checkpoint.Checkpoint:
        """
        Get the :class:`~determined.experimental.Checkpoint` representing the
        checkpoint with the provided UUID.
        """
        r = self._session.get(f"/api/v1/checkpoints/{uuid}").json()
        return checkpoint.Checkpoint._from_json(r["checkpoint"], self._session)

    def create_model(
        self,
        name: str,
        description: Optional[str] = "",
        metadata: Optional[Dict[str, Any]] = None,
        labels: Optional[List[str]] = None,
    ) -> model.Model:
        """
        Add a model to the model registry.

        Arguments:
            name (string): The name of the model. This name must be unique.
            description (string, optional): A description of the model.
            metadata (dict, optional): Dictionary of metadata to add to the model.
        """
        r = self._session.post(
            "/api/v1/models",
            json={"description": description, "metadata": metadata, "name": name, "labels": labels},
        )

        return model.Model._from_json(r.json().get("model"), self._session)

    def get_model(self, identifier: Union[str, int]) -> model.Model:
        """
        Get the :class:`~determined.experimental.Model` from the model registry
        with the provided identifer, which is either a string-type name or an
        integer-type model ID. If no corresponding model is found in the registry,
        an exception is raised.

        Arguments:
            identifier (string, int): The unique name or ID of the model.
        """
        r = self._session.get(f"/api/v1/models/{identifier}").json()
        assert r.get("model", False)
        return model.Model._from_json(r.get("model"), self._session)

    def get_model_by_id(self, model_id: int) -> model.Model:
        """
        Get the :class:`~determined.experimental.Model` from the model registry
        with the provided id. If no model with that id is found in the registry,
        an exception is raised.

        .. warning::
           Determined.get_model_by_id() has been deprecated and will be removed
           in a future version.
           Please call Determined.get_model() with either a string-type name or
           an integer-type model ID.
        """
        warnings.warn(
            "Determined.get_model_by_id() has been deprecated and will be removed"
            "in a future version.\n"
            "Please call Determined.get_model() with either a string-type name or"
            "an integer-type model ID.",
            FutureWarning,
        )
        return self.get_model(model_id)

    def get_models(
        self,
        sort_by: model.ModelSortBy = model.ModelSortBy.NAME,
        order_by: model.ModelOrderBy = model.ModelOrderBy.ASCENDING,
        name: str = "",
        description: str = "",
        model_id: int = 0,
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
            model_id: If this paramter is set, models will be filtered to
                only include the model with this unique numeric id.
        """
        r = self._session.get(
            "/api/v1/models/",
            params={
                "sort_by": sort_by.value,
                "order_by": order_by.value,
                "name": name,
                "description": description,
                "id": model_id,
            },
        )

        models = r.json().get("models")
        return [model.Model._from_json(m, self._session) for m in models]

    def get_model_labels(self) -> List[str]:
        """
        Get a list of labels used on any models, sorted from most-popular to least-popular.
        """
        r = self._session.get("/api/v1/model/labels")

        labels = r.json().get("labels")
        return cast(List[str], labels)
