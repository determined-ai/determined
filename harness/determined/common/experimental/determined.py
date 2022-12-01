import pathlib
import warnings
from typing import Any, Dict, Iterable, List, Optional, Sequence, Union

from determined.common import api, context, util, yaml
from determined.common.api import authentication, bindings, certs
from determined.common.experimental import checkpoint, experiment, model, trial, user


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
        self._session = api.Session(master, user, auth, cert)

    def _from_bindings(self, raw: bindings.v1User) -> user.User:
        assert raw.id is not None
        if raw.agentUserGroup is not None:
            return user.User(
                user_id=raw.id,
                username=raw.username,
                admin=raw.admin,
                session=self._session,
                active=raw.active,
                display_name=raw.displayName,
                agent_uid=raw.agentUserGroup.agentUid,
                agent_gid=raw.agentUserGroup.agentGid,
                agent_user=raw.agentUserGroup.agentUser,
                agent_group=raw.agentUserGroup.agentGroup,
            )
        else:
            return user.User(
                user_id=raw.id,
                username=raw.username,
                admin=raw.admin,
                session=self._session,
                active=raw.active,
                display_name=raw.displayName,
            )

    def create_user(self, username: str, admin: bool, password: Optional[str]) -> user.User:
        create_user = bindings.v1User(username=username, admin=admin, active=True)
        hashedPassword = None
        if password is not None:
            hashedPassword = api.salt_and_hash(password)
        req = bindings.v1PostUserRequest(password=hashedPassword, user=create_user, isHashed=True)
        resp = bindings.post_PostUser(self._session, body=req)
        assert resp.user is not None
        return self._from_bindings(resp.user)

    def get_user_by_id(self, user_id: int) -> user.User:
        resp = bindings.get_GetUser(session=self._session, userId=user_id)
        assert user_id is not None
        return self._from_bindings(resp.user)

    def get_user_by_name(self, user_name: str) -> user.User:
        resp = bindings.get_GetUserByUsername(session=self._session, username=user_name)
        return self._from_bindings(resp.user)

    def whoami(self) -> user.User:
        resp = bindings.get_GetMe(self._session)
        return self._from_bindings(resp.user)

    def logout(self) -> None:
        auth = self._session._auth
        # auth should only be None in the special login Session, which must not be used in a
        # Determined object.
        assert auth, "Determined.logout() found an unauthorized Session"

        user = auth.get_session_user()
        # get_session_user() is allowed to return an empty string, which seems dumb, but in that
        # case we do not want to trigger the authentication.logout default username lookup logic.
        assert user, "Determined.logout() couldn't find a valid username"

        authentication.logout(self._session._master, user, self._session._cert)

    def list_users(self) -> Sequence[user.User]:
        users_bindings = bindings.get_GetUsers(session=self._session).users
        users: List[user.User] = []
        if users_bindings is None:
            return users
        for user_b in users_bindings:
            user_obj = self._from_bindings(user_b)
            users.append(user_obj)
        return users

    def create_experiment(
        self,
        config: Union[str, pathlib.Path, Dict],
        model_dir: Union[str, pathlib.Path],
        includes: Optional[Iterable[Union[str, pathlib.Path]]] = None,
    ) -> experiment.ExperimentReference:
        """
        Create an experiment with config parameters and model directory. The function
        returns :class:`~determined.experimental.ExperimentReference` of the experiment.

        Arguments:
            config(string, pathlib.Path, dictionary): experiment config filename (.yaml)
                or a dict.
            model_dir(string): directory containing model definition.
            includes (Iterable[Union[str, pathlib.Path]], optional): Additional files or
            directories to include in the model definition.  (default: ``None``)
        """
        if isinstance(config, str):
            with open(config) as f:
                config_text = f.read()
            _ = util.safe_load_yaml_with_exceptions(config_text)
        elif isinstance(config, pathlib.Path):
            with config.open() as f:
                config_text = f.read()
            _ = util.safe_load_yaml_with_exceptions(config_text)
        elif isinstance(config, Dict):
            yaml_dump = yaml.dump(config)
            assert yaml_dump is not None
            config_text = yaml_dump
        else:
            raise ValueError("config parameter must be dictionary or path")

        if isinstance(model_dir, str):
            model_dir = pathlib.Path(model_dir)

        path_includes = (pathlib.Path(i) for i in includes or [])
        model_context = context.read_v1_context(model_dir, includes=path_includes)

        req = bindings.v1CreateExperimentRequest(
            # TODO: add this as a param to create_experiment()
            activate=True,
            config=config_text,
            modelDefinition=model_context,
            # TODO: add these as params to create_experiment()
            parentId=None,
            projectId=None,
        )

        resp = bindings.post_CreateExperiment(self._session, body=req)

        exp_id = resp.experiment.id
        exp = experiment.ExperimentReference(exp_id, self._session)

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
        resp = bindings.get_GetCheckpoint(self._session, checkpointUuid=uuid)
        return checkpoint.Checkpoint._from_bindings(resp.checkpoint, self._session)

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

        # TODO: add notes param to create_model()
        req = bindings.v1PostModelRequest(
            name=name, description=description, labels=labels, metadata=metadata, notes=None
        )

        resp = bindings.post_PostModel(self._session, body=req)

        return model.Model._from_bindings(resp.model, self._session)

    def get_model(self, identifier: Union[str, int]) -> model.Model:
        """
        Get the :class:`~determined.experimental.Model` from the model registry
        with the provided identifer, which is either a string-type name or an
        integer-type model ID. If no corresponding model is found in the registry,
        an exception is raised.

        Arguments:
            identifier (string, int): The unique name or ID of the model.
        """

        resp = bindings.get_GetModel(self._session, modelName=str(identifier))
        return model.Model._from_bindings(resp.model, self._session)

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
            "Determined.get_model_by_id() has been deprecated and will be removed "
            "in a future version.\n"
            "Please call Determined.get_model() with either a string-type name or "
            "an integer-type model ID.",
            FutureWarning,
        )
        return self.get_model(model_id)

    def get_models(
        self,
        sort_by: model.ModelSortBy = model.ModelSortBy.NAME,
        order_by: model.ModelOrderBy = model.ModelOrderBy.ASCENDING,
        name: Optional[str] = None,
        description: Optional[str] = None,
        model_id: Optional[int] = None,
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
        # TODO: more parameters?
        #   - archived
        #   - labels
        #   - userIds
        #   - users
        def get_with_offset(offset: int) -> bindings.v1GetModelsResponse:
            return bindings.get_GetModels(
                self._session,
                archived=None,
                description=description,
                id=model_id,
                labels=None,
                name=name,
                offset=offset,
                orderBy=order_by._to_bindings(),
                sortBy=sort_by._to_bindings(),
                userIds=None,
                users=None,
            )

        resps = api.read_paginated(get_with_offset)

        return [model.Model._from_bindings(m, self._session) for r in resps for m in r.models]

    def get_model_labels(self) -> List[str]:
        """
        Get a list of labels used on any models, sorted from most-popular to least-popular.
        """
        return list(bindings.get_GetModelLabels(self._session).labels)
