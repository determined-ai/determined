from daist.models.config import Config
from daist.models.session import session
from daist.utils.det import DetAPIClient


class Resources:
    def __init__(self, user_id=None, user_name=None,
                 model_name=None, model_version_number=None,
                 experiment_id=None, task_id=None, checkpoint_id=None, experiment_file=None,
                 metric_name=None, metric_type=None,
                 batches=None, batches_margin=100,
                 trial_id=None,
                 template_name=None,
                 resource_pool=None,
                 workspace_id=None, project_id=None,
                 group_id=None, rbac_enabled=False):

        self.user_id = user_id
        self.user_name = user_name
        self.model_name = model_name
        self.model_version_number = model_version_number
        self.experiment_id = experiment_id
        self.trial_id = trial_id
        self.checkpoint_id = checkpoint_id
        self.experiment_file = experiment_file
        self.metric_name = metric_name
        self.metric_type = metric_type
        self.batches = batches
        self.batches_margin = batches_margin
        self.task_id = task_id
        self.template_name = template_name
        self.resource_pool = resource_pool
        self.workspace_id = workspace_id
        self.project_id = project_id
        self.group_id = group_id
        self.rbac_enabled = rbac_enabled


def autodiscover() -> Resources:
    client = DetAPIClient(session.determined.host,
                          session.determined.user,
                          session.determined.password)

    resources = Resources()

    user = client.get("/api/v1/me")["user"]
    resources.user_id = user["id"]
    resources.user_name = user["username"]

    # TODO: loop through experiments and select an experiment with the most trials that meet
    #  our criteria
    paginated_experiments = client.get("/api/v1/experiments?showTrialData=true")
    for experiment in paginated_experiments["experiments"]:
        if experiment["state"] != "STATE_COMPLETED":
            continue
        experiment_id = experiment['id']
        for trialId in client.get(
                f"/api/v1/experiments/{experiment_id}")['experiment']['trialIds']:
            trial = client.get(f"/api/v1/trials/{trialId}")["trial"]
            batches = trial["bestCheckpoint"]["totalBatches"]
            resources.checkpoint_id = trial["bestCheckpoint"]["uuid"]
            resources.experiment_id = experiment_id
            resources.trial_id = trialId
            resources.batches = batches
            resources.batches_margin = 0
            resources.metric_name = list(trial["summaryMetrics"]["avg_metrics"].keys())[0]
            resources.metric_type = "METRIC_TYPE_TRAINING"
            for file in client.get(
                    f"/api/v1/experiments/{resources.experiment_id}/file_tree")["files"]:
                if not file["isDir"]:
                    resources.experiment_file = file["path"]
            break
        if resources.experiment_id is not None:
            break
    
    for model in client.get("/api/v1/models")["models"]:
        if model["numVersions"] > 0:
            model_id = model["id"]
            model_name = model["name"]
            model_version_number = (
                client.get(f"/api/v1/models/{model_id}/versions"))["modelVersions"][0]["id"]
            resources.model_name = model_name
            resources.model_version_number = model_version_number
            break

    resource_pool = client.get("/api/v1/resource-pools")["resourcePools"][0]["name"]
    resources.resource_pool = resource_pool

    allocations = client.get("/api/v1/tasks")["allocationIdToSummary"]
    for allocation in allocations:
        resources.task_id = allocations[allocation]["taskId"]
        break

    templates = client.get("/api/v1/templates")["templates"]
    for template in templates:
        resources.template_name = template["name"]
        break

    for workspace in client.get("/api/v1/workspaces")["workspaces"]:
        workspace_id = workspace["id"]
        for project in client.get(f"/api/v1/workspaces/{workspace_id}/projects")["projects"]:
            project_id = project["id"]
            resources.workspace_id = workspace_id
            resources.project_id = project_id
            break
        if resources.workspace_id is not None:
            break
    
    resources.rbac_enabled = client.get("/api/v1/master")["rbacEnabled"]

    # TODO: set group_id if rbac is enabled. This is currently NEVER set for K6 tests.
    # It will appear in the following URLs:
    #     f"/api/v1/groups/{resources.group_id}"
    #     f"/api/v1/roles/search/by-group/{resources.group_id}"

    return resources


ci_snapshot_resources = Resources(
    user_id=20, user_name="admin",
    model_name="tnjpuojqzbluqiyyqilftulsw", model_version_number=1,
    experiment_id=100, trial_id=8282,
    metric_name="85c9", metric_type="METRIC_TYPE_TRAINING",
    batches=1800, batches_margin=99,
    task_id="backported.8282",
    # The uncategorized workspace ID.
    workspace_id=1,
    # The uncategorized project ID.
    project_id=1,
    rbac_enabled=False,
    # there's a checkpoint listed in the database, but we don't have the checkpoint
    checkpoint_id=None,
    experiment_file=None,
    template_name=None,
    resource_pool='default'
)


def ci_snapshot_on_det_deploy(*_args) -> Resources:
    resources = ci_snapshot_resources
    resources.resource_pool = "compute-pool"
    return resources


def ci_snapshot_on_local_mode(*_args) -> Resources:
    resources = ci_snapshot_resources
    resources.resource_pool = "default"
    return resources


def get_resource_profile(config: Config) -> Resources:
    available_profiles = ["autodiscover", "ci_snapshot_on_local_mode", "ci_snapshot_on_det_deploy"]
    if config.determined.resource_profile in available_profiles:
        return globals()[config.determined.resource_profile](config)
    raise Exception(f"Unknown profile specified: {config.determined.resource_profile}."
                    f"Must be one of {', '.join(available_profiles)}")
