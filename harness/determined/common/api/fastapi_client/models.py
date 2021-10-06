from datetime import datetime
from enum import Enum
from typing import Any  # noqa
from typing import Dict, List, Optional

from determined.common.api.fapi import BaseModel


class Determinedcheckpointv1State(str, Enum):
    UNSPECIFIED = "STATE_UNSPECIFIED"
    ACTIVE = "STATE_ACTIVE"
    COMPLETED = "STATE_COMPLETED"
    ERROR = "STATE_ERROR"
    DELETED = "STATE_DELETED"


class Determinedcontainerv1State(str, Enum):
    UNSPECIFIED = "STATE_UNSPECIFIED"
    ASSIGNED = "STATE_ASSIGNED"
    PULLING = "STATE_PULLING"
    STARTING = "STATE_STARTING"
    RUNNING = "STATE_RUNNING"
    TERMINATED = "STATE_TERMINATED"


class Determineddevicev1Type(str, Enum):
    UNSPECIFIED = "TYPE_UNSPECIFIED"
    CPU = "TYPE_CPU"
    GPU = "TYPE_GPU"


class Determinedexperimentv1State(str, Enum):
    UNSPECIFIED = "STATE_UNSPECIFIED"
    ACTIVE = "STATE_ACTIVE"
    PAUSED = "STATE_PAUSED"
    STOPPING_COMPLETED = "STATE_STOPPING_COMPLETED"
    STOPPING_CANCELED = "STATE_STOPPING_CANCELED"
    STOPPING_ERROR = "STATE_STOPPING_ERROR"
    COMPLETED = "STATE_COMPLETED"
    CANCELED = "STATE_CANCELED"
    ERROR = "STATE_ERROR"
    DELETED = "STATE_DELETED"
    DELETING = "STATE_DELETING"
    DELETE_FAILED = "STATE_DELETE_FAILED"


class Determinedjobv1State(str, Enum):
    UNSPECIFIED = "STATE_UNSPECIFIED"
    QUEUED = "STATE_QUEUED"
    SCHEDULED = "STATE_SCHEDULED"
    SCHEDULED_BACKFILLED = "STATE_SCHEDULED_BACKFILLED"


class Determinedjobv1Type(str, Enum):
    UNSPECIFIED = "TYPE_UNSPECIFIED"
    EXPERIMENT = "TYPE_EXPERIMENT"
    NOTEBOOK = "TYPE_NOTEBOOK"
    TENSORBOARD = "TYPE_TENSORBOARD"
    SHELL = "TYPE_SHELL"
    COMMAND = "TYPE_COMMAND"


class Determinedtaskv1State(str, Enum):
    UNSPECIFIED = "STATE_UNSPECIFIED"
    PENDING = "STATE_PENDING"
    ASSIGNED = "STATE_ASSIGNED"
    PULLING = "STATE_PULLING"
    STARTING = "STATE_STARTING"
    RUNNING = "STATE_RUNNING"
    TERMINATED = "STATE_TERMINATED"
    TERMINATING = "STATE_TERMINATING"


class GetHPImportanceResponseMetricHPImportance(BaseModel):
    hp_importance: "Optional[Dict[str, float]]"
    experiment_progress: "Optional[float]"
    error: "Optional[str]"
    pending: "Optional[bool]"
    in_progress: "Optional[bool]"


class GetTrialResponseWorkloadContainer(BaseModel):
    training: "Optional[V1MetricsWorkload]"
    validation: "Optional[V1MetricsWorkload]"
    checkpoint: "Optional[V1CheckpointWorkload]"


class ProtobufAny(BaseModel):
    type_url: "Optional[str]"
    value: "Optional[str]"


class ProtobufFieldMask(BaseModel):
    paths: "Optional[List[str]]"


class ProtobufNullValue(str, Enum):
    NULL_VALUE = "NULL_VALUE"


class RuntimeError(BaseModel):
    error: "Optional[str]"
    code: "Optional[int]"
    message: "Optional[str]"
    details: "Optional[List[ProtobufAny]]"


class RuntimeStreamError(BaseModel):
    grpc_code: "Optional[int]"
    http_code: "Optional[int]"
    message: "Optional[str]"
    http_status: "Optional[str]"
    details: "Optional[List[ProtobufAny]]"


class StreamResultOfV1GetHPImportanceResponse(BaseModel):
    result: "Optional[V1GetHPImportanceResponse]"
    error: "Optional[RuntimeStreamError]"


class StreamResultOfV1GetTrialProfilerAvailableSeriesResponse(BaseModel):
    result: "Optional[V1GetTrialProfilerAvailableSeriesResponse]"
    error: "Optional[RuntimeStreamError]"


class StreamResultOfV1GetTrialProfilerMetricsResponse(BaseModel):
    result: "Optional[V1GetTrialProfilerMetricsResponse]"
    error: "Optional[RuntimeStreamError]"


class StreamResultOfV1MasterLogsResponse(BaseModel):
    result: "Optional[V1MasterLogsResponse]"
    error: "Optional[RuntimeStreamError]"


class StreamResultOfV1MetricBatchesResponse(BaseModel):
    result: "Optional[V1MetricBatchesResponse]"
    error: "Optional[RuntimeStreamError]"


class StreamResultOfV1MetricNamesResponse(BaseModel):
    result: "Optional[V1MetricNamesResponse]"
    error: "Optional[RuntimeStreamError]"


class StreamResultOfV1NotebookLogsResponse(BaseModel):
    result: "Optional[V1NotebookLogsResponse]"
    error: "Optional[RuntimeStreamError]"


class StreamResultOfV1TrialLogsFieldsResponse(BaseModel):
    result: "Optional[V1TrialLogsFieldsResponse]"
    error: "Optional[RuntimeStreamError]"


class StreamResultOfV1TrialLogsResponse(BaseModel):
    result: "Optional[V1TrialLogsResponse]"
    error: "Optional[RuntimeStreamError]"


class StreamResultOfV1TrialsSampleResponse(BaseModel):
    result: "Optional[V1TrialsSampleResponse]"
    error: "Optional[RuntimeStreamError]"


class StreamResultOfV1TrialsSnapshotResponse(BaseModel):
    result: "Optional[V1TrialsSnapshotResponse]"
    error: "Optional[RuntimeStreamError]"


class TrainingLengthUnit(str, Enum):
    UNSPECIFIED = "UNIT_UNSPECIFIED"
    RECORDS = "UNIT_RECORDS"
    BATCHES = "UNIT_BATCHES"
    EPOCHS = "UNIT_EPOCHS"


class TrialEarlyExitExitedReason(str, Enum):
    UNSPECIFIED = "EXITED_REASON_UNSPECIFIED"
    INVALID_HP = "EXITED_REASON_INVALID_HP"
    USER_REQUESTED_STOP = "EXITED_REASON_USER_REQUESTED_STOP"
    INIT_INVALID_HP = "EXITED_REASON_INIT_INVALID_HP"


class TrialProfilerMetricLabelsProfilerMetricType(str, Enum):
    UNSPECIFIED = "PROFILER_METRIC_TYPE_UNSPECIFIED"
    SYSTEM = "PROFILER_METRIC_TYPE_SYSTEM"
    TIMING = "PROFILER_METRIC_TYPE_TIMING"
    MISC = "PROFILER_METRIC_TYPE_MISC"


class TrialsSampleResponseDataPoint(BaseModel):
    batches: "int"
    value: "float"


class Trialv1Trial(BaseModel):
    id: "int"
    experiment_id: "int"
    start_time: "datetime"
    end_time: "Optional[datetime]"
    state: "Determinedexperimentv1State"
    hparams: "Any"
    total_batches_processed: "int"
    best_validation: "Optional[V1MetricsWorkload]"
    latest_validation: "Optional[V1MetricsWorkload]"
    best_checkpoint: "Optional[V1CheckpointWorkload]"
    runner_state: "Optional[str]"


class V1AckAllocationPreemptionSignalRequest(BaseModel):
    allocation_id: "str"


class V1Agent(BaseModel):
    id: "Optional[str]"
    registered_time: "Optional[datetime]"
    slots: "Optional[Dict[str, V1Slot]]"
    containers: "Optional[Dict[str, V1Container]]"
    label: "Optional[str]"
    resource_pool: "Optional[str]"
    addresses: "Optional[List[str]]"
    enabled: "Optional[bool]"
    draining: "Optional[bool]"


class V1AgentUserGroup(BaseModel):
    agent_uid: "Optional[int]"
    agent_gid: "Optional[int]"


class V1AllocationPreemptionSignalResponse(BaseModel):
    preempt: "Optional[bool]"


class V1AllocationRendezvousInfoResponse(BaseModel):
    rendezvous_info: "V1RendezvousInfo"


class V1AwsCustomTag(BaseModel):
    key: "str"
    value: "str"


class V1Checkpoint(BaseModel):
    uuid: "Optional[str]"
    experiment_config: "Optional[Any]"
    experiment_id: "int"
    trial_id: "int"
    hparams: "Optional[Any]"
    batch_number: "int"
    end_time: "Optional[datetime]"
    resources: "Optional[Dict[str, str]]"
    metadata: "Optional[Any]"
    framework: "Optional[str]"
    format: "Optional[str]"
    determined_version: "Optional[str]"
    metrics: "Optional[V1Metrics]"
    validation_state: "Optional[Determinedcheckpointv1State]"
    state: "Determinedcheckpointv1State"
    searcher_metric: "Optional[float]"


class V1CheckpointMetadata(BaseModel):
    trial_id: "int"
    trial_run_id: "int"
    uuid: "str"
    resources: "Dict[str, str]"
    framework: "str"
    format: "str"
    determined_version: "str"
    latest_batch: "Optional[int]"


class V1CheckpointWorkload(BaseModel):
    uuid: "Optional[str]"
    end_time: "Optional[datetime]"
    state: "Determinedcheckpointv1State"
    resources: "Optional[Dict[str, str]]"
    total_batches: "int"


class V1Command(BaseModel):
    id: "str"
    description: "str"
    state: "Determinedtaskv1State"
    start_time: "datetime"
    container: "Optional[V1Container]"
    username: "str"
    resource_pool: "str"
    exit_status: "Optional[str]"
    job_summary: "Optional[V1JobSummary]"


class V1CompleteValidateAfterOperation(BaseModel):
    op: "Optional[V1ValidateAfterOperation]"
    searcher_metric: "Optional[float]"


class V1Container(BaseModel):
    parent: "Optional[str]"
    id: "str"
    state: "Determinedcontainerv1State"
    devices: "Optional[List[V1Device]]"


class V1CreateExperimentRequest(BaseModel):
    model_definition: "Optional[List[V1File]]"
    config: "Optional[str]"
    validate_only: "Optional[bool]"
    parent_id: "Optional[int]"


class V1CreateExperimentResponse(BaseModel):
    experiment: "V1Experiment"
    config: "Any"


class V1CurrentUserResponse(BaseModel):
    user: "V1User"


class V1Device(BaseModel):
    id: "Optional[int]"
    brand: "Optional[str]"
    uuid: "Optional[str]"
    type: "Optional[Determineddevicev1Type]"


class V1DisableAgentRequest(BaseModel):
    agent_id: "Optional[str]"
    drain: "Optional[bool]"


class V1DisableAgentResponse(BaseModel):
    agent: "Optional[V1Agent]"


class V1DisableSlotResponse(BaseModel):
    slot: "Optional[V1Slot]"


class V1EnableAgentResponse(BaseModel):
    agent: "Optional[V1Agent]"


class V1EnableSlotResponse(BaseModel):
    slot: "Optional[V1Slot]"


class V1Experiment(BaseModel):
    id: "int"
    description: "Optional[str]"
    labels: "Optional[List[str]]"
    start_time: "datetime"
    end_time: "Optional[datetime]"
    state: "Determinedexperimentv1State"
    archived: "bool"
    num_trials: "int"
    progress: "Optional[float]"
    username: "str"
    resource_pool: "Optional[str]"
    searcher_type: "str"
    name: "str"
    notes: "Optional[str]"


class V1ExperimentSimulation(BaseModel):
    config: "Optional[Any]"
    seed: "Optional[int]"
    trials: "Optional[List[V1TrialSimulation]]"


class V1File(BaseModel):
    path: "str"
    type: "int"
    content: "str"
    mtime: "str"
    mode: "int"
    uid: "int"
    gid: "int"


class V1FittingPolicy(str, Enum):
    UNSPECIFIED = "FITTING_POLICY_UNSPECIFIED"
    BEST = "FITTING_POLICY_BEST"
    WORST = "FITTING_POLICY_WORST"
    KUBERNETES = "FITTING_POLICY_KUBERNETES"


class V1GetAgentResponse(BaseModel):
    agent: "Optional[V1Agent]"


class V1GetAgentsRequestSortBy(str, Enum):
    UNSPECIFIED = "SORT_BY_UNSPECIFIED"
    ID = "SORT_BY_ID"
    TIME = "SORT_BY_TIME"


class V1GetAgentsResponse(BaseModel):
    agents: "Optional[List[V1Agent]]"
    pagination: "Optional[V1Pagination]"


class V1GetBestSearcherValidationMetricResponse(BaseModel):
    metric: "Optional[float]"


class V1GetCheckpointResponse(BaseModel):
    checkpoint: "Optional[V1Checkpoint]"


class V1GetCommandResponse(BaseModel):
    command: "Optional[V1Command]"
    config: "Optional[Any]"


class V1GetCommandsRequestSortBy(str, Enum):
    UNSPECIFIED = "SORT_BY_UNSPECIFIED"
    ID = "SORT_BY_ID"
    DESCRIPTION = "SORT_BY_DESCRIPTION"
    START_TIME = "SORT_BY_START_TIME"


class V1GetCommandsResponse(BaseModel):
    commands: "Optional[List[V1Command]]"
    pagination: "Optional[V1Pagination]"


class V1GetCurrentTrialSearcherOperationResponse(BaseModel):
    op: "Optional[V1SearcherOperation]"
    completed: "Optional[bool]"


class V1GetExperimentCheckpointsRequestSortBy(str, Enum):
    UNSPECIFIED = "SORT_BY_UNSPECIFIED"
    UUID = "SORT_BY_UUID"
    TRIAL_ID = "SORT_BY_TRIAL_ID"
    BATCH_NUMBER = "SORT_BY_BATCH_NUMBER"
    START_TIME = "SORT_BY_START_TIME"
    END_TIME = "SORT_BY_END_TIME"
    VALIDATION_STATE = "SORT_BY_VALIDATION_STATE"
    STATE = "SORT_BY_STATE"
    SEARCHER_METRIC = "SORT_BY_SEARCHER_METRIC"


class V1GetExperimentCheckpointsResponse(BaseModel):
    checkpoints: "Optional[List[V1Checkpoint]]"
    pagination: "Optional[V1Pagination]"


class V1GetExperimentLabelsResponse(BaseModel):
    labels: "Optional[List[str]]"


class V1GetExperimentResponse(BaseModel):
    experiment: "V1Experiment"
    config: "Any"
    job_summary: "Optional[V1JobSummary]"


class V1GetExperimentTrialsRequestSortBy(str, Enum):
    UNSPECIFIED = "SORT_BY_UNSPECIFIED"
    ID = "SORT_BY_ID"
    START_TIME = "SORT_BY_START_TIME"
    END_TIME = "SORT_BY_END_TIME"
    STATE = "SORT_BY_STATE"
    BEST_VALIDATION_METRIC = "SORT_BY_BEST_VALIDATION_METRIC"
    LATEST_VALIDATION_METRIC = "SORT_BY_LATEST_VALIDATION_METRIC"
    BATCHES_PROCESSED = "SORT_BY_BATCHES_PROCESSED"
    DURATION = "SORT_BY_DURATION"


class V1GetExperimentTrialsResponse(BaseModel):
    trials: "List[Trialv1Trial]"
    pagination: "V1Pagination"


class V1GetExperimentValidationHistoryResponse(BaseModel):
    validation_history: "Optional[List[V1ValidationHistoryEntry]]"


class V1GetExperimentsRequestSortBy(str, Enum):
    UNSPECIFIED = "SORT_BY_UNSPECIFIED"
    ID = "SORT_BY_ID"
    DESCRIPTION = "SORT_BY_DESCRIPTION"
    START_TIME = "SORT_BY_START_TIME"
    END_TIME = "SORT_BY_END_TIME"
    STATE = "SORT_BY_STATE"
    NUM_TRIALS = "SORT_BY_NUM_TRIALS"
    PROGRESS = "SORT_BY_PROGRESS"
    USER = "SORT_BY_USER"
    NAME = "SORT_BY_NAME"


class V1GetExperimentsResponse(BaseModel):
    experiments: "List[V1Experiment]"
    pagination: "V1Pagination"


class V1GetHPImportanceResponse(BaseModel):
    training_metrics: "Dict[str, GetHPImportanceResponseMetricHPImportance]"
    validation_metrics: "Dict[str, GetHPImportanceResponseMetricHPImportance]"


class V1GetJobQueueStatsResponse(BaseModel):
    results: "List[V1RPQueueStat]"


class V1GetJobsResponse(BaseModel):
    pagination: "V1Pagination"
    jobs: "List[V1Job]"


class V1GetMasterConfigResponse(BaseModel):
    config: "Any"


class V1GetMasterResponse(BaseModel):
    version: "str"
    master_id: "str"
    cluster_id: "str"
    cluster_name: "str"
    telemetry_enabled: "Optional[bool]"
    sso_providers: "Optional[List[V1SSOProvider]]"


class V1GetModelDefResponse(BaseModel):
    b64_tgz: "str"


class V1GetModelResponse(BaseModel):
    model: "Optional[V1Model]"


class V1GetModelVersionResponse(BaseModel):
    model_version: "Optional[V1ModelVersion]"


class V1GetModelVersionsRequestSortBy(str, Enum):
    UNSPECIFIED = "SORT_BY_UNSPECIFIED"
    VERSION = "SORT_BY_VERSION"
    CREATION_TIME = "SORT_BY_CREATION_TIME"


class V1GetModelVersionsResponse(BaseModel):
    model: "Optional[V1Model]"
    model_versions: "Optional[List[V1ModelVersion]]"
    pagination: "Optional[V1Pagination]"


class V1GetModelsRequestSortBy(str, Enum):
    UNSPECIFIED = "SORT_BY_UNSPECIFIED"
    NAME = "SORT_BY_NAME"
    DESCRIPTION = "SORT_BY_DESCRIPTION"
    CREATION_TIME = "SORT_BY_CREATION_TIME"
    LAST_UPDATED_TIME = "SORT_BY_LAST_UPDATED_TIME"


class V1GetModelsResponse(BaseModel):
    models: "Optional[List[V1Model]]"
    pagination: "Optional[V1Pagination]"


class V1GetNotebookResponse(BaseModel):
    notebook: "Optional[V1Notebook]"
    config: "Optional[Any]"


class V1GetNotebooksRequestSortBy(str, Enum):
    UNSPECIFIED = "SORT_BY_UNSPECIFIED"
    ID = "SORT_BY_ID"
    DESCRIPTION = "SORT_BY_DESCRIPTION"
    START_TIME = "SORT_BY_START_TIME"


class V1GetNotebooksResponse(BaseModel):
    notebooks: "Optional[List[V1Notebook]]"
    pagination: "Optional[V1Pagination]"


class V1GetResourcePoolsResponse(BaseModel):
    resource_pools: "Optional[List[V1ResourcePool]]"
    pagination: "Optional[V1Pagination]"


class V1GetShellResponse(BaseModel):
    shell: "Optional[V1Shell]"
    config: "Optional[Any]"


class V1GetShellsRequestSortBy(str, Enum):
    UNSPECIFIED = "SORT_BY_UNSPECIFIED"
    ID = "SORT_BY_ID"
    DESCRIPTION = "SORT_BY_DESCRIPTION"
    START_TIME = "SORT_BY_START_TIME"


class V1GetShellsResponse(BaseModel):
    shells: "Optional[List[V1Shell]]"
    pagination: "Optional[V1Pagination]"


class V1GetSlotResponse(BaseModel):
    slot: "Optional[V1Slot]"


class V1GetSlotsResponse(BaseModel):
    slots: "Optional[List[V1Slot]]"


class V1GetTelemetryResponse(BaseModel):
    enabled: "bool"
    segment_key: "Optional[str]"


class V1GetTemplateResponse(BaseModel):
    template: "Optional[V1Template]"


class V1GetTemplatesRequestSortBy(str, Enum):
    UNSPECIFIED = "SORT_BY_UNSPECIFIED"
    NAME = "SORT_BY_NAME"


class V1GetTemplatesResponse(BaseModel):
    templates: "Optional[List[V1Template]]"
    pagination: "Optional[V1Pagination]"


class V1GetTensorboardResponse(BaseModel):
    tensorboard: "Optional[V1Tensorboard]"
    config: "Optional[Any]"


class V1GetTensorboardsRequestSortBy(str, Enum):
    UNSPECIFIED = "SORT_BY_UNSPECIFIED"
    ID = "SORT_BY_ID"
    DESCRIPTION = "SORT_BY_DESCRIPTION"
    START_TIME = "SORT_BY_START_TIME"


class V1GetTensorboardsResponse(BaseModel):
    tensorboards: "Optional[List[V1Tensorboard]]"
    pagination: "Optional[V1Pagination]"


class V1GetTrialCheckpointsRequestSortBy(str, Enum):
    UNSPECIFIED = "SORT_BY_UNSPECIFIED"
    UUID = "SORT_BY_UUID"
    BATCH_NUMBER = "SORT_BY_BATCH_NUMBER"
    START_TIME = "SORT_BY_START_TIME"
    END_TIME = "SORT_BY_END_TIME"
    VALIDATION_STATE = "SORT_BY_VALIDATION_STATE"
    STATE = "SORT_BY_STATE"


class V1GetTrialCheckpointsResponse(BaseModel):
    checkpoints: "Optional[List[V1Checkpoint]]"
    pagination: "Optional[V1Pagination]"


class V1GetTrialProfilerAvailableSeriesResponse(BaseModel):
    labels: "List[V1TrialProfilerMetricLabels]"


class V1GetTrialProfilerMetricsResponse(BaseModel):
    batch: "V1TrialProfilerMetricsBatch"


class V1GetTrialResponse(BaseModel):
    trial: "Trialv1Trial"
    workloads: "Optional[List[GetTrialResponseWorkloadContainer]]"


class V1GetUserResponse(BaseModel):
    user: "Optional[V1User]"


class V1GetUsersResponse(BaseModel):
    users: "Optional[List[V1User]]"


class V1IdleNotebookRequest(BaseModel):
    notebook_id: "Optional[str]"
    idle: "Optional[bool]"


class V1Job(BaseModel):
    summary: "V1JobSummary"
    type: "Determinedjobv1Type"
    submission_time: "datetime"
    user: "str"
    resource_pool: "str"
    is_preemptible: "bool"
    priority: "Optional[int]"
    weight: "Optional[float]"
    entity_id: "str"


class V1JobSummary(BaseModel):
    job_id: "str"
    state: "Determinedjobv1State"


class V1K8PriorityClass(BaseModel):
    priority_class: "Optional[str]"
    priority_value: "Optional[int]"


class V1KillCommandResponse(BaseModel):
    command: "Optional[V1Command]"


class V1KillNotebookResponse(BaseModel):
    notebook: "Optional[V1Notebook]"


class V1KillShellResponse(BaseModel):
    shell: "Optional[V1Shell]"


class V1KillTensorboardResponse(BaseModel):
    tensorboard: "Optional[V1Tensorboard]"


class V1LaunchCommandRequest(BaseModel):
    config: "Optional[Any]"
    template_name: "Optional[str]"
    files: "Optional[List[V1File]]"
    data: "Optional[str]"


class V1LaunchCommandResponse(BaseModel):
    command: "Optional[V1Command]"
    config: "Any"


class V1LaunchNotebookRequest(BaseModel):
    config: "Optional[Any]"
    template_name: "Optional[str]"
    files: "Optional[List[V1File]]"
    preview: "Optional[bool]"


class V1LaunchNotebookResponse(BaseModel):
    notebook: "V1Notebook"
    config: "Any"


class V1LaunchShellRequest(BaseModel):
    config: "Optional[Any]"
    template_name: "Optional[str]"
    files: "Optional[List[V1File]]"
    data: "Optional[str]"


class V1LaunchShellResponse(BaseModel):
    shell: "Optional[V1Shell]"
    config: "Any"


class V1LaunchTensorboardRequest(BaseModel):
    experiment_ids: "Optional[List[int]]"
    trial_ids: "Optional[List[int]]"
    config: "Optional[Any]"
    template_name: "Optional[str]"
    files: "Optional[List[V1File]]"


class V1LaunchTensorboardResponse(BaseModel):
    tensorboard: "V1Tensorboard"
    config: "Any"


class V1LogEntry(BaseModel):
    id: "int"
    message: "Optional[str]"


class V1LogLevel(str, Enum):
    UNSPECIFIED = "LOG_LEVEL_UNSPECIFIED"
    TRACE = "LOG_LEVEL_TRACE"
    DEBUG = "LOG_LEVEL_DEBUG"
    INFO = "LOG_LEVEL_INFO"
    WARNING = "LOG_LEVEL_WARNING"
    ERROR = "LOG_LEVEL_ERROR"
    CRITICAL = "LOG_LEVEL_CRITICAL"


class V1LoginRequest(BaseModel):
    username: "str"
    password: "str"
    is_hashed: "Optional[bool]"


class V1LoginResponse(BaseModel):
    token: "str"
    user: "V1User"


class V1MarkAllocationReservationDaemonRequest(BaseModel):
    allocation_id: "str"
    container_id: "str"


class V1MasterLogsResponse(BaseModel):
    log_entry: "Optional[V1LogEntry]"


class V1MetricBatchesResponse(BaseModel):
    batches: "Optional[List[int]]"


class V1MetricNamesResponse(BaseModel):
    searcher_metric: "Optional[str]"
    training_metrics: "Optional[List[str]]"
    validation_metrics: "Optional[List[str]]"


class V1MetricType(str, Enum):
    UNSPECIFIED = "METRIC_TYPE_UNSPECIFIED"
    TRAINING = "METRIC_TYPE_TRAINING"
    VALIDATION = "METRIC_TYPE_VALIDATION"


class V1Metrics(BaseModel):
    num_inputs: "Optional[int]"
    validation_metrics: "Optional[Any]"


class V1MetricsWorkload(BaseModel):
    end_time: "Optional[datetime]"
    state: "Determinedexperimentv1State"
    metrics: "Optional[Any]"
    num_inputs: "int"
    total_batches: "int"


class V1Model(BaseModel):
    name: "str"
    description: "Optional[str]"
    metadata: "Any"
    creation_time: "datetime"
    last_updated_time: "datetime"


class V1ModelVersion(BaseModel):
    model: "Optional[V1Model]"
    checkpoint: "Optional[V1Checkpoint]"
    version: "Optional[int]"
    creation_time: "Optional[datetime]"


class V1Notebook(BaseModel):
    id: "str"
    description: "str"
    state: "Determinedtaskv1State"
    start_time: "datetime"
    container: "Optional[V1Container]"
    username: "str"
    service_address: "Optional[str]"
    resource_pool: "str"
    exit_status: "Optional[str]"
    job_summary: "Optional[V1JobSummary]"


class V1NotebookLogsResponse(BaseModel):
    log_entry: "Optional[V1LogEntry]"


class V1OrderBy(str, Enum):
    UNSPECIFIED = "ORDER_BY_UNSPECIFIED"
    ASC = "ORDER_BY_ASC"
    DESC = "ORDER_BY_DESC"


class V1Pagination(BaseModel):
    offset: "Optional[int]"
    limit: "Optional[int]"
    start_index: "Optional[int]"
    end_index: "Optional[int]"
    total: "Optional[int]"


class V1PaginationRequest(BaseModel):
    offset: "Optional[int]"
    limit: "Optional[int]"


class V1PatchExperimentResponse(BaseModel):
    experiment: "Optional[V1Experiment]"


class V1PatchModelRequest(BaseModel):
    model: "Optional[V1Model]"


class V1PatchModelResponse(BaseModel):
    model: "Optional[V1Model]"


class V1PostCheckpointMetadataRequest(BaseModel):
    checkpoint: "Optional[V1Checkpoint]"


class V1PostCheckpointMetadataResponse(BaseModel):
    checkpoint: "Optional[V1Checkpoint]"


class V1PostModelResponse(BaseModel):
    model: "Optional[V1Model]"


class V1PostModelVersionRequest(BaseModel):
    model_name: "Optional[str]"
    checkpoint_uuid: "Optional[str]"


class V1PostModelVersionResponse(BaseModel):
    model_version: "Optional[V1ModelVersion]"


class V1PostTrialProfilerMetricsBatchRequest(BaseModel):
    batches: "Optional[List[V1TrialProfilerMetricsBatch]]"


class V1PostUserRequest(BaseModel):
    user: "Optional[V1User]"
    password: "Optional[str]"


class V1PostUserResponse(BaseModel):
    user: "Optional[V1User]"


class V1PreviewHPSearchRequest(BaseModel):
    config: "Optional[Any]"
    seed: "Optional[int]"


class V1PreviewHPSearchResponse(BaseModel):
    simulation: "Optional[V1ExperimentSimulation]"


class V1PutTemplateResponse(BaseModel):
    template: "Optional[V1Template]"


class V1QueueControl(BaseModel):
    job_id: "str"
    queue_position: "Optional[int]"
    priority: "Optional[int]"
    weight: "Optional[int]"
    resource_pool: "Optional[str]"


class V1QueueStats(BaseModel):
    queued_count: "Optional[int]"
    scheduled_count: "Optional[int]"
    preemptible_count: "Optional[int]"


class V1RendezvousInfo(BaseModel):
    addresses: "List[str]"
    rank: "int"


class V1ResourceAllocationAggregatedEntry(BaseModel):
    period_start: "str"
    period: "V1ResourceAllocationAggregationPeriod"
    seconds: "float"
    by_username: "Dict[str, float]"
    by_experiment_label: "Dict[str, float]"
    by_resource_pool: "Dict[str, float]"
    by_agent_label: "Dict[str, float]"


class V1ResourceAllocationAggregatedResponse(BaseModel):
    resource_entries: "List[V1ResourceAllocationAggregatedEntry]"


class V1ResourceAllocationAggregationPeriod(str, Enum):
    UNSPECIFIED = "RESOURCE_ALLOCATION_AGGREGATION_PERIOD_UNSPECIFIED"
    DAILY = "RESOURCE_ALLOCATION_AGGREGATION_PERIOD_DAILY"
    MONTHLY = "RESOURCE_ALLOCATION_AGGREGATION_PERIOD_MONTHLY"


class V1ResourceAllocationRawEntry(BaseModel):
    kind: "Optional[str]"
    start_time: "Optional[datetime]"
    end_time: "Optional[datetime]"
    experiment_id: "Optional[int]"
    username: "Optional[str]"
    labels: "Optional[List[str]]"
    seconds: "Optional[float]"
    slots: "Optional[int]"


class V1ResourceAllocationRawResponse(BaseModel):
    resource_entries: "Optional[List[V1ResourceAllocationRawEntry]]"


class V1ResourcePool(BaseModel):
    name: "str"
    description: "str"
    type: "V1ResourcePoolType"
    num_agents: "int"
    slots_available: "int"
    slots_used: "int"
    slot_type: "Determineddevicev1Type"
    aux_container_capacity: "int"
    aux_containers_running: "int"
    default_compute_pool: "bool"
    default_aux_pool: "bool"
    preemptible: "bool"
    min_agents: "int"
    max_agents: "int"
    slots_per_agent: "Optional[int]"
    aux_container_capacity_per_agent: "int"
    scheduler_type: "V1SchedulerType"
    scheduler_fitting_policy: "V1FittingPolicy"
    location: "str"
    image_id: "str"
    instance_type: "str"
    master_url: "str"
    master_cert_name: "str"
    startup_script: "str"
    container_startup_script: "str"
    agent_docker_network: "str"
    agent_docker_runtime: "str"
    agent_docker_image: "str"
    agent_fluent_image: "str"
    max_idle_agent_period: "float"
    max_agent_starting_period: "float"
    details: "V1ResourcePoolDetail"


class V1ResourcePoolAwsDetail(BaseModel):
    region: "str"
    root_volume_size: "int"
    image_id: "str"
    tag_key: "str"
    tag_value: "str"
    instance_name: "str"
    ssh_key_name: "str"
    public_ip: "bool"
    subnet_id: "Optional[str]"
    security_group_id: "str"
    iam_instance_profile_arn: "str"
    instance_type: "Optional[str]"
    log_group: "Optional[str]"
    log_stream: "Optional[str]"
    spot_enabled: "bool"
    spot_max_price: "Optional[str]"
    custom_tags: "Optional[List[V1AwsCustomTag]]"


class V1ResourcePoolDetail(BaseModel):
    aws: "Optional[V1ResourcePoolAwsDetail]"
    gcp: "Optional[V1ResourcePoolGcpDetail]"
    priority_scheduler: "Optional[V1ResourcePoolPrioritySchedulerDetail]"


class V1ResourcePoolGcpDetail(BaseModel):
    project: "str"
    zone: "str"
    boot_disk_size: "int"
    boot_disk_source_image: "str"
    label_key: "str"
    label_value: "str"
    name_prefix: "str"
    network: "str"
    subnetwork: "Optional[str]"
    external_ip: "bool"
    network_tags: "Optional[List[str]]"
    service_account_email: "str"
    service_account_scopes: "List[str]"
    machine_type: "str"
    gpu_type: "str"
    gpu_num: "int"
    preemptible: "bool"
    operation_timeout_period: "float"


class V1ResourcePoolPrioritySchedulerDetail(BaseModel):
    preemption: "bool"
    default_priority: "int"
    k8_priorities: "Optional[List[V1K8PriorityClass]]"


class V1ResourcePoolType(str, Enum):
    UNSPECIFIED = "RESOURCE_POOL_TYPE_UNSPECIFIED"
    AWS = "RESOURCE_POOL_TYPE_AWS"
    GCP = "RESOURCE_POOL_TYPE_GCP"
    STATIC = "RESOURCE_POOL_TYPE_STATIC"
    K8S = "RESOURCE_POOL_TYPE_K8S"


class V1RPQueueStat(BaseModel):
    stats: "V1QueueStats"
    resource_pool: "str"


class V1RunnableOperation(BaseModel):
    type: "Optional[V1RunnableType]"
    length: "Optional[V1TrainingLength]"


class V1RunnableType(str, Enum):
    UNSPECIFIED = "RUNNABLE_TYPE_UNSPECIFIED"
    TRAIN = "RUNNABLE_TYPE_TRAIN"
    VALIDATE = "RUNNABLE_TYPE_VALIDATE"


class V1SchedulerType(str, Enum):
    UNSPECIFIED = "SCHEDULER_TYPE_UNSPECIFIED"
    PRIORITY = "SCHEDULER_TYPE_PRIORITY"
    FAIR_SHARE = "SCHEDULER_TYPE_FAIR_SHARE"
    ROUND_ROBIN = "SCHEDULER_TYPE_ROUND_ROBIN"
    KUBERNETES = "SCHEDULER_TYPE_KUBERNETES"


class V1SearcherOperation(BaseModel):
    validate_after: "Optional[V1ValidateAfterOperation]"


class V1SetCommandPriorityRequest(BaseModel):
    command_id: "Optional[str]"
    priority: "Optional[int]"


class V1SetCommandPriorityResponse(BaseModel):
    command: "Optional[V1Command]"


class V1SetNotebookPriorityRequest(BaseModel):
    notebook_id: "Optional[str]"
    priority: "Optional[int]"


class V1SetNotebookPriorityResponse(BaseModel):
    notebook: "Optional[V1Notebook]"


class V1SetShellPriorityRequest(BaseModel):
    shell_id: "Optional[str]"
    priority: "Optional[int]"


class V1SetShellPriorityResponse(BaseModel):
    shell: "Optional[V1Shell]"


class V1SetTensorboardPriorityRequest(BaseModel):
    tensorboard_id: "Optional[str]"
    priority: "Optional[int]"


class V1SetTensorboardPriorityResponse(BaseModel):
    tensorboard: "Optional[V1Tensorboard]"


class V1SetUserPasswordResponse(BaseModel):
    user: "Optional[V1User]"


class V1Shell(BaseModel):
    id: "str"
    description: "str"
    state: "Determinedtaskv1State"
    start_time: "datetime"
    container: "Optional[V1Container]"
    private_key: "Optional[str]"
    public_key: "Optional[str]"
    username: "str"
    resource_pool: "str"
    exit_status: "Optional[str]"
    addresses: "Optional[List[Any]]"
    agent_user_group: "Optional[Any]"
    job_summary: "Optional[V1JobSummary]"


class V1Slot(BaseModel):
    id: "Optional[str]"
    device: "Optional[V1Device]"
    enabled: "Optional[bool]"
    container: "Optional[V1Container]"
    draining: "Optional[bool]"


class V1SSOProvider(BaseModel):
    name: "str"
    sso_url: "str"


class V1Template(BaseModel):
    name: "str"
    config: "Any"


class V1Tensorboard(BaseModel):
    id: "str"
    description: "str"
    state: "Determinedtaskv1State"
    start_time: "datetime"
    container: "Optional[V1Container]"
    experiment_ids: "Optional[List[int]]"
    trial_ids: "Optional[List[int]]"
    username: "str"
    service_address: "Optional[str]"
    resource_pool: "str"
    exit_status: "Optional[str]"
    job_summary: "Optional[V1JobSummary]"


class V1TrainingLength(BaseModel):
    unit: "TrainingLengthUnit"
    length: "int"


class V1TrialEarlyExit(BaseModel):
    reason: "TrialEarlyExitExitedReason"


class V1TrialLogsFieldsResponse(BaseModel):
    agent_ids: "Optional[List[str]]"
    container_ids: "Optional[List[str]]"
    rank_ids: "Optional[List[int]]"
    stdtypes: "Optional[List[str]]"
    sources: "Optional[List[str]]"


class V1TrialLogsResponse(BaseModel):
    id: "str"
    timestamp: "datetime"
    message: "str"
    level: "V1LogLevel"


class V1TrialMetrics(BaseModel):
    trial_id: "int"
    trial_run_id: "int"
    latest_batch: "int"
    metrics: "Any"
    batch_metrics: "Optional[List[Any]]"


class V1TrialProfilerMetricLabels(BaseModel):
    trial_id: "int"
    name: "str"
    agent_id: "Optional[str]"
    gpu_uuid: "Optional[str]"
    metric_type: "Optional[TrialProfilerMetricLabelsProfilerMetricType]"


class V1TrialProfilerMetricsBatch(BaseModel):
    values: "List[float]"
    batches: "List[int]"
    timestamps: "List[datetime]"
    labels: "V1TrialProfilerMetricLabels"


class V1TrialRunnerMetadata(BaseModel):
    state: "str"


class V1TrialSimulation(BaseModel):
    operations: "Optional[List[V1RunnableOperation]]"
    occurrences: "Optional[int]"


class V1TrialsSampleResponse(BaseModel):
    trials: "List[V1TrialsSampleResponseTrial]"
    promoted_trials: "List[int]"
    demoted_trials: "List[int]"


class V1TrialsSampleResponseTrial(BaseModel):
    trial_id: "int"
    hparams: "Any"
    data: "List[TrialsSampleResponseDataPoint]"


class V1TrialsSnapshotResponse(BaseModel):
    trials: "List[V1TrialsSnapshotResponseTrial]"


class V1TrialsSnapshotResponseTrial(BaseModel):
    trial_id: "int"
    hparams: "Any"
    metric: "float"
    batches_processed: "int"


class V1User(BaseModel):
    id: "int"
    username: "str"
    admin: "bool"
    active: "bool"
    agent_user_group: "Optional[V1AgentUserGroup]"


class V1ValidateAfterOperation(BaseModel):
    length: "Optional[V1TrainingLength]"


class V1ValidationHistoryEntry(BaseModel):
    trial_id: "int"
    end_time: "datetime"
    searcher_metric: "float"
