import itertools
import logging
import pathlib
import warnings
from typing import Any, Dict, Iterable, List, Optional, Sequence, Union

import determined as det
from determined.common import api, context, util
from determined.common.api import authentication, bindings, certs, errors
from determined.common.experimental import (
    checkpoint,
    experiment,
    metrics,
    model,
    oauth2_scim_client,
    trial,
    user,
    workspace,
)

# TODO (MLG-1087): move OrderBy to experimental.client namespace
from determined.common.experimental._util import OrderBy  # noqa: I2041

logger = logging.getLogger("determined.client")


class Determined:
    # Dev note: Determined is basically a wrapper around Session that calls generated bindings.
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
        self._master = api.canonicalize_master_url(master or api.get_default_master_url())

        cert = certs.default_load(
            master_url=self._master,
            explicit_path=cert_path,
            explicit_cert_name=cert_name,
            explicit_noverify=noverify,
        )

        self._session = authentication.login_with_cache(self._master, user, password, cert=cert)

    @classmethod
    def _from_session(cls, session: api.Session) -> "Determined":
        """Create a new Determined object that uses an existing session.

        This constructor exists to help the CLI transition to using SDK methods, most of which are
        derived from a Determined object at some point in their lifespan.
        """
        # mypy gives new_det "Any" type, even if cls is annotated
        new_det = cls.__new__(cls)  # type: Determined
        new_det._session = session
        return new_det

    def create_user(
        self, username: str, admin: bool, password: Optional[str] = None, remote: bool = False
    ) -> user.User:
        """Creates a user.

        The user's credentials may be managed by a remote service (Enterprise edition only),
        in which case the `remote` argument should be set to `true`, and then SSO should be
        configured for the user. A remote user has no password and cannot log in except via SSO.
        Otherwise, a password must be set that meets complexity requirements.

        The complexity requirements are:
            - Must be at least 8 characters long.
            - Must contain at least one upper-case letter.
            - Must contain at least one lower-case letter.
            - Must contain at least one number.

        Arg:
            username: username of the user.
            admin: indicates whether the user is an admin.
            password: password of the user.
            remote: indicates whether the user is managed by a remote service.

        Returns:
            A :class:`~determined.experimental.client.User` of the created user.

        Raises:
            ValueError: an error describing why the password does not meet complexity requirements.
        """
        create_user = bindings.v1User(username=username, admin=admin, active=True, remote=remote)
        hashedPassword = None
        if not remote:
            authentication.check_password_complexity(password)
            if password is not None:
                hashedPassword = api.salt_and_hash(password)
        req = bindings.v1PostUserRequest(password=hashedPassword, user=create_user, isHashed=True)
        resp = bindings.post_PostUser(self._session, body=req)
        assert resp.user is not None
        return user.User._from_bindings(resp.user, self._session)

    def get_user_by_id(self, user_id: int) -> user.User:
        resp = bindings.get_GetUser(session=self._session, userId=user_id)
        assert user_id is not None
        return user.User._from_bindings(resp.user, self._session)

    def get_user_by_name(self, user_name: str) -> user.User:
        resp = bindings.get_GetUserByUsername(session=self._session, username=user_name)
        return user.User._from_bindings(resp.user, self._session)

    def whoami(self) -> user.User:
        resp = bindings.get_GetMe(self._session)
        return user.User._from_bindings(resp.user, self._session)

    def get_session_username(self) -> str:
        return self._session.username

    def logout(self) -> None:
        """Log out of the current session.

        This results in dropping any cached credentials and sending a request to master to
        invalidate the session's token.
        """
        authentication.logout(self._session.master, self._session.username, self._session.cert)

    def list_users(self, active: Optional[bool] = None) -> List[user.User]:
        def get_with_offset(offset: int) -> bindings.v1GetUsersResponse:
            return bindings.get_GetUsers(session=self._session, offset=offset, active=active)

        resps = api.read_paginated(get_with_offset)

        users = []
        for r in resps:
            if not r.users:
                continue
            for u in r.users:
                users.append(user.User._from_bindings(u, self._session))

        return users

    def create_experiment(
        self,
        config: Union[str, pathlib.Path, Dict],
        model_dir: Optional[Union[str, pathlib.Path]] = None,
        includes: Optional[Iterable[Union[str, pathlib.Path]]] = None,
        project_id: Optional[int] = None,
        template: Optional[str] = None,
    ) -> experiment.Experiment:
        """
        Create an experiment with config parameters and model directory. The function
        returns an :class:`~determined.experimental.Experiment`.

        Arguments:
            config(string, pathlib.Path, dictionary): experiment config filename (.yaml)
                or a dict.
            model_dir(string, optional): directory containing model definition. (default: ``None``)
            includes(Iterable[Union[str, pathlib.Path]], optional): Additional files or
                directories to include in the model definition. (default: ``None``)
            project_id(int, optional): The id of the project this experiment should belong to.
            (default: ``None``)
            template(string, optional): The name of the template for the experiment.
                See :ref:`config-template` for moredetails.
            (default: ``None``)
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
            yaml_dump = util.yaml_safe_dump(config)
            assert yaml_dump is not None
            config_text = yaml_dump
        else:
            raise ValueError("config parameter must be dictionary or path")

        if isinstance(model_dir, str):
            model_dir = pathlib.Path(model_dir)

        path_includes = (pathlib.Path(i) for i in includes or [])

        model_context = None
        if model_dir is not None:
            model_context = context.read_v1_context(model_dir, includes=path_includes)

        req = bindings.v1CreateExperimentRequest(
            # TODO: add this as a param to create_experiment()
            activate=True,
            config=config_text,
            modelDefinition=model_context,
            projectId=project_id,
            template=template,
        )

        resp = bindings.post_CreateExperiment(self._session, body=req)

        if resp.warnings:
            for w in resp.warnings:
                logger.warning(api.WARNING_MESSAGE_MAP[w])

        return experiment.Experiment._from_bindings(resp.experiment, self._session)

    def get_experiment(self, experiment_id: int) -> experiment.Experiment:
        """
        Get an experiment (:class:`~determined.experimental.Experiment`) by experiment ID.
        """
        resp = bindings.get_GetExperiment(session=self._session, experimentId=experiment_id)
        return experiment.Experiment._from_bindings(resp.experiment, self._session)

    def list_experiments(
        self,
        sort_by: Optional[experiment.ExperimentSortBy] = None,
        order_by: Optional[OrderBy] = None,
        experiment_ids: Optional[List[int]] = None,
        labels: Optional[List[str]] = None,
        users: Optional[List[str]] = None,
        states: Optional[List[experiment.ExperimentState]] = None,
        name: Optional[str] = None,
        project_id: Optional[int] = None,
    ) -> List[experiment.Experiment]:
        """Get a list of experiments (:class:`~determined.experimental.Experiment`).

        Arguments:
            sort_by: Which field to sort by. See
                :class:`~determined.experimental.ExperimentSortBy`.
            order_by: Whether to sort in ascending or descending order. See
                :class:`~determined.experimental.OrderBy`.
            name: If this parameter is set, experiments will be filtered to only include those
                with names matching this parameter.
            experiment_ids: Only return experiments with these IDs.
            labels: Only return experiments with a label in this list.
            users: Only return experiments belonging to these users. Defaults to all users.
            states: Only return experiments that are in these states.
            project_id: Only return experiments associated with this project ID.

        Returns:
            A list of experiments.
        """

        def get_with_offset(offset: int) -> bindings.v1GetExperimentsResponse:
            return bindings.get_GetExperiments(
                session=self._session,
                sortBy=sort_by and sort_by._to_bindings() or None,
                orderBy=order_by and order_by._to_bindings() or None,
                archived=None,
                description=None,
                labels=labels,
                experimentIdFilter_incl=experiment_ids,
                offset=offset,
                limit=None,
                name=name,
                states=[state._to_bindings() for state in states] if states else None,
                users=users,
                projectId=project_id,
            )

        bindings_exps: Iterable[bindings.v1Experiment] = itertools.chain.from_iterable(
            r.experiments for r in api.read_paginated(get_with_offset)
        )
        return [experiment.Experiment._from_bindings(b, self._session) for b in bindings_exps]

    def get_trial(self, trial_id: int) -> trial.Trial:
        """
        Get the :class:`~determined.experimental.Trial` representing the
        trial with the provided trial ID.
        """
        resp = bindings.get_GetTrial(session=self._session, trialId=trial_id)
        return trial.Trial._from_bindings(resp.trial, self._session)

    def get_checkpoint(self, uuid: str) -> checkpoint.Checkpoint:
        """
        Get the :class:`~determined.experimental.Checkpoint` representing the
        checkpoint with the provided UUID.
        """
        resp = bindings.get_GetCheckpoint(self._session, checkpointUuid=uuid)
        return checkpoint.Checkpoint._from_bindings(resp.checkpoint, self._session)

    def get_workspace(self, name: str) -> workspace.Workspace:
        resp = bindings.get_GetWorkspaces(self._session, name=name)
        if len(resp.workspaces) == 0:
            raise errors.NotFoundException(f"Workspace {name} not found.")
        assert len(resp.workspaces) == 1, f"Multiple workspaces found with name {name}"
        return workspace.Workspace._from_bindings(resp.workspaces[0], self._session)

    def list_workspaces(self) -> List[workspace.Workspace]:
        def get_with_offset(offset: int) -> bindings.v1GetWorkspacesResponse:
            return bindings.get_GetWorkspaces(self._session, offset=offset)

        iter_workspaces = itertools.chain.from_iterable(
            r.workspaces for r in api.read_paginated(get_with_offset)
        )
        return [workspace.Workspace._from_bindings(w, self._session) for w in iter_workspaces]

    def create_workspace(self, name: str) -> workspace.Workspace:
        """Create a new workspace with the provided name.

        Args:
            name: The name of the workspace to create.

        Returns:
            The newly-created :class:`~determined.experimental.Workspace`.

        Raises:
            errors.APIException: If a workspace with the provided name already exists.
        """
        req = bindings.v1PostWorkspaceRequest(name=name)
        resp = bindings.post_PostWorkspace(self._session, body=req)
        return workspace.Workspace._from_bindings(resp.workspace, self._session)

    def delete_workspace(self, name: str) -> None:
        """Delete the workspace with the provided name.

        Args:
            name: The name of the workspace to delete.

        Raises:
            errors.NotFoundException: If no workspace with the provided name exists.
        """
        workspace_id = self.get_workspace(name).id
        bindings.delete_DeleteWorkspace(self._session, id=workspace_id)

    def create_model(
        self,
        name: str,
        description: Optional[str] = "",
        metadata: Optional[Dict[str, Any]] = None,
        labels: Optional[List[str]] = None,
        workspace_name: Optional[str] = None,
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
            name=name,
            description=description,
            labels=labels,
            metadata=metadata,
            notes=None,
            workspaceName=workspace_name,
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
            stacklevel=2,
        )
        return self.get_model(model_id)

    def get_models(
        self,
        sort_by: model.ModelSortBy = model.ModelSortBy.NAME,
        order_by: OrderBy = OrderBy.ASCENDING,
        name: Optional[str] = None,
        description: Optional[str] = None,
        model_id: Optional[int] = None,
        workspace_names: Optional[List[str]] = None,
        workspace_ids: Optional[List[int]] = None,
    ) -> List[model.Model]:
        warnings.warn(
            "Determined.get_models() has been deprecated and will be removed in a future version."
            "Please call Determined.list_models() instead.",
            FutureWarning,
            stacklevel=2,
        )
        return list(
            self.list_models(
                sort_by=sort_by,
                order_by=order_by,
                name=name,
                description=description,
                model_id=model_id,
                workspace_names=workspace_names,
                workspace_ids=workspace_ids,
            )
        )

    def list_models(
        self,
        sort_by: model.ModelSortBy = model.ModelSortBy.NAME,
        order_by: OrderBy = OrderBy.ASCENDING,
        name: Optional[str] = None,
        description: Optional[str] = None,
        model_id: Optional[int] = None,
        workspace_names: Optional[List[str]] = None,
        workspace_ids: Optional[List[int]] = None,
    ) -> List[model.Model]:
        """
        Get a list of all models in the model registry.

        Arguments:
            sort_by: Which field to sort by. See :class:`~determined.experimental.ModelSortBy`.
            order_by: Whether to sort in ascending or descending order. See
                :class:`~determined.experimental.OrderBy`.
            name: If this parameter is set, models will be filtered to only
                include models with names matching this parameter.
            description: If this parameter is set, models will be filtered to
                only include models with descriptions matching this parameter.
            model_id: If this parameter is set, models will be filtered to
                only include the model with this unique numeric id.
            workspace_names: Only return models with names in this list.
            workspace_ids: Only return models with workspace IDs in this list.

        Returns:
            A list of models.
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
                limit=None,
                userIds=None,
                users=None,
                workspaceNames=workspace_names,
                workspaceIds=workspace_ids,
            )

        bindings_models: Iterable[bindings.v1Model] = itertools.chain.from_iterable(
            r.models for r in api.read_paginated(get_with_offset)
        )

        return [model.Model._from_bindings(m, self._session) for m in bindings_models]

    def get_model_labels(self) -> List[str]:
        """
        Get a list of labels used on any models, sorted from most-popular to least-popular.
        """
        return list(bindings.get_GetModelLabels(self._session).labels)

    def list_oauth_clients(self) -> Sequence[oauth2_scim_client.Oauth2ScimClient]:
        try:
            oauth2_scim_clients: List[oauth2_scim_client.Oauth2ScimClient] = []
            clients = self._session.get("oauth2/clients").json()
            for client in clients:
                osc: oauth2_scim_client.Oauth2ScimClient = oauth2_scim_client.Oauth2ScimClient(
                    name=client["name"], client_id=client["id"], domain=client["domain"]
                )
                oauth2_scim_clients.append(osc)
            return oauth2_scim_clients
        except api.errors.NotFoundException:
            raise det.errors.EnterpriseOnlyError("API not found: oauth2/clients")

    def add_oauth_client(self, domain: str, name: str) -> oauth2_scim_client.Oauth2ScimClient:
        try:
            client = self._session.post(
                "oauth2/clients", json={"domain": domain, "name": name}
            ).json()
            return oauth2_scim_client.Oauth2ScimClient(
                client_id=str(client["id"]), secret=str(client["secret"]), domain=domain, name=name
            )

        except api.errors.NotFoundException:
            raise det.errors.EnterpriseOnlyError("API not found: oauth2/clients")

    def remove_oauth_client(self, client_id: str) -> None:
        try:
            self._session.delete(f"oauth2/clients/{client_id}")
        except api.errors.NotFoundException:
            raise det.errors.EnterpriseOnlyError("API not found: oauth2/clients")

    def stream_trials_metrics(
        self, trial_ids: List[int], group: str
    ) -> Iterable[metrics.TrialMetrics]:
        warnings.warn(
            "Determined.stream_training_metrics is deprecated."
            "Use Determined.iter_trials_metrics instead",
            FutureWarning,
            stacklevel=2,
        )
        return self.iter_trials_metrics(trial_ids=trial_ids, group=group)

    def iter_trials_metrics(
        self, trial_ids: List[int], group: str
    ) -> Iterable[metrics.TrialMetrics]:
        """Generate an iterator of metrics for the passed trials.

        This function opens up a persistent connection to the Determined master to receive trial
        metrics. For as long as the connection remains open, the generator it returns yields the
        TrialMetrics it receives.

        Arguments:
            trial_ids: The trial IDs to iterate over metrics for.
            group: The metric group to iterate over.  Common values are "validation" and "training",
                but group can be any value passed to master when reporting metrics during training
                (usually via a context's `report_metrics`).

        Returns:
            An iterable of :class:`~determined.experimental.TrialMetrics` objects.
        """
        return trial._stream_trials_metrics(self._session, trial_ids, group=group)

    def stream_trials_training_metrics(
        self, trial_ids: List[int]
    ) -> Iterable[metrics.TrainingMetrics]:
        """Streams training metrics for this trial.

        DEPRECATED: Use iter_trials_metrics instead with `group` set to "training"
        """
        warnings.warn(
            "Determined.stream_trials_training_metrics is deprecated."
            "Use Determined.iter_trials_metrics instead with `group` set to 'training'",
            FutureWarning,
            stacklevel=2,
        )
        return trial._stream_training_metrics(self._session, trial_ids)

    def stream_trials_validation_metrics(
        self, trial_ids: List[int]
    ) -> Iterable[metrics.ValidationMetrics]:
        """Streams validation metrics for this trial.

        DEPRECATED: Use iter_trials_metrics instead with `group` set to "validation"
        """
        warnings.warn(
            "Determined.stream_trials_validation_metrics is deprecated."
            "Use Determined.iter_trials_metrics instead with `group` set to 'validation'",
            FutureWarning,
            stacklevel=2,
        )
        return trial._stream_validation_metrics(self._session, trial_ids)
