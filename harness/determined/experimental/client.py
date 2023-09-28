"""
The ``client`` module exposes many of the same capabilities as the ``det`` CLI tool directly to
Python code with an object-oriented interface.

As a simple example, let's walk through the most basic workflow for creating an experiment,
waiting for it to complete, and finding the top-performing checkpoint.

The first step is to import the ``client`` module and possibly to call
:func:`~determined.experimental.client.login`:

.. code:: python

   from determined.experimental import client

   # If you have called `det user login`, environment variables will have been set such that
   # logging in with `login` is unnecessary:
   # client.login(master=..., user=..., password=...)

The next step is to call :func:`~determined.experimental.client.create_experiment`:

.. code:: python

   # config can be a path to a config file or a python dict of the config.
   exp = client.create_experiment(config="my_config.yaml", model_dir=".")
   print(f"started experiment {exp.id}")

The returned object will be an :class:`~determined.experimental.client.Experiment`
which has methods for controlling the lifetime of the experiment running on the cluster.
In this example, we will just wait for the experiment to complete.

.. code:: python

   exit_status = exp.wait()
   print(f"experiment completed with status {exit_status}")

Now that the experiment has completed, you can grab the top-performing checkpoint from training:

.. code:: python

   best_checkpoint = exp.list_checkpoints()[0]
   print(f"best checkpoint was {best_checkpoint.uuid}")


See :ref:`use-trained-models` for more ideas on what to do next.
"""

import functools
import logging
import pathlib
import warnings
from typing import Any, Callable, Dict, Iterable, List, Optional, Sequence, Union

from determined.common.api import Session  # noqa: F401
from determined.common.experimental.checkpoint import (  # noqa: F401
    Checkpoint,
    CheckpointOrderBy,
    CheckpointSortBy,
    CheckpointState,
    DownloadMode,
)
from determined.common.experimental.determined import Determined
from determined.common.experimental.experiment import Experiment, ExperimentState  # noqa: F401
from determined.common.experimental.metrics import TrainingMetrics, TrialMetrics, ValidationMetrics
from determined.common.experimental.model import Model, ModelOrderBy, ModelSortBy
from determined.common.experimental.oauth2_scim_client import Oauth2ScimClient
from determined.common.experimental.trial import Trial, TrialOrderBy, TrialSortBy  # noqa: F401
from determined.common.experimental.user import User
from determined.common.experimental.workspace import Workspace

_determined = None  # type: Optional[Determined]


def _require_singleton(fn: Callable) -> Callable:
    @functools.wraps(fn)
    def _fn(*args: Any, **kwargs: Any) -> Any:
        global _determined
        if _determined is None:
            _determined = Determined()
        return fn(*args, **kwargs)

    return _fn


def login(
    master: Optional[str] = None,
    user: Optional[str] = None,
    password: Optional[str] = None,
    cert_path: Optional[str] = None,
    cert_name: Optional[str] = None,
    noverify: bool = False,
) -> None:
    """
    ``login()`` will configure the default Determined() singleton used by all of the other functions
    in the client module.

    It is often unnecessary to call ``login()``.  If you have configured your environment so that
    the Determined CLI works without any extra arguments or environment variables, you should not
    have to call ``login()`` at all.

    If you do need to call ``login()``, it must be called before any calling any other functions
    from this module, otherwise it will fail.

    If you have reason to connect to multiple masters, you should use explicit
    :class:`~determined.experimental.client.Determined` objects instead.  Each explicit
    :class:`~determined.experimental.client.Determined` object accepts the same parameters as
    ``login()``, and offers the same functions as what are offered in this module.

    .. note::

       Try to avoid having your password in your python code.  If you are running on your local
       machine, you should always be able to use ``det user login`` on the CLI, and ``login()``
       will not need either a user or a password.  If you have ran ``det user login`` with multiple
       users (and you have not ran ``det user logout``), then you should be able to run
       ``login(user=...)`` for any of those users without putting your password in your code.

    Args:
        master (string, optional): The URL of the Determined master.
            If this argument is not specified, the environment variables
            DET_MASTER and DET_MASTER_ADDR will be checked for the master URL in that order.
        user (string, optional): The Determined username used for
            authentication. (default: ``determined``)
        password (string, optional): The password associated with the user.
        cert_path (string, optional): A path to a custom PEM-encoded certificate,
            against which to validate the master.  (default: ``None``)
        cert_name (string, optional): The name of the master hostname to use during certificate
            validation. Normally this is taken from the master URL, but there may be cases where
            the master is exposed on multiple networks that this value might need to be
            overridden. (default: ``None``)
        noverify (boolean, optional): disable all TLS verification entirely.  (default: ``False``)
    """
    global _determined

    if _determined is not None:
        raise ValueError(
            "You can only call login() once, before any other calls to any functions in the client "
            "module.  If you have reason to connect to multiple masters, you should use explicit "
            "client.Determined() objects, which each expose the same functions as this module."
        )

    _determined = Determined(master, user, password, cert_path, cert_name, noverify)


@_require_singleton
def create_experiment(
    config: Union[str, pathlib.Path, Dict],
    model_dir: str,
    includes: Optional[Iterable[Union[str, pathlib.Path]]] = None,
) -> Experiment:
    """Create an experiment with config parameters and model directory.

    Args:
        config: Experiment config filename (.yaml) or a dict.
        model_dir: Directory containing model definition.
        iterables: Additional files or directories to include in the model definition.

    Returns:
        An :class:`~determined.experimental.client.Experiment` of the created experiment.
    """
    assert _determined is not None
    return _determined.create_experiment(config, model_dir, includes)


@_require_singleton
def get_experiment(experiment_id: int) -> Experiment:
    """Get the Experiment representing the experiment with the provided experiment ID.

    Args:
        experiment_id (int): The experiment ID.

    Returns:
        The fetched :class:`~determined.experimental.client.Experiment`.
    """
    assert _determined is not None
    return _determined.get_experiment(experiment_id)


@_require_singleton
def create_user(
    username: str, admin: bool, password: Optional[str] = None, remote: bool = False
) -> User:
    """Create an user with username and password, admin.

    Arg:
        username: username of the user.
        password: password of the user.
        admin: indicates whether the user is an admin.

    Returns:
        A :class:`~determined.experimental.client.User` of the created user.
    """
    assert _determined is not None
    return _determined.create_user(username, admin, password, remote)


@_require_singleton
def get_user_by_id(user_id: int) -> User:
    """Get the User with the provided user id."""
    assert _determined is not None
    return _determined.get_user_by_id(user_id)


@_require_singleton
def get_user_by_name(user_name: str) -> User:
    """Get the User with the provided username."""
    assert _determined is not None
    return _determined.get_user_by_name(user_name)


@_require_singleton
def get_session_username() -> str:
    """Get the username of the currently signed in user."""
    assert _determined is not None
    return _determined.get_session_username()


@_require_singleton
def whoami() -> User:
    """Get the current User."""
    assert _determined is not None
    return _determined.whoami()


# DOES NOT REQUIRE SINGLETON (don't force a login in order to log out).
def logout() -> None:
    """Log out of the current session."""
    if _determined is not None:
        return _determined.logout()

    logging.warning(
        "client has not been logged in, either explicitly by client.login() or implicitly by any "
        "other client.* function, so client.logout() has no session to log out of and is a no-op. "
        "If you would like to log out of the default active session, try "
        "client.Determined().logout() instead."
    )


@_require_singleton
def list_users() -> List[User]:
    """Get a list of all Users."""
    assert _determined is not None
    return _determined.list_users()


@_require_singleton
def get_trial(trial_id: int) -> Trial:
    """Get the Trial representing the trial with the provided ID.

    Arg:
        trial_id: The trial ID.

    Returns:
        The fetched :class:`~determined.experimental.client.Trial`.
    """
    assert _determined is not None
    return _determined.get_trial(trial_id)


@_require_singleton
def get_checkpoint(uuid: str) -> Checkpoint:
    """Get the Checkpoint representing with the provided UUID.

    Args:
        uuid: The checkpoint UUID.

    Returns:
        The fetched :class:`~determined.experimental.client.Checkpoint`.
    """
    assert _determined is not None
    return _determined.get_checkpoint(uuid)


@_require_singleton
def get_workspace(name: str) -> Workspace:
    """Get the Workspace with the provided name.

    Args:
        name: The workspace name.

    Returns:
        The fetched :class:`~determined.experimental.client.Workspace`.
    """
    assert _determined is not None
    return _determined.get_workspace(name)


@_require_singleton
def create_model(
    name: str, description: Optional[str] = "", metadata: Optional[Dict[str, Any]] = None
) -> Model:
    """Add a model to the model registry.

    Args:
        name: The name of the model. This name must be unique.
        description: A description of the model.
        metadata: Dictionary of metadata to add to the model.

    Returns:
        A :class:`~determined.experimental.client.Model` of the created model.
    """
    assert _determined is not None
    return _determined.create_model(name, description, metadata)


@_require_singleton
def get_model(identifier: Union[str, int]) -> Model:
    """Get the model from the model registry with the provided numeric id.

    If no model with that name is found in the registry, an exception is raised.

    Args:
        identifier: The unique name or numeric ID of the model.

    Returns:
        The fetched :class:`~determined.experimental.client.Model`.
    """
    assert _determined is not None
    return _determined.get_model(identifier)


@_require_singleton
def get_model_by_id(model_id: int) -> Model:
    """Get the model from the model registry with the provided numeric id.

    If no model with that id is found in the registry, an exception is raised.

    Args:
        model_id: The unique id of the model.

    Returns:
        The fetched :class:`~determined.experimental.client.Model`.

    .. warning::
       client.get_model_by_id() has been deprecated and will be removed
       in a future version.
       Please call client.get_model() with either a string-type name or
       an integer-type model ID.
    """
    warnings.warn(
        "client.get_model_by_id() has been deprecated and will be removed "
        "in a future version.\n"
        "Please call client.get_model() with either a string-type name or "
        "an integer-type model ID.",
        FutureWarning,
        stacklevel=2,
    )
    assert _determined is not None
    return _determined.get_model(model_id)


@_require_singleton
def get_models(
    sort_by: ModelSortBy = ModelSortBy.NAME,
    order_by: ModelOrderBy = ModelOrderBy.ASCENDING,
    name: str = "",
    description: str = "",
) -> List[Model]:
    """Get a list of all models in the model registry.

    Args:
        sort_by: Which field to sort by. See :class:`~determined.experimental.client.ModelSortBy`.
        order_by: Whether to sort in ascending or descending order. See
            :class:`~determined.experimental.client.ModelOrderBy`.
        name: If this parameter is set, models will be filtered to only
            include models with names matching this parameter.
        description: If this parameter is set, models will be filtered to
            only include models with descriptions matching this parameter.

    Returns:
        A list of :class:`~determined.experimental.client.Model` objects matching any passed
        filters.
    """
    assert _determined is not None
    return _determined.get_models(sort_by, order_by, name, description)


@_require_singleton
def get_model_labels() -> List[str]:
    """Get a list of labels used on any models in the model registry.

    Returns:
        A list of model labels sorted from most-popular to least-popular.
    """
    assert _determined is not None
    return _determined.get_model_labels()


@_require_singleton
def list_oauth_clients() -> Sequence[Oauth2ScimClient]:
    """Get a list of Oauth2 Scim clients."""
    assert _determined is not None
    return _determined.list_oauth_clients()


@_require_singleton
def add_oauth_client(domain: str, name: str) -> Oauth2ScimClient:
    """Add an oauth client.

    Args:
        domain: Domain of OAuth client.
        name: Name of OAuth client.

    Returns:
        A :class:`~determined.experimental.client.Oauth2ScimClient` of the created client.
    """
    assert _determined is not None
    return _determined.add_oauth_client(domain, name)


@_require_singleton
def remove_oauth_client(client_id: str) -> None:
    """Remove an oauth client."""
    assert _determined is not None
    return _determined.remove_oauth_client(client_id)


@_require_singleton
def stream_trials_metrics(trial_ids: List[int], group: str) -> Iterable[TrialMetrics]:
    """Iterate over the metrics for one or more trials.

    This function collects TrialMetrics from a trial, sorted by `trial_id`, `trial_run_id`
    and `steps_completed`.

    .. warning::
       Contrary to its name, no streaming is actually done in this function. As more metrics are
       computed on the master, they will not be appended to the iterator this function returns.

    Args:
        trial_ids: List of trial IDs to get metrics for.
        group: The metrics group to stream. Must either "training" or "validation".

    Returns:
        An iterable of :class:`~determined.experimental.client.TrialMetrics` objects.
    """
    assert _determined is not None
    return _determined.stream_trials_metrics(trial_ids, group=group)


@_require_singleton
def stream_trials_training_metrics(trial_ids: List[int]) -> Iterable[TrainingMetrics]:
    """Iterate over training metrics for one or more trials.

    Args:
        trial_ids: List of trial IDs to get metrics for.

    .. warning::
       client.stream_trials_training_metrics() has been deprecated and will be removed
       in a future version.
       Please call client.stream_trials_metrics() with `group` set to "training".
    """
    assert _determined is not None
    return _determined.stream_trials_training_metrics(trial_ids)


@_require_singleton
def stream_trials_validation_metrics(trial_ids: List[int]) -> Iterable[ValidationMetrics]:
    """Iterate over validation metrics for one or more trials.

    Args:
        trial_ids: List of trial IDs to get metrics for.

    .. warning::
       client.stream_trials_validation_metrics() has been deprecated and will be removed
       in a future version.
       Please call client.stream_trials_metrics() with `group` set to "validation".
    """
    assert _determined is not None
    return _determined.stream_trials_validation_metrics(trial_ids)


@_require_singleton
def _get_singleton_session() -> Session:
    assert _determined is not None
    return _determined._session
