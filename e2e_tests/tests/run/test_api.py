import time

import pytest

from determined.common import api
from determined.common.api import bindings
from tests import api_utils
from tests.experiment import noop


def wait_for_run_state(
    test_session: api.Session,
    run_id: int,
    expected_state: bindings.trialv1State,
    timeout: int = 30,
) -> None:
    deadline = time.time() + timeout
    while time.time() < deadline:
        resp = bindings.post_SearchRuns(
            test_session,
            body=bindings.v1SearchRunsRequest(
                limit=1,
                filter="""
                {"filterGroup": {
                    "children": [
                    {
                        "columnName": "id",
                        "kind": "field",
                        "location": "LOCATION_TYPE_RUN",
                        "operator": "=",
                        "type": "COLUMN_TYPE_NUMBER",
                        "value": %s
                    }
                    ],
                    "conjunction": "and",
                    "kind": "group"
                },
                "showArchived": false
                }
                """
                % run_id,
            ),
        )
        if expected_state == resp.runs[0].state:
            return
        time.sleep(0.1)
    pytest.fail(f"task failed to complete after {timeout} seconds")


@pytest.mark.e2e_cpu
def test_run_kill() -> None:
    sess = api_utils.user_session()
    exp_ref = noop.create_experiment(sess)

    searchResp = bindings.post_SearchRuns(
        sess,
        body=bindings.v1SearchRunsRequest(
            limit=1,
            filter="""{
    "filterGroup": {
        "children": [
        {
            "columnName": "experimentId",
            "kind": "field",
            "location": "LOCATION_TYPE_RUN",
            "operator": "=",
            "type": "COLUMN_TYPE_NUMBER",
            "value": %s
        }
        ],
        "conjunction": "and",
        "kind": "group"
    },
    "showArchived": false
    }"""
            % exp_ref.id,
        ),
    )
    assert searchResp.runs[0].state == bindings.trialv1State.ACTIVE

    run_id = searchResp.runs[0].id
    killResp = bindings.post_KillRuns(
        sess, body=bindings.v1KillRunsRequest(runIds=[run_id], projectId=1)
    )
    assert len(killResp.results) == 1, f"failed to kill run {run_id} from exp {exp_ref.id}"
    assert killResp.results[0].id == run_id
    assert killResp.results[0].error == ""

    # ensure that run is canceled
    wait_for_run_state(sess, run_id, bindings.trialv1State.CANCELED)

    # cancelling an already terminated run should be fine
    killResp = bindings.post_KillRuns(
        sess, body=bindings.v1KillRunsRequest(runIds=[run_id], projectId=1)
    )

    # validate response
    assert (
        len(killResp.results) == 1
    ), f"error when trying to terminate run {run_id} from exp {exp_ref.id} a second time"
    assert killResp.results[0].id == run_id
    assert killResp.results[0].error == ""


@pytest.mark.e2e_cpu
def test_run_kill_filter() -> None:
    sess = api_utils.user_session()
    config = {
        "searcher": {"name": "grid"},
        "hyperparameters": {
            "x": {"type": "categorical", "vals": [1, 2]},
        },
    }
    exp_ref = noop.create_experiment(sess, config=config)

    runFilter = (
        """{
  "filterGroup": {
    "children": [
      {
        "columnName": "experimentId",
        "kind": "field",
        "location": "LOCATION_TYPE_RUN",
        "operator": "=",
        "type": "COLUMN_TYPE_NUMBER",
        "value": %d
      },
      {
        "columnName": "x",
        "kind": "field",
        "location": "LOCATION_TYPE_RUN_HYPERPARAMETERS",
        "operator": ">=",
        "type": "COLUMN_TYPE_NUMBER",
        "value": 2
      }
    ],
    "conjunction": "and",
    "kind": "group"
  },
  "showArchived": false
}"""
        % exp_ref.id
    )

    killResp = bindings.post_KillRuns(
        sess, body=bindings.v1KillRunsRequest(runIds=[], filter=runFilter, projectId=1)
    )
    searchResp = bindings.post_SearchRuns(sess, body=bindings.v1SearchRunsRequest(filter=runFilter))

    # Expect one of the two grid runs to match our filter.
    assert len(killResp.results) == 1, killResp.results
    assert len(searchResp.runs) == 1, searchResp.runs
    res = killResp.results[0]
    assert res.error == "", res.error
    wait_for_run_state(sess, res.id, bindings.trialv1State.CANCELED)
    exp_ref.kill()


@pytest.mark.e2e_cpu
def test_run_pause_and_resume() -> None:
    sess = api_utils.user_session()
    exp_ref = noop.create_experiment(sess)

    searchResp = bindings.post_SearchRuns(
        sess,
        body=bindings.v1SearchRunsRequest(
            limit=1,
            filter="""{"filterGroup":{"children":[{"columnName":"experimentId","kind":"field",
            "location":"LOCATION_TYPE_RUN","operator":"=","type":"COLUMN_TYPE_NUMBER","value":%d}],
            "conjunction":"and","kind":"group"},"showArchived":false}"""
            % (exp_ref.id),
        ),
    )

    assert searchResp.runs[0].state == bindings.trialv1State.ACTIVE
    run_id = searchResp.runs[0].id
    pauseResp = bindings.post_PauseRuns(
        sess, body=bindings.v1PauseRunsRequest(runIds=[run_id], projectId=1)
    )

    # validate response
    assert len(pauseResp.results) == 1
    assert pauseResp.results[0].id == run_id
    assert pauseResp.results[0].error == ""

    # ensure that run is paused
    wait_for_run_state(sess, run_id, bindings.trialv1State.PAUSED)

    resumeResp = bindings.post_ResumeRuns(
        sess, body=bindings.v1ResumeRunsRequest(runIds=[run_id], projectId=1)
    )

    assert len(resumeResp.results) == 1
    assert resumeResp.results[0].id == run_id
    assert resumeResp.results[0].error == ""

    # ensure that run is unpaused
    wait_for_run_state(sess, run_id, bindings.trialv1State.ACTIVE)

    # kill run for cleanup
    _ = bindings.post_KillRuns(sess, body=bindings.v1KillRunsRequest(runIds=[run_id], projectId=1))
    wait_for_run_state(sess, run_id, bindings.trialv1State.CANCELED)


@pytest.mark.e2e_cpu
def test_run_in_search_not_pausable_or_resumable() -> None:
    sess = api_utils.user_session()
    config = {"searcher": {"name": "random", "max_trials": 2}}
    exp_ref = noop.create_experiment(sess, config=config)

    runFilter = (
        """{
  "filterGroup": {
    "children": [
      {
        "columnName": "experimentId",
        "kind": "field",
        "location": "LOCATION_TYPE_RUN",
        "operator": "=",
        "type": "COLUMN_TYPE_NUMBER",
        "value": %s
      }
    ],
    "conjunction": "and",
    "kind": "group"
  },
  "showArchived": false
}"""
        % exp_ref.id
    )
    pauseResp = bindings.post_PauseRuns(
        sess,
        body=bindings.v1PauseRunsRequest(
            runIds=[],
            filter=runFilter,
            projectId=1,
        ),
    )

    assert pauseResp.results
    for r in pauseResp.results:
        assert r.error == "Cannot pause/unpause run '" + str(r.id) + "' (part of multi-trial)."

    resumeResp = bindings.post_ResumeRuns(
        sess,
        body=bindings.v1ResumeRunsRequest(runIds=[], projectId=1, filter=runFilter),
    )

    assert resumeResp.results
    for res in resumeResp.results:
        assert res.error == "Cannot pause/unpause run '" + str(res.id) + "' (part of multi-trial)."

    exp_ref.kill()
