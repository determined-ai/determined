import functools
import pathlib
from typing import Any, Callable, Dict, List, Optional, Union

from determined.common.experimental.checkpoint import Checkpoint
from determined.common.experimental.determined import Determined
from determined.common.experimental.experiment import (  # noqa: F401
    ExperimentReference,
    ExperimentState,
)
from determined.common.experimental.model import Model, ModelOrderBy, ModelSortBy
from determined.common.experimental.trial import TrialReference

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
    login will configure the default Determined() singleton used by all of the other functions in
    the client module.

    It is often unnecessary to call login().  If you have configured your environment so that
    the Determined CLI works without any extra arguments or environment variables, you should not
    have to call login at all.

    If you do need to call login(), it must be called before any calling any other functions from
    this module, otherwise it will fail.

    If you have reason to connect to multiple masters, you should use explicit Determined() objects
    instead.  Each explicit Determined() object accepts the same parameters as login(), and offers
    the same functions as what are offered in this module.

    .. note::

       Try to avoid having your password in your python code.  If you are running on your local
       machine, you should always be able to use ``det user login`` on the CLI, and login()
       will not need either a user or a password.  If you have ran ``det user login`` with multiple
       users (and you have not ran ``det user logout``), then you should be able to run
       ``login(user=...)`` for any of those users without putting your password in your code.
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
) -> ExperimentReference:
    """
    Create an experiment with config parameters and model directory. The function
    returns :class:`~determined.experimental.ExperimentReference` of the experiment.

    Arguments:
        config(string, pathlib.Path, dictionary): experiment config filename (.yaml)
            or a dict.
        model_dir(string): directory containing model definition.
    """
    assert _determined is not None
    return _determined.create_experiment(config, model_dir)


@_require_singleton
def get_experiment(experiment_id: int) -> ExperimentReference:
    """
    Get the :class:`~determined.experimental.ExperimentReference` representing the
    experiment with the provided experiment ID.
    """
    assert _determined is not None
    return _determined.get_experiment(experiment_id)


@_require_singleton
def get_trial(trial_id: int) -> TrialReference:
    """
    Get the :class:`~determined.experimental.TrialReference` representing the
    trial with the provided trial ID.
    """
    assert _determined is not None
    return _determined.get_trial(trial_id)


@_require_singleton
def get_checkpoint(uuid: str) -> Checkpoint:
    """
    Get the :class:`~determined.experimental.Checkpoint` representing the
    checkpoint with the provided UUID.
    """
    assert _determined is not None
    return _determined.get_checkpoint(uuid)


@_require_singleton
def create_model(
    name: str, description: Optional[str] = "", metadata: Optional[Dict[str, Any]] = None
) -> Model:
    """
    Add a model to the model registry.

    Arguments:
        name (string): The name of the model. This name must be unique.
        description (string, optional): A description of the model.
        metadata (dict, optional): Dictionary of metadata to add to the model.
    """
    assert _determined is not None
    return _determined.create_model(name, description, metadata)


@_require_singleton
def get_model(name: str) -> Model:
    """
    Get the :class:`~determined.experimental.Model` from the model registry
    with the provided name. If no model with that name is found in the registry,
    an exception is raised.
    """
    assert _determined is not None
    return _determined.get_model(name)


@_require_singleton
def get_models(
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
    assert _determined is not None
    return _determined.get_models(sort_by, order_by, name, description)
