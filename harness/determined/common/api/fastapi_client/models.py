from datetime import datetime
from enum import Enum
from typing import Any  # noqa
from typing import Dict, List, Optional

from determined.common.api.fapi import BaseModel, Field


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


class Determinedtaskv1State(str, Enum):
    UNSPECIFIED = "STATE_UNSPECIFIED"
    PENDING = "STATE_PENDING"
    ASSIGNED = "STATE_ASSIGNED"
    PULLING = "STATE_PULLING"
    STARTING = "STATE_STARTING"
    RUNNING = "STATE_RUNNING"
    TERMINATED = "STATE_TERMINATED"
    TERMINATING = "STATE_TERMINATING"


class Devicev1Type(str, Enum):
    UNSPECIFIED = "TYPE_UNSPECIFIED"
    CPU = "TYPE_CPU"
    GPU = "TYPE_GPU"


class GetHPImportanceResponseMetricHPImportance(BaseModel):
    hp_importance: "Optional[Dict[str, float]]" = Field(None, alias="hpImportance")
    experiment_progress: "Optional[float]" = Field(None, alias="experimentProgress")
    error: "Optional[str]" = Field(None, alias="error")
    pending: "Optional[bool]" = Field(None, alias="pending")
    in_progress: "Optional[bool]" = Field(None, alias="inProgress")


class GetTrialResponseWorkloadContainer(BaseModel):
    training: "Optional[V1MetricsWorkload]" = Field(None, alias="training")
    validation: "Optional[V1MetricsWorkload]" = Field(None, alias="validation")
    checkpoint: "Optional[V1CheckpointWorkload]" = Field(None, alias="checkpoint")


class ProtobufAny(BaseModel):
    type_url: "Optional[str]" = Field(None, alias="typeUrl")
    value: "Optional[str]" = Field(None, alias="value")


class ProtobufFieldMask(BaseModel):
    paths: "Optional[List[str]]" = Field(None, alias="paths")


class ProtobufNullValue(str, Enum):
    NULL_VALUE = "NULL_VALUE"


class RuntimeError(BaseModel):
    error: "Optional[str]" = Field(None, alias="error")
    code: "Optional[int]" = Field(None, alias="code")
    message: "Optional[str]" = Field(None, alias="message")
    details: "Optional[List[ProtobufAny]]" = Field(None, alias="details")


class RuntimeStreamError(BaseModel):
    grpc_code: "Optional[int]" = Field(None, alias="grpcCode")
    http_code: "Optional[int]" = Field(None, alias="httpCode")
    message: "Optional[str]" = Field(None, alias="message")
    http_status: "Optional[str]" = Field(None, alias="httpStatus")
    details: "Optional[List[ProtobufAny]]" = Field(None, alias="details")


class StreamResultOfV1GetHPImportanceResponse(BaseModel):
    result: "Optional[V1GetHPImportanceResponse]" = Field(None, alias="result")
    error: "Optional[RuntimeStreamError]" = Field(None, alias="error")


class StreamResultOfV1GetTrialProfilerAvailableSeriesResponse(BaseModel):
    result: "Optional[V1GetTrialProfilerAvailableSeriesResponse]" = Field(None, alias="result")
    error: "Optional[RuntimeStreamError]" = Field(None, alias="error")


class StreamResultOfV1GetTrialProfilerMetricsResponse(BaseModel):
    result: "Optional[V1GetTrialProfilerMetricsResponse]" = Field(None, alias="result")
    error: "Optional[RuntimeStreamError]" = Field(None, alias="error")


class StreamResultOfV1MasterLogsResponse(BaseModel):
    result: "Optional[V1MasterLogsResponse]" = Field(None, alias="result")
    error: "Optional[RuntimeStreamError]" = Field(None, alias="error")


class StreamResultOfV1MetricBatchesResponse(BaseModel):
    result: "Optional[V1MetricBatchesResponse]" = Field(None, alias="result")
    error: "Optional[RuntimeStreamError]" = Field(None, alias="error")


class StreamResultOfV1MetricNamesResponse(BaseModel):
    result: "Optional[V1MetricNamesResponse]" = Field(None, alias="result")
    error: "Optional[RuntimeStreamError]" = Field(None, alias="error")


class StreamResultOfV1NotebookLogsResponse(BaseModel):
    result: "Optional[V1NotebookLogsResponse]" = Field(None, alias="result")
    error: "Optional[RuntimeStreamError]" = Field(None, alias="error")


class StreamResultOfV1TrialLogsFieldsResponse(BaseModel):
    result: "Optional[V1TrialLogsFieldsResponse]" = Field(None, alias="result")
    error: "Optional[RuntimeStreamError]" = Field(None, alias="error")


class StreamResultOfV1TrialLogsResponse(BaseModel):
    result: "Optional[V1TrialLogsResponse]" = Field(None, alias="result")
    error: "Optional[RuntimeStreamError]" = Field(None, alias="error")


class StreamResultOfV1TrialsSampleResponse(BaseModel):
    result: "Optional[V1TrialsSampleResponse]" = Field(None, alias="result")
    error: "Optional[RuntimeStreamError]" = Field(None, alias="error")


class StreamResultOfV1TrialsSnapshotResponse(BaseModel):
    result: "Optional[V1TrialsSnapshotResponse]" = Field(None, alias="result")
    error: "Optional[RuntimeStreamError]" = Field(None, alias="error")


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
    batches: "int" = Field(..., alias="batches")
    value: "float" = Field(..., alias="value")


class Trialv1Trial(BaseModel):
    id: "int" = Field(..., alias="id")
    experiment_id: "int" = Field(..., alias="experimentId")
    start_time: "datetime" = Field(..., alias="startTime")
    end_time: "Optional[datetime]" = Field(None, alias="endTime")
    state: "Determinedexperimentv1State" = Field(..., alias="state")
    hparams: "Any" = Field(..., alias="hparams")
    total_batches_processed: "int" = Field(..., alias="totalBatchesProcessed")
    best_validation: "Optional[V1MetricsWorkload]" = Field(None, alias="bestValidation")
    latest_validation: "Optional[V1MetricsWorkload]" = Field(None, alias="latestValidation")
    best_checkpoint: "Optional[V1CheckpointWorkload]" = Field(None, alias="bestCheckpoint")
    runner_state: "Optional[str]" = Field(None, alias="runnerState")


class V1AckAllocationPreemptionSignalRequest(BaseModel):
    allocation_id: "str" = Field(..., alias="allocationId")


class V1Agent(BaseModel):
    id: "Optional[str]" = Field(None, alias="id")
    registered_time: "Optional[datetime]" = Field(None, alias="registeredTime")
    slots: "Optional[Dict[str, V1Slot]]" = Field(None, alias="slots")
    containers: "Optional[Dict[str, V1Container]]" = Field(None, alias="containers")
    label: "Optional[str]" = Field(None, alias="label")
    resource_pool: "Optional[str]" = Field(None, alias="resourcePool")
    addresses: "Optional[List[str]]" = Field(None, alias="addresses")
    enabled: "Optional[bool]" = Field(None, alias="enabled")
    draining: "Optional[bool]" = Field(None, alias="draining")


class V1AgentUserGroup(BaseModel):
    agent_uid: "Optional[int]" = Field(None, alias="agentUid")
    agent_gid: "Optional[int]" = Field(None, alias="agentGid")


class V1AllocationPreemptionSignalResponse(BaseModel):
    preempt: "Optional[bool]" = Field(None, alias="preempt")


class V1AllocationRendezvousInfoResponse(BaseModel):
    rendezvous_info: "V1RendezvousInfo" = Field(..., alias="rendezvousInfo")


class V1AwsCustomTag(BaseModel):
    key: "str" = Field(..., alias="key")
    value: "str" = Field(..., alias="value")


class V1Checkpoint(BaseModel):
    uuid: "Optional[str]" = Field(None, alias="uuid")
    experiment_config: "Optional[Any]" = Field(None, alias="experimentConfig")
    experiment_id: "int" = Field(..., alias="experimentId")
    trial_id: "int" = Field(..., alias="trialId")
    hparams: "Optional[Any]" = Field(None, alias="hparams")
    batch_number: "int" = Field(..., alias="batchNumber")
    end_time: "Optional[datetime]" = Field(None, alias="endTime")
    resources: "Optional[Dict[str, str]]" = Field(None, alias="resources")
    metadata: "Optional[Any]" = Field(None, alias="metadata")
    framework: "Optional[str]" = Field(None, alias="framework")
    format: "Optional[str]" = Field(None, alias="format")
    determined_version: "Optional[str]" = Field(None, alias="determinedVersion")
    metrics: "Optional[V1Metrics]" = Field(None, alias="metrics")
    validation_state: "Optional[Determinedcheckpointv1State]" = Field(None, alias="validationState")
    state: "Determinedcheckpointv1State" = Field(..., alias="state")
    searcher_metric: "Optional[float]" = Field(None, alias="searcherMetric")


class V1CheckpointMetadata(BaseModel):
    trial_id: "int" = Field(..., alias="trialId")
    trial_run_id: "int" = Field(..., alias="trialRunId")
    uuid: "str" = Field(..., alias="uuid")
    resources: "Dict[str, str]" = Field(..., alias="resources")
    framework: "str" = Field(..., alias="framework")
    format: "str" = Field(..., alias="format")
    determined_version: "str" = Field(..., alias="determinedVersion")
    latest_batch: "Optional[int]" = Field(None, alias="latestBatch")


class V1CheckpointWorkload(BaseModel):
    uuid: "Optional[str]" = Field(None, alias="uuid")
    end_time: "Optional[datetime]" = Field(None, alias="endTime")
    state: "Determinedcheckpointv1State" = Field(..., alias="state")
    resources: "Optional[Dict[str, str]]" = Field(None, alias="resources")
    total_batches: "int" = Field(..., alias="totalBatches")


class V1Command(BaseModel):
    id: "str" = Field(..., alias="id")
    description: "str" = Field(..., alias="description")
    state: "Determinedtaskv1State" = Field(..., alias="state")
    start_time: "datetime" = Field(..., alias="startTime")
    container: "Optional[V1Container]" = Field(None, alias="container")
    username: "str" = Field(..., alias="username")
    resource_pool: "str" = Field(..., alias="resourcePool")
    exit_status: "Optional[str]" = Field(None, alias="exitStatus")


class V1CompleteValidateAfterOperation(BaseModel):
    op: "Optional[V1ValidateAfterOperation]" = Field(None, alias="op")
    searcher_metric: "Optional[float]" = Field(None, alias="searcherMetric")


class V1Container(BaseModel):
    parent: "Optional[str]" = Field(None, alias="parent")
    id: "str" = Field(..., alias="id")
    state: "Determinedcontainerv1State" = Field(..., alias="state")
    devices: "Optional[List[V1Device]]" = Field(None, alias="devices")


class V1CreateExperimentRequest(BaseModel):
    model_definition: "Optional[List[V1File]]" = Field(None, alias="modelDefinition")
    config: "Optional[str]" = Field(None, alias="config")
    validate_only: "Optional[bool]" = Field(None, alias="validateOnly")
    parent_id: "Optional[int]" = Field(None, alias="parentId")


class V1CreateExperimentResponse(BaseModel):
    experiment: "V1Experiment" = Field(..., alias="experiment")
    config: "Any" = Field(..., alias="config")


class V1CurrentUserResponse(BaseModel):
    user: "V1User" = Field(..., alias="user")


class V1Device(BaseModel):
    id: "Optional[int]" = Field(None, alias="id")
    brand: "Optional[str]" = Field(None, alias="brand")
    uuid: "Optional[str]" = Field(None, alias="uuid")
    type: "Optional[Devicev1Type]" = Field(None, alias="type")


class V1DisableAgentRequest(BaseModel):
    agent_id: "Optional[str]" = Field(None, alias="agentId")
    drain: "Optional[bool]" = Field(None, alias="drain")


class V1DisableAgentResponse(BaseModel):
    agent: "Optional[V1Agent]" = Field(None, alias="agent")


class V1DisableSlotResponse(BaseModel):
    slot: "Optional[V1Slot]" = Field(None, alias="slot")


class V1EnableAgentResponse(BaseModel):
    agent: "Optional[V1Agent]" = Field(None, alias="agent")


class V1EnableSlotResponse(BaseModel):
    slot: "Optional[V1Slot]" = Field(None, alias="slot")


class V1Experiment(BaseModel):
    id: "int" = Field(..., alias="id")
    description: "Optional[str]" = Field(None, alias="description")
    labels: "Optional[List[str]]" = Field(None, alias="labels")
    start_time: "datetime" = Field(..., alias="startTime")
    end_time: "Optional[datetime]" = Field(None, alias="endTime")
    state: "Determinedexperimentv1State" = Field(..., alias="state")
    archived: "bool" = Field(..., alias="archived")
    num_trials: "int" = Field(..., alias="numTrials")
    progress: "Optional[float]" = Field(None, alias="progress")
    username: "str" = Field(..., alias="username")
    resource_pool: "Optional[str]" = Field(None, alias="resourcePool")
    searcher_type: "str" = Field(..., alias="searcherType")
    name: "str" = Field(..., alias="name")
    notes: "Optional[str]" = Field(None, alias="notes")


class V1ExperimentSimulation(BaseModel):
    config: "Optional[Any]" = Field(None, alias="config")
    seed: "Optional[int]" = Field(None, alias="seed")
    trials: "Optional[List[V1TrialSimulation]]" = Field(None, alias="trials")


class V1File(BaseModel):
    path: "str" = Field(..., alias="path")
    type: "int" = Field(..., alias="type")
    content: "str" = Field(..., alias="content")
    mtime: "str" = Field(..., alias="mtime")
    mode: "int" = Field(..., alias="mode")
    uid: "int" = Field(..., alias="uid")
    gid: "int" = Field(..., alias="gid")


class V1FittingPolicy(str, Enum):
    UNSPECIFIED = "FITTING_POLICY_UNSPECIFIED"
    BEST = "FITTING_POLICY_BEST"
    WORST = "FITTING_POLICY_WORST"
    KUBERNETES = "FITTING_POLICY_KUBERNETES"


class V1GetAgentResponse(BaseModel):
    agent: "Optional[V1Agent]" = Field(None, alias="agent")


class V1GetAgentsRequestSortBy(str, Enum):
    UNSPECIFIED = "SORT_BY_UNSPECIFIED"
    ID = "SORT_BY_ID"
    TIME = "SORT_BY_TIME"


class V1GetAgentsResponse(BaseModel):
    agents: "Optional[List[V1Agent]]" = Field(None, alias="agents")
    pagination: "Optional[V1Pagination]" = Field(None, alias="pagination")


class V1GetBestSearcherValidationMetricResponse(BaseModel):
    metric: "Optional[float]" = Field(None, alias="metric")


class V1GetCheckpointResponse(BaseModel):
    checkpoint: "Optional[V1Checkpoint]" = Field(None, alias="checkpoint")


class V1GetCommandResponse(BaseModel):
    command: "Optional[V1Command]" = Field(None, alias="command")
    config: "Optional[Any]" = Field(None, alias="config")


class V1GetCommandsRequestSortBy(str, Enum):
    UNSPECIFIED = "SORT_BY_UNSPECIFIED"
    ID = "SORT_BY_ID"
    DESCRIPTION = "SORT_BY_DESCRIPTION"
    START_TIME = "SORT_BY_START_TIME"


class V1GetCommandsResponse(BaseModel):
    commands: "Optional[List[V1Command]]" = Field(None, alias="commands")
    pagination: "Optional[V1Pagination]" = Field(None, alias="pagination")


class V1GetCurrentTrialSearcherOperationResponse(BaseModel):
    op: "Optional[V1SearcherOperation]" = Field(None, alias="op")
    completed: "Optional[bool]" = Field(None, alias="completed")


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
    checkpoints: "Optional[List[V1Checkpoint]]" = Field(None, alias="checkpoints")
    pagination: "Optional[V1Pagination]" = Field(None, alias="pagination")


class V1GetExperimentLabelsResponse(BaseModel):
    labels: "Optional[List[str]]" = Field(None, alias="labels")


class V1GetExperimentResponse(BaseModel):
    experiment: "V1Experiment" = Field(..., alias="experiment")
    config: "Any" = Field(..., alias="config")


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
    trials: "List[Trialv1Trial]" = Field(..., alias="trials")
    pagination: "V1Pagination" = Field(..., alias="pagination")


class V1GetExperimentValidationHistoryResponse(BaseModel):
    validation_history: "Optional[List[V1ValidationHistoryEntry]]" = Field(None, alias="validationHistory")


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
    experiments: "List[V1Experiment]" = Field(..., alias="experiments")
    pagination: "V1Pagination" = Field(..., alias="pagination")


class V1GetHPImportanceResponse(BaseModel):
    training_metrics: "Dict[str, GetHPImportanceResponseMetricHPImportance]" = Field(..., alias="trainingMetrics")
    validation_metrics: "Dict[str, GetHPImportanceResponseMetricHPImportance]" = Field(..., alias="validationMetrics")


class V1GetMasterConfigResponse(BaseModel):
    config: "Any" = Field(..., alias="config")


class V1GetMasterResponse(BaseModel):
    version: "str" = Field(..., alias="version")
    master_id: "str" = Field(..., alias="masterId")
    cluster_id: "str" = Field(..., alias="clusterId")
    cluster_name: "str" = Field(..., alias="clusterName")
    telemetry_enabled: "Optional[bool]" = Field(None, alias="telemetryEnabled")
    sso_providers: "Optional[List[V1SSOProvider]]" = Field(None, alias="ssoProviders")


class V1GetModelDefResponse(BaseModel):
    b64_tgz: "str" = Field(..., alias="b64Tgz")


class V1GetModelResponse(BaseModel):
    model: "Optional[V1Model]" = Field(None, alias="model")


class V1GetModelVersionResponse(BaseModel):
    model_version: "Optional[V1ModelVersion]" = Field(None, alias="modelVersion")


class V1GetModelVersionsRequestSortBy(str, Enum):
    UNSPECIFIED = "SORT_BY_UNSPECIFIED"
    VERSION = "SORT_BY_VERSION"
    CREATION_TIME = "SORT_BY_CREATION_TIME"


class V1GetModelVersionsResponse(BaseModel):
    model: "Optional[V1Model]" = Field(None, alias="model")
    model_versions: "Optional[List[V1ModelVersion]]" = Field(None, alias="modelVersions")
    pagination: "Optional[V1Pagination]" = Field(None, alias="pagination")


class V1GetModelsRequestSortBy(str, Enum):
    UNSPECIFIED = "SORT_BY_UNSPECIFIED"
    NAME = "SORT_BY_NAME"
    DESCRIPTION = "SORT_BY_DESCRIPTION"
    CREATION_TIME = "SORT_BY_CREATION_TIME"
    LAST_UPDATED_TIME = "SORT_BY_LAST_UPDATED_TIME"


class V1GetModelsResponse(BaseModel):
    models: "Optional[List[V1Model]]" = Field(None, alias="models")
    pagination: "Optional[V1Pagination]" = Field(None, alias="pagination")


class V1GetNotebookResponse(BaseModel):
    notebook: "Optional[V1Notebook]" = Field(None, alias="notebook")
    config: "Optional[Any]" = Field(None, alias="config")


class V1GetNotebooksRequestSortBy(str, Enum):
    UNSPECIFIED = "SORT_BY_UNSPECIFIED"
    ID = "SORT_BY_ID"
    DESCRIPTION = "SORT_BY_DESCRIPTION"
    START_TIME = "SORT_BY_START_TIME"


class V1GetNotebooksResponse(BaseModel):
    notebooks: "Optional[List[V1Notebook]]" = Field(None, alias="notebooks")
    pagination: "Optional[V1Pagination]" = Field(None, alias="pagination")


class V1GetResourcePoolsResponse(BaseModel):
    resource_pools: "Optional[List[V1ResourcePool]]" = Field(None, alias="resourcePools")
    pagination: "Optional[V1Pagination]" = Field(None, alias="pagination")


class V1GetShellResponse(BaseModel):
    shell: "Optional[V1Shell]" = Field(None, alias="shell")
    config: "Optional[Any]" = Field(None, alias="config")


class V1GetShellsRequestSortBy(str, Enum):
    UNSPECIFIED = "SORT_BY_UNSPECIFIED"
    ID = "SORT_BY_ID"
    DESCRIPTION = "SORT_BY_DESCRIPTION"
    START_TIME = "SORT_BY_START_TIME"


class V1GetShellsResponse(BaseModel):
    shells: "Optional[List[V1Shell]]" = Field(None, alias="shells")
    pagination: "Optional[V1Pagination]" = Field(None, alias="pagination")


class V1GetSlotResponse(BaseModel):
    slot: "Optional[V1Slot]" = Field(None, alias="slot")


class V1GetSlotsResponse(BaseModel):
    slots: "Optional[List[V1Slot]]" = Field(None, alias="slots")


class V1GetTelemetryResponse(BaseModel):
    enabled: "bool" = Field(..., alias="enabled")
    segment_key: "Optional[str]" = Field(None, alias="segmentKey")


class V1GetTemplateResponse(BaseModel):
    template: "Optional[V1Template]" = Field(None, alias="template")


class V1GetTemplatesRequestSortBy(str, Enum):
    UNSPECIFIED = "SORT_BY_UNSPECIFIED"
    NAME = "SORT_BY_NAME"


class V1GetTemplatesResponse(BaseModel):
    templates: "Optional[List[V1Template]]" = Field(None, alias="templates")
    pagination: "Optional[V1Pagination]" = Field(None, alias="pagination")


class V1GetTensorboardResponse(BaseModel):
    tensorboard: "Optional[V1Tensorboard]" = Field(None, alias="tensorboard")
    config: "Optional[Any]" = Field(None, alias="config")


class V1GetTensorboardsRequestSortBy(str, Enum):
    UNSPECIFIED = "SORT_BY_UNSPECIFIED"
    ID = "SORT_BY_ID"
    DESCRIPTION = "SORT_BY_DESCRIPTION"
    START_TIME = "SORT_BY_START_TIME"


class V1GetTensorboardsResponse(BaseModel):
    tensorboards: "Optional[List[V1Tensorboard]]" = Field(None, alias="tensorboards")
    pagination: "Optional[V1Pagination]" = Field(None, alias="pagination")


class V1GetTrialCheckpointsRequestSortBy(str, Enum):
    UNSPECIFIED = "SORT_BY_UNSPECIFIED"
    UUID = "SORT_BY_UUID"
    BATCH_NUMBER = "SORT_BY_BATCH_NUMBER"
    START_TIME = "SORT_BY_START_TIME"
    END_TIME = "SORT_BY_END_TIME"
    VALIDATION_STATE = "SORT_BY_VALIDATION_STATE"
    STATE = "SORT_BY_STATE"


class V1GetTrialCheckpointsResponse(BaseModel):
    checkpoints: "Optional[List[V1Checkpoint]]" = Field(None, alias="checkpoints")
    pagination: "Optional[V1Pagination]" = Field(None, alias="pagination")


class V1GetTrialProfilerAvailableSeriesResponse(BaseModel):
    labels: "List[V1TrialProfilerMetricLabels]" = Field(..., alias="labels")


class V1GetTrialProfilerMetricsResponse(BaseModel):
    batch: "V1TrialProfilerMetricsBatch" = Field(..., alias="batch")


class V1GetTrialResponse(BaseModel):
    trial: "Trialv1Trial" = Field(..., alias="trial")
    workloads: "Optional[List[GetTrialResponseWorkloadContainer]]" = Field(None, alias="workloads")


class V1GetUserResponse(BaseModel):
    user: "Optional[V1User]" = Field(None, alias="user")


class V1GetUsersResponse(BaseModel):
    users: "Optional[List[V1User]]" = Field(None, alias="users")


class V1IdleNotebookRequest(BaseModel):
    notebook_id: "Optional[str]" = Field(None, alias="notebookId")
    idle: "Optional[bool]" = Field(None, alias="idle")


class V1KillCommandResponse(BaseModel):
    command: "Optional[V1Command]" = Field(None, alias="command")


class V1KillNotebookResponse(BaseModel):
    notebook: "Optional[V1Notebook]" = Field(None, alias="notebook")


class V1KillShellResponse(BaseModel):
    shell: "Optional[V1Shell]" = Field(None, alias="shell")


class V1KillTensorboardResponse(BaseModel):
    tensorboard: "Optional[V1Tensorboard]" = Field(None, alias="tensorboard")


class V1LaunchCommandRequest(BaseModel):
    config: "Optional[Any]" = Field(None, alias="config")
    template_name: "Optional[str]" = Field(None, alias="templateName")
    files: "Optional[List[V1File]]" = Field(None, alias="files")
    data: "Optional[str]" = Field(None, alias="data")


class V1LaunchCommandResponse(BaseModel):
    command: "Optional[V1Command]" = Field(None, alias="command")
    config: "Any" = Field(..., alias="config")


class V1LaunchNotebookRequest(BaseModel):
    config: "Optional[Any]" = Field(None, alias="config")
    template_name: "Optional[str]" = Field(None, alias="templateName")
    files: "Optional[List[V1File]]" = Field(None, alias="files")
    preview: "Optional[bool]" = Field(None, alias="preview")


class V1LaunchNotebookResponse(BaseModel):
    notebook: "V1Notebook" = Field(..., alias="notebook")
    config: "Any" = Field(..., alias="config")


class V1LaunchShellRequest(BaseModel):
    config: "Optional[Any]" = Field(None, alias="config")
    template_name: "Optional[str]" = Field(None, alias="templateName")
    files: "Optional[List[V1File]]" = Field(None, alias="files")
    data: "Optional[str]" = Field(None, alias="data")


class V1LaunchShellResponse(BaseModel):
    shell: "Optional[V1Shell]" = Field(None, alias="shell")
    config: "Any" = Field(..., alias="config")


class V1LaunchTensorboardRequest(BaseModel):
    experiment_ids: "Optional[List[int]]" = Field(None, alias="experimentIds")
    trial_ids: "Optional[List[int]]" = Field(None, alias="trialIds")
    config: "Optional[Any]" = Field(None, alias="config")
    template_name: "Optional[str]" = Field(None, alias="templateName")
    files: "Optional[List[V1File]]" = Field(None, alias="files")


class V1LaunchTensorboardResponse(BaseModel):
    tensorboard: "V1Tensorboard" = Field(..., alias="tensorboard")
    config: "Any" = Field(..., alias="config")


class V1LogEntry(BaseModel):
    id: "int" = Field(..., alias="id")
    message: "Optional[str]" = Field(None, alias="message")


class V1LogLevel(str, Enum):
    UNSPECIFIED = "LOG_LEVEL_UNSPECIFIED"
    TRACE = "LOG_LEVEL_TRACE"
    DEBUG = "LOG_LEVEL_DEBUG"
    INFO = "LOG_LEVEL_INFO"
    WARNING = "LOG_LEVEL_WARNING"
    ERROR = "LOG_LEVEL_ERROR"
    CRITICAL = "LOG_LEVEL_CRITICAL"


class V1LoginRequest(BaseModel):
    username: "str" = Field(..., alias="username")
    password: "str" = Field(..., alias="password")
    is_hashed: "Optional[bool]" = Field(None, alias="isHashed")


class V1LoginResponse(BaseModel):
    token: "str" = Field(..., alias="token")
    user: "V1User" = Field(..., alias="user")


class V1MarkAllocationReservationDaemonRequest(BaseModel):
    allocation_id: "str" = Field(..., alias="allocationId")
    container_id: "str" = Field(..., alias="containerId")


class V1MasterLogsResponse(BaseModel):
    log_entry: "Optional[V1LogEntry]" = Field(None, alias="logEntry")


class V1MetricBatchesResponse(BaseModel):
    batches: "Optional[List[int]]" = Field(None, alias="batches")


class V1MetricNamesResponse(BaseModel):
    searcher_metric: "Optional[str]" = Field(None, alias="searcherMetric")
    training_metrics: "Optional[List[str]]" = Field(None, alias="trainingMetrics")
    validation_metrics: "Optional[List[str]]" = Field(None, alias="validationMetrics")


class V1MetricType(str, Enum):
    UNSPECIFIED = "METRIC_TYPE_UNSPECIFIED"
    TRAINING = "METRIC_TYPE_TRAINING"
    VALIDATION = "METRIC_TYPE_VALIDATION"


class V1Metrics(BaseModel):
    num_inputs: "Optional[int]" = Field(None, alias="numInputs")
    validation_metrics: "Optional[Any]" = Field(None, alias="validationMetrics")


class V1MetricsWorkload(BaseModel):
    end_time: "Optional[datetime]" = Field(None, alias="endTime")
    state: "Determinedexperimentv1State" = Field(..., alias="state")
    metrics: "Optional[Any]" = Field(None, alias="metrics")
    num_inputs: "int" = Field(..., alias="numInputs")
    total_batches: "int" = Field(..., alias="totalBatches")


class V1Model(BaseModel):
    name: "str" = Field(..., alias="name")
    description: "Optional[str]" = Field(None, alias="description")
    metadata: "Any" = Field(..., alias="metadata")
    # creation_time: "datetime" = Field(..., alias="creationTime")
    # last_updated_time: "datetime" = Field(..., alias="lastUpdatedTime")


class V1ModelVersion(BaseModel):
    model: "Optional[V1Model]" = Field(None, alias="model")
    checkpoint: "Optional[V1Checkpoint]" = Field(None, alias="checkpoint")
    version: "Optional[int]" = Field(None, alias="version")
    creation_time: "Optional[datetime]" = Field(None, alias="creationTime")


class V1Notebook(BaseModel):
    id: "str" = Field(..., alias="id")
    description: "str" = Field(..., alias="description")
    state: "Determinedtaskv1State" = Field(..., alias="state")
    start_time: "datetime" = Field(..., alias="startTime")
    container: "Optional[V1Container]" = Field(None, alias="container")
    username: "str" = Field(..., alias="username")
    service_address: "Optional[str]" = Field(None, alias="serviceAddress")
    resource_pool: "str" = Field(..., alias="resourcePool")
    exit_status: "Optional[str]" = Field(None, alias="exitStatus")


class V1NotebookLogsResponse(BaseModel):
    log_entry: "Optional[V1LogEntry]" = Field(None, alias="logEntry")


class V1OrderBy(str, Enum):
    UNSPECIFIED = "ORDER_BY_UNSPECIFIED"
    ASC = "ORDER_BY_ASC"
    DESC = "ORDER_BY_DESC"


class V1Pagination(BaseModel):
    offset: "Optional[int]" = Field(None, alias="offset")
    limit: "Optional[int]" = Field(None, alias="limit")
    start_index: "Optional[int]" = Field(None, alias="startIndex")
    end_index: "Optional[int]" = Field(None, alias="endIndex")
    total: "Optional[int]" = Field(None, alias="total")


class V1PatchExperimentResponse(BaseModel):
    experiment: "Optional[V1Experiment]" = Field(None, alias="experiment")


class V1PatchModelRequest(BaseModel):
    model: "Optional[V1Model]" = Field(None, alias="model")


class V1PatchModelResponse(BaseModel):
    model: "Optional[V1Model]" = Field(None, alias="model")


class V1PostCheckpointMetadataRequest(BaseModel):
    checkpoint: "Optional[V1Checkpoint]" = Field(None, alias="checkpoint")


class V1PostCheckpointMetadataResponse(BaseModel):
    checkpoint: "Optional[V1Checkpoint]" = Field(None, alias="checkpoint")


class V1PostModelResponse(BaseModel):
    model: "Optional[V1Model]" = Field(None, alias="model")


class V1PostModelVersionRequest(BaseModel):
    model_name: "Optional[str]" = Field(None, alias="modelName")
    checkpoint_uuid: "Optional[str]" = Field(None, alias="checkpointUuid")


class V1PostModelVersionResponse(BaseModel):
    model_version: "Optional[V1ModelVersion]" = Field(None, alias="modelVersion")


class V1PostTrialProfilerMetricsBatchRequest(BaseModel):
    batches: "Optional[List[V1TrialProfilerMetricsBatch]]" = Field(None, alias="batches")


class V1PostUserRequest(BaseModel):
    user: "Optional[V1User]" = Field(None, alias="user")
    password: "Optional[str]" = Field(None, alias="password")


class V1PostUserResponse(BaseModel):
    user: "Optional[V1User]" = Field(None, alias="user")


class V1PreviewHPSearchRequest(BaseModel):
    config: "Optional[Any]" = Field(None, alias="config")
    seed: "Optional[int]" = Field(None, alias="seed")


class V1PreviewHPSearchResponse(BaseModel):
    simulation: "Optional[V1ExperimentSimulation]" = Field(None, alias="simulation")


class V1PutTemplateResponse(BaseModel):
    template: "Optional[V1Template]" = Field(None, alias="template")


class V1RendezvousInfo(BaseModel):
    addresses: "List[str]" = Field(..., alias="addresses")
    rank: "int" = Field(..., alias="rank")


class V1ResourceAllocationAggregatedEntry(BaseModel):
    period_start: "str" = Field(..., alias="periodStart")
    period: "V1ResourceAllocationAggregationPeriod" = Field(..., alias="period")
    seconds: "float" = Field(..., alias="seconds")
    by_username: "Dict[str, float]" = Field(..., alias="byUsername")
    by_experiment_label: "Dict[str, float]" = Field(..., alias="byExperimentLabel")
    by_resource_pool: "Dict[str, float]" = Field(..., alias="byResourcePool")
    by_agent_label: "Dict[str, float]" = Field(..., alias="byAgentLabel")


class V1ResourceAllocationAggregatedResponse(BaseModel):
    resource_entries: "List[V1ResourceAllocationAggregatedEntry]" = Field(..., alias="resourceEntries")


class V1ResourceAllocationAggregationPeriod(str, Enum):
    UNSPECIFIED = "RESOURCE_ALLOCATION_AGGREGATION_PERIOD_UNSPECIFIED"
    DAILY = "RESOURCE_ALLOCATION_AGGREGATION_PERIOD_DAILY"
    MONTHLY = "RESOURCE_ALLOCATION_AGGREGATION_PERIOD_MONTHLY"


class V1ResourceAllocationRawEntry(BaseModel):
    kind: "Optional[str]" = Field(None, alias="kind")
    start_time: "Optional[datetime]" = Field(None, alias="startTime")
    end_time: "Optional[datetime]" = Field(None, alias="endTime")
    experiment_id: "Optional[int]" = Field(None, alias="experimentId")
    username: "Optional[str]" = Field(None, alias="username")
    labels: "Optional[List[str]]" = Field(None, alias="labels")
    seconds: "Optional[float]" = Field(None, alias="seconds")
    slots: "Optional[int]" = Field(None, alias="slots")


class V1ResourceAllocationRawResponse(BaseModel):
    resource_entries: "Optional[List[V1ResourceAllocationRawEntry]]" = Field(None, alias="resourceEntries")


class V1ResourcePool(BaseModel):
    name: "str" = Field(..., alias="name")
    description: "str" = Field(..., alias="description")
    type: "V1ResourcePoolType" = Field(..., alias="type")
    num_agents: "int" = Field(..., alias="numAgents")
    slots_available: "int" = Field(..., alias="slotsAvailable")
    slots_used: "int" = Field(..., alias="slotsUsed")
    slot_type: "Devicev1Type" = Field(..., alias="slotType")
    aux_container_capacity: "int" = Field(..., alias="auxContainerCapacity")
    aux_containers_running: "int" = Field(..., alias="auxContainersRunning")
    default_compute_pool: "bool" = Field(..., alias="defaultComputePool")
    default_aux_pool: "bool" = Field(..., alias="defaultAuxPool")
    preemptible: "bool" = Field(..., alias="preemptible")
    min_agents: "int" = Field(..., alias="minAgents")
    max_agents: "int" = Field(..., alias="maxAgents")
    slots_per_agent: "Optional[int]" = Field(None, alias="slotsPerAgent")
    aux_container_capacity_per_agent: "int" = Field(..., alias="auxContainerCapacityPerAgent")
    scheduler_type: "V1SchedulerType" = Field(..., alias="schedulerType")
    scheduler_fitting_policy: "V1FittingPolicy" = Field(..., alias="schedulerFittingPolicy")
    location: "str" = Field(..., alias="location")
    image_id: "str" = Field(..., alias="imageId")
    instance_type: "str" = Field(..., alias="instanceType")
    master_url: "str" = Field(..., alias="masterUrl")
    master_cert_name: "str" = Field(..., alias="masterCertName")
    startup_script: "str" = Field(..., alias="startupScript")
    container_startup_script: "str" = Field(..., alias="containerStartupScript")
    agent_docker_network: "str" = Field(..., alias="agentDockerNetwork")
    agent_docker_runtime: "str" = Field(..., alias="agentDockerRuntime")
    agent_docker_image: "str" = Field(..., alias="agentDockerImage")
    agent_fluent_image: "str" = Field(..., alias="agentFluentImage")
    max_idle_agent_period: "float" = Field(..., alias="maxIdleAgentPeriod")
    max_agent_starting_period: "float" = Field(..., alias="maxAgentStartingPeriod")
    details: "V1ResourcePoolDetail" = Field(..., alias="details")


class V1ResourcePoolAwsDetail(BaseModel):
    region: "str" = Field(..., alias="region")
    root_volume_size: "int" = Field(..., alias="rootVolumeSize")
    image_id: "str" = Field(..., alias="imageId")
    tag_key: "str" = Field(..., alias="tagKey")
    tag_value: "str" = Field(..., alias="tagValue")
    instance_name: "str" = Field(..., alias="instanceName")
    ssh_key_name: "str" = Field(..., alias="sshKeyName")
    public_ip: "bool" = Field(..., alias="publicIp")
    subnet_id: "Optional[str]" = Field(None, alias="subnetId")
    security_group_id: "str" = Field(..., alias="securityGroupId")
    iam_instance_profile_arn: "str" = Field(..., alias="iamInstanceProfileArn")
    instance_type: "Optional[str]" = Field(None, alias="instanceType")
    log_group: "Optional[str]" = Field(None, alias="logGroup")
    log_stream: "Optional[str]" = Field(None, alias="logStream")
    spot_enabled: "bool" = Field(..., alias="spotEnabled")
    spot_max_price: "Optional[str]" = Field(None, alias="spotMaxPrice")
    custom_tags: "Optional[List[V1AwsCustomTag]]" = Field(None, alias="customTags")


class V1ResourcePoolDetail(BaseModel):
    aws: "Optional[V1ResourcePoolAwsDetail]" = Field(None, alias="aws")
    gcp: "Optional[V1ResourcePoolGcpDetail]" = Field(None, alias="gcp")
    priority_scheduler: "Optional[V1ResourcePoolPrioritySchedulerDetail]" = Field(None, alias="priorityScheduler")


class V1ResourcePoolGcpDetail(BaseModel):
    project: "str" = Field(..., alias="project")
    zone: "str" = Field(..., alias="zone")
    boot_disk_size: "int" = Field(..., alias="bootDiskSize")
    boot_disk_source_image: "str" = Field(..., alias="bootDiskSourceImage")
    label_key: "str" = Field(..., alias="labelKey")
    label_value: "str" = Field(..., alias="labelValue")
    name_prefix: "str" = Field(..., alias="namePrefix")
    network: "str" = Field(..., alias="network")
    subnetwork: "Optional[str]" = Field(None, alias="subnetwork")
    external_ip: "bool" = Field(..., alias="externalIp")
    network_tags: "Optional[List[str]]" = Field(None, alias="networkTags")
    service_account_email: "str" = Field(..., alias="serviceAccountEmail")
    service_account_scopes: "List[str]" = Field(..., alias="serviceAccountScopes")
    machine_type: "str" = Field(..., alias="machineType")
    gpu_type: "str" = Field(..., alias="gpuType")
    gpu_num: "int" = Field(..., alias="gpuNum")
    preemptible: "bool" = Field(..., alias="preemptible")
    operation_timeout_period: "float" = Field(..., alias="operationTimeoutPeriod")


class V1ResourcePoolPrioritySchedulerDetail(BaseModel):
    preemption: "bool" = Field(..., alias="preemption")
    default_priority: "int" = Field(..., alias="defaultPriority")


class V1ResourcePoolType(str, Enum):
    UNSPECIFIED = "RESOURCE_POOL_TYPE_UNSPECIFIED"
    AWS = "RESOURCE_POOL_TYPE_AWS"
    GCP = "RESOURCE_POOL_TYPE_GCP"
    STATIC = "RESOURCE_POOL_TYPE_STATIC"
    K8S = "RESOURCE_POOL_TYPE_K8S"


class V1RunnableOperation(BaseModel):
    type: "Optional[V1RunnableType]" = Field(None, alias="type")
    length: "Optional[V1TrainingLength]" = Field(None, alias="length")


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
    validate_after: "Optional[V1ValidateAfterOperation]" = Field(None, alias="validateAfter")


class V1SetCommandPriorityRequest(BaseModel):
    command_id: "Optional[str]" = Field(None, alias="commandId")
    priority: "Optional[int]" = Field(None, alias="priority")


class V1SetCommandPriorityResponse(BaseModel):
    command: "Optional[V1Command]" = Field(None, alias="command")


class V1SetNotebookPriorityRequest(BaseModel):
    notebook_id: "Optional[str]" = Field(None, alias="notebookId")
    priority: "Optional[int]" = Field(None, alias="priority")


class V1SetNotebookPriorityResponse(BaseModel):
    notebook: "Optional[V1Notebook]" = Field(None, alias="notebook")


class V1SetShellPriorityRequest(BaseModel):
    shell_id: "Optional[str]" = Field(None, alias="shellId")
    priority: "Optional[int]" = Field(None, alias="priority")


class V1SetShellPriorityResponse(BaseModel):
    shell: "Optional[V1Shell]" = Field(None, alias="shell")


class V1SetTensorboardPriorityRequest(BaseModel):
    tensorboard_id: "Optional[str]" = Field(None, alias="tensorboardId")
    priority: "Optional[int]" = Field(None, alias="priority")


class V1SetTensorboardPriorityResponse(BaseModel):
    tensorboard: "Optional[V1Tensorboard]" = Field(None, alias="tensorboard")


class V1SetUserPasswordResponse(BaseModel):
    user: "Optional[V1User]" = Field(None, alias="user")


class V1Shell(BaseModel):
    id: "str" = Field(..., alias="id")
    description: "str" = Field(..., alias="description")
    state: "Determinedtaskv1State" = Field(..., alias="state")
    start_time: "datetime" = Field(..., alias="startTime")
    container: "Optional[V1Container]" = Field(None, alias="container")
    private_key: "Optional[str]" = Field(None, alias="privateKey")
    public_key: "Optional[str]" = Field(None, alias="publicKey")
    username: "str" = Field(..., alias="username")
    resource_pool: "str" = Field(..., alias="resourcePool")
    exit_status: "Optional[str]" = Field(None, alias="exitStatus")
    addresses: "Optional[List[Any]]" = Field(None, alias="addresses")
    agent_user_group: "Optional[Any]" = Field(None, alias="agentUserGroup")


class V1Slot(BaseModel):
    id: "Optional[str]" = Field(None, alias="id")
    device: "Optional[V1Device]" = Field(None, alias="device")
    enabled: "Optional[bool]" = Field(None, alias="enabled")
    container: "Optional[V1Container]" = Field(None, alias="container")
    draining: "Optional[bool]" = Field(None, alias="draining")


class V1SSOProvider(BaseModel):
    name: "str" = Field(..., alias="name")
    sso_url: "str" = Field(..., alias="ssoUrl")


class V1Template(BaseModel):
    name: "str" = Field(..., alias="name")
    config: "Any" = Field(..., alias="config")


class V1Tensorboard(BaseModel):
    id: "str" = Field(..., alias="id")
    description: "str" = Field(..., alias="description")
    state: "Determinedtaskv1State" = Field(..., alias="state")
    start_time: "datetime" = Field(..., alias="startTime")
    container: "Optional[V1Container]" = Field(None, alias="container")
    experiment_ids: "Optional[List[int]]" = Field(None, alias="experimentIds")
    trial_ids: "Optional[List[int]]" = Field(None, alias="trialIds")
    username: "str" = Field(..., alias="username")
    service_address: "Optional[str]" = Field(None, alias="serviceAddress")
    resource_pool: "str" = Field(..., alias="resourcePool")
    exit_status: "Optional[str]" = Field(None, alias="exitStatus")


class V1TrainingLength(BaseModel):
    unit: "TrainingLengthUnit" = Field(..., alias="unit")
    length: "int" = Field(..., alias="length")


class V1TrialEarlyExit(BaseModel):
    reason: "TrialEarlyExitExitedReason" = Field(..., alias="reason")


class V1TrialLogsFieldsResponse(BaseModel):
    agent_ids: "Optional[List[str]]" = Field(None, alias="agentIds")
    container_ids: "Optional[List[str]]" = Field(None, alias="containerIds")
    rank_ids: "Optional[List[int]]" = Field(None, alias="rankIds")
    stdtypes: "Optional[List[str]]" = Field(None, alias="stdtypes")
    sources: "Optional[List[str]]" = Field(None, alias="sources")


class V1TrialLogsResponse(BaseModel):
    id: "str" = Field(..., alias="id")
    timestamp: "datetime" = Field(..., alias="timestamp")
    message: "str" = Field(..., alias="message")
    level: "V1LogLevel" = Field(..., alias="level")


class V1TrialMetrics(BaseModel):
    trial_id: "int" = Field(..., alias="trialId")
    trial_run_id: "int" = Field(..., alias="trialRunId")
    latest_batch: "int" = Field(..., alias="latestBatch")
    metrics: "Any" = Field(..., alias="metrics")
    batch_metrics: "Optional[List[Any]]" = Field(None, alias="batchMetrics")


class V1TrialProfilerMetricLabels(BaseModel):
    trial_id: "int" = Field(..., alias="trialId")
    name: "str" = Field(..., alias="name")
    agent_id: "Optional[str]" = Field(None, alias="agentId")
    gpu_uuid: "Optional[str]" = Field(None, alias="gpuUuid")
    metric_type: "Optional[TrialProfilerMetricLabelsProfilerMetricType]" = Field(None, alias="metricType")


class V1TrialProfilerMetricsBatch(BaseModel):
    values: "List[float]" = Field(..., alias="values")
    batches: "List[int]" = Field(..., alias="batches")
    timestamps: "List[datetime]" = Field(..., alias="timestamps")
    labels: "V1TrialProfilerMetricLabels" = Field(..., alias="labels")


class V1TrialRunnerMetadata(BaseModel):
    state: "str" = Field(..., alias="state")


class V1TrialSimulation(BaseModel):
    operations: "Optional[List[V1RunnableOperation]]" = Field(None, alias="operations")
    occurrences: "Optional[int]" = Field(None, alias="occurrences")


class V1TrialsSampleResponse(BaseModel):
    trials: "List[V1TrialsSampleResponseTrial]" = Field(..., alias="trials")
    promoted_trials: "List[int]" = Field(..., alias="promotedTrials")
    demoted_trials: "List[int]" = Field(..., alias="demotedTrials")


class V1TrialsSampleResponseTrial(BaseModel):
    trial_id: "int" = Field(..., alias="trialId")
    hparams: "Any" = Field(..., alias="hparams")
    data: "List[TrialsSampleResponseDataPoint]" = Field(..., alias="data")


class V1TrialsSnapshotResponse(BaseModel):
    trials: "List[V1TrialsSnapshotResponseTrial]" = Field(..., alias="trials")


class V1TrialsSnapshotResponseTrial(BaseModel):
    trial_id: "int" = Field(..., alias="trialId")
    hparams: "Any" = Field(..., alias="hparams")
    metric: "float" = Field(..., alias="metric")
    batches_processed: "int" = Field(..., alias="batchesProcessed")


class V1User(BaseModel):
    id: "int" = Field(..., alias="id")
    username: "str" = Field(..., alias="username")
    admin: "bool" = Field(..., alias="admin")
    active: "bool" = Field(..., alias="active")
    agent_user_group: "Optional[V1AgentUserGroup]" = Field(None, alias="agentUserGroup")


class V1ValidateAfterOperation(BaseModel):
    length: "Optional[V1TrainingLength]" = Field(None, alias="length")


class V1ValidationHistoryEntry(BaseModel):
    trial_id: "int" = Field(..., alias="trialId")
    end_time: "datetime" = Field(..., alias="endTime")
    searcher_metric: "float" = Field(..., alias="searcherMetric")

# def eval_model_types(text: str) -> type:
#     return eval(text)
