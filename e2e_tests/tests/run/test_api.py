import time

import pytest

from determined.common import api
from determined.common.api import bindings
from tests import api_utils
from tests import config as conf
from tests import experiment as exp


def wait_for_run_state(
    test_session: api.Session,
    run_id: int,
    expected_state: bindings.trialv1State,
    timeout: int = 30,
) -> None:
    deadline = time.time() + timeout
    while time.time() < deadline:
        resp = bindings.get_SearchRuns(
            test_session,
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
        )
        if expected_state == resp.runs[0].state:
            return
        time.sleep(0.1)
    pytest.fail(f"task failed to complete after {timeout} seconds")


@pytest.mark.e2e_cpu
def test_run_kill() -> None:
    sess = api_utils.user_session()
    exp_id = exp.create_experiment(
        sess, conf.fixtures_path("no_op/single.yaml"), conf.fixtures_path("no_op")
    )

    searchResp = bindings.get_SearchRuns(
        sess,
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
        % exp_id,
    )
    assert searchResp.runs[0].state == bindings.trialv1State.ACTIVE

    run_id = searchResp.runs[0].id
    killResp = bindings.post_KillRuns(
        sess, body=bindings.v1KillRunsRequest(runIds=[run_id], projectId=1)
    )
    assert len(killResp.results) == 1
    assert killResp.results[0].id == run_id
    assert killResp.results[0].error == ""

    # ensure that run is canceled
    wait_for_run_state(sess, run_id, bindings.trialv1State.CANCELED)

    # cancelling an already terminated run should be fine
    killResp = bindings.post_KillRuns(
        sess, body=bindings.v1KillRunsRequest(runIds=[run_id], projectId=1)
    )

    # validate response
    assert len(killResp.results) == 1
    assert killResp.results[0].id == run_id
    assert killResp.results[0].error == ""


@pytest.mark.e2e_cpu
def test_run_kill_filter() -> None:
    sess = api_utils.user_session()
    exp_id = exp.create_experiment(
        sess,
        conf.fixtures_path("no_op/grid-long-run.yaml"),
        conf.fixtures_path("no_op"),
    )

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
      },
      {
        "columnName": "hp.unique_id",
        "kind": "field",
        "location": "LOCATION_TYPE_RUN_HYPERPARAMETERS",
        "operator": ">=",
        "type": "COLUMN_TYPE_NUMBER",
        "value": 3
      }
    ],
    "conjunction": "and",
    "kind": "group"
  },
  "showArchived": false
}"""
        % exp_id
    )

    killResp = bindings.post_KillRuns(
        sess, body=bindings.v1KillRunsRequest(runIds=[], filter=runFilter, projectId=1)
    )

    searchResp = bindings.get_SearchRuns(sess, filter=runFilter)

    # validate response
    assert len(searchResp.runs) > 0
    assert len(killResp.results) > 0
    assert len(killResp.results) == len(searchResp.runs)
    for res in killResp.results:
        assert res.error == ""
        wait_for_run_state(sess, res.id, bindings.trialv1State.CANCELED)
