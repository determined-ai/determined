import datetime
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
        return trial.Trial(trial_id, standard_session)

    return _make_trialref


@responses.activate
def test_trial_logs_converts_epoch_time(make_trialref: Callable[[int], trial.Trial]) -> None:
    trialref = make_trialref(1)
    trial_logs = api_responses.sample_get_trial_logs()
    trial_logs_url = f"{_MASTER}/api/v1/trials/{trialref.id}/logs"

    before_ts = 1689285229
    after_ts = 1689280229

    timestamp_before = datetime.datetime.fromtimestamp(before_ts).isoformat("T") + "Z"
    timestamp_after = datetime.datetime.fromtimestamp(after_ts).isoformat("T") + "Z"
    params = {
        "timestampBefore": timestamp_before,
        "timestampAfter": timestamp_after,
    }
    responses.get(
        trial_logs_url,
        stream=True,
        body=trial_logs,
        match=[responses.matchers.query_param_matcher(params, strict_match=False)],
    )

    resp = trialref.logs(timestamp_before=before_ts, timestamp_after=after_ts)

    assert len(list(resp)) > 1
    assert len(responses.calls) == 1

    call = responses.calls[0]
    assert call.request.params["timestampBefore"] == params["timestampBefore"]
    assert call.request.params["timestampAfter"] == params["timestampAfter"]


@pytest.mark.parametrize(
    "timestamp",
    ["2021-10-26T23:17:12+04:00", "2021-10-26 23:17:12Z", "2021-10-26T23:17:12"],
)
@responses.activate
def test_trial_logs_invalid_timestamps(
    make_trialref: Callable[[int], trial.Trial], timestamp: str
) -> None:
    trialref = make_trialref(1)

    with pytest.raises(ValueError):
        list(trialref.logs(timestamp_before=timestamp, timestamp_after=timestamp))
