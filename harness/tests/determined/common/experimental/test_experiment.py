import base64
import math
import os
from typing import Callable
from unittest import mock

import pytest
import requests
import responses

from determined.common import api
from determined.common.api import bindings
from determined.common.experimental import checkpoint, determined, experiment
from tests.fixtures import api_responses

_MASTER = "http://localhost:8080"


@pytest.fixture
def standard_session() -> api.Session:
    return api.Session(master=_MASTER, user=None, auth=None, cert=None)


@pytest.fixture
def make_expref(standard_session: api.Session) -> Callable[[int], experiment.Experiment]:
    def _make_expref(exp_id: int) -> experiment.Experiment:
        return experiment.Experiment(exp_id, standard_session)

    return _make_expref


@responses.activate
def test_await_waits_for_first_trial_to_start(
    make_expref: Callable[[int], experiment.Experiment]
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
    make_expref: Callable[[int], experiment.Experiment],
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

    # Register an extra response so the mock can keep serving the experiment
    #   (necessary for the `expref.reload() call below)
    responses.get(f"{_MASTER}/api/v1/experiments/{expref.id}", json=exp_resp_terminal.to_json())
    expref.reload()
    assert expref.state == terminal_state


@responses.activate
def test_wait_raises_exception_when_experiment_is_paused(
    make_expref: Callable[[int], experiment.Experiment]
) -> None:
    expref = make_expref(1)

    exp_resp = api_responses.sample_get_experiment(
        id=expref.id, state=bindings.experimentv1State.PAUSED
    )

    responses.get(f"{_MASTER}/api/v1/experiments/{expref.id}", json=exp_resp.to_json())

    with pytest.raises(ValueError):
        expref.wait()


@responses.activate
def test_list_trials_iterates_through_all_trials(
    make_expref: Callable[[int], experiment.Experiment]
) -> None:
    expref = make_expref(1)
    page_size = 2

    tr_resp = api_responses.sample_get_experiment_trials()

    assert len(tr_resp.trials) >= 2, "Test expects sample trial response to contain >= 2 Trials."
    for trial in tr_resp.trials:
        trial.experimentId = expref.id

    responses.add_callback(
        responses.GET,
        f"{_MASTER}/api/v1/experiments/{expref.id}/trials",
        callback=api_responses.serve_by_page(tr_resp, "trials", max_page_size=page_size),
    )

    trials = expref.list_trials(limit=page_size)

    assert len(list(trials)) == len(tr_resp.trials)


@responses.activate
def test_list_trials_requests_pages_lazily(
    make_expref: Callable[[int], experiment.Experiment]
) -> None:
    expref = make_expref(1)
    page_size = 2

    tr_resp = api_responses.sample_get_experiment_trials()

    assert len(tr_resp.trials) >= 2, "Test expects sample trial response to contain >= 2 Trials."
    for trial in tr_resp.trials:
        trial.experimentId = expref.id

    responses.add_callback(
        responses.GET,
        f"{_MASTER}/api/v1/experiments/{expref.id}/trials",
        callback=api_responses.serve_by_page(tr_resp, "trials", max_page_size=page_size),
    )

    trials = expref.list_trials(limit=page_size)

    # Iterate through each item in generator and ensure API is called to fetch new pages.
    for i, _ in enumerate(trials):
        page_num = math.ceil((i + 1) / page_size)
        assert len(responses.calls) == page_num
    total_pages = math.ceil((len(tr_resp.trials)) / page_size)
    assert len(responses.calls) == total_pages


@pytest.mark.parametrize(
    "attr_name,attr_value",
    [
        ("name", "test_name"),
        ("description", "test description"),
        ("notes", "test notes"),
    ],
)
@mock.patch("determined.common.api.bindings.patch_PatchExperiment")
def test_experiment_sets_attributes(
    mock_bindings: mock.MagicMock,
    make_expref: Callable[[int], experiment.Experiment],
    attr_name: str,
    attr_value: str,
) -> None:
    expref = make_expref(1)

    # Call associated set_ method for attribute.
    attr_setter = getattr(expref, f"set_{attr_name}")
    attr_setter(attr_value)
    _, kwargs = mock_bindings.call_args
    assert getattr(kwargs["body"], attr_name) == attr_value


@mock.patch("determined.common.api.bindings.post_ArchiveExperiment")
def test_archive_doesnt_update_local_on_rest_failure(
    mock_bindings: mock.MagicMock,
    make_expref: Callable[[int], experiment.Experiment],
) -> None:
    expref = make_expref(1)

    mock_bindings.side_effect = bindings.APIHttpError("post_ArchiveExperiment", requests.Response())

    assert expref.archived is None
    try:
        expref.archive()
        raise AssertionError("bindings API call should raise an exception")
    except bindings.APIHttpError:
        assert expref.archived is None


@mock.patch("determined.common.api.bindings.post_UnarchiveExperiment")
def test_unarchive_doesnt_update_local_on_rest_failure(
    mock_bindings: mock.MagicMock,
    make_expref: Callable[[int], experiment.Experiment],
) -> None:
    expref = make_expref(1)

    mock_bindings.side_effect = bindings.APIHttpError(
        "post_UnarchiveExperiment", requests.Response()
    )

    assert expref.archived is None
    try:
        expref.unarchive()
        raise AssertionError("bindings API call should raise an exception")
    except bindings.APIHttpError:
        assert expref.archived is None


@mock.patch("determined.common.api.bindings.get_GetModelDef")
def test_download_code_writes_output_to_file(
    mock_bindings: mock.MagicMock,
    make_expref: Callable[[int], experiment.Experiment],
    tmp_path: os.PathLike,
) -> None:
    expref = make_expref(1)
    # Encode sample response to base64, decode bytes to string.
    sample_tgz_content = base64.b64encode(b"b64TgzResponse").decode()
    mock_bindings.return_value = bindings.v1GetModelDefResponse(b64Tgz=sample_tgz_content)

    output_file = expref.download_code(output_dir=str(tmp_path))

    with open(output_file, "rb") as f:
        file_content = f.read()
        assert file_content == b"b64TgzResponse"


@mock.patch("determined.common.api.bindings.get_GetExperimentCheckpoints")
def test_list_checkpoints_calls_bindings_sortByMetric_with_sort_by_str(
    mock_bindings: mock.MagicMock,
    make_expref: Callable[[int], experiment.Experiment],
) -> None:
    expref = make_expref(1)
    ckpt_resp = api_responses.sample_get_experiment_checkpoints()
    mock_bindings.side_effect = [ckpt_resp]

    sort_by_metric = "val_metric"
    expref.list_checkpoints(sort_by=sort_by_metric, order_by=determined.OrderBy.ASC)

    _, call_kwargs = mock_bindings.call_args_list[0]

    assert call_kwargs["sortByMetric"] == sort_by_metric


@mock.patch("determined.common.api.bindings.get_GetExperimentCheckpoints")
def test_list_checkpoints_calls_bindings_sortByAttr_with_sort_by_attr(
    mock_bindings: mock.MagicMock,
    make_expref: Callable[[int], experiment.Experiment],
) -> None:
    expref = make_expref(1)
    ckpt_resp = api_responses.sample_get_experiment_checkpoints()
    mock_bindings.side_effect = [ckpt_resp]

    sort_by_attr = checkpoint.CheckpointSortBy.SEARCHER_METRIC
    expref.list_checkpoints(sort_by=sort_by_attr, order_by=determined.OrderBy.ASC)

    _, call_kwargs = mock_bindings.call_args_list[0]

    assert call_kwargs["sortByAttr"] == sort_by_attr._to_bindings()


def test_list_checkpoints_errors_on_only_order_by_set(
    make_expref: Callable[[int], experiment.Experiment],
) -> None:
    expref = make_expref(1)

    with pytest.raises(AssertionError):
        expref.list_checkpoints(sort_by=None, order_by=determined.OrderBy.ASC, max_results=5)


def test_list_checkpoints_errors_on_only_sort_by_set(
    make_expref: Callable[[int], experiment.Experiment],
) -> None:
    expref = make_expref(1)

    with pytest.raises(AssertionError):
        expref.list_checkpoints(
            sort_by=checkpoint.CheckpointSortBy.UUID, order_by=None, max_results=5
        )
