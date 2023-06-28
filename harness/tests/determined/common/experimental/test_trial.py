from typing import Callable

import pytest
import responses

from determined.common import api
from determined.common.experimental import trial
from tests.fixtures import api_responses

_MASTER = "http://localhost:8080"


@pytest.fixture
def standard_session() -> api.Session:
    return api.Session(master=_MASTER, user=None, auth=None, cert=None)


@pytest.fixture
def make_trialref(standard_session: api.Session) -> Callable[[int], trial.Trial]:
    def _make_trialref(trial_id: int) -> trial.Trial:
        return trial.Trial(trial_id=trial_id, session=standard_session)

    return _make_trialref


@responses.activate
def test_trial_includes_summary_metrics(make_trialref: Callable[[int], trial.Trial]) -> None:
    trial_ref = make_trialref(1)

    tr_resp = api_responses.sample_get_trial()

    trial_url = f"{_MASTER}/api/v1/trials/{trial_ref.id}"
    responses.get(trial_url, json=tr_resp.to_json())

    trial_ref.reload()

    responses.assert_call_count(trial_url, 1)
    assert trial_ref.summary
    assert trial_ref.summary["avg_metrics"]
    assert trial_ref.summary["validation_metrics"]
