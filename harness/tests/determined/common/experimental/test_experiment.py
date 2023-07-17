from typing import Callable

import pytest
import responses

from determined.common import api
from determined.common.api import bindings
from determined.common.experimental import experiment
from tests.fixtures import api_responses

_MASTER = "http://localhost:8080"


@pytest.fixture
def standard_session() -> api.Session:
    return api.Session(master=_MASTER, user=None, auth=None, cert=None)


@pytest.fixture
def make_expref(standard_session: api.Session) -> Callable[[int], experiment.ExperimentReference]:
    def _make_expref(exp_id: int) -> experiment.ExperimentReference:
        """Make an experiment reference with the given ID."""
        return experiment.ExperimentReference(exp_id, standard_session)

    return _make_expref


@responses.activate
def test_await_waits_for_first_trial_to_start(
    make_expref: Callable[[int], experiment.ExperimentReference]
) -> None:
    expref = make_expref(1)

    tr_resp = api_responses.sample_get_experiment_trials()
    for trial in tr_resp.trials:
        trial.experimentId = expref.id
    empty_tr_resp = bindings.v1GetExperimentTrialsResponse(
        trials=[], pagination=api_responses.empty_get_pagination()
    )

    responses.get(f"{_MASTER}/api/v1/experiments/{expref.id}/trials", json=empty_tr_resp.to_json())
    responses.get(f"{_MASTER}/api/v1/experiments/{expref.id}/trials", json=empty_tr_resp.to_json())
    responses.get(f"{_MASTER}/api/v1/experiments/{expref.id}/trials", json=tr_resp.to_json())

    expref.await_first_trial(interval=0.01)
    assert len(responses.calls) > 2


@pytest.mark.parametrize(
    "terminal_state",
    [
        bindings.experimentv1State.CANCELED,
        bindings.experimentv1State.COMPLETED,
        bindings.experimentv1State.DELETED,
        bindings.experimentv1State.ERROR,
    ],
)
@responses.activate
def test_wait_waits_until_terminal_state(
    make_expref: Callable[[int], experiment.ExperimentReference],
    terminal_state: bindings.experimentv1State,
) -> None:
    expref = make_expref(1)

    exp_resp_running = api_responses.sample_get_experiment(
        id=expref.id, state=bindings.experimentv1State.RUNNING
    )
    exp_resp_terminal = api_responses.sample_get_experiment(id=expref.id, state=terminal_state)

    responses.get(f"{_MASTER}/api/v1/experiments/{expref.id}", json=exp_resp_running.to_json())
    responses.get(f"{_MASTER}/api/v1/experiments/{expref.id}", json=exp_resp_running.to_json())
    responses.get(f"{_MASTER}/api/v1/experiments/{expref.id}", json=exp_resp_terminal.to_json())

    expref.wait(interval=0.01)
    assert expref._get().state == terminal_state


@responses.activate
def test_wait_raises_exception_when_experiment_is_paused(
    make_expref: Callable[[int], experiment.ExperimentReference]
) -> None:
    expref = make_expref(1)

    exp_resp = api_responses.sample_get_experiment(
        id=expref.id, state=bindings.experimentv1State.PAUSED
    )

    responses.get(f"{_MASTER}/api/v1/experiments/{expref.id}", json=exp_resp.to_json())

    with pytest.raises(ValueError):
        expref.wait()
