import uuid
from typing import Any, Dict, List

import pytest

from determined.common.api import authentication, bindings, errors
from determined.common.experimental import session
from tests import config as conf
from tests import experiment as exp


def gen_string():
    return str(uuid.uuid4())


def queryTrials(sess, filters: Dict[str, Any]) -> List[Dict[str, Any]]:
    body = bindings.v1QueryTrialsRequest.from_json({"filters": filters, "limit": 1000})
    res = bindings.post_QueryTrials(sess, body=body)
    return res.to_json()["trials"]


def queryTrialsIds(sess, filters) -> List[int]:
    return [t["trialId"] for t in queryTrials(sess, filters)]


def tagPayload(tags) -> List[Dict[str, Any]]:
    return [{"key": tag} for tag in tags]


def patchTrials(sess, trials, addTags=None, removeTags=None):
    target = {"trial": {"ids": trials}} if isinstance(trials, list) else {"filters": trials}
    patch = {"patch": {"addTag": addTags, "removeTag": removeTags}}
    body = bindings.v1PatchTrialsRequest.from_json({**target, **patch})
    bindings.patch_PatchTrials(sess, body=body)


def assert_same_ids(a, b):
    assert sorted(a) == sorted(b)


def create_collection(
    sess,
    name,
    filters,
    sorter=None,
    project_id=1,
):
    if sorter is None:
        sorter = {
            "field": "trial_id",
            "namespace": "NAMESPACE_UNSPECIFIED",
            "orderBy": "ORDER_BY_DESC",
        }
    body = bindings.v1CreateTrialsCollectionRequest.from_json(
        {"name": name, "filters": filters, "sorter": sorter, "projectId": project_id}
    )
    return bindings.post_CreateTrialsCollection(sess, body=body).to_json()["collection"]


def patch_collection(sess, collection_id, name="", filters=None, sorter=None):
    body = bindings.v1PatchTrialsCollectionRequest.from_json(
        {"id": collection_id, "name": name, "filters": filters, "sorter": sorter}
    )
    return bindings.patch_PatchTrialsCollection(sess, body=body).to_json()["collection"]


def get_collections(sess, project_id=1):
    return bindings.get_GetTrialsCollections(sess, projectId=project_id).to_json()["collections"]


def get_collection_names(sess, project_id=1):
    return [c["name"] for c in get_collections(sess, project_id)]


def sorted_filter_items(filters):
    return [(k, v) for k, v in sorted(filters.items(), key=lambda kv: kv[0]) if v]


def assert_same_filters(f1, f2):
    assert sorted_filter_items(f1) == sorted_filter_items(f2)


def empty_range_filters():
    return {
        "trainingMetrics": [],
        "validationMetrics": [],
        "hparams": [],
    }


def assert_collection_is_uniquely_represented_in_collections(sess, collection):
    collections = get_collections(sess, project_id=collection["projectId"])
    matching_collections = [c for c in collections if c["id"] == collection["id"]]
    assert len(matching_collections) == 1
    assert matching_collections[0] == collection


@pytest.mark.e2e_cpu
def test_trial_collections() -> None:

    master_url = conf.make_master_url()
    authentication.cli_auth = authentication.Authentication(master_url, try_reauth=True)
    sess = session.Session(master_url, None, None, None)

    experiment_id = exp.create_experiment(
        conf.fixtures_path("no_op/adaptive.yaml"), conf.fixtures_path("no_op")
    )

    exp.wait_for_experiment_state(
        experiment_id,
        bindings.determinedexperimentv1State.STATE_COMPLETED,
        max_wait_secs=conf.DEFAULT_MAX_WAIT_SECS,
    )

    # query by experiment gets you the right trials
    trials = bindings.get_GetExperimentTrials(sess, experimentId=experiment_id).trials
    t_ids = [t.id for t in trials]

    assert_same_ids(t_ids, queryTrialsIds(sess, {"experimentIds": [experiment_id]}))

    # query by trial id gets you the right trials
    assert_same_ids(t_ids, queryTrialsIds(sess, {"trialIds": t_ids}))

    # querying for non-existent tags get you no trials
    tags = [{"key": gen_string()} for tag in range(3)]
    assert len(queryTrialsIds(sess, {"tags": tags})) == 0

    # after adding those same tags, you get the relevant trials
    patchTrials(sess, t_ids, addTags=tags)
    assert_same_ids(t_ids, queryTrialsIds(sess, {"tags": tags}))

    # and they have the relevant tags
    trials_resp = queryTrials(sess, {"tags": tags})
    tags_as_strings = [t["key"] for t in tags]
    assert all(sorted(tags_as_strings) == sorted(t["tags"].keys()) for t in trials_resp)

    # querying for list of tags returns partial matches
    patchTrials(sess, t_ids, removeTags=tags[:1])
    assert_same_ids(t_ids, queryTrialsIds(sess, {"tags": tags}))

    # after removing all tags, querying by tags comes up empty
    patchTrials(sess, {"tags": tags}, removeTags=tags)
    query_ids = queryTrialsIds(sess, {"tags": tags})
    assert len(query_ids) == 0

    # and the trials themselves have the tags removed
    trials_resp = queryTrials(sess, {"experimentIds": [experiment_id]})
    assert all(not t["tags"] for t in trials_resp)

    resp_trials = queryTrials(sess, {"experimentIds": [experiment_id]})
    sorted_by_searcher_metric = sorted(resp_trials, key=lambda x: x["searcherMetricLoss"])
    top_three_ids = [t["trialId"] for t in sorted_by_searcher_metric[:3]]
    assert_same_ids(
        top_three_ids,
        queryTrialsIds(
            sess,
            {
                "experimentIds": [experiment_id],
                "rankWithinExp": {
                    "sorter": {
                        "namespace": "NAMESPACE_UNSPECIFIED",
                        "field": "searcher_metric_loss",
                        "orderBy": "ORDER_BY_ASC",
                    },
                    "rank": 3,
                },
            },
        ),
    )

    exp_trials = exp.experiment_trials(experiment_id)
    t_ids = [t.trial.id for t in exp_trials]

    trials = queryTrials(sess, {"trialIds": t_ids})

    for trial in trials:
        t_id = trial["trialId"]
        good_range_filters = empty_range_filters()
        bad_range_filters = empty_range_filters()

        for namespace in ["trainingMetrics", "validationMetrics", "hparams"]:
            for name, val in trial[namespace].items():
                try:
                    val = float(val)
                except Exception:
                    continue
                good_range_filters[namespace].append(
                    {"name": name, "min": val - 0.00001, "max": val + 0.00001}
                )
                bad_range_filters[namespace].append(
                    {"name": name, "min": val + 1, "max": val + 1.0001}
                )

        assert t_id in queryTrialsIds(sess, good_range_filters)
        assert t_id not in queryTrialsIds(sess, bad_range_filters)
        assert t_id not in queryTrialsIds(
            sess, {**bad_range_filters, "hparams": good_range_filters["hparams"]}
        )
        assert t_id not in queryTrialsIds(
            sess,
            {**bad_range_filters, "validationMetrics": good_range_filters["validationMetrics"]},
        )
        assert t_id not in queryTrialsIds(
            sess, {**bad_range_filters, "trainingMetrics": good_range_filters["trainingMetrics"]}
        )

    # collection does not exist yet
    existing_collection_names = get_collection_names(sess)
    original_name_for_collection = gen_string()
    original_filters_for_collection = {"experimentIds": [experiment_id]}
    assert original_name_for_collection not in existing_collection_names

    # create a new collection
    collection = create_collection(
        sess, original_name_for_collection, original_filters_for_collection
    )
    assert_collection_is_uniquely_represented_in_collections(sess, collection)
    assert_same_filters(original_filters_for_collection, collection["filters"])
    assert collection["name"] == original_name_for_collection

    # patch the filters for the collection
    collection_id = collection["id"]
    new_filters_for_collection = {"trialIds": t_ids}
    patched_collection = patch_collection(sess, collection_id, filters=new_filters_for_collection)
    assert_collection_is_uniquely_represented_in_collections(sess, patched_collection)

    # unpatched fields remain unchanged
    assert patched_collection["name"] == original_name_for_collection

    # patched fields are updated
    assert_same_filters(new_filters_for_collection, patched_collection["filters"])

    # patch a different field
    new_name_for_collection = gen_string()
    twice_patched_collection = patch_collection(sess, collection_id, name=new_name_for_collection)
    assert_collection_is_uniquely_represented_in_collections(sess, twice_patched_collection)
    assert twice_patched_collection["name"] == new_name_for_collection
    assert twice_patched_collection["name"] != original_name_for_collection

    # unpatched fields remain unchanged
    assert_same_filters(twice_patched_collection["filters"], new_filters_for_collection)

    with pytest.raises(AssertionError):
        assert_same_filters(twice_patched_collection["filters"], original_filters_for_collection)

    with pytest.raises(errors.APIException):
        create_collection(sess, new_name_for_collection, original_filters_for_collection)
    other_collection = create_collection(
        sess, original_name_for_collection, original_filters_for_collection
    )
    with pytest.raises(errors.APIException):
        patch_collection(sess, other_collection["id"], name=new_name_for_collection)
