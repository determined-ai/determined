import datetime
from typing import Callable, Union
from unittest import mock

import pytest
import responses

from determined.common import api
from determined.common.experimental import checkpoint, trial
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
    trial_logs = api_responses.sample_trial_logs()
    trial_logs_url = f"{_MASTER}/api/v1/trials/{trialref.id}/logs"

    before_ts = 1689285229
    after_ts = 1689280229

    timestamp_before = datetime.datetime.fromtimestamp(before_ts).isoformat("T") + "Z"
    timestamp_after = datetime.datetime.fromtimestamp(after_ts).isoformat("T") + "Z"

    responses.get(trial_logs_url, stream=True, body=trial_logs)

    list(trialref.logs(timestamp_before=before_ts, timestamp_after=after_ts))

    call = responses.calls[0]
    assert call.request.params["timestampBefore"] == timestamp_before
    assert call.request.params["timestampAfter"] == timestamp_after


@pytest.mark.parametrize(
    "name,timestamp",
    [
        ("invalid_timezone", "2021-10-26T23:17:12+04:00"),
        ("missing_T", "2021-10-26 23:17:12Z"),
        ("missing_Z", "2021-10-26T23:17:12"),
    ],
)
@responses.activate
def test_trial_logs_invalid_timestamps(
    make_trialref: Callable[[int], trial.Trial], name: str, timestamp: str
) -> None:
    trialref = make_trialref(1)

    with pytest.raises(ValueError):
        list(trialref.logs(timestamp_before=timestamp, timestamp_after=None))
        list(trialref.logs(timestamp_before=None, timestamp_after=timestamp))


@pytest.mark.parametrize(
    "name,valid_timestamp",
    [
        ("no_microseconds", "2021-10-26T23:17:12Z"),
        ("microseconds", "2021-10-26T23:17:12.0000Z"),
    ],
)
@responses.activate
def test_trial_logs_accepts_valid_timestamps(
    make_trialref: Callable[[int], trial.Trial], name: str, valid_timestamp: str
) -> None:
    trialref = make_trialref(1)
    trial_logs = api_responses.sample_trial_logs()
    trial_logs_url = f"{_MASTER}/api/v1/trials/{trialref.id}/logs"

    responses.get(
        trial_logs_url,
        stream=True,
        body=trial_logs,
    )
    list(trialref.logs(timestamp_before=valid_timestamp, timestamp_after=None))

    responses.get(
        trial_logs_url,
        stream=True,
        body=trial_logs,
    )

    list(trialref.logs(timestamp_before=None, timestamp_after=valid_timestamp))


@mock.patch("determined.common.api.bindings.get_GetTrialCheckpoints")
def test_list_checkpoints_calls_bindings_sortByMetric_with_sort_by_str(
    mock_bindings: mock.MagicMock,
    make_trialref: Callable[[int], trial.Trial],
) -> None:
    trialref = make_trialref(1)
    ckpt_resp = api_responses.sample_get_trial_checkpoints()
    mock_bindings.side_effect = [ckpt_resp]

    sort_by_metric = "val_metric"
    trialref.list_checkpoints(sort_by=sort_by_metric, order_by=checkpoint.CheckpointOrderBy.ASC)

    _, call_kwargs = mock_bindings.call_args_list[0]

    assert call_kwargs["sortByMetric"] == sort_by_metric


@mock.patch("determined.common.api.bindings.get_GetTrialCheckpoints")
def test_list_checkpoints_calls_bindings_sortByAttr_with_sort_by_attr(
    mock_bindings: mock.MagicMock,
    make_trialref: Callable[[int], trial.Trial],
) -> None:
    trialref = make_trialref(1)
    ckpt_resp = api_responses.sample_get_trial_checkpoints()
    mock_bindings.side_effect = [ckpt_resp]

    sort_by_attr = checkpoint.CheckpointSortBy.SEARCHER_METRIC
    trialref.list_checkpoints(sort_by=sort_by_attr, order_by=checkpoint.CheckpointOrderBy.ASC)

    _, call_kwargs = mock_bindings.call_args_list[0]

    assert call_kwargs["sortByAttr"] == sort_by_attr._to_bindings()
