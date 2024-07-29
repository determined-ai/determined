import datetime
import logging
import json
import random
from urllib import parse

from . import locust_utils
from .locust_utils import LocustTasksWithMeta, LocustGetTaskWithMeta, LocustPostTaskWithMeta
from .resources import Resources
from ..utils import flags


logger = logging.getLogger(__name__)


def read_only_tasks(resources: Resources) -> LocustTasksWithMeta:
    tasks = locust_utils.LocustTasksWithMeta()

    tasks.append(LocustGetTaskWithMeta("/api/v1/master", test_name="get master configuration"))
    tasks.append(LocustGetTaskWithMeta("/api/v1/master/config",
                                       test_name="get master config file"))

    tasks.append(LocustGetTaskWithMeta("/api/v1/agents", test_name="get agents"))
    tasks.append(LocustGetTaskWithMeta("/api/v1/workspaces", test_name="get workspaces"))
    tasks.append(LocustGetTaskWithMeta("/api/v1/users/setting", test_name="get user settings"))
    tasks.append(LocustGetTaskWithMeta("/api/v1/resource-pools", test_name="get resource pools"))
    tasks.append(LocustGetTaskWithMeta("/api/v1/users", test_name="get users"))
    tasks.append(LocustGetTaskWithMeta("/api/v1/models", test_name="get models"))
    tasks.append(LocustGetTaskWithMeta("/api/v1/master/telemetry", test_name="get telemetry"))
    tasks.append(LocustGetTaskWithMeta("/api/v1/tensorboards", test_name="get tensorboards"))
    tasks.append(LocustGetTaskWithMeta("/api/v1/shells", test_name="get shells"))
    tasks.append(LocustGetTaskWithMeta("/api/v1/notebooks", test_name="get notebooks"))
    tasks.append(LocustGetTaskWithMeta("/api/v1/commands", test_name="get commands"))
    tasks.append(LocustGetTaskWithMeta("/api/v1/job-queues", test_name="get job queues"))
    tasks.append(LocustGetTaskWithMeta("/api/v1/job-queues/stats", test_name="get job queue stats"))
    tasks.append(LocustGetTaskWithMeta("/api/v1/user/projects/activity",
                                       test_name="get user activity"))
    tasks.append(LocustGetTaskWithMeta("/api/v1/webhooks", test_name="get webhooks"))
    tasks.append(LocustGetTaskWithMeta("/api/v1/model/labels", test_name="get model labels"))
    tasks.append(LocustGetTaskWithMeta("/api/v1/tasks", test_name="get tasks"))
    tasks.append(LocustGetTaskWithMeta("/api/v1/tasks/count", test_name="get task count"))
    tasks.append(LocustGetTaskWithMeta("/api/v1/experiments", test_name="get experiments",
                                       params={"showTrialData": True}))
    tasks.append(LocustGetTaskWithMeta("/api/v1/master/logs", test_name="get master logs",
                                       params={
                                           "offset": -1,
                                           "limit": 0
                                       }))
    tasks.append(LocustGetTaskWithMeta("/api/v1/auth/user", test_name="get authenticated user"))
    tasks.append(LocustGetTaskWithMeta("/api/v1/me", test_name="get me"))
    tasks.append(LocustGetTaskWithMeta("/api/v1/experiment/labels",
                                       test_name="get experiment labels"))
    tasks.append(LocustGetTaskWithMeta("/api/v1/templates", test_name="get templates"))

    start_date = "2000-01-01"
    end_date = datetime.date.today().isoformat()
    tasks.append(LocustGetTaskWithMeta(
        "/api/v1/resources/allocation/aggregated", test_name="get resource allocations",
        params={
            "start_date": start_date,
            "end_date": end_date,
            "period": "RESOURCE_ALLOCATION_AGGREGATION_PERIOD_DAILY"
        }))

    if resources.user_id is not None:
        tasks.append(LocustGetTaskWithMeta(f"/api/v1/users/{resources.user_id}",
                                           test_name="get user by ID"))
    if resources.user_name is not None:
        tasks.append(LocustGetTaskWithMeta(f"/api/v1/users/{resources.user_name}/by-username",
                                           test_name="get user by username"))

    if resources.resource_pool is not None:
        tasks.append(LocustGetTaskWithMeta(
            f"/api/v1/resource-pools/{resources.resource_pool}/workspace-bindings",
            test_name="get workspace bindings"))
        tasks.append(LocustGetTaskWithMeta(
            "/api/v1/job-queues-v2", test_name="get v2 job queue",
            params={"resourcePool": resources.resource_pool}))

    if resources.project_id is not None:
        tasks.append(LocustGetTaskWithMeta(f"/api/v1/projects/{resources.project_id}",
                                           test_name="get project"))
        tasks.append(LocustGetTaskWithMeta(
            f"/api/v1/projects/{resources.project_id}/experiments/metric-ranges",
            test_name="get project metric ranges"))
        # .. todo:: this should be moved to a different tasks list as this is supposed to be
        #           read only. However, this does not affect the state of the system, making it
        #           effectively read only.
        tasks.append(LocustPostTaskWithMeta(
            "/api/v1/experiments-search", test_name="search experiments",
            body={"projectId": resources.project_id}))
        if flags.SCALE_33:
            tasks.append(LocustGetTaskWithMeta(
                f"/api/v1/projects/{resources.project_id}/columns",
                test_name="get project columns"))

    if resources.workspace_id is not None:
        tasks.append(LocustGetTaskWithMeta(
            "/api/v1/tensorboards", test_name="get workspace tensorboards",
            params={"sortBy": "SORT_BY_UNSPECIFIED",
                    "orderBy": "ORDER_BY_UNSPECIFIED",
                    "workspaceId": resources.workspace_id}))
        tasks.append(LocustGetTaskWithMeta("/api/v1/shells", test_name="get workspace shells",
                                           params={"sortBy": "SORT_BY_UNSPECIFIED",
                                                   "orderBy": "ORDER_BY_UNSPECIFIED",
                                                   "workspaceId": resources.workspace_id}))
        tasks.append(LocustGetTaskWithMeta(
            "/api/v1/notebooks", test_name="get workspace notebooks",
            params={"limit": 1000, "workspaceId": resources.workspace_id}))
        # This was formerly "get workspace commands" in legacy K6 testing.
        tasks.append(LocustGetTaskWithMeta("/api/v1/shells", test_name="shells",
                                           params={"limit": 1000,
                                                   "workspaceId": resources.workspace_id}))
        tasks.append(LocustGetTaskWithMeta(
            f"/api/v1/workspaces/{resources.workspace_id}/projects",
            test_name="get workspace projects"))
        tasks.append(LocustGetTaskWithMeta(
            f"/api/v1/workspaces/{resources.workspace_id}/available-resource-pools",
            test_name="get available workspace resource pools"))
        tasks.append(LocustGetTaskWithMeta(
            "/api/v1/model/labels", test_name="get workspace model labels",
            params={"workspaceId": resources.workspace_id}))
        tasks.append(LocustGetTaskWithMeta(f"/api/v1/workspaces/{resources.workspace_id}",
                                           test_name="get workspace"))

    if resources.model_name is not None:
        tasks.append(LocustGetTaskWithMeta(f"/api/v1/models/{resources.model_name}",
                                           test_name="get model"))
        tasks.append(LocustGetTaskWithMeta(f"/api/v1/models/{resources.model_name}/versions",
                                           test_name="get model versions"))
        if resources.model_version_number is not None:
            tasks.append(LocustGetTaskWithMeta(
                f"/api/v1/models/{resources.model_name}/versions/{resources.model_version_number}",
                test_name="get model version"))
            tasks.append(LocustGetTaskWithMeta(
                f"/api/v1/models/{resources.model_name}/versions/"
                f"{resources.model_version_number}/metrics",
                test_name="get model metrics"))

    if resources.trial_id is not None:
        tasks.append(LocustGetTaskWithMeta(f"/api/v1/trials/{resources.trial_id}",
                                           test_name="get trial"))
        tasks.append(LocustGetTaskWithMeta(f"/api/v1/trials/{resources.trial_id}/workloads",
                                           test_name="get trial workloads"))
        # This was formerly "get trial logs" in legacy K6 testings
        tasks.append(LocustGetTaskWithMeta(f"/api/v1/trials/{resources.trial_id}/logs/fields",
                                           test_name="get trial log fields"))
        tasks.append(LocustGetTaskWithMeta(
            "/api/v1/trials/time-series", test_name="get trials time series",
            params={
                "trialIds": resources.trial_id,
                "startBatches": 0,
                "metricType": "METRIC_TYPE_UNSPECIFIED"}))

        tasks.append(LocustGetTaskWithMeta(f"/api/v1/trials/{resources.trial_id}/checkpoints",
                                           test_name="get trial checkpoints"))
        tasks.append(LocustGetTaskWithMeta(f"/api/v1/trials/{resources.trial_id}/profiler/metrics",
                                           test_name="get trial metrics"
                                           ))
        tasks.append(LocustGetTaskWithMeta(f"/api/v1/trials/{resources.trial_id}/logs",
                                           test_name="get logs by trial"))
        tasks.append(LocustGetTaskWithMeta(
            f"/api/v1/trials/{resources.trial_id}/profiler/available_series",
            test_name="get profiler available series"))
        tasks.append(LocustGetTaskWithMeta(
            "/api/v1/trials/metrics/training_metrics", test_name="get training metrics",
            params={"trialIds": resources.trial_id}))
        tasks.append(LocustGetTaskWithMeta(
            "/api/v1/trials/metrics/validation_metrics", test_name="get validation metrics",
            params={"trialIds": resources.trial_id}))
        # Redundant?
        tasks.append(LocustGetTaskWithMeta(
            "/api/v1/trials/metrics/trial_metrics", test_name="get trial metrics 2",
            params={"trialIds": resources.trial_id, "group": "training"}))

    if resources.experiment_id is not None:
        tasks.append(LocustGetTaskWithMeta(f"/api/v1/experiments/{resources.experiment_id}",
                                           test_name="get experiment"))
        tasks.append(LocustGetTaskWithMeta(
            "/api/v1/experiments/metrics-stream/metric-names",
            test_name="get experiment metric names",
            params={"ids": resources.experiment_id}))
        tasks.append(LocustGetTaskWithMeta(f"/api/v1/experiments/{resources.experiment_id}/trials",
                                           test_name="get experiment trials"))
        tasks.append(LocustGetTaskWithMeta(
            f"/api/v1/experiments/{resources.experiment_id}/file_tree",
            test_name="get experiment file tree"))

        tasks.append(LocustGetTaskWithMeta(
            f"/api/v1/experiments/{resources.experiment_id}/model_def",
            test_name="get experiment model definition"))
        tasks.append(LocustGetTaskWithMeta(
            f"/api/v1/experiments/{resources.experiment_id}/searcher"
            f"/best_searcher_validation_metric",
            test_name="get experiment best searcher validation metric"))
        tasks.append(LocustGetTaskWithMeta(
            f"/api/v1/experiments/{resources.experiment_id}/searcher_events",
            test_name="get experiment searcher events"))
        tasks.append(LocustGetTaskWithMeta(
            f"/api/v1/experiments/{resources.experiment_id}/validation-history",
            test_name="get experiment validation history"))
        if resources.experiment_file is not None:
            tasks.append(LocustGetTaskWithMeta(
                f"/experiments/{resources.experiment_id}/file/download",
                test_name="get experiment file",
                params={"path": resources.experiment_file}))
            # .. todo:: this should be moved to a different tasks list as this is supposed to be
        #               read only.
            tasks.append(LocustPostTaskWithMeta(
                f"/api/v1/experiments/{resources.experiment_id}/file",
                test_name="put experiment file",
                body={"experimentId": resources.experiment_id, "path": resources.experiment_file}))

    if resources.checkpoint_id is not None:
        # Big download?
        tasks.append(LocustGetTaskWithMeta(f"/checkpoints/{resources.checkpoint_id}"))
        tasks.append(LocustGetTaskWithMeta(f"/api/v1/checkpoints/{resources.checkpoint_id}"))
        tasks.append(LocustGetTaskWithMeta(
            f"/api/v1/checkpoints/{resources.checkpoint_id}/metrics"))

    if resources.task_id is not None:
        tasks.append(LocustGetTaskWithMeta(f"/api/v1/tasks/{resources.task_id}",
                                           test_name="get task"))
        tasks.append(LocustGetTaskWithMeta(f"/api/v1/tasks/{resources.task_id}/logs/fields",
                                           test_name="get task log fields"))
        tasks.append(LocustGetTaskWithMeta(f"/api/v1/tasks/{resources.task_id}/logs",
                                           test_name="get task logs"))
        tasks.append(LocustGetTaskWithMeta(f"/api/v1/tasks/{resources.task_id}/acceleratorData",
                                           test_name="get task accelerator data"))
        tasks.append(LocustGetTaskWithMeta(f"/api/v1/tasks/{resources.task_id}/config",
                                           test_name="get task config"))
        tasks.append(LocustGetTaskWithMeta(f"/api/v1/tasks/{resources.task_id}/context_directory",
                                           test_name="get task context directory"))

    if resources.template_name is not None:
        tasks.append(LocustGetTaskWithMeta(f"/api/v1/templates/{resources.template_name}",
                                           test_name="get template"))

    if resources.rbac_enabled:
        tasks.append(LocustGetTaskWithMeta("/api/v1/permissions-summary",
                                           test_name="get permissions summary")),
        tasks.append(LocustGetTaskWithMeta("/api/v1/groups/search", test_name="search groups"))
        tasks.append(LocustGetTaskWithMeta("/api/v1/roles/search/by-assignability",
                                           test_name="get roles by assignability"))
        tasks.append(LocustGetTaskWithMeta("/api/v1/roles-search", test_name="search roles"))
        tasks.append(LocustGetTaskWithMeta(f"/api/v1/roles/search/by-user/{resources.user_id}",
                                           test_name="search by user ID"))
        if resources.workspace_id is not None:
            tasks.append(LocustGetTaskWithMeta(f"/api/v1/roles/workspace/{resources.workspace_id}",
                                               test_name="get workspace roles"))

        if resources.group_id is not None:
            tasks.append(LocustGetTaskWithMeta(f"/api/v1/groups/{resources.group_id}",
                                               test_name="get group"))
            tasks.append(LocustGetTaskWithMeta(
                f"/api/v1/roles/search/by-group/{resources.group_id}",
                test_name="search roles by group"))

    filter_ = {
        "filterGroup": {
            "children": [
                {
                    "columnName": "hp.global_batch_size",
                    "kind": "field",
                    "location": "LOCATION_TYPE_HYPERPARAMETERS",
                    "operator": "<=",
                    "type": "COLUMN_TYPE_NUMBER",
                    "value": 60
                }
            ],
            "conjunction": "and",
            "kind": "group"
        },
        "showArchived": False
    }
    serialized_filter = json.dumps(filter_)
    encoded_filter = parse.quote_plus(serialized_filter)
    tasks.append(LocustPostTaskWithMeta(f"/api/v1/runs?filter={encoded_filter}",
                                        test_name="search flat runs w/ hparam"))
    if resources.project_id is not None:
        tasks.append(LocustPostTaskWithMeta(f"/api/v1/runs", test_name="search flat runs",
                                            body={"projectId": resources.project_id}))

    # The following require launching things to test:
    #   commands
    #   notebooks
    #   shells
    #   tensorboards

    logger.info(f"There are {len(tasks)} tasks programmed")
    # TODO: weighting?

    random.shuffle(tasks)

    return tasks
