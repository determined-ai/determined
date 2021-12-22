# The contents of this file are programatically generated.
import enum
import math
import typing as t

import requests

# flake8: noqa
Json = t.Any


def dump_float(val: t.Any) -> t.Any:
    if math.isnan(val):
        return "Nan"
    if math.isinf(val):
        return "Infinity" if val > 0 else "-Infinity"
    return val


Request = t.Callable[
    [
        str,  # method
        str,  # path
        t.Optional[t.Dict[str, t.Any]],  # params
        t.Any,  # json
        t.Optional[str],  # data
        t.Optional[t.Dict[str, t.Any]],  # headers
        t.Optional[int],  # timeout
    ],
    requests.Response,
]


class GetHPImportanceResponseMetricHPImportance:
    def __init__(
        self,
        error: "t.Optional[str]" = None,
        experimentProgress: "t.Optional[float]" = None,
        hpImportance: "t.Optional[t.Dict[str, float]]" = None,
        inProgress: "t.Optional[bool]" = None,
        pending: "t.Optional[bool]" = None,
    ):
        self.hpImportance = hpImportance
        self.experimentProgress = experimentProgress
        self.error = error
        self.pending = pending
        self.inProgress = inProgress

    @classmethod
    def from_json(cls, obj: Json) -> "GetHPImportanceResponseMetricHPImportance":
        return cls(
            hpImportance={k: float(v) for k, v in obj["hpImportance"].items()} if obj.get("hpImportance", None) is not None else None,
            experimentProgress=float(obj["experimentProgress"]) if obj.get("experimentProgress", None) is not None else None,
            error=obj["error"] if obj.get("error", None) is not None else None,
            pending=obj["pending"] if obj.get("pending", None) is not None else None,
            inProgress=obj["inProgress"] if obj.get("inProgress", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "hpImportance": {k: dump_float(v) for k, v in self.hpImportance.items()} if self.hpImportance is not None else None,
            "experimentProgress": dump_float(self.experimentProgress) if self.experimentProgress is not None else None,
            "error": self.error if self.error is not None else None,
            "pending": self.pending if self.pending is not None else None,
            "inProgress": self.inProgress if self.inProgress is not None else None,
        }

class GetTrialResponseWorkloadContainer:
    def __init__(
        self,
        checkpoint: "t.Optional[v1CheckpointWorkload]" = None,
        training: "t.Optional[v1MetricsWorkload]" = None,
        validation: "t.Optional[v1MetricsWorkload]" = None,
    ):
        self.training = training
        self.validation = validation
        self.checkpoint = checkpoint

    @classmethod
    def from_json(cls, obj: Json) -> "GetTrialResponseWorkloadContainer":
        return cls(
            training=v1MetricsWorkload.from_json(obj["training"]) if obj.get("training", None) is not None else None,
            validation=v1MetricsWorkload.from_json(obj["validation"]) if obj.get("validation", None) is not None else None,
            checkpoint=v1CheckpointWorkload.from_json(obj["checkpoint"]) if obj.get("checkpoint", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "training": self.training.to_json() if self.training is not None else None,
            "validation": self.validation.to_json() if self.validation is not None else None,
            "checkpoint": self.checkpoint.to_json() if self.checkpoint is not None else None,
        }

class TrainingLengthUnit(enum.Enum):
    UNIT_UNSPECIFIED = "UNIT_UNSPECIFIED"
    UNIT_RECORDS = "UNIT_RECORDS"
    UNIT_BATCHES = "UNIT_BATCHES"
    UNIT_EPOCHS = "UNIT_EPOCHS"

class TrialEarlyExitExitedReason(enum.Enum):
    EXITED_REASON_UNSPECIFIED = "EXITED_REASON_UNSPECIFIED"
    EXITED_REASON_INVALID_HP = "EXITED_REASON_INVALID_HP"
    EXITED_REASON_USER_REQUESTED_STOP = "EXITED_REASON_USER_REQUESTED_STOP"
    EXITED_REASON_INIT_INVALID_HP = "EXITED_REASON_INIT_INVALID_HP"

class TrialProfilerMetricLabelsProfilerMetricType(enum.Enum):
    PROFILER_METRIC_TYPE_UNSPECIFIED = "PROFILER_METRIC_TYPE_UNSPECIFIED"
    PROFILER_METRIC_TYPE_SYSTEM = "PROFILER_METRIC_TYPE_SYSTEM"
    PROFILER_METRIC_TYPE_TIMING = "PROFILER_METRIC_TYPE_TIMING"
    PROFILER_METRIC_TYPE_MISC = "PROFILER_METRIC_TYPE_MISC"

class TrialsSampleResponseDataPoint:
    def __init__(
        self,
        batches: int,
        value: float,
    ):
        self.batches = batches
        self.value = value

    @classmethod
    def from_json(cls, obj: Json) -> "TrialsSampleResponseDataPoint":
        return cls(
            batches=obj["batches"],
            value=float(obj["value"]),
        )

    def to_json(self) -> t.Any:
        return {
            "batches": self.batches,
            "value": dump_float(self.value),
        }

class determinedcheckpointv1State(enum.Enum):
    STATE_UNSPECIFIED = "STATE_UNSPECIFIED"
    STATE_ACTIVE = "STATE_ACTIVE"
    STATE_COMPLETED = "STATE_COMPLETED"
    STATE_ERROR = "STATE_ERROR"
    STATE_DELETED = "STATE_DELETED"

class determinedcontainerv1State(enum.Enum):
    STATE_UNSPECIFIED = "STATE_UNSPECIFIED"
    STATE_ASSIGNED = "STATE_ASSIGNED"
    STATE_PULLING = "STATE_PULLING"
    STATE_STARTING = "STATE_STARTING"
    STATE_RUNNING = "STATE_RUNNING"
    STATE_TERMINATED = "STATE_TERMINATED"

class determineddevicev1Type(enum.Enum):
    TYPE_UNSPECIFIED = "TYPE_UNSPECIFIED"
    TYPE_CPU = "TYPE_CPU"
    TYPE_GPU = "TYPE_GPU"

class determinedexperimentv1State(enum.Enum):
    STATE_UNSPECIFIED = "STATE_UNSPECIFIED"
    STATE_ACTIVE = "STATE_ACTIVE"
    STATE_PAUSED = "STATE_PAUSED"
    STATE_STOPPING_COMPLETED = "STATE_STOPPING_COMPLETED"
    STATE_STOPPING_CANCELED = "STATE_STOPPING_CANCELED"
    STATE_STOPPING_ERROR = "STATE_STOPPING_ERROR"
    STATE_COMPLETED = "STATE_COMPLETED"
    STATE_CANCELED = "STATE_CANCELED"
    STATE_ERROR = "STATE_ERROR"
    STATE_DELETED = "STATE_DELETED"
    STATE_DELETING = "STATE_DELETING"
    STATE_DELETE_FAILED = "STATE_DELETE_FAILED"
    STATE_STOPPING_KILLED = "STATE_STOPPING_KILLED"

class determinedjobv1State(enum.Enum):
    STATE_UNSPECIFIED = "STATE_UNSPECIFIED"
    STATE_QUEUED = "STATE_QUEUED"
    STATE_SCHEDULED = "STATE_SCHEDULED"
    STATE_SCHEDULED_BACKFILLED = "STATE_SCHEDULED_BACKFILLED"

class determinedjobv1Type(enum.Enum):
    TYPE_UNSPECIFIED = "TYPE_UNSPECIFIED"
    TYPE_EXPERIMENT = "TYPE_EXPERIMENT"
    TYPE_NOTEBOOK = "TYPE_NOTEBOOK"
    TYPE_TENSORBOARD = "TYPE_TENSORBOARD"
    TYPE_SHELL = "TYPE_SHELL"
    TYPE_COMMAND = "TYPE_COMMAND"

class determinedtaskv1State(enum.Enum):
    STATE_UNSPECIFIED = "STATE_UNSPECIFIED"
    STATE_PENDING = "STATE_PENDING"
    STATE_ASSIGNED = "STATE_ASSIGNED"
    STATE_PULLING = "STATE_PULLING"
    STATE_STARTING = "STATE_STARTING"
    STATE_RUNNING = "STATE_RUNNING"
    STATE_TERMINATED = "STATE_TERMINATED"
    STATE_TERMINATING = "STATE_TERMINATING"

class protobufAny:
    def __init__(
        self,
        typeUrl: "t.Optional[str]" = None,
        value: "t.Optional[str]" = None,
    ):
        self.typeUrl = typeUrl
        self.value = value

    @classmethod
    def from_json(cls, obj: Json) -> "protobufAny":
        return cls(
            typeUrl=obj["typeUrl"] if obj.get("typeUrl", None) is not None else None,
            value=obj["value"] if obj.get("value", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "typeUrl": self.typeUrl if self.typeUrl is not None else None,
            "value": self.value if self.value is not None else None,
        }

class protobufFieldMask:
    def __init__(
        self,
        paths: "t.Optional[t.Sequence[str]]" = None,
    ):
        self.paths = paths

    @classmethod
    def from_json(cls, obj: Json) -> "protobufFieldMask":
        return cls(
            paths=obj["paths"] if obj.get("paths", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "paths": self.paths if self.paths is not None else None,
        }

class protobufNullValue(enum.Enum):
    NULL_VALUE = "NULL_VALUE"

class runtimeError:
    def __init__(
        self,
        code: "t.Optional[int]" = None,
        details: "t.Optional[t.Sequence[protobufAny]]" = None,
        error: "t.Optional[str]" = None,
        message: "t.Optional[str]" = None,
    ):
        self.error = error
        self.code = code
        self.message = message
        self.details = details

    @classmethod
    def from_json(cls, obj: Json) -> "runtimeError":
        return cls(
            error=obj["error"] if obj.get("error", None) is not None else None,
            code=obj["code"] if obj.get("code", None) is not None else None,
            message=obj["message"] if obj.get("message", None) is not None else None,
            details=[protobufAny.from_json(x) for x in obj["details"]] if obj.get("details", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "error": self.error if self.error is not None else None,
            "code": self.code if self.code is not None else None,
            "message": self.message if self.message is not None else None,
            "details": [x.to_json() for x in self.details] if self.details is not None else None,
        }

class runtimeStreamError:
    def __init__(
        self,
        details: "t.Optional[t.Sequence[protobufAny]]" = None,
        grpcCode: "t.Optional[int]" = None,
        httpCode: "t.Optional[int]" = None,
        httpStatus: "t.Optional[str]" = None,
        message: "t.Optional[str]" = None,
    ):
        self.grpcCode = grpcCode
        self.httpCode = httpCode
        self.message = message
        self.httpStatus = httpStatus
        self.details = details

    @classmethod
    def from_json(cls, obj: Json) -> "runtimeStreamError":
        return cls(
            grpcCode=obj["grpcCode"] if obj.get("grpcCode", None) is not None else None,
            httpCode=obj["httpCode"] if obj.get("httpCode", None) is not None else None,
            message=obj["message"] if obj.get("message", None) is not None else None,
            httpStatus=obj["httpStatus"] if obj.get("httpStatus", None) is not None else None,
            details=[protobufAny.from_json(x) for x in obj["details"]] if obj.get("details", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "grpcCode": self.grpcCode if self.grpcCode is not None else None,
            "httpCode": self.httpCode if self.httpCode is not None else None,
            "message": self.message if self.message is not None else None,
            "httpStatus": self.httpStatus if self.httpStatus is not None else None,
            "details": [x.to_json() for x in self.details] if self.details is not None else None,
        }

class trialv1Trial:
    def __init__(
        self,
        experimentId: int,
        hparams: "t.Dict[str, t.Any]",
        id: int,
        startTime: str,
        state: "determinedexperimentv1State",
        totalBatchesProcessed: int,
        bestCheckpoint: "t.Optional[v1CheckpointWorkload]" = None,
        bestValidation: "t.Optional[v1MetricsWorkload]" = None,
        endTime: "t.Optional[str]" = None,
        latestValidation: "t.Optional[v1MetricsWorkload]" = None,
        runnerState: "t.Optional[str]" = None,
        wallClockTime: "t.Optional[float]" = None,
    ):
        self.id = id
        self.experimentId = experimentId
        self.startTime = startTime
        self.endTime = endTime
        self.state = state
        self.hparams = hparams
        self.totalBatchesProcessed = totalBatchesProcessed
        self.bestValidation = bestValidation
        self.latestValidation = latestValidation
        self.bestCheckpoint = bestCheckpoint
        self.runnerState = runnerState
        self.wallClockTime = wallClockTime

    @classmethod
    def from_json(cls, obj: Json) -> "trialv1Trial":
        return cls(
            id=obj["id"],
            experimentId=obj["experimentId"],
            startTime=obj["startTime"],
            endTime=obj["endTime"] if obj.get("endTime", None) is not None else None,
            state=obj["state"],
            hparams=obj["hparams"],
            totalBatchesProcessed=obj["totalBatchesProcessed"],
            bestValidation=v1MetricsWorkload.from_json(obj["bestValidation"]) if obj.get("bestValidation", None) is not None else None,
            latestValidation=v1MetricsWorkload.from_json(obj["latestValidation"]) if obj.get("latestValidation", None) is not None else None,
            bestCheckpoint=v1CheckpointWorkload.from_json(obj["bestCheckpoint"]) if obj.get("bestCheckpoint", None) is not None else None,
            runnerState=obj["runnerState"] if obj.get("runnerState", None) is not None else None,
            wallClockTime=float(obj["wallClockTime"]) if obj.get("wallClockTime", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "id": self.id,
            "experimentId": self.experimentId,
            "startTime": self.startTime,
            "endTime": self.endTime if self.endTime is not None else None,
            "state": self.state,
            "hparams": self.hparams,
            "totalBatchesProcessed": self.totalBatchesProcessed,
            "bestValidation": self.bestValidation.to_json() if self.bestValidation is not None else None,
            "latestValidation": self.latestValidation.to_json() if self.latestValidation is not None else None,
            "bestCheckpoint": self.bestCheckpoint.to_json() if self.bestCheckpoint is not None else None,
            "runnerState": self.runnerState if self.runnerState is not None else None,
            "wallClockTime": dump_float(self.wallClockTime) if self.wallClockTime is not None else None,
        }

class v1AckAllocationPreemptionSignalRequest:
    def __init__(
        self,
        allocationId: str,
    ):
        self.allocationId = allocationId

    @classmethod
    def from_json(cls, obj: Json) -> "v1AckAllocationPreemptionSignalRequest":
        return cls(
            allocationId=obj["allocationId"],
        )

    def to_json(self) -> t.Any:
        return {
            "allocationId": self.allocationId,
        }

class v1Agent:
    def __init__(
        self,
        addresses: "t.Optional[t.Sequence[str]]" = None,
        containers: "t.Optional[t.Dict[str, v1Container]]" = None,
        draining: "t.Optional[bool]" = None,
        enabled: "t.Optional[bool]" = None,
        id: "t.Optional[str]" = None,
        label: "t.Optional[str]" = None,
        registeredTime: "t.Optional[str]" = None,
        resourcePool: "t.Optional[str]" = None,
        slots: "t.Optional[t.Dict[str, v1Slot]]" = None,
    ):
        self.id = id
        self.registeredTime = registeredTime
        self.slots = slots
        self.containers = containers
        self.label = label
        self.resourcePool = resourcePool
        self.addresses = addresses
        self.enabled = enabled
        self.draining = draining

    @classmethod
    def from_json(cls, obj: Json) -> "v1Agent":
        return cls(
            id=obj["id"] if obj.get("id", None) is not None else None,
            registeredTime=obj["registeredTime"] if obj.get("registeredTime", None) is not None else None,
            slots={k: v1Slot.from_json(v) for k, v in obj["slots"].items()} if obj.get("slots", None) is not None else None,
            containers={k: v1Container.from_json(v) for k, v in obj["containers"].items()} if obj.get("containers", None) is not None else None,
            label=obj["label"] if obj.get("label", None) is not None else None,
            resourcePool=obj["resourcePool"] if obj.get("resourcePool", None) is not None else None,
            addresses=obj["addresses"] if obj.get("addresses", None) is not None else None,
            enabled=obj["enabled"] if obj.get("enabled", None) is not None else None,
            draining=obj["draining"] if obj.get("draining", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "id": self.id if self.id is not None else None,
            "registeredTime": self.registeredTime if self.registeredTime is not None else None,
            "slots": {k: v.to_json() for k, v in self.slots.items()} if self.slots is not None else None,
            "containers": {k: v.to_json() for k, v in self.containers.items()} if self.containers is not None else None,
            "label": self.label if self.label is not None else None,
            "resourcePool": self.resourcePool if self.resourcePool is not None else None,
            "addresses": self.addresses if self.addresses is not None else None,
            "enabled": self.enabled if self.enabled is not None else None,
            "draining": self.draining if self.draining is not None else None,
        }

class v1AgentUserGroup:
    def __init__(
        self,
        agentGid: "t.Optional[int]" = None,
        agentUid: "t.Optional[int]" = None,
    ):
        self.agentUid = agentUid
        self.agentGid = agentGid

    @classmethod
    def from_json(cls, obj: Json) -> "v1AgentUserGroup":
        return cls(
            agentUid=obj["agentUid"] if obj.get("agentUid", None) is not None else None,
            agentGid=obj["agentGid"] if obj.get("agentGid", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "agentUid": self.agentUid if self.agentUid is not None else None,
            "agentGid": self.agentGid if self.agentGid is not None else None,
        }

class v1AllocationPreemptionSignalResponse:
    def __init__(
        self,
        preempt: "t.Optional[bool]" = None,
    ):
        self.preempt = preempt

    @classmethod
    def from_json(cls, obj: Json) -> "v1AllocationPreemptionSignalResponse":
        return cls(
            preempt=obj["preempt"] if obj.get("preempt", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "preempt": self.preempt if self.preempt is not None else None,
        }

class v1AllocationRendezvousInfoResponse:
    def __init__(
        self,
        rendezvousInfo: "v1RendezvousInfo",
    ):
        self.rendezvousInfo = rendezvousInfo

    @classmethod
    def from_json(cls, obj: Json) -> "v1AllocationRendezvousInfoResponse":
        return cls(
            rendezvousInfo=v1RendezvousInfo.from_json(obj["rendezvousInfo"]),
        )

    def to_json(self) -> t.Any:
        return {
            "rendezvousInfo": self.rendezvousInfo.to_json(),
        }

class v1AwsCustomTag:
    def __init__(
        self,
        key: str,
        value: str,
    ):
        self.key = key
        self.value = value

    @classmethod
    def from_json(cls, obj: Json) -> "v1AwsCustomTag":
        return cls(
            key=obj["key"],
            value=obj["value"],
        )

    def to_json(self) -> t.Any:
        return {
            "key": self.key,
            "value": self.value,
        }

class v1Checkpoint:
    def __init__(
        self,
        batchNumber: int,
        experimentId: int,
        state: "determinedcheckpointv1State",
        trialId: int,
        determinedVersion: "t.Optional[str]" = None,
        endTime: "t.Optional[str]" = None,
        experimentConfig: "t.Optional[t.Dict[str, t.Any]]" = None,
        format: "t.Optional[str]" = None,
        framework: "t.Optional[str]" = None,
        hparams: "t.Optional[t.Dict[str, t.Any]]" = None,
        metadata: "t.Optional[t.Dict[str, t.Any]]" = None,
        metrics: "t.Optional[v1Metrics]" = None,
        resources: "t.Optional[t.Dict[str, str]]" = None,
        searcherMetric: "t.Optional[float]" = None,
        uuid: "t.Optional[str]" = None,
        validationState: "t.Optional[determinedcheckpointv1State]" = None,
    ):
        self.uuid = uuid
        self.experimentConfig = experimentConfig
        self.experimentId = experimentId
        self.trialId = trialId
        self.hparams = hparams
        self.batchNumber = batchNumber
        self.endTime = endTime
        self.resources = resources
        self.metadata = metadata
        self.framework = framework
        self.format = format
        self.determinedVersion = determinedVersion
        self.metrics = metrics
        self.validationState = validationState
        self.state = state
        self.searcherMetric = searcherMetric

    @classmethod
    def from_json(cls, obj: Json) -> "v1Checkpoint":
        return cls(
            uuid=obj["uuid"] if obj.get("uuid", None) is not None else None,
            experimentConfig=obj["experimentConfig"] if obj.get("experimentConfig", None) is not None else None,
            experimentId=obj["experimentId"],
            trialId=obj["trialId"],
            hparams=obj["hparams"] if obj.get("hparams", None) is not None else None,
            batchNumber=obj["batchNumber"],
            endTime=obj["endTime"] if obj.get("endTime", None) is not None else None,
            resources=obj["resources"] if obj.get("resources", None) is not None else None,
            metadata=obj["metadata"] if obj.get("metadata", None) is not None else None,
            framework=obj["framework"] if obj.get("framework", None) is not None else None,
            format=obj["format"] if obj.get("format", None) is not None else None,
            determinedVersion=obj["determinedVersion"] if obj.get("determinedVersion", None) is not None else None,
            metrics=v1Metrics.from_json(obj["metrics"]) if obj.get("metrics", None) is not None else None,
            validationState=obj["validationState"] if obj.get("validationState", None) is not None else None,
            state=obj["state"],
            searcherMetric=float(obj["searcherMetric"]) if obj.get("searcherMetric", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "uuid": self.uuid if self.uuid is not None else None,
            "experimentConfig": self.experimentConfig if self.experimentConfig is not None else None,
            "experimentId": self.experimentId,
            "trialId": self.trialId,
            "hparams": self.hparams if self.hparams is not None else None,
            "batchNumber": self.batchNumber,
            "endTime": self.endTime if self.endTime is not None else None,
            "resources": self.resources if self.resources is not None else None,
            "metadata": self.metadata if self.metadata is not None else None,
            "framework": self.framework if self.framework is not None else None,
            "format": self.format if self.format is not None else None,
            "determinedVersion": self.determinedVersion if self.determinedVersion is not None else None,
            "metrics": self.metrics.to_json() if self.metrics is not None else None,
            "validationState": self.validationState if self.validationState is not None else None,
            "state": self.state,
            "searcherMetric": dump_float(self.searcherMetric) if self.searcherMetric is not None else None,
        }

class v1CheckpointMetadata:
    def __init__(
        self,
        determinedVersion: str,
        format: str,
        framework: str,
        resources: "t.Dict[str, str]",
        trialId: int,
        trialRunId: int,
        uuid: str,
        latestBatch: "t.Optional[int]" = None,
    ):
        self.trialId = trialId
        self.trialRunId = trialRunId
        self.uuid = uuid
        self.resources = resources
        self.framework = framework
        self.format = format
        self.determinedVersion = determinedVersion
        self.latestBatch = latestBatch

    @classmethod
    def from_json(cls, obj: Json) -> "v1CheckpointMetadata":
        return cls(
            trialId=obj["trialId"],
            trialRunId=obj["trialRunId"],
            uuid=obj["uuid"],
            resources=obj["resources"],
            framework=obj["framework"],
            format=obj["format"],
            determinedVersion=obj["determinedVersion"],
            latestBatch=obj["latestBatch"] if obj.get("latestBatch", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "trialId": self.trialId,
            "trialRunId": self.trialRunId,
            "uuid": self.uuid,
            "resources": self.resources,
            "framework": self.framework,
            "format": self.format,
            "determinedVersion": self.determinedVersion,
            "latestBatch": self.latestBatch if self.latestBatch is not None else None,
        }

class v1CheckpointWorkload:
    def __init__(
        self,
        state: "determinedcheckpointv1State",
        totalBatches: int,
        endTime: "t.Optional[str]" = None,
        resources: "t.Optional[t.Dict[str, str]]" = None,
        uuid: "t.Optional[str]" = None,
    ):
        self.uuid = uuid
        self.endTime = endTime
        self.state = state
        self.resources = resources
        self.totalBatches = totalBatches

    @classmethod
    def from_json(cls, obj: Json) -> "v1CheckpointWorkload":
        return cls(
            uuid=obj["uuid"] if obj.get("uuid", None) is not None else None,
            endTime=obj["endTime"] if obj.get("endTime", None) is not None else None,
            state=obj["state"],
            resources=obj["resources"] if obj.get("resources", None) is not None else None,
            totalBatches=obj["totalBatches"],
        )

    def to_json(self) -> t.Any:
        return {
            "uuid": self.uuid if self.uuid is not None else None,
            "endTime": self.endTime if self.endTime is not None else None,
            "state": self.state,
            "resources": self.resources if self.resources is not None else None,
            "totalBatches": self.totalBatches,
        }

class v1Command:
    def __init__(
        self,
        description: str,
        id: str,
        jobId: str,
        resourcePool: str,
        startTime: str,
        state: "determinedtaskv1State",
        username: str,
        container: "t.Optional[v1Container]" = None,
        exitStatus: "t.Optional[str]" = None,
    ):
        self.id = id
        self.description = description
        self.state = state
        self.startTime = startTime
        self.container = container
        self.username = username
        self.resourcePool = resourcePool
        self.exitStatus = exitStatus
        self.jobId = jobId

    @classmethod
    def from_json(cls, obj: Json) -> "v1Command":
        return cls(
            id=obj["id"],
            description=obj["description"],
            state=obj["state"],
            startTime=obj["startTime"],
            container=v1Container.from_json(obj["container"]) if obj.get("container", None) is not None else None,
            username=obj["username"],
            resourcePool=obj["resourcePool"],
            exitStatus=obj["exitStatus"] if obj.get("exitStatus", None) is not None else None,
            jobId=obj["jobId"],
        )

    def to_json(self) -> t.Any:
        return {
            "id": self.id,
            "description": self.description,
            "state": self.state,
            "startTime": self.startTime,
            "container": self.container.to_json() if self.container is not None else None,
            "username": self.username,
            "resourcePool": self.resourcePool,
            "exitStatus": self.exitStatus if self.exitStatus is not None else None,
            "jobId": self.jobId,
        }

class v1CompleteValidateAfterOperation:
    def __init__(
        self,
        op: "t.Optional[v1ValidateAfterOperation]" = None,
        searcherMetric: "t.Optional[float]" = None,
    ):
        self.op = op
        self.searcherMetric = searcherMetric

    @classmethod
    def from_json(cls, obj: Json) -> "v1CompleteValidateAfterOperation":
        return cls(
            op=v1ValidateAfterOperation.from_json(obj["op"]) if obj.get("op", None) is not None else None,
            searcherMetric=float(obj["searcherMetric"]) if obj.get("searcherMetric", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "op": self.op.to_json() if self.op is not None else None,
            "searcherMetric": dump_float(self.searcherMetric) if self.searcherMetric is not None else None,
        }

class v1Container:
    def __init__(
        self,
        id: str,
        state: "determinedcontainerv1State",
        devices: "t.Optional[t.Sequence[v1Device]]" = None,
        parent: "t.Optional[str]" = None,
    ):
        self.parent = parent
        self.id = id
        self.state = state
        self.devices = devices

    @classmethod
    def from_json(cls, obj: Json) -> "v1Container":
        return cls(
            parent=obj["parent"] if obj.get("parent", None) is not None else None,
            id=obj["id"],
            state=obj["state"],
            devices=[v1Device.from_json(x) for x in obj["devices"]] if obj.get("devices", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "parent": self.parent if self.parent is not None else None,
            "id": self.id,
            "state": self.state,
            "devices": [x.to_json() for x in self.devices] if self.devices is not None else None,
        }

class v1CreateExperimentRequest:
    def __init__(
        self,
        activate: "t.Optional[bool]" = None,
        config: "t.Optional[str]" = None,
        modelDefinition: "t.Optional[t.Sequence[v1File]]" = None,
        parentId: "t.Optional[int]" = None,
        validateOnly: "t.Optional[bool]" = None,
    ):
        self.modelDefinition = modelDefinition
        self.config = config
        self.validateOnly = validateOnly
        self.parentId = parentId
        self.activate = activate

    @classmethod
    def from_json(cls, obj: Json) -> "v1CreateExperimentRequest":
        return cls(
            modelDefinition=[v1File.from_json(x) for x in obj["modelDefinition"]] if obj.get("modelDefinition", None) is not None else None,
            config=obj["config"] if obj.get("config", None) is not None else None,
            validateOnly=obj["validateOnly"] if obj.get("validateOnly", None) is not None else None,
            parentId=obj["parentId"] if obj.get("parentId", None) is not None else None,
            activate=obj["activate"] if obj.get("activate", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "modelDefinition": [x.to_json() for x in self.modelDefinition] if self.modelDefinition is not None else None,
            "config": self.config if self.config is not None else None,
            "validateOnly": self.validateOnly if self.validateOnly is not None else None,
            "parentId": self.parentId if self.parentId is not None else None,
            "activate": self.activate if self.activate is not None else None,
        }

class v1CreateExperimentResponse:
    def __init__(
        self,
        config: "t.Dict[str, t.Any]",
        experiment: "v1Experiment",
    ):
        self.experiment = experiment
        self.config = config

    @classmethod
    def from_json(cls, obj: Json) -> "v1CreateExperimentResponse":
        return cls(
            experiment=v1Experiment.from_json(obj["experiment"]),
            config=obj["config"],
        )

    def to_json(self) -> t.Any:
        return {
            "experiment": self.experiment.to_json(),
            "config": self.config,
        }

class v1CurrentUserResponse:
    def __init__(
        self,
        user: "v1User",
    ):
        self.user = user

    @classmethod
    def from_json(cls, obj: Json) -> "v1CurrentUserResponse":
        return cls(
            user=v1User.from_json(obj["user"]),
        )

    def to_json(self) -> t.Any:
        return {
            "user": self.user.to_json(),
        }

class v1Device:
    def __init__(
        self,
        brand: "t.Optional[str]" = None,
        id: "t.Optional[int]" = None,
        type: "t.Optional[determineddevicev1Type]" = None,
        uuid: "t.Optional[str]" = None,
    ):
        self.id = id
        self.brand = brand
        self.uuid = uuid
        self.type = type

    @classmethod
    def from_json(cls, obj: Json) -> "v1Device":
        return cls(
            id=obj["id"] if obj.get("id", None) is not None else None,
            brand=obj["brand"] if obj.get("brand", None) is not None else None,
            uuid=obj["uuid"] if obj.get("uuid", None) is not None else None,
            type=obj["type"] if obj.get("type", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "id": self.id if self.id is not None else None,
            "brand": self.brand if self.brand is not None else None,
            "uuid": self.uuid if self.uuid is not None else None,
            "type": self.type if self.type is not None else None,
        }

class v1DisableAgentRequest:
    def __init__(
        self,
        agentId: "t.Optional[str]" = None,
        drain: "t.Optional[bool]" = None,
    ):
        self.agentId = agentId
        self.drain = drain

    @classmethod
    def from_json(cls, obj: Json) -> "v1DisableAgentRequest":
        return cls(
            agentId=obj["agentId"] if obj.get("agentId", None) is not None else None,
            drain=obj["drain"] if obj.get("drain", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "agentId": self.agentId if self.agentId is not None else None,
            "drain": self.drain if self.drain is not None else None,
        }

class v1DisableAgentResponse:
    def __init__(
        self,
        agent: "t.Optional[v1Agent]" = None,
    ):
        self.agent = agent

    @classmethod
    def from_json(cls, obj: Json) -> "v1DisableAgentResponse":
        return cls(
            agent=v1Agent.from_json(obj["agent"]) if obj.get("agent", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "agent": self.agent.to_json() if self.agent is not None else None,
        }

class v1DisableSlotResponse:
    def __init__(
        self,
        slot: "t.Optional[v1Slot]" = None,
    ):
        self.slot = slot

    @classmethod
    def from_json(cls, obj: Json) -> "v1DisableSlotResponse":
        return cls(
            slot=v1Slot.from_json(obj["slot"]) if obj.get("slot", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "slot": self.slot.to_json() if self.slot is not None else None,
        }

class v1EnableAgentResponse:
    def __init__(
        self,
        agent: "t.Optional[v1Agent]" = None,
    ):
        self.agent = agent

    @classmethod
    def from_json(cls, obj: Json) -> "v1EnableAgentResponse":
        return cls(
            agent=v1Agent.from_json(obj["agent"]) if obj.get("agent", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "agent": self.agent.to_json() if self.agent is not None else None,
        }

class v1EnableSlotResponse:
    def __init__(
        self,
        slot: "t.Optional[v1Slot]" = None,
    ):
        self.slot = slot

    @classmethod
    def from_json(cls, obj: Json) -> "v1EnableSlotResponse":
        return cls(
            slot=v1Slot.from_json(obj["slot"]) if obj.get("slot", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "slot": self.slot.to_json() if self.slot is not None else None,
        }

class v1Experiment:
    def __init__(
        self,
        archived: bool,
        id: int,
        jobId: str,
        name: str,
        numTrials: int,
        searcherType: str,
        startTime: str,
        state: "determinedexperimentv1State",
        username: str,
        description: "t.Optional[str]" = None,
        endTime: "t.Optional[str]" = None,
        labels: "t.Optional[t.Sequence[str]]" = None,
        notes: "t.Optional[str]" = None,
        progress: "t.Optional[float]" = None,
        resourcePool: "t.Optional[str]" = None,
    ):
        self.id = id
        self.description = description
        self.labels = labels
        self.startTime = startTime
        self.endTime = endTime
        self.state = state
        self.archived = archived
        self.numTrials = numTrials
        self.progress = progress
        self.username = username
        self.resourcePool = resourcePool
        self.searcherType = searcherType
        self.name = name
        self.notes = notes
        self.jobId = jobId

    @classmethod
    def from_json(cls, obj: Json) -> "v1Experiment":
        return cls(
            id=obj["id"],
            description=obj["description"] if obj.get("description", None) is not None else None,
            labels=obj["labels"] if obj.get("labels", None) is not None else None,
            startTime=obj["startTime"],
            endTime=obj["endTime"] if obj.get("endTime", None) is not None else None,
            state=obj["state"],
            archived=obj["archived"],
            numTrials=obj["numTrials"],
            progress=float(obj["progress"]) if obj.get("progress", None) is not None else None,
            username=obj["username"],
            resourcePool=obj["resourcePool"] if obj.get("resourcePool", None) is not None else None,
            searcherType=obj["searcherType"],
            name=obj["name"],
            notes=obj["notes"] if obj.get("notes", None) is not None else None,
            jobId=obj["jobId"],
        )

    def to_json(self) -> t.Any:
        return {
            "id": self.id,
            "description": self.description if self.description is not None else None,
            "labels": self.labels if self.labels is not None else None,
            "startTime": self.startTime,
            "endTime": self.endTime if self.endTime is not None else None,
            "state": self.state,
            "archived": self.archived,
            "numTrials": self.numTrials,
            "progress": dump_float(self.progress) if self.progress is not None else None,
            "username": self.username,
            "resourcePool": self.resourcePool if self.resourcePool is not None else None,
            "searcherType": self.searcherType,
            "name": self.name,
            "notes": self.notes if self.notes is not None else None,
            "jobId": self.jobId,
        }

class v1ExperimentSimulation:
    def __init__(
        self,
        config: "t.Optional[t.Dict[str, t.Any]]" = None,
        seed: "t.Optional[int]" = None,
        trials: "t.Optional[t.Sequence[v1TrialSimulation]]" = None,
    ):
        self.config = config
        self.seed = seed
        self.trials = trials

    @classmethod
    def from_json(cls, obj: Json) -> "v1ExperimentSimulation":
        return cls(
            config=obj["config"] if obj.get("config", None) is not None else None,
            seed=obj["seed"] if obj.get("seed", None) is not None else None,
            trials=[v1TrialSimulation.from_json(x) for x in obj["trials"]] if obj.get("trials", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "config": self.config if self.config is not None else None,
            "seed": self.seed if self.seed is not None else None,
            "trials": [x.to_json() for x in self.trials] if self.trials is not None else None,
        }

class v1File:
    def __init__(
        self,
        content: str,
        gid: int,
        mode: int,
        mtime: str,
        path: str,
        type: int,
        uid: int,
    ):
        self.path = path
        self.type = type
        self.content = content
        self.mtime = mtime
        self.mode = mode
        self.uid = uid
        self.gid = gid

    @classmethod
    def from_json(cls, obj: Json) -> "v1File":
        return cls(
            path=obj["path"],
            type=obj["type"],
            content=obj["content"],
            mtime=obj["mtime"],
            mode=obj["mode"],
            uid=obj["uid"],
            gid=obj["gid"],
        )

    def to_json(self) -> t.Any:
        return {
            "path": self.path,
            "type": self.type,
            "content": self.content,
            "mtime": self.mtime,
            "mode": self.mode,
            "uid": self.uid,
            "gid": self.gid,
        }

class v1FittingPolicy(enum.Enum):
    FITTING_POLICY_UNSPECIFIED = "FITTING_POLICY_UNSPECIFIED"
    FITTING_POLICY_BEST = "FITTING_POLICY_BEST"
    FITTING_POLICY_WORST = "FITTING_POLICY_WORST"
    FITTING_POLICY_KUBERNETES = "FITTING_POLICY_KUBERNETES"

class v1GetAgentResponse:
    def __init__(
        self,
        agent: "t.Optional[v1Agent]" = None,
    ):
        self.agent = agent

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetAgentResponse":
        return cls(
            agent=v1Agent.from_json(obj["agent"]) if obj.get("agent", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "agent": self.agent.to_json() if self.agent is not None else None,
        }

class v1GetAgentsRequestSortBy(enum.Enum):
    SORT_BY_UNSPECIFIED = "SORT_BY_UNSPECIFIED"
    SORT_BY_ID = "SORT_BY_ID"
    SORT_BY_TIME = "SORT_BY_TIME"

class v1GetAgentsResponse:
    def __init__(
        self,
        agents: "t.Optional[t.Sequence[v1Agent]]" = None,
        pagination: "t.Optional[v1Pagination]" = None,
    ):
        self.agents = agents
        self.pagination = pagination

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetAgentsResponse":
        return cls(
            agents=[v1Agent.from_json(x) for x in obj["agents"]] if obj.get("agents", None) is not None else None,
            pagination=v1Pagination.from_json(obj["pagination"]) if obj.get("pagination", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "agents": [x.to_json() for x in self.agents] if self.agents is not None else None,
            "pagination": self.pagination.to_json() if self.pagination is not None else None,
        }

class v1GetBestSearcherValidationMetricResponse:
    def __init__(
        self,
        metric: "t.Optional[float]" = None,
    ):
        self.metric = metric

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetBestSearcherValidationMetricResponse":
        return cls(
            metric=float(obj["metric"]) if obj.get("metric", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "metric": dump_float(self.metric) if self.metric is not None else None,
        }

class v1GetCheckpointResponse:
    def __init__(
        self,
        checkpoint: "t.Optional[v1Checkpoint]" = None,
    ):
        self.checkpoint = checkpoint

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetCheckpointResponse":
        return cls(
            checkpoint=v1Checkpoint.from_json(obj["checkpoint"]) if obj.get("checkpoint", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "checkpoint": self.checkpoint.to_json() if self.checkpoint is not None else None,
        }

class v1GetCommandResponse:
    def __init__(
        self,
        command: "t.Optional[v1Command]" = None,
        config: "t.Optional[t.Dict[str, t.Any]]" = None,
    ):
        self.command = command
        self.config = config

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetCommandResponse":
        return cls(
            command=v1Command.from_json(obj["command"]) if obj.get("command", None) is not None else None,
            config=obj["config"] if obj.get("config", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "command": self.command.to_json() if self.command is not None else None,
            "config": self.config if self.config is not None else None,
        }

class v1GetCommandsRequestSortBy(enum.Enum):
    SORT_BY_UNSPECIFIED = "SORT_BY_UNSPECIFIED"
    SORT_BY_ID = "SORT_BY_ID"
    SORT_BY_DESCRIPTION = "SORT_BY_DESCRIPTION"
    SORT_BY_START_TIME = "SORT_BY_START_TIME"

class v1GetCommandsResponse:
    def __init__(
        self,
        commands: "t.Optional[t.Sequence[v1Command]]" = None,
        pagination: "t.Optional[v1Pagination]" = None,
    ):
        self.commands = commands
        self.pagination = pagination

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetCommandsResponse":
        return cls(
            commands=[v1Command.from_json(x) for x in obj["commands"]] if obj.get("commands", None) is not None else None,
            pagination=v1Pagination.from_json(obj["pagination"]) if obj.get("pagination", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "commands": [x.to_json() for x in self.commands] if self.commands is not None else None,
            "pagination": self.pagination.to_json() if self.pagination is not None else None,
        }

class v1GetCurrentTrialSearcherOperationResponse:
    def __init__(
        self,
        completed: "t.Optional[bool]" = None,
        op: "t.Optional[v1SearcherOperation]" = None,
    ):
        self.op = op
        self.completed = completed

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetCurrentTrialSearcherOperationResponse":
        return cls(
            op=v1SearcherOperation.from_json(obj["op"]) if obj.get("op", None) is not None else None,
            completed=obj["completed"] if obj.get("completed", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "op": self.op.to_json() if self.op is not None else None,
            "completed": self.completed if self.completed is not None else None,
        }

class v1GetExperimentCheckpointsRequestSortBy(enum.Enum):
    SORT_BY_UNSPECIFIED = "SORT_BY_UNSPECIFIED"
    SORT_BY_UUID = "SORT_BY_UUID"
    SORT_BY_TRIAL_ID = "SORT_BY_TRIAL_ID"
    SORT_BY_BATCH_NUMBER = "SORT_BY_BATCH_NUMBER"
    SORT_BY_START_TIME = "SORT_BY_START_TIME"
    SORT_BY_END_TIME = "SORT_BY_END_TIME"
    SORT_BY_VALIDATION_STATE = "SORT_BY_VALIDATION_STATE"
    SORT_BY_STATE = "SORT_BY_STATE"
    SORT_BY_SEARCHER_METRIC = "SORT_BY_SEARCHER_METRIC"

class v1GetExperimentCheckpointsResponse:
    def __init__(
        self,
        checkpoints: "t.Optional[t.Sequence[v1Checkpoint]]" = None,
        pagination: "t.Optional[v1Pagination]" = None,
    ):
        self.checkpoints = checkpoints
        self.pagination = pagination

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetExperimentCheckpointsResponse":
        return cls(
            checkpoints=[v1Checkpoint.from_json(x) for x in obj["checkpoints"]] if obj.get("checkpoints", None) is not None else None,
            pagination=v1Pagination.from_json(obj["pagination"]) if obj.get("pagination", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "checkpoints": [x.to_json() for x in self.checkpoints] if self.checkpoints is not None else None,
            "pagination": self.pagination.to_json() if self.pagination is not None else None,
        }

class v1GetExperimentLabelsResponse:
    def __init__(
        self,
        labels: "t.Optional[t.Sequence[str]]" = None,
    ):
        self.labels = labels

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetExperimentLabelsResponse":
        return cls(
            labels=obj["labels"] if obj.get("labels", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "labels": self.labels if self.labels is not None else None,
        }

class v1GetExperimentResponse:
    def __init__(
        self,
        config: "t.Dict[str, t.Any]",
        experiment: "v1Experiment",
        jobSummary: "t.Optional[v1JobSummary]" = None,
    ):
        self.experiment = experiment
        self.config = config
        self.jobSummary = jobSummary

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetExperimentResponse":
        return cls(
            experiment=v1Experiment.from_json(obj["experiment"]),
            config=obj["config"],
            jobSummary=v1JobSummary.from_json(obj["jobSummary"]) if obj.get("jobSummary", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "experiment": self.experiment.to_json(),
            "config": self.config,
            "jobSummary": self.jobSummary.to_json() if self.jobSummary is not None else None,
        }

class v1GetExperimentTrialsRequestSortBy(enum.Enum):
    SORT_BY_UNSPECIFIED = "SORT_BY_UNSPECIFIED"
    SORT_BY_ID = "SORT_BY_ID"
    SORT_BY_START_TIME = "SORT_BY_START_TIME"
    SORT_BY_END_TIME = "SORT_BY_END_TIME"
    SORT_BY_STATE = "SORT_BY_STATE"
    SORT_BY_BEST_VALIDATION_METRIC = "SORT_BY_BEST_VALIDATION_METRIC"
    SORT_BY_LATEST_VALIDATION_METRIC = "SORT_BY_LATEST_VALIDATION_METRIC"
    SORT_BY_BATCHES_PROCESSED = "SORT_BY_BATCHES_PROCESSED"
    SORT_BY_DURATION = "SORT_BY_DURATION"

class v1GetExperimentTrialsResponse:
    def __init__(
        self,
        pagination: "v1Pagination",
        trials: "t.Sequence[trialv1Trial]",
    ):
        self.trials = trials
        self.pagination = pagination

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetExperimentTrialsResponse":
        return cls(
            trials=[trialv1Trial.from_json(x) for x in obj["trials"]],
            pagination=v1Pagination.from_json(obj["pagination"]),
        )

    def to_json(self) -> t.Any:
        return {
            "trials": [x.to_json() for x in self.trials],
            "pagination": self.pagination.to_json(),
        }

class v1GetExperimentValidationHistoryResponse:
    def __init__(
        self,
        validationHistory: "t.Optional[t.Sequence[v1ValidationHistoryEntry]]" = None,
    ):
        self.validationHistory = validationHistory

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetExperimentValidationHistoryResponse":
        return cls(
            validationHistory=[v1ValidationHistoryEntry.from_json(x) for x in obj["validationHistory"]] if obj.get("validationHistory", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "validationHistory": [x.to_json() for x in self.validationHistory] if self.validationHistory is not None else None,
        }

class v1GetExperimentsRequestSortBy(enum.Enum):
    SORT_BY_UNSPECIFIED = "SORT_BY_UNSPECIFIED"
    SORT_BY_ID = "SORT_BY_ID"
    SORT_BY_DESCRIPTION = "SORT_BY_DESCRIPTION"
    SORT_BY_START_TIME = "SORT_BY_START_TIME"
    SORT_BY_END_TIME = "SORT_BY_END_TIME"
    SORT_BY_STATE = "SORT_BY_STATE"
    SORT_BY_NUM_TRIALS = "SORT_BY_NUM_TRIALS"
    SORT_BY_PROGRESS = "SORT_BY_PROGRESS"
    SORT_BY_USER = "SORT_BY_USER"
    SORT_BY_NAME = "SORT_BY_NAME"

class v1GetExperimentsResponse:
    def __init__(
        self,
        experiments: "t.Sequence[v1Experiment]",
        pagination: "v1Pagination",
    ):
        self.experiments = experiments
        self.pagination = pagination

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetExperimentsResponse":
        return cls(
            experiments=[v1Experiment.from_json(x) for x in obj["experiments"]],
            pagination=v1Pagination.from_json(obj["pagination"]),
        )

    def to_json(self) -> t.Any:
        return {
            "experiments": [x.to_json() for x in self.experiments],
            "pagination": self.pagination.to_json(),
        }

class v1GetHPImportanceResponse:
    def __init__(
        self,
        trainingMetrics: "t.Dict[str, GetHPImportanceResponseMetricHPImportance]",
        validationMetrics: "t.Dict[str, GetHPImportanceResponseMetricHPImportance]",
    ):
        self.trainingMetrics = trainingMetrics
        self.validationMetrics = validationMetrics

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetHPImportanceResponse":
        return cls(
            trainingMetrics={k: GetHPImportanceResponseMetricHPImportance.from_json(v) for k, v in obj["trainingMetrics"].items()},
            validationMetrics={k: GetHPImportanceResponseMetricHPImportance.from_json(v) for k, v in obj["validationMetrics"].items()},
        )

    def to_json(self) -> t.Any:
        return {
            "trainingMetrics": {k: v.to_json() for k, v in self.trainingMetrics.items()},
            "validationMetrics": {k: v.to_json() for k, v in self.validationMetrics.items()},
        }

class v1GetJobQueueStatsResponse:
    def __init__(
        self,
        results: "t.Sequence[v1RPQueueStat]",
    ):
        self.results = results

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetJobQueueStatsResponse":
        return cls(
            results=[v1RPQueueStat.from_json(x) for x in obj["results"]],
        )

    def to_json(self) -> t.Any:
        return {
            "results": [x.to_json() for x in self.results],
        }

class v1GetJobsResponse:
    def __init__(
        self,
        jobs: "t.Sequence[v1Job]",
        pagination: "v1Pagination",
    ):
        self.pagination = pagination
        self.jobs = jobs

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetJobsResponse":
        return cls(
            pagination=v1Pagination.from_json(obj["pagination"]),
            jobs=[v1Job.from_json(x) for x in obj["jobs"]],
        )

    def to_json(self) -> t.Any:
        return {
            "pagination": self.pagination.to_json(),
            "jobs": [x.to_json() for x in self.jobs],
        }

class v1GetMasterConfigResponse:
    def __init__(
        self,
        config: "t.Dict[str, t.Any]",
    ):
        self.config = config

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetMasterConfigResponse":
        return cls(
            config=obj["config"],
        )

    def to_json(self) -> t.Any:
        return {
            "config": self.config,
        }

class v1GetMasterResponse:
    def __init__(
        self,
        clusterId: str,
        clusterName: str,
        masterId: str,
        version: str,
        branding: "t.Optional[str]" = None,
        externalLoginUri: "t.Optional[str]" = None,
        externalLogoutUri: "t.Optional[str]" = None,
        ssoProviders: "t.Optional[t.Sequence[v1SSOProvider]]" = None,
        telemetryEnabled: "t.Optional[bool]" = None,
    ):
        self.version = version
        self.masterId = masterId
        self.clusterId = clusterId
        self.clusterName = clusterName
        self.telemetryEnabled = telemetryEnabled
        self.ssoProviders = ssoProviders
        self.externalLoginUri = externalLoginUri
        self.externalLogoutUri = externalLogoutUri
        self.branding = branding

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetMasterResponse":
        return cls(
            version=obj["version"],
            masterId=obj["masterId"],
            clusterId=obj["clusterId"],
            clusterName=obj["clusterName"],
            telemetryEnabled=obj["telemetryEnabled"] if obj.get("telemetryEnabled", None) is not None else None,
            ssoProviders=[v1SSOProvider.from_json(x) for x in obj["ssoProviders"]] if obj.get("ssoProviders", None) is not None else None,
            externalLoginUri=obj["externalLoginUri"] if obj.get("externalLoginUri", None) is not None else None,
            externalLogoutUri=obj["externalLogoutUri"] if obj.get("externalLogoutUri", None) is not None else None,
            branding=obj["branding"] if obj.get("branding", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "version": self.version,
            "masterId": self.masterId,
            "clusterId": self.clusterId,
            "clusterName": self.clusterName,
            "telemetryEnabled": self.telemetryEnabled if self.telemetryEnabled is not None else None,
            "ssoProviders": [x.to_json() for x in self.ssoProviders] if self.ssoProviders is not None else None,
            "externalLoginUri": self.externalLoginUri if self.externalLoginUri is not None else None,
            "externalLogoutUri": self.externalLogoutUri if self.externalLogoutUri is not None else None,
            "branding": self.branding if self.branding is not None else None,
        }

class v1GetModelDefResponse:
    def __init__(
        self,
        b64Tgz: str,
    ):
        self.b64Tgz = b64Tgz

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetModelDefResponse":
        return cls(
            b64Tgz=obj["b64Tgz"],
        )

    def to_json(self) -> t.Any:
        return {
            "b64Tgz": self.b64Tgz,
        }

class v1GetModelLabelsResponse:
    def __init__(
        self,
        labels: "t.Optional[t.Sequence[str]]" = None,
    ):
        self.labels = labels

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetModelLabelsResponse":
        return cls(
            labels=obj["labels"] if obj.get("labels", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "labels": self.labels if self.labels is not None else None,
        }

class v1GetModelResponse:
    def __init__(
        self,
        model: "t.Optional[v1Model]" = None,
    ):
        self.model = model

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetModelResponse":
        return cls(
            model=v1Model.from_json(obj["model"]) if obj.get("model", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "model": self.model.to_json() if self.model is not None else None,
        }

class v1GetModelVersionResponse:
    def __init__(
        self,
        modelVersion: "t.Optional[v1ModelVersion]" = None,
    ):
        self.modelVersion = modelVersion

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetModelVersionResponse":
        return cls(
            modelVersion=v1ModelVersion.from_json(obj["modelVersion"]) if obj.get("modelVersion", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "modelVersion": self.modelVersion.to_json() if self.modelVersion is not None else None,
        }

class v1GetModelVersionsRequestSortBy(enum.Enum):
    SORT_BY_UNSPECIFIED = "SORT_BY_UNSPECIFIED"
    SORT_BY_VERSION = "SORT_BY_VERSION"
    SORT_BY_CREATION_TIME = "SORT_BY_CREATION_TIME"

class v1GetModelVersionsResponse:
    def __init__(
        self,
        model: "t.Optional[v1Model]" = None,
        modelVersions: "t.Optional[t.Sequence[v1ModelVersion]]" = None,
        pagination: "t.Optional[v1Pagination]" = None,
    ):
        self.model = model
        self.modelVersions = modelVersions
        self.pagination = pagination

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetModelVersionsResponse":
        return cls(
            model=v1Model.from_json(obj["model"]) if obj.get("model", None) is not None else None,
            modelVersions=[v1ModelVersion.from_json(x) for x in obj["modelVersions"]] if obj.get("modelVersions", None) is not None else None,
            pagination=v1Pagination.from_json(obj["pagination"]) if obj.get("pagination", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "model": self.model.to_json() if self.model is not None else None,
            "modelVersions": [x.to_json() for x in self.modelVersions] if self.modelVersions is not None else None,
            "pagination": self.pagination.to_json() if self.pagination is not None else None,
        }

class v1GetModelsRequestSortBy(enum.Enum):
    SORT_BY_UNSPECIFIED = "SORT_BY_UNSPECIFIED"
    SORT_BY_NAME = "SORT_BY_NAME"
    SORT_BY_DESCRIPTION = "SORT_BY_DESCRIPTION"
    SORT_BY_CREATION_TIME = "SORT_BY_CREATION_TIME"
    SORT_BY_LAST_UPDATED_TIME = "SORT_BY_LAST_UPDATED_TIME"
    SORT_BY_NUM_VERSIONS = "SORT_BY_NUM_VERSIONS"

class v1GetModelsResponse:
    def __init__(
        self,
        models: "t.Sequence[v1Model]",
        pagination: "v1Pagination",
    ):
        self.models = models
        self.pagination = pagination

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetModelsResponse":
        return cls(
            models=[v1Model.from_json(x) for x in obj["models"]],
            pagination=v1Pagination.from_json(obj["pagination"]),
        )

    def to_json(self) -> t.Any:
        return {
            "models": [x.to_json() for x in self.models],
            "pagination": self.pagination.to_json(),
        }

class v1GetNotebookResponse:
    def __init__(
        self,
        config: "t.Optional[t.Dict[str, t.Any]]" = None,
        notebook: "t.Optional[v1Notebook]" = None,
    ):
        self.notebook = notebook
        self.config = config

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetNotebookResponse":
        return cls(
            notebook=v1Notebook.from_json(obj["notebook"]) if obj.get("notebook", None) is not None else None,
            config=obj["config"] if obj.get("config", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "notebook": self.notebook.to_json() if self.notebook is not None else None,
            "config": self.config if self.config is not None else None,
        }

class v1GetNotebooksRequestSortBy(enum.Enum):
    SORT_BY_UNSPECIFIED = "SORT_BY_UNSPECIFIED"
    SORT_BY_ID = "SORT_BY_ID"
    SORT_BY_DESCRIPTION = "SORT_BY_DESCRIPTION"
    SORT_BY_START_TIME = "SORT_BY_START_TIME"

class v1GetNotebooksResponse:
    def __init__(
        self,
        notebooks: "t.Optional[t.Sequence[v1Notebook]]" = None,
        pagination: "t.Optional[v1Pagination]" = None,
    ):
        self.notebooks = notebooks
        self.pagination = pagination

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetNotebooksResponse":
        return cls(
            notebooks=[v1Notebook.from_json(x) for x in obj["notebooks"]] if obj.get("notebooks", None) is not None else None,
            pagination=v1Pagination.from_json(obj["pagination"]) if obj.get("pagination", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "notebooks": [x.to_json() for x in self.notebooks] if self.notebooks is not None else None,
            "pagination": self.pagination.to_json() if self.pagination is not None else None,
        }

class v1GetResourcePoolsResponse:
    def __init__(
        self,
        pagination: "t.Optional[v1Pagination]" = None,
        resourcePools: "t.Optional[t.Sequence[v1ResourcePool]]" = None,
    ):
        self.resourcePools = resourcePools
        self.pagination = pagination

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetResourcePoolsResponse":
        return cls(
            resourcePools=[v1ResourcePool.from_json(x) for x in obj["resourcePools"]] if obj.get("resourcePools", None) is not None else None,
            pagination=v1Pagination.from_json(obj["pagination"]) if obj.get("pagination", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "resourcePools": [x.to_json() for x in self.resourcePools] if self.resourcePools is not None else None,
            "pagination": self.pagination.to_json() if self.pagination is not None else None,
        }

class v1GetShellResponse:
    def __init__(
        self,
        config: "t.Optional[t.Dict[str, t.Any]]" = None,
        shell: "t.Optional[v1Shell]" = None,
    ):
        self.shell = shell
        self.config = config

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetShellResponse":
        return cls(
            shell=v1Shell.from_json(obj["shell"]) if obj.get("shell", None) is not None else None,
            config=obj["config"] if obj.get("config", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "shell": self.shell.to_json() if self.shell is not None else None,
            "config": self.config if self.config is not None else None,
        }

class v1GetShellsRequestSortBy(enum.Enum):
    SORT_BY_UNSPECIFIED = "SORT_BY_UNSPECIFIED"
    SORT_BY_ID = "SORT_BY_ID"
    SORT_BY_DESCRIPTION = "SORT_BY_DESCRIPTION"
    SORT_BY_START_TIME = "SORT_BY_START_TIME"

class v1GetShellsResponse:
    def __init__(
        self,
        pagination: "t.Optional[v1Pagination]" = None,
        shells: "t.Optional[t.Sequence[v1Shell]]" = None,
    ):
        self.shells = shells
        self.pagination = pagination

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetShellsResponse":
        return cls(
            shells=[v1Shell.from_json(x) for x in obj["shells"]] if obj.get("shells", None) is not None else None,
            pagination=v1Pagination.from_json(obj["pagination"]) if obj.get("pagination", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "shells": [x.to_json() for x in self.shells] if self.shells is not None else None,
            "pagination": self.pagination.to_json() if self.pagination is not None else None,
        }

class v1GetSlotResponse:
    def __init__(
        self,
        slot: "t.Optional[v1Slot]" = None,
    ):
        self.slot = slot

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetSlotResponse":
        return cls(
            slot=v1Slot.from_json(obj["slot"]) if obj.get("slot", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "slot": self.slot.to_json() if self.slot is not None else None,
        }

class v1GetSlotsResponse:
    def __init__(
        self,
        slots: "t.Optional[t.Sequence[v1Slot]]" = None,
    ):
        self.slots = slots

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetSlotsResponse":
        return cls(
            slots=[v1Slot.from_json(x) for x in obj["slots"]] if obj.get("slots", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "slots": [x.to_json() for x in self.slots] if self.slots is not None else None,
        }

class v1GetTelemetryResponse:
    def __init__(
        self,
        enabled: bool,
        segmentKey: "t.Optional[str]" = None,
    ):
        self.enabled = enabled
        self.segmentKey = segmentKey

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetTelemetryResponse":
        return cls(
            enabled=obj["enabled"],
            segmentKey=obj["segmentKey"] if obj.get("segmentKey", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "enabled": self.enabled,
            "segmentKey": self.segmentKey if self.segmentKey is not None else None,
        }

class v1GetTemplateResponse:
    def __init__(
        self,
        template: "t.Optional[v1Template]" = None,
    ):
        self.template = template

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetTemplateResponse":
        return cls(
            template=v1Template.from_json(obj["template"]) if obj.get("template", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "template": self.template.to_json() if self.template is not None else None,
        }

class v1GetTemplatesRequestSortBy(enum.Enum):
    SORT_BY_UNSPECIFIED = "SORT_BY_UNSPECIFIED"
    SORT_BY_NAME = "SORT_BY_NAME"

class v1GetTemplatesResponse:
    def __init__(
        self,
        pagination: "t.Optional[v1Pagination]" = None,
        templates: "t.Optional[t.Sequence[v1Template]]" = None,
    ):
        self.templates = templates
        self.pagination = pagination

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetTemplatesResponse":
        return cls(
            templates=[v1Template.from_json(x) for x in obj["templates"]] if obj.get("templates", None) is not None else None,
            pagination=v1Pagination.from_json(obj["pagination"]) if obj.get("pagination", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "templates": [x.to_json() for x in self.templates] if self.templates is not None else None,
            "pagination": self.pagination.to_json() if self.pagination is not None else None,
        }

class v1GetTensorboardResponse:
    def __init__(
        self,
        config: "t.Optional[t.Dict[str, t.Any]]" = None,
        tensorboard: "t.Optional[v1Tensorboard]" = None,
    ):
        self.tensorboard = tensorboard
        self.config = config

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetTensorboardResponse":
        return cls(
            tensorboard=v1Tensorboard.from_json(obj["tensorboard"]) if obj.get("tensorboard", None) is not None else None,
            config=obj["config"] if obj.get("config", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "tensorboard": self.tensorboard.to_json() if self.tensorboard is not None else None,
            "config": self.config if self.config is not None else None,
        }

class v1GetTensorboardsRequestSortBy(enum.Enum):
    SORT_BY_UNSPECIFIED = "SORT_BY_UNSPECIFIED"
    SORT_BY_ID = "SORT_BY_ID"
    SORT_BY_DESCRIPTION = "SORT_BY_DESCRIPTION"
    SORT_BY_START_TIME = "SORT_BY_START_TIME"

class v1GetTensorboardsResponse:
    def __init__(
        self,
        pagination: "t.Optional[v1Pagination]" = None,
        tensorboards: "t.Optional[t.Sequence[v1Tensorboard]]" = None,
    ):
        self.tensorboards = tensorboards
        self.pagination = pagination

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetTensorboardsResponse":
        return cls(
            tensorboards=[v1Tensorboard.from_json(x) for x in obj["tensorboards"]] if obj.get("tensorboards", None) is not None else None,
            pagination=v1Pagination.from_json(obj["pagination"]) if obj.get("pagination", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "tensorboards": [x.to_json() for x in self.tensorboards] if self.tensorboards is not None else None,
            "pagination": self.pagination.to_json() if self.pagination is not None else None,
        }

class v1GetTrialCheckpointsRequestSortBy(enum.Enum):
    SORT_BY_UNSPECIFIED = "SORT_BY_UNSPECIFIED"
    SORT_BY_UUID = "SORT_BY_UUID"
    SORT_BY_BATCH_NUMBER = "SORT_BY_BATCH_NUMBER"
    SORT_BY_START_TIME = "SORT_BY_START_TIME"
    SORT_BY_END_TIME = "SORT_BY_END_TIME"
    SORT_BY_VALIDATION_STATE = "SORT_BY_VALIDATION_STATE"
    SORT_BY_STATE = "SORT_BY_STATE"

class v1GetTrialCheckpointsResponse:
    def __init__(
        self,
        checkpoints: "t.Optional[t.Sequence[v1Checkpoint]]" = None,
        pagination: "t.Optional[v1Pagination]" = None,
    ):
        self.checkpoints = checkpoints
        self.pagination = pagination

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetTrialCheckpointsResponse":
        return cls(
            checkpoints=[v1Checkpoint.from_json(x) for x in obj["checkpoints"]] if obj.get("checkpoints", None) is not None else None,
            pagination=v1Pagination.from_json(obj["pagination"]) if obj.get("pagination", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "checkpoints": [x.to_json() for x in self.checkpoints] if self.checkpoints is not None else None,
            "pagination": self.pagination.to_json() if self.pagination is not None else None,
        }

class v1GetTrialProfilerAvailableSeriesResponse:
    def __init__(
        self,
        labels: "t.Sequence[v1TrialProfilerMetricLabels]",
    ):
        self.labels = labels

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetTrialProfilerAvailableSeriesResponse":
        return cls(
            labels=[v1TrialProfilerMetricLabels.from_json(x) for x in obj["labels"]],
        )

    def to_json(self) -> t.Any:
        return {
            "labels": [x.to_json() for x in self.labels],
        }

class v1GetTrialProfilerMetricsResponse:
    def __init__(
        self,
        batch: "v1TrialProfilerMetricsBatch",
    ):
        self.batch = batch

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetTrialProfilerMetricsResponse":
        return cls(
            batch=v1TrialProfilerMetricsBatch.from_json(obj["batch"]),
        )

    def to_json(self) -> t.Any:
        return {
            "batch": self.batch.to_json(),
        }

class v1GetTrialResponse:
    def __init__(
        self,
        trial: "trialv1Trial",
        workloads: "t.Optional[t.Sequence[GetTrialResponseWorkloadContainer]]" = None,
    ):
        self.trial = trial
        self.workloads = workloads

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetTrialResponse":
        return cls(
            trial=trialv1Trial.from_json(obj["trial"]),
            workloads=[GetTrialResponseWorkloadContainer.from_json(x) for x in obj["workloads"]] if obj.get("workloads", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "trial": self.trial.to_json(),
            "workloads": [x.to_json() for x in self.workloads] if self.workloads is not None else None,
        }

class v1GetUserResponse:
    def __init__(
        self,
        user: "t.Optional[v1User]" = None,
    ):
        self.user = user

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetUserResponse":
        return cls(
            user=v1User.from_json(obj["user"]) if obj.get("user", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "user": self.user.to_json() if self.user is not None else None,
        }

class v1GetUsersResponse:
    def __init__(
        self,
        users: "t.Optional[t.Sequence[v1User]]" = None,
    ):
        self.users = users

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetUsersResponse":
        return cls(
            users=[v1User.from_json(x) for x in obj["users"]] if obj.get("users", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "users": [x.to_json() for x in self.users] if self.users is not None else None,
        }

class v1IdleNotebookRequest:
    def __init__(
        self,
        idle: "t.Optional[bool]" = None,
        notebookId: "t.Optional[str]" = None,
    ):
        self.notebookId = notebookId
        self.idle = idle

    @classmethod
    def from_json(cls, obj: Json) -> "v1IdleNotebookRequest":
        return cls(
            notebookId=obj["notebookId"] if obj.get("notebookId", None) is not None else None,
            idle=obj["idle"] if obj.get("idle", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "notebookId": self.notebookId if self.notebookId is not None else None,
            "idle": self.idle if self.idle is not None else None,
        }

class v1Job:
    def __init__(
        self,
        allocatedSlots: int,
        entityId: str,
        isPreemptible: bool,
        jobId: str,
        name: str,
        requestedSlots: int,
        resourcePool: str,
        submissionTime: str,
        type: "determinedjobv1Type",
        username: str,
        priority: "t.Optional[int]" = None,
        progress: "t.Optional[float]" = None,
        summary: "t.Optional[v1JobSummary]" = None,
        weight: "t.Optional[float]" = None,
    ):
        self.summary = summary
        self.type = type
        self.submissionTime = submissionTime
        self.username = username
        self.resourcePool = resourcePool
        self.isPreemptible = isPreemptible
        self.priority = priority
        self.weight = weight
        self.entityId = entityId
        self.jobId = jobId
        self.requestedSlots = requestedSlots
        self.allocatedSlots = allocatedSlots
        self.name = name
        self.progress = progress

    @classmethod
    def from_json(cls, obj: Json) -> "v1Job":
        return cls(
            summary=v1JobSummary.from_json(obj["summary"]) if obj.get("summary", None) is not None else None,
            type=obj["type"],
            submissionTime=obj["submissionTime"],
            username=obj["username"],
            resourcePool=obj["resourcePool"],
            isPreemptible=obj["isPreemptible"],
            priority=obj["priority"] if obj.get("priority", None) is not None else None,
            weight=float(obj["weight"]) if obj.get("weight", None) is not None else None,
            entityId=obj["entityId"],
            jobId=obj["jobId"],
            requestedSlots=obj["requestedSlots"],
            allocatedSlots=obj["allocatedSlots"],
            name=obj["name"],
            progress=float(obj["progress"]) if obj.get("progress", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "summary": self.summary.to_json() if self.summary is not None else None,
            "type": self.type,
            "submissionTime": self.submissionTime,
            "username": self.username,
            "resourcePool": self.resourcePool,
            "isPreemptible": self.isPreemptible,
            "priority": self.priority if self.priority is not None else None,
            "weight": dump_float(self.weight) if self.weight is not None else None,
            "entityId": self.entityId,
            "jobId": self.jobId,
            "requestedSlots": self.requestedSlots,
            "allocatedSlots": self.allocatedSlots,
            "name": self.name,
            "progress": dump_float(self.progress) if self.progress is not None else None,
        }

class v1JobSummary:
    def __init__(
        self,
        jobsAhead: int,
        state: "determinedjobv1State",
    ):
        self.state = state
        self.jobsAhead = jobsAhead

    @classmethod
    def from_json(cls, obj: Json) -> "v1JobSummary":
        return cls(
            state=obj["state"],
            jobsAhead=obj["jobsAhead"],
        )

    def to_json(self) -> t.Any:
        return {
            "state": self.state,
            "jobsAhead": self.jobsAhead,
        }

class v1K8PriorityClass:
    def __init__(
        self,
        priorityClass: "t.Optional[str]" = None,
        priorityValue: "t.Optional[int]" = None,
    ):
        self.priorityClass = priorityClass
        self.priorityValue = priorityValue

    @classmethod
    def from_json(cls, obj: Json) -> "v1K8PriorityClass":
        return cls(
            priorityClass=obj["priorityClass"] if obj.get("priorityClass", None) is not None else None,
            priorityValue=obj["priorityValue"] if obj.get("priorityValue", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "priorityClass": self.priorityClass if self.priorityClass is not None else None,
            "priorityValue": self.priorityValue if self.priorityValue is not None else None,
        }

class v1KillCommandResponse:
    def __init__(
        self,
        command: "t.Optional[v1Command]" = None,
    ):
        self.command = command

    @classmethod
    def from_json(cls, obj: Json) -> "v1KillCommandResponse":
        return cls(
            command=v1Command.from_json(obj["command"]) if obj.get("command", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "command": self.command.to_json() if self.command is not None else None,
        }

class v1KillNotebookResponse:
    def __init__(
        self,
        notebook: "t.Optional[v1Notebook]" = None,
    ):
        self.notebook = notebook

    @classmethod
    def from_json(cls, obj: Json) -> "v1KillNotebookResponse":
        return cls(
            notebook=v1Notebook.from_json(obj["notebook"]) if obj.get("notebook", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "notebook": self.notebook.to_json() if self.notebook is not None else None,
        }

class v1KillShellResponse:
    def __init__(
        self,
        shell: "t.Optional[v1Shell]" = None,
    ):
        self.shell = shell

    @classmethod
    def from_json(cls, obj: Json) -> "v1KillShellResponse":
        return cls(
            shell=v1Shell.from_json(obj["shell"]) if obj.get("shell", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "shell": self.shell.to_json() if self.shell is not None else None,
        }

class v1KillTensorboardResponse:
    def __init__(
        self,
        tensorboard: "t.Optional[v1Tensorboard]" = None,
    ):
        self.tensorboard = tensorboard

    @classmethod
    def from_json(cls, obj: Json) -> "v1KillTensorboardResponse":
        return cls(
            tensorboard=v1Tensorboard.from_json(obj["tensorboard"]) if obj.get("tensorboard", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "tensorboard": self.tensorboard.to_json() if self.tensorboard is not None else None,
        }

class v1LaunchCommandRequest:
    def __init__(
        self,
        config: "t.Optional[t.Dict[str, t.Any]]" = None,
        data: "t.Optional[str]" = None,
        files: "t.Optional[t.Sequence[v1File]]" = None,
        templateName: "t.Optional[str]" = None,
    ):
        self.config = config
        self.templateName = templateName
        self.files = files
        self.data = data

    @classmethod
    def from_json(cls, obj: Json) -> "v1LaunchCommandRequest":
        return cls(
            config=obj["config"] if obj.get("config", None) is not None else None,
            templateName=obj["templateName"] if obj.get("templateName", None) is not None else None,
            files=[v1File.from_json(x) for x in obj["files"]] if obj.get("files", None) is not None else None,
            data=obj["data"] if obj.get("data", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "config": self.config if self.config is not None else None,
            "templateName": self.templateName if self.templateName is not None else None,
            "files": [x.to_json() for x in self.files] if self.files is not None else None,
            "data": self.data if self.data is not None else None,
        }

class v1LaunchCommandResponse:
    def __init__(
        self,
        command: "v1Command",
        config: "t.Dict[str, t.Any]",
    ):
        self.command = command
        self.config = config

    @classmethod
    def from_json(cls, obj: Json) -> "v1LaunchCommandResponse":
        return cls(
            command=v1Command.from_json(obj["command"]),
            config=obj["config"],
        )

    def to_json(self) -> t.Any:
        return {
            "command": self.command.to_json(),
            "config": self.config,
        }

class v1LaunchNotebookRequest:
    def __init__(
        self,
        config: "t.Optional[t.Dict[str, t.Any]]" = None,
        files: "t.Optional[t.Sequence[v1File]]" = None,
        preview: "t.Optional[bool]" = None,
        templateName: "t.Optional[str]" = None,
    ):
        self.config = config
        self.templateName = templateName
        self.files = files
        self.preview = preview

    @classmethod
    def from_json(cls, obj: Json) -> "v1LaunchNotebookRequest":
        return cls(
            config=obj["config"] if obj.get("config", None) is not None else None,
            templateName=obj["templateName"] if obj.get("templateName", None) is not None else None,
            files=[v1File.from_json(x) for x in obj["files"]] if obj.get("files", None) is not None else None,
            preview=obj["preview"] if obj.get("preview", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "config": self.config if self.config is not None else None,
            "templateName": self.templateName if self.templateName is not None else None,
            "files": [x.to_json() for x in self.files] if self.files is not None else None,
            "preview": self.preview if self.preview is not None else None,
        }

class v1LaunchNotebookResponse:
    def __init__(
        self,
        config: "t.Dict[str, t.Any]",
        notebook: "v1Notebook",
    ):
        self.notebook = notebook
        self.config = config

    @classmethod
    def from_json(cls, obj: Json) -> "v1LaunchNotebookResponse":
        return cls(
            notebook=v1Notebook.from_json(obj["notebook"]),
            config=obj["config"],
        )

    def to_json(self) -> t.Any:
        return {
            "notebook": self.notebook.to_json(),
            "config": self.config,
        }

class v1LaunchShellRequest:
    def __init__(
        self,
        config: "t.Optional[t.Dict[str, t.Any]]" = None,
        data: "t.Optional[str]" = None,
        files: "t.Optional[t.Sequence[v1File]]" = None,
        templateName: "t.Optional[str]" = None,
    ):
        self.config = config
        self.templateName = templateName
        self.files = files
        self.data = data

    @classmethod
    def from_json(cls, obj: Json) -> "v1LaunchShellRequest":
        return cls(
            config=obj["config"] if obj.get("config", None) is not None else None,
            templateName=obj["templateName"] if obj.get("templateName", None) is not None else None,
            files=[v1File.from_json(x) for x in obj["files"]] if obj.get("files", None) is not None else None,
            data=obj["data"] if obj.get("data", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "config": self.config if self.config is not None else None,
            "templateName": self.templateName if self.templateName is not None else None,
            "files": [x.to_json() for x in self.files] if self.files is not None else None,
            "data": self.data if self.data is not None else None,
        }

class v1LaunchShellResponse:
    def __init__(
        self,
        config: "t.Dict[str, t.Any]",
        shell: "v1Shell",
    ):
        self.shell = shell
        self.config = config

    @classmethod
    def from_json(cls, obj: Json) -> "v1LaunchShellResponse":
        return cls(
            shell=v1Shell.from_json(obj["shell"]),
            config=obj["config"],
        )

    def to_json(self) -> t.Any:
        return {
            "shell": self.shell.to_json(),
            "config": self.config,
        }

class v1LaunchTensorboardRequest:
    def __init__(
        self,
        config: "t.Optional[t.Dict[str, t.Any]]" = None,
        experimentIds: "t.Optional[t.Sequence[int]]" = None,
        files: "t.Optional[t.Sequence[v1File]]" = None,
        templateName: "t.Optional[str]" = None,
        trialIds: "t.Optional[t.Sequence[int]]" = None,
    ):
        self.experimentIds = experimentIds
        self.trialIds = trialIds
        self.config = config
        self.templateName = templateName
        self.files = files

    @classmethod
    def from_json(cls, obj: Json) -> "v1LaunchTensorboardRequest":
        return cls(
            experimentIds=obj["experimentIds"] if obj.get("experimentIds", None) is not None else None,
            trialIds=obj["trialIds"] if obj.get("trialIds", None) is not None else None,
            config=obj["config"] if obj.get("config", None) is not None else None,
            templateName=obj["templateName"] if obj.get("templateName", None) is not None else None,
            files=[v1File.from_json(x) for x in obj["files"]] if obj.get("files", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "experimentIds": self.experimentIds if self.experimentIds is not None else None,
            "trialIds": self.trialIds if self.trialIds is not None else None,
            "config": self.config if self.config is not None else None,
            "templateName": self.templateName if self.templateName is not None else None,
            "files": [x.to_json() for x in self.files] if self.files is not None else None,
        }

class v1LaunchTensorboardResponse:
    def __init__(
        self,
        config: "t.Dict[str, t.Any]",
        tensorboard: "v1Tensorboard",
    ):
        self.tensorboard = tensorboard
        self.config = config

    @classmethod
    def from_json(cls, obj: Json) -> "v1LaunchTensorboardResponse":
        return cls(
            tensorboard=v1Tensorboard.from_json(obj["tensorboard"]),
            config=obj["config"],
        )

    def to_json(self) -> t.Any:
        return {
            "tensorboard": self.tensorboard.to_json(),
            "config": self.config,
        }

class v1LogEntry:
    def __init__(
        self,
        id: int,
        level: "t.Optional[v1LogLevel]" = None,
        message: "t.Optional[str]" = None,
        timestamp: "t.Optional[str]" = None,
    ):
        self.id = id
        self.message = message
        self.timestamp = timestamp
        self.level = level

    @classmethod
    def from_json(cls, obj: Json) -> "v1LogEntry":
        return cls(
            id=obj["id"],
            message=obj["message"] if obj.get("message", None) is not None else None,
            timestamp=obj["timestamp"] if obj.get("timestamp", None) is not None else None,
            level=obj["level"] if obj.get("level", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "id": self.id,
            "message": self.message if self.message is not None else None,
            "timestamp": self.timestamp if self.timestamp is not None else None,
            "level": self.level if self.level is not None else None,
        }

class v1LogLevel(enum.Enum):
    LOG_LEVEL_UNSPECIFIED = "LOG_LEVEL_UNSPECIFIED"
    LOG_LEVEL_TRACE = "LOG_LEVEL_TRACE"
    LOG_LEVEL_DEBUG = "LOG_LEVEL_DEBUG"
    LOG_LEVEL_INFO = "LOG_LEVEL_INFO"
    LOG_LEVEL_WARNING = "LOG_LEVEL_WARNING"
    LOG_LEVEL_ERROR = "LOG_LEVEL_ERROR"
    LOG_LEVEL_CRITICAL = "LOG_LEVEL_CRITICAL"

class v1LoginRequest:
    def __init__(
        self,
        password: str,
        username: str,
        isHashed: "t.Optional[bool]" = None,
    ):
        self.username = username
        self.password = password
        self.isHashed = isHashed

    @classmethod
    def from_json(cls, obj: Json) -> "v1LoginRequest":
        return cls(
            username=obj["username"],
            password=obj["password"],
            isHashed=obj["isHashed"] if obj.get("isHashed", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "username": self.username,
            "password": self.password,
            "isHashed": self.isHashed if self.isHashed is not None else None,
        }

class v1LoginResponse:
    def __init__(
        self,
        token: str,
        user: "v1User",
    ):
        self.token = token
        self.user = user

    @classmethod
    def from_json(cls, obj: Json) -> "v1LoginResponse":
        return cls(
            token=obj["token"],
            user=v1User.from_json(obj["user"]),
        )

    def to_json(self) -> t.Any:
        return {
            "token": self.token,
            "user": self.user.to_json(),
        }

class v1MarkAllocationReservationDaemonRequest:
    def __init__(
        self,
        allocationId: str,
        containerId: str,
    ):
        self.allocationId = allocationId
        self.containerId = containerId

    @classmethod
    def from_json(cls, obj: Json) -> "v1MarkAllocationReservationDaemonRequest":
        return cls(
            allocationId=obj["allocationId"],
            containerId=obj["containerId"],
        )

    def to_json(self) -> t.Any:
        return {
            "allocationId": self.allocationId,
            "containerId": self.containerId,
        }

class v1MasterLogsResponse:
    def __init__(
        self,
        logEntry: "t.Optional[v1LogEntry]" = None,
    ):
        self.logEntry = logEntry

    @classmethod
    def from_json(cls, obj: Json) -> "v1MasterLogsResponse":
        return cls(
            logEntry=v1LogEntry.from_json(obj["logEntry"]) if obj.get("logEntry", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "logEntry": self.logEntry.to_json() if self.logEntry is not None else None,
        }

class v1MetricBatchesResponse:
    def __init__(
        self,
        batches: "t.Optional[t.Sequence[int]]" = None,
    ):
        self.batches = batches

    @classmethod
    def from_json(cls, obj: Json) -> "v1MetricBatchesResponse":
        return cls(
            batches=obj["batches"] if obj.get("batches", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "batches": self.batches if self.batches is not None else None,
        }

class v1MetricNamesResponse:
    def __init__(
        self,
        searcherMetric: "t.Optional[str]" = None,
        trainingMetrics: "t.Optional[t.Sequence[str]]" = None,
        validationMetrics: "t.Optional[t.Sequence[str]]" = None,
    ):
        self.searcherMetric = searcherMetric
        self.trainingMetrics = trainingMetrics
        self.validationMetrics = validationMetrics

    @classmethod
    def from_json(cls, obj: Json) -> "v1MetricNamesResponse":
        return cls(
            searcherMetric=obj["searcherMetric"] if obj.get("searcherMetric", None) is not None else None,
            trainingMetrics=obj["trainingMetrics"] if obj.get("trainingMetrics", None) is not None else None,
            validationMetrics=obj["validationMetrics"] if obj.get("validationMetrics", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "searcherMetric": self.searcherMetric if self.searcherMetric is not None else None,
            "trainingMetrics": self.trainingMetrics if self.trainingMetrics is not None else None,
            "validationMetrics": self.validationMetrics if self.validationMetrics is not None else None,
        }

class v1MetricType(enum.Enum):
    METRIC_TYPE_UNSPECIFIED = "METRIC_TYPE_UNSPECIFIED"
    METRIC_TYPE_TRAINING = "METRIC_TYPE_TRAINING"
    METRIC_TYPE_VALIDATION = "METRIC_TYPE_VALIDATION"

class v1Metrics:
    def __init__(
        self,
        numInputs: "t.Optional[int]" = None,
        validationMetrics: "t.Optional[t.Dict[str, t.Any]]" = None,
    ):
        self.numInputs = numInputs
        self.validationMetrics = validationMetrics

    @classmethod
    def from_json(cls, obj: Json) -> "v1Metrics":
        return cls(
            numInputs=obj["numInputs"] if obj.get("numInputs", None) is not None else None,
            validationMetrics=obj["validationMetrics"] if obj.get("validationMetrics", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "numInputs": self.numInputs if self.numInputs is not None else None,
            "validationMetrics": self.validationMetrics if self.validationMetrics is not None else None,
        }

class v1MetricsWorkload:
    def __init__(
        self,
        numInputs: int,
        state: "determinedexperimentv1State",
        totalBatches: int,
        endTime: "t.Optional[str]" = None,
        metrics: "t.Optional[t.Dict[str, t.Any]]" = None,
    ):
        self.endTime = endTime
        self.state = state
        self.metrics = metrics
        self.numInputs = numInputs
        self.totalBatches = totalBatches

    @classmethod
    def from_json(cls, obj: Json) -> "v1MetricsWorkload":
        return cls(
            endTime=obj["endTime"] if obj.get("endTime", None) is not None else None,
            state=obj["state"],
            metrics=obj["metrics"] if obj.get("metrics", None) is not None else None,
            numInputs=obj["numInputs"],
            totalBatches=obj["totalBatches"],
        )

    def to_json(self) -> t.Any:
        return {
            "endTime": self.endTime if self.endTime is not None else None,
            "state": self.state,
            "metrics": self.metrics if self.metrics is not None else None,
            "numInputs": self.numInputs,
            "totalBatches": self.totalBatches,
        }

class v1Model:
    def __init__(
        self,
        creationTime: str,
        id: int,
        lastUpdatedTime: str,
        metadata: "t.Dict[str, t.Any]",
        name: str,
        numVersions: int,
        username: str,
        archived: "t.Optional[bool]" = None,
        description: "t.Optional[str]" = None,
        labels: "t.Optional[t.Sequence[str]]" = None,
        notes: "t.Optional[str]" = None,
    ):
        self.name = name
        self.description = description
        self.metadata = metadata
        self.creationTime = creationTime
        self.lastUpdatedTime = lastUpdatedTime
        self.id = id
        self.numVersions = numVersions
        self.labels = labels
        self.username = username
        self.archived = archived
        self.notes = notes

    @classmethod
    def from_json(cls, obj: Json) -> "v1Model":
        return cls(
            name=obj["name"],
            description=obj["description"] if obj.get("description", None) is not None else None,
            metadata=obj["metadata"],
            creationTime=obj["creationTime"],
            lastUpdatedTime=obj["lastUpdatedTime"],
            id=obj["id"],
            numVersions=obj["numVersions"],
            labels=obj["labels"] if obj.get("labels", None) is not None else None,
            username=obj["username"],
            archived=obj["archived"] if obj.get("archived", None) is not None else None,
            notes=obj["notes"] if obj.get("notes", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "name": self.name,
            "description": self.description if self.description is not None else None,
            "metadata": self.metadata,
            "creationTime": self.creationTime,
            "lastUpdatedTime": self.lastUpdatedTime,
            "id": self.id,
            "numVersions": self.numVersions,
            "labels": self.labels if self.labels is not None else None,
            "username": self.username,
            "archived": self.archived if self.archived is not None else None,
            "notes": self.notes if self.notes is not None else None,
        }

class v1ModelVersion:
    def __init__(
        self,
        checkpoint: "v1Checkpoint",
        creationTime: str,
        id: int,
        model: "v1Model",
        username: str,
        version: int,
        comment: "t.Optional[str]" = None,
        labels: "t.Optional[t.Sequence[str]]" = None,
        lastUpdatedTime: "t.Optional[str]" = None,
        metadata: "t.Optional[t.Dict[str, t.Any]]" = None,
        name: "t.Optional[str]" = None,
        notes: "t.Optional[str]" = None,
    ):
        self.model = model
        self.checkpoint = checkpoint
        self.version = version
        self.creationTime = creationTime
        self.id = id
        self.name = name
        self.metadata = metadata
        self.lastUpdatedTime = lastUpdatedTime
        self.comment = comment
        self.username = username
        self.labels = labels
        self.notes = notes

    @classmethod
    def from_json(cls, obj: Json) -> "v1ModelVersion":
        return cls(
            model=v1Model.from_json(obj["model"]),
            checkpoint=v1Checkpoint.from_json(obj["checkpoint"]),
            version=obj["version"],
            creationTime=obj["creationTime"],
            id=obj["id"],
            name=obj["name"] if obj.get("name", None) is not None else None,
            metadata=obj["metadata"] if obj.get("metadata", None) is not None else None,
            lastUpdatedTime=obj["lastUpdatedTime"] if obj.get("lastUpdatedTime", None) is not None else None,
            comment=obj["comment"] if obj.get("comment", None) is not None else None,
            username=obj["username"],
            labels=obj["labels"] if obj.get("labels", None) is not None else None,
            notes=obj["notes"] if obj.get("notes", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "model": self.model.to_json(),
            "checkpoint": self.checkpoint.to_json(),
            "version": self.version,
            "creationTime": self.creationTime,
            "id": self.id,
            "name": self.name if self.name is not None else None,
            "metadata": self.metadata if self.metadata is not None else None,
            "lastUpdatedTime": self.lastUpdatedTime if self.lastUpdatedTime is not None else None,
            "comment": self.comment if self.comment is not None else None,
            "username": self.username,
            "labels": self.labels if self.labels is not None else None,
            "notes": self.notes if self.notes is not None else None,
        }

class v1Notebook:
    def __init__(
        self,
        description: str,
        id: str,
        jobId: str,
        resourcePool: str,
        startTime: str,
        state: "determinedtaskv1State",
        username: str,
        container: "t.Optional[v1Container]" = None,
        exitStatus: "t.Optional[str]" = None,
        serviceAddress: "t.Optional[str]" = None,
    ):
        self.id = id
        self.description = description
        self.state = state
        self.startTime = startTime
        self.container = container
        self.username = username
        self.serviceAddress = serviceAddress
        self.resourcePool = resourcePool
        self.exitStatus = exitStatus
        self.jobId = jobId

    @classmethod
    def from_json(cls, obj: Json) -> "v1Notebook":
        return cls(
            id=obj["id"],
            description=obj["description"],
            state=obj["state"],
            startTime=obj["startTime"],
            container=v1Container.from_json(obj["container"]) if obj.get("container", None) is not None else None,
            username=obj["username"],
            serviceAddress=obj["serviceAddress"] if obj.get("serviceAddress", None) is not None else None,
            resourcePool=obj["resourcePool"],
            exitStatus=obj["exitStatus"] if obj.get("exitStatus", None) is not None else None,
            jobId=obj["jobId"],
        )

    def to_json(self) -> t.Any:
        return {
            "id": self.id,
            "description": self.description,
            "state": self.state,
            "startTime": self.startTime,
            "container": self.container.to_json() if self.container is not None else None,
            "username": self.username,
            "serviceAddress": self.serviceAddress if self.serviceAddress is not None else None,
            "resourcePool": self.resourcePool,
            "exitStatus": self.exitStatus if self.exitStatus is not None else None,
            "jobId": self.jobId,
        }

class v1NotebookLogsResponse:
    def __init__(
        self,
        logEntry: "t.Optional[v1LogEntry]" = None,
    ):
        self.logEntry = logEntry

    @classmethod
    def from_json(cls, obj: Json) -> "v1NotebookLogsResponse":
        return cls(
            logEntry=v1LogEntry.from_json(obj["logEntry"]) if obj.get("logEntry", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "logEntry": self.logEntry.to_json() if self.logEntry is not None else None,
        }

class v1OrderBy(enum.Enum):
    ORDER_BY_UNSPECIFIED = "ORDER_BY_UNSPECIFIED"
    ORDER_BY_ASC = "ORDER_BY_ASC"
    ORDER_BY_DESC = "ORDER_BY_DESC"

class v1Pagination:
    def __init__(
        self,
        endIndex: "t.Optional[int]" = None,
        limit: "t.Optional[int]" = None,
        offset: "t.Optional[int]" = None,
        startIndex: "t.Optional[int]" = None,
        total: "t.Optional[int]" = None,
    ):
        self.offset = offset
        self.limit = limit
        self.startIndex = startIndex
        self.endIndex = endIndex
        self.total = total

    @classmethod
    def from_json(cls, obj: Json) -> "v1Pagination":
        return cls(
            offset=obj["offset"] if obj.get("offset", None) is not None else None,
            limit=obj["limit"] if obj.get("limit", None) is not None else None,
            startIndex=obj["startIndex"] if obj.get("startIndex", None) is not None else None,
            endIndex=obj["endIndex"] if obj.get("endIndex", None) is not None else None,
            total=obj["total"] if obj.get("total", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "offset": self.offset if self.offset is not None else None,
            "limit": self.limit if self.limit is not None else None,
            "startIndex": self.startIndex if self.startIndex is not None else None,
            "endIndex": self.endIndex if self.endIndex is not None else None,
            "total": self.total if self.total is not None else None,
        }

class v1PaginationRequest:
    def __init__(
        self,
        limit: "t.Optional[int]" = None,
        offset: "t.Optional[int]" = None,
    ):
        self.offset = offset
        self.limit = limit

    @classmethod
    def from_json(cls, obj: Json) -> "v1PaginationRequest":
        return cls(
            offset=obj["offset"] if obj.get("offset", None) is not None else None,
            limit=obj["limit"] if obj.get("limit", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "offset": self.offset if self.offset is not None else None,
            "limit": self.limit if self.limit is not None else None,
        }

class v1PatchExperimentResponse:
    def __init__(
        self,
        experiment: "t.Optional[v1Experiment]" = None,
    ):
        self.experiment = experiment

    @classmethod
    def from_json(cls, obj: Json) -> "v1PatchExperimentResponse":
        return cls(
            experiment=v1Experiment.from_json(obj["experiment"]) if obj.get("experiment", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "experiment": self.experiment.to_json() if self.experiment is not None else None,
        }

class v1PatchModel:
    def __init__(
        self,
        description: "t.Optional[str]" = None,
        labels: "t.Optional[t.Sequence[t.Dict[str, t.Any]]]" = None,
        metadata: "t.Optional[t.Dict[str, t.Any]]" = None,
        name: "t.Optional[str]" = None,
        notes: "t.Optional[str]" = None,
    ):
        self.name = name
        self.description = description
        self.metadata = metadata
        self.labels = labels
        self.notes = notes

    @classmethod
    def from_json(cls, obj: Json) -> "v1PatchModel":
        return cls(
            name=obj["name"] if obj.get("name", None) is not None else None,
            description=obj["description"] if obj.get("description", None) is not None else None,
            metadata=obj["metadata"] if obj.get("metadata", None) is not None else None,
            labels=obj["labels"] if obj.get("labels", None) is not None else None,
            notes=obj["notes"] if obj.get("notes", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "name": self.name if self.name is not None else None,
            "description": self.description if self.description is not None else None,
            "metadata": self.metadata if self.metadata is not None else None,
            "labels": self.labels if self.labels is not None else None,
            "notes": self.notes if self.notes is not None else None,
        }

class v1PatchModelRequest:
    def __init__(
        self,
        model: "t.Optional[v1PatchModel]" = None,
        modelId: "t.Optional[int]" = None,
    ):
        self.model = model
        self.modelId = modelId

    @classmethod
    def from_json(cls, obj: Json) -> "v1PatchModelRequest":
        return cls(
            model=v1PatchModel.from_json(obj["model"]) if obj.get("model", None) is not None else None,
            modelId=obj["modelId"] if obj.get("modelId", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "model": self.model.to_json() if self.model is not None else None,
            "modelId": self.modelId if self.modelId is not None else None,
        }

class v1PatchModelResponse:
    def __init__(
        self,
        model: "t.Optional[v1Model]" = None,
    ):
        self.model = model

    @classmethod
    def from_json(cls, obj: Json) -> "v1PatchModelResponse":
        return cls(
            model=v1Model.from_json(obj["model"]) if obj.get("model", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "model": self.model.to_json() if self.model is not None else None,
        }

class v1PatchModelVersion:
    def __init__(
        self,
        checkpoint: "t.Optional[v1Checkpoint]" = None,
        comment: "t.Optional[str]" = None,
        labels: "t.Optional[t.Sequence[t.Dict[str, t.Any]]]" = None,
        metadata: "t.Optional[t.Dict[str, t.Any]]" = None,
        name: "t.Optional[str]" = None,
        notes: "t.Optional[str]" = None,
    ):
        self.checkpoint = checkpoint
        self.name = name
        self.metadata = metadata
        self.comment = comment
        self.labels = labels
        self.notes = notes

    @classmethod
    def from_json(cls, obj: Json) -> "v1PatchModelVersion":
        return cls(
            checkpoint=v1Checkpoint.from_json(obj["checkpoint"]) if obj.get("checkpoint", None) is not None else None,
            name=obj["name"] if obj.get("name", None) is not None else None,
            metadata=obj["metadata"] if obj.get("metadata", None) is not None else None,
            comment=obj["comment"] if obj.get("comment", None) is not None else None,
            labels=obj["labels"] if obj.get("labels", None) is not None else None,
            notes=obj["notes"] if obj.get("notes", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "checkpoint": self.checkpoint.to_json() if self.checkpoint is not None else None,
            "name": self.name if self.name is not None else None,
            "metadata": self.metadata if self.metadata is not None else None,
            "comment": self.comment if self.comment is not None else None,
            "labels": self.labels if self.labels is not None else None,
            "notes": self.notes if self.notes is not None else None,
        }

class v1PatchModelVersionRequest:
    def __init__(
        self,
        modelId: "t.Optional[int]" = None,
        modelVersion: "t.Optional[v1PatchModelVersion]" = None,
        modelVersionId: "t.Optional[int]" = None,
    ):
        self.modelId = modelId
        self.modelVersion = modelVersion
        self.modelVersionId = modelVersionId

    @classmethod
    def from_json(cls, obj: Json) -> "v1PatchModelVersionRequest":
        return cls(
            modelId=obj["modelId"] if obj.get("modelId", None) is not None else None,
            modelVersion=v1PatchModelVersion.from_json(obj["modelVersion"]) if obj.get("modelVersion", None) is not None else None,
            modelVersionId=obj["modelVersionId"] if obj.get("modelVersionId", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "modelId": self.modelId if self.modelId is not None else None,
            "modelVersion": self.modelVersion.to_json() if self.modelVersion is not None else None,
            "modelVersionId": self.modelVersionId if self.modelVersionId is not None else None,
        }

class v1PatchModelVersionResponse:
    def __init__(
        self,
        modelVersion: "t.Optional[v1ModelVersion]" = None,
    ):
        self.modelVersion = modelVersion

    @classmethod
    def from_json(cls, obj: Json) -> "v1PatchModelVersionResponse":
        return cls(
            modelVersion=v1ModelVersion.from_json(obj["modelVersion"]) if obj.get("modelVersion", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "modelVersion": self.modelVersion.to_json() if self.modelVersion is not None else None,
        }

class v1PostCheckpointMetadataRequest:
    def __init__(
        self,
        checkpoint: "t.Optional[v1Checkpoint]" = None,
    ):
        self.checkpoint = checkpoint

    @classmethod
    def from_json(cls, obj: Json) -> "v1PostCheckpointMetadataRequest":
        return cls(
            checkpoint=v1Checkpoint.from_json(obj["checkpoint"]) if obj.get("checkpoint", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "checkpoint": self.checkpoint.to_json() if self.checkpoint is not None else None,
        }

class v1PostCheckpointMetadataResponse:
    def __init__(
        self,
        checkpoint: "t.Optional[v1Checkpoint]" = None,
    ):
        self.checkpoint = checkpoint

    @classmethod
    def from_json(cls, obj: Json) -> "v1PostCheckpointMetadataResponse":
        return cls(
            checkpoint=v1Checkpoint.from_json(obj["checkpoint"]) if obj.get("checkpoint", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "checkpoint": self.checkpoint.to_json() if self.checkpoint is not None else None,
        }

class v1PostModelRequest:
    def __init__(
        self,
        name: str,
        description: "t.Optional[str]" = None,
        labels: "t.Optional[t.Sequence[str]]" = None,
        metadata: "t.Optional[t.Dict[str, t.Any]]" = None,
        notes: "t.Optional[str]" = None,
        username: "t.Optional[str]" = None,
    ):
        self.name = name
        self.description = description
        self.metadata = metadata
        self.labels = labels
        self.username = username
        self.notes = notes

    @classmethod
    def from_json(cls, obj: Json) -> "v1PostModelRequest":
        return cls(
            name=obj["name"],
            description=obj["description"] if obj.get("description", None) is not None else None,
            metadata=obj["metadata"] if obj.get("metadata", None) is not None else None,
            labels=obj["labels"] if obj.get("labels", None) is not None else None,
            username=obj["username"] if obj.get("username", None) is not None else None,
            notes=obj["notes"] if obj.get("notes", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "name": self.name,
            "description": self.description if self.description is not None else None,
            "metadata": self.metadata if self.metadata is not None else None,
            "labels": self.labels if self.labels is not None else None,
            "username": self.username if self.username is not None else None,
            "notes": self.notes if self.notes is not None else None,
        }

class v1PostModelResponse:
    def __init__(
        self,
        model: "t.Optional[v1Model]" = None,
    ):
        self.model = model

    @classmethod
    def from_json(cls, obj: Json) -> "v1PostModelResponse":
        return cls(
            model=v1Model.from_json(obj["model"]) if obj.get("model", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "model": self.model.to_json() if self.model is not None else None,
        }

class v1PostModelVersionRequest:
    def __init__(
        self,
        checkpointUuid: str,
        modelId: int,
        comment: "t.Optional[str]" = None,
        labels: "t.Optional[t.Sequence[str]]" = None,
        metadata: "t.Optional[t.Dict[str, t.Any]]" = None,
        name: "t.Optional[str]" = None,
        notes: "t.Optional[str]" = None,
    ):
        self.modelId = modelId
        self.checkpointUuid = checkpointUuid
        self.name = name
        self.comment = comment
        self.metadata = metadata
        self.labels = labels
        self.notes = notes

    @classmethod
    def from_json(cls, obj: Json) -> "v1PostModelVersionRequest":
        return cls(
            modelId=obj["modelId"],
            checkpointUuid=obj["checkpointUuid"],
            name=obj["name"] if obj.get("name", None) is not None else None,
            comment=obj["comment"] if obj.get("comment", None) is not None else None,
            metadata=obj["metadata"] if obj.get("metadata", None) is not None else None,
            labels=obj["labels"] if obj.get("labels", None) is not None else None,
            notes=obj["notes"] if obj.get("notes", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "modelId": self.modelId,
            "checkpointUuid": self.checkpointUuid,
            "name": self.name if self.name is not None else None,
            "comment": self.comment if self.comment is not None else None,
            "metadata": self.metadata if self.metadata is not None else None,
            "labels": self.labels if self.labels is not None else None,
            "notes": self.notes if self.notes is not None else None,
        }

class v1PostModelVersionResponse:
    def __init__(
        self,
        modelVersion: "t.Optional[v1ModelVersion]" = None,
    ):
        self.modelVersion = modelVersion

    @classmethod
    def from_json(cls, obj: Json) -> "v1PostModelVersionResponse":
        return cls(
            modelVersion=v1ModelVersion.from_json(obj["modelVersion"]) if obj.get("modelVersion", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "modelVersion": self.modelVersion.to_json() if self.modelVersion is not None else None,
        }

class v1PostTrialProfilerMetricsBatchRequest:
    def __init__(
        self,
        batches: "t.Optional[t.Sequence[v1TrialProfilerMetricsBatch]]" = None,
    ):
        self.batches = batches

    @classmethod
    def from_json(cls, obj: Json) -> "v1PostTrialProfilerMetricsBatchRequest":
        return cls(
            batches=[v1TrialProfilerMetricsBatch.from_json(x) for x in obj["batches"]] if obj.get("batches", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "batches": [x.to_json() for x in self.batches] if self.batches is not None else None,
        }

class v1PostUserRequest:
    def __init__(
        self,
        password: "t.Optional[str]" = None,
        user: "t.Optional[v1User]" = None,
    ):
        self.user = user
        self.password = password

    @classmethod
    def from_json(cls, obj: Json) -> "v1PostUserRequest":
        return cls(
            user=v1User.from_json(obj["user"]) if obj.get("user", None) is not None else None,
            password=obj["password"] if obj.get("password", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "user": self.user.to_json() if self.user is not None else None,
            "password": self.password if self.password is not None else None,
        }

class v1PostUserResponse:
    def __init__(
        self,
        user: "t.Optional[v1User]" = None,
    ):
        self.user = user

    @classmethod
    def from_json(cls, obj: Json) -> "v1PostUserResponse":
        return cls(
            user=v1User.from_json(obj["user"]) if obj.get("user", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "user": self.user.to_json() if self.user is not None else None,
        }

class v1PreviewHPSearchRequest:
    def __init__(
        self,
        config: "t.Optional[t.Dict[str, t.Any]]" = None,
        seed: "t.Optional[int]" = None,
    ):
        self.config = config
        self.seed = seed

    @classmethod
    def from_json(cls, obj: Json) -> "v1PreviewHPSearchRequest":
        return cls(
            config=obj["config"] if obj.get("config", None) is not None else None,
            seed=obj["seed"] if obj.get("seed", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "config": self.config if self.config is not None else None,
            "seed": self.seed if self.seed is not None else None,
        }

class v1PreviewHPSearchResponse:
    def __init__(
        self,
        simulation: "t.Optional[v1ExperimentSimulation]" = None,
    ):
        self.simulation = simulation

    @classmethod
    def from_json(cls, obj: Json) -> "v1PreviewHPSearchResponse":
        return cls(
            simulation=v1ExperimentSimulation.from_json(obj["simulation"]) if obj.get("simulation", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "simulation": self.simulation.to_json() if self.simulation is not None else None,
        }

class v1PutTemplateResponse:
    def __init__(
        self,
        template: "t.Optional[v1Template]" = None,
    ):
        self.template = template

    @classmethod
    def from_json(cls, obj: Json) -> "v1PutTemplateResponse":
        return cls(
            template=v1Template.from_json(obj["template"]) if obj.get("template", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "template": self.template.to_json() if self.template is not None else None,
        }

class v1QueueControl:
    def __init__(
        self,
        jobId: str,
        priority: "t.Optional[int]" = None,
        queuePosition: "t.Optional[float]" = None,
        resourcePool: "t.Optional[str]" = None,
        sourceResourcePool: "t.Optional[str]" = None,
        weight: "t.Optional[float]" = None,
    ):
        self.jobId = jobId
        self.sourceResourcePool = sourceResourcePool
        self.resourcePool = resourcePool
        self.queuePosition = queuePosition
        self.priority = priority
        self.weight = weight

    @classmethod
    def from_json(cls, obj: Json) -> "v1QueueControl":
        return cls(
            jobId=obj["jobId"],
            sourceResourcePool=obj["sourceResourcePool"] if obj.get("sourceResourcePool", None) is not None else None,
            resourcePool=obj["resourcePool"] if obj.get("resourcePool", None) is not None else None,
            queuePosition=float(obj["queuePosition"]) if obj.get("queuePosition", None) is not None else None,
            priority=obj["priority"] if obj.get("priority", None) is not None else None,
            weight=float(obj["weight"]) if obj.get("weight", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "jobId": self.jobId,
            "sourceResourcePool": self.sourceResourcePool if self.sourceResourcePool is not None else None,
            "resourcePool": self.resourcePool if self.resourcePool is not None else None,
            "queuePosition": dump_float(self.queuePosition) if self.queuePosition is not None else None,
            "priority": self.priority if self.priority is not None else None,
            "weight": dump_float(self.weight) if self.weight is not None else None,
        }

class v1QueueStats:
    def __init__(
        self,
        queuedCount: int,
        scheduledCount: int,
    ):
        self.queuedCount = queuedCount
        self.scheduledCount = scheduledCount

    @classmethod
    def from_json(cls, obj: Json) -> "v1QueueStats":
        return cls(
            queuedCount=obj["queuedCount"],
            scheduledCount=obj["scheduledCount"],
        )

    def to_json(self) -> t.Any:
        return {
            "queuedCount": self.queuedCount,
            "scheduledCount": self.scheduledCount,
        }

class v1RPQueueStat:
    def __init__(
        self,
        resourcePool: str,
        stats: "v1QueueStats",
    ):
        self.stats = stats
        self.resourcePool = resourcePool

    @classmethod
    def from_json(cls, obj: Json) -> "v1RPQueueStat":
        return cls(
            stats=v1QueueStats.from_json(obj["stats"]),
            resourcePool=obj["resourcePool"],
        )

    def to_json(self) -> t.Any:
        return {
            "stats": self.stats.to_json(),
            "resourcePool": self.resourcePool,
        }

class v1RendezvousInfo:
    def __init__(
        self,
        addresses: "t.Sequence[str]",
        rank: int,
    ):
        self.addresses = addresses
        self.rank = rank

    @classmethod
    def from_json(cls, obj: Json) -> "v1RendezvousInfo":
        return cls(
            addresses=obj["addresses"],
            rank=obj["rank"],
        )

    def to_json(self) -> t.Any:
        return {
            "addresses": self.addresses,
            "rank": self.rank,
        }

class v1ResourceAllocationAggregatedEntry:
    def __init__(
        self,
        byAgentLabel: "t.Dict[str, float]",
        byExperimentLabel: "t.Dict[str, float]",
        byResourcePool: "t.Dict[str, float]",
        byUsername: "t.Dict[str, float]",
        period: "v1ResourceAllocationAggregationPeriod",
        periodStart: str,
        seconds: float,
    ):
        self.periodStart = periodStart
        self.period = period
        self.seconds = seconds
        self.byUsername = byUsername
        self.byExperimentLabel = byExperimentLabel
        self.byResourcePool = byResourcePool
        self.byAgentLabel = byAgentLabel

    @classmethod
    def from_json(cls, obj: Json) -> "v1ResourceAllocationAggregatedEntry":
        return cls(
            periodStart=obj["periodStart"],
            period=obj["period"],
            seconds=float(obj["seconds"]),
            byUsername={k: float(v) for k, v in obj["byUsername"].items()},
            byExperimentLabel={k: float(v) for k, v in obj["byExperimentLabel"].items()},
            byResourcePool={k: float(v) for k, v in obj["byResourcePool"].items()},
            byAgentLabel={k: float(v) for k, v in obj["byAgentLabel"].items()},
        )

    def to_json(self) -> t.Any:
        return {
            "periodStart": self.periodStart,
            "period": self.period,
            "seconds": dump_float(self.seconds),
            "byUsername": {k: dump_float(v) for k, v in self.byUsername.items()},
            "byExperimentLabel": {k: dump_float(v) for k, v in self.byExperimentLabel.items()},
            "byResourcePool": {k: dump_float(v) for k, v in self.byResourcePool.items()},
            "byAgentLabel": {k: dump_float(v) for k, v in self.byAgentLabel.items()},
        }

class v1ResourceAllocationAggregatedResponse:
    def __init__(
        self,
        resourceEntries: "t.Sequence[v1ResourceAllocationAggregatedEntry]",
    ):
        self.resourceEntries = resourceEntries

    @classmethod
    def from_json(cls, obj: Json) -> "v1ResourceAllocationAggregatedResponse":
        return cls(
            resourceEntries=[v1ResourceAllocationAggregatedEntry.from_json(x) for x in obj["resourceEntries"]],
        )

    def to_json(self) -> t.Any:
        return {
            "resourceEntries": [x.to_json() for x in self.resourceEntries],
        }

class v1ResourceAllocationAggregationPeriod(enum.Enum):
    RESOURCE_ALLOCATION_AGGREGATION_PERIOD_UNSPECIFIED = "RESOURCE_ALLOCATION_AGGREGATION_PERIOD_UNSPECIFIED"
    RESOURCE_ALLOCATION_AGGREGATION_PERIOD_DAILY = "RESOURCE_ALLOCATION_AGGREGATION_PERIOD_DAILY"
    RESOURCE_ALLOCATION_AGGREGATION_PERIOD_MONTHLY = "RESOURCE_ALLOCATION_AGGREGATION_PERIOD_MONTHLY"

class v1ResourceAllocationRawEntry:
    def __init__(
        self,
        endTime: "t.Optional[str]" = None,
        experimentId: "t.Optional[int]" = None,
        kind: "t.Optional[str]" = None,
        labels: "t.Optional[t.Sequence[str]]" = None,
        seconds: "t.Optional[float]" = None,
        slots: "t.Optional[int]" = None,
        startTime: "t.Optional[str]" = None,
        username: "t.Optional[str]" = None,
    ):
        self.kind = kind
        self.startTime = startTime
        self.endTime = endTime
        self.experimentId = experimentId
        self.username = username
        self.labels = labels
        self.seconds = seconds
        self.slots = slots

    @classmethod
    def from_json(cls, obj: Json) -> "v1ResourceAllocationRawEntry":
        return cls(
            kind=obj["kind"] if obj.get("kind", None) is not None else None,
            startTime=obj["startTime"] if obj.get("startTime", None) is not None else None,
            endTime=obj["endTime"] if obj.get("endTime", None) is not None else None,
            experimentId=obj["experimentId"] if obj.get("experimentId", None) is not None else None,
            username=obj["username"] if obj.get("username", None) is not None else None,
            labels=obj["labels"] if obj.get("labels", None) is not None else None,
            seconds=float(obj["seconds"]) if obj.get("seconds", None) is not None else None,
            slots=obj["slots"] if obj.get("slots", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "kind": self.kind if self.kind is not None else None,
            "startTime": self.startTime if self.startTime is not None else None,
            "endTime": self.endTime if self.endTime is not None else None,
            "experimentId": self.experimentId if self.experimentId is not None else None,
            "username": self.username if self.username is not None else None,
            "labels": self.labels if self.labels is not None else None,
            "seconds": dump_float(self.seconds) if self.seconds is not None else None,
            "slots": self.slots if self.slots is not None else None,
        }

class v1ResourceAllocationRawResponse:
    def __init__(
        self,
        resourceEntries: "t.Optional[t.Sequence[v1ResourceAllocationRawEntry]]" = None,
    ):
        self.resourceEntries = resourceEntries

    @classmethod
    def from_json(cls, obj: Json) -> "v1ResourceAllocationRawResponse":
        return cls(
            resourceEntries=[v1ResourceAllocationRawEntry.from_json(x) for x in obj["resourceEntries"]] if obj.get("resourceEntries", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "resourceEntries": [x.to_json() for x in self.resourceEntries] if self.resourceEntries is not None else None,
        }

class v1ResourcePool:
    def __init__(
        self,
        agentDockerImage: str,
        agentDockerNetwork: str,
        agentDockerRuntime: str,
        agentFluentImage: str,
        auxContainerCapacity: int,
        auxContainerCapacityPerAgent: int,
        auxContainersRunning: int,
        containerStartupScript: str,
        defaultAuxPool: bool,
        defaultComputePool: bool,
        description: str,
        details: "v1ResourcePoolDetail",
        imageId: str,
        instanceType: str,
        location: str,
        masterCertName: str,
        masterUrl: str,
        maxAgentStartingPeriod: float,
        maxAgents: int,
        maxIdleAgentPeriod: float,
        minAgents: int,
        name: str,
        numAgents: int,
        preemptible: bool,
        schedulerFittingPolicy: "v1FittingPolicy",
        schedulerType: "v1SchedulerType",
        slotType: "determineddevicev1Type",
        slotsAvailable: int,
        slotsUsed: int,
        startupScript: str,
        type: "v1ResourcePoolType",
        slotsPerAgent: "t.Optional[int]" = None,
    ):
        self.name = name
        self.description = description
        self.type = type
        self.numAgents = numAgents
        self.slotsAvailable = slotsAvailable
        self.slotsUsed = slotsUsed
        self.slotType = slotType
        self.auxContainerCapacity = auxContainerCapacity
        self.auxContainersRunning = auxContainersRunning
        self.defaultComputePool = defaultComputePool
        self.defaultAuxPool = defaultAuxPool
        self.preemptible = preemptible
        self.minAgents = minAgents
        self.maxAgents = maxAgents
        self.slotsPerAgent = slotsPerAgent
        self.auxContainerCapacityPerAgent = auxContainerCapacityPerAgent
        self.schedulerType = schedulerType
        self.schedulerFittingPolicy = schedulerFittingPolicy
        self.location = location
        self.imageId = imageId
        self.instanceType = instanceType
        self.masterUrl = masterUrl
        self.masterCertName = masterCertName
        self.startupScript = startupScript
        self.containerStartupScript = containerStartupScript
        self.agentDockerNetwork = agentDockerNetwork
        self.agentDockerRuntime = agentDockerRuntime
        self.agentDockerImage = agentDockerImage
        self.agentFluentImage = agentFluentImage
        self.maxIdleAgentPeriod = maxIdleAgentPeriod
        self.maxAgentStartingPeriod = maxAgentStartingPeriod
        self.details = details

    @classmethod
    def from_json(cls, obj: Json) -> "v1ResourcePool":
        return cls(
            name=obj["name"],
            description=obj["description"],
            type=obj["type"],
            numAgents=obj["numAgents"],
            slotsAvailable=obj["slotsAvailable"],
            slotsUsed=obj["slotsUsed"],
            slotType=obj["slotType"],
            auxContainerCapacity=obj["auxContainerCapacity"],
            auxContainersRunning=obj["auxContainersRunning"],
            defaultComputePool=obj["defaultComputePool"],
            defaultAuxPool=obj["defaultAuxPool"],
            preemptible=obj["preemptible"],
            minAgents=obj["minAgents"],
            maxAgents=obj["maxAgents"],
            slotsPerAgent=obj["slotsPerAgent"] if obj.get("slotsPerAgent", None) is not None else None,
            auxContainerCapacityPerAgent=obj["auxContainerCapacityPerAgent"],
            schedulerType=obj["schedulerType"],
            schedulerFittingPolicy=obj["schedulerFittingPolicy"],
            location=obj["location"],
            imageId=obj["imageId"],
            instanceType=obj["instanceType"],
            masterUrl=obj["masterUrl"],
            masterCertName=obj["masterCertName"],
            startupScript=obj["startupScript"],
            containerStartupScript=obj["containerStartupScript"],
            agentDockerNetwork=obj["agentDockerNetwork"],
            agentDockerRuntime=obj["agentDockerRuntime"],
            agentDockerImage=obj["agentDockerImage"],
            agentFluentImage=obj["agentFluentImage"],
            maxIdleAgentPeriod=float(obj["maxIdleAgentPeriod"]),
            maxAgentStartingPeriod=float(obj["maxAgentStartingPeriod"]),
            details=v1ResourcePoolDetail.from_json(obj["details"]),
        )

    def to_json(self) -> t.Any:
        return {
            "name": self.name,
            "description": self.description,
            "type": self.type,
            "numAgents": self.numAgents,
            "slotsAvailable": self.slotsAvailable,
            "slotsUsed": self.slotsUsed,
            "slotType": self.slotType,
            "auxContainerCapacity": self.auxContainerCapacity,
            "auxContainersRunning": self.auxContainersRunning,
            "defaultComputePool": self.defaultComputePool,
            "defaultAuxPool": self.defaultAuxPool,
            "preemptible": self.preemptible,
            "minAgents": self.minAgents,
            "maxAgents": self.maxAgents,
            "slotsPerAgent": self.slotsPerAgent if self.slotsPerAgent is not None else None,
            "auxContainerCapacityPerAgent": self.auxContainerCapacityPerAgent,
            "schedulerType": self.schedulerType,
            "schedulerFittingPolicy": self.schedulerFittingPolicy,
            "location": self.location,
            "imageId": self.imageId,
            "instanceType": self.instanceType,
            "masterUrl": self.masterUrl,
            "masterCertName": self.masterCertName,
            "startupScript": self.startupScript,
            "containerStartupScript": self.containerStartupScript,
            "agentDockerNetwork": self.agentDockerNetwork,
            "agentDockerRuntime": self.agentDockerRuntime,
            "agentDockerImage": self.agentDockerImage,
            "agentFluentImage": self.agentFluentImage,
            "maxIdleAgentPeriod": dump_float(self.maxIdleAgentPeriod),
            "maxAgentStartingPeriod": dump_float(self.maxAgentStartingPeriod),
            "details": self.details.to_json(),
        }

class v1ResourcePoolAwsDetail:
    def __init__(
        self,
        iamInstanceProfileArn: str,
        imageId: str,
        instanceName: str,
        publicIp: bool,
        region: str,
        rootVolumeSize: int,
        securityGroupId: str,
        spotEnabled: bool,
        sshKeyName: str,
        tagKey: str,
        tagValue: str,
        customTags: "t.Optional[t.Sequence[v1AwsCustomTag]]" = None,
        instanceType: "t.Optional[str]" = None,
        logGroup: "t.Optional[str]" = None,
        logStream: "t.Optional[str]" = None,
        spotMaxPrice: "t.Optional[str]" = None,
        subnetId: "t.Optional[str]" = None,
    ):
        self.region = region
        self.rootVolumeSize = rootVolumeSize
        self.imageId = imageId
        self.tagKey = tagKey
        self.tagValue = tagValue
        self.instanceName = instanceName
        self.sshKeyName = sshKeyName
        self.publicIp = publicIp
        self.subnetId = subnetId
        self.securityGroupId = securityGroupId
        self.iamInstanceProfileArn = iamInstanceProfileArn
        self.instanceType = instanceType
        self.logGroup = logGroup
        self.logStream = logStream
        self.spotEnabled = spotEnabled
        self.spotMaxPrice = spotMaxPrice
        self.customTags = customTags

    @classmethod
    def from_json(cls, obj: Json) -> "v1ResourcePoolAwsDetail":
        return cls(
            region=obj["region"],
            rootVolumeSize=obj["rootVolumeSize"],
            imageId=obj["imageId"],
            tagKey=obj["tagKey"],
            tagValue=obj["tagValue"],
            instanceName=obj["instanceName"],
            sshKeyName=obj["sshKeyName"],
            publicIp=obj["publicIp"],
            subnetId=obj["subnetId"] if obj.get("subnetId", None) is not None else None,
            securityGroupId=obj["securityGroupId"],
            iamInstanceProfileArn=obj["iamInstanceProfileArn"],
            instanceType=obj["instanceType"] if obj.get("instanceType", None) is not None else None,
            logGroup=obj["logGroup"] if obj.get("logGroup", None) is not None else None,
            logStream=obj["logStream"] if obj.get("logStream", None) is not None else None,
            spotEnabled=obj["spotEnabled"],
            spotMaxPrice=obj["spotMaxPrice"] if obj.get("spotMaxPrice", None) is not None else None,
            customTags=[v1AwsCustomTag.from_json(x) for x in obj["customTags"]] if obj.get("customTags", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "region": self.region,
            "rootVolumeSize": self.rootVolumeSize,
            "imageId": self.imageId,
            "tagKey": self.tagKey,
            "tagValue": self.tagValue,
            "instanceName": self.instanceName,
            "sshKeyName": self.sshKeyName,
            "publicIp": self.publicIp,
            "subnetId": self.subnetId if self.subnetId is not None else None,
            "securityGroupId": self.securityGroupId,
            "iamInstanceProfileArn": self.iamInstanceProfileArn,
            "instanceType": self.instanceType if self.instanceType is not None else None,
            "logGroup": self.logGroup if self.logGroup is not None else None,
            "logStream": self.logStream if self.logStream is not None else None,
            "spotEnabled": self.spotEnabled,
            "spotMaxPrice": self.spotMaxPrice if self.spotMaxPrice is not None else None,
            "customTags": [x.to_json() for x in self.customTags] if self.customTags is not None else None,
        }

class v1ResourcePoolDetail:
    def __init__(
        self,
        aws: "t.Optional[v1ResourcePoolAwsDetail]" = None,
        gcp: "t.Optional[v1ResourcePoolGcpDetail]" = None,
        priorityScheduler: "t.Optional[v1ResourcePoolPrioritySchedulerDetail]" = None,
    ):
        self.aws = aws
        self.gcp = gcp
        self.priorityScheduler = priorityScheduler

    @classmethod
    def from_json(cls, obj: Json) -> "v1ResourcePoolDetail":
        return cls(
            aws=v1ResourcePoolAwsDetail.from_json(obj["aws"]) if obj.get("aws", None) is not None else None,
            gcp=v1ResourcePoolGcpDetail.from_json(obj["gcp"]) if obj.get("gcp", None) is not None else None,
            priorityScheduler=v1ResourcePoolPrioritySchedulerDetail.from_json(obj["priorityScheduler"]) if obj.get("priorityScheduler", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "aws": self.aws.to_json() if self.aws is not None else None,
            "gcp": self.gcp.to_json() if self.gcp is not None else None,
            "priorityScheduler": self.priorityScheduler.to_json() if self.priorityScheduler is not None else None,
        }

class v1ResourcePoolGcpDetail:
    def __init__(
        self,
        bootDiskSize: int,
        bootDiskSourceImage: str,
        externalIp: bool,
        gpuNum: int,
        gpuType: str,
        labelKey: str,
        labelValue: str,
        machineType: str,
        namePrefix: str,
        network: str,
        operationTimeoutPeriod: float,
        preemptible: bool,
        project: str,
        serviceAccountEmail: str,
        serviceAccountScopes: "t.Sequence[str]",
        zone: str,
        networkTags: "t.Optional[t.Sequence[str]]" = None,
        subnetwork: "t.Optional[str]" = None,
    ):
        self.project = project
        self.zone = zone
        self.bootDiskSize = bootDiskSize
        self.bootDiskSourceImage = bootDiskSourceImage
        self.labelKey = labelKey
        self.labelValue = labelValue
        self.namePrefix = namePrefix
        self.network = network
        self.subnetwork = subnetwork
        self.externalIp = externalIp
        self.networkTags = networkTags
        self.serviceAccountEmail = serviceAccountEmail
        self.serviceAccountScopes = serviceAccountScopes
        self.machineType = machineType
        self.gpuType = gpuType
        self.gpuNum = gpuNum
        self.preemptible = preemptible
        self.operationTimeoutPeriod = operationTimeoutPeriod

    @classmethod
    def from_json(cls, obj: Json) -> "v1ResourcePoolGcpDetail":
        return cls(
            project=obj["project"],
            zone=obj["zone"],
            bootDiskSize=obj["bootDiskSize"],
            bootDiskSourceImage=obj["bootDiskSourceImage"],
            labelKey=obj["labelKey"],
            labelValue=obj["labelValue"],
            namePrefix=obj["namePrefix"],
            network=obj["network"],
            subnetwork=obj["subnetwork"] if obj.get("subnetwork", None) is not None else None,
            externalIp=obj["externalIp"],
            networkTags=obj["networkTags"] if obj.get("networkTags", None) is not None else None,
            serviceAccountEmail=obj["serviceAccountEmail"],
            serviceAccountScopes=obj["serviceAccountScopes"],
            machineType=obj["machineType"],
            gpuType=obj["gpuType"],
            gpuNum=obj["gpuNum"],
            preemptible=obj["preemptible"],
            operationTimeoutPeriod=float(obj["operationTimeoutPeriod"]),
        )

    def to_json(self) -> t.Any:
        return {
            "project": self.project,
            "zone": self.zone,
            "bootDiskSize": self.bootDiskSize,
            "bootDiskSourceImage": self.bootDiskSourceImage,
            "labelKey": self.labelKey,
            "labelValue": self.labelValue,
            "namePrefix": self.namePrefix,
            "network": self.network,
            "subnetwork": self.subnetwork if self.subnetwork is not None else None,
            "externalIp": self.externalIp,
            "networkTags": self.networkTags if self.networkTags is not None else None,
            "serviceAccountEmail": self.serviceAccountEmail,
            "serviceAccountScopes": self.serviceAccountScopes,
            "machineType": self.machineType,
            "gpuType": self.gpuType,
            "gpuNum": self.gpuNum,
            "preemptible": self.preemptible,
            "operationTimeoutPeriod": dump_float(self.operationTimeoutPeriod),
        }

class v1ResourcePoolPrioritySchedulerDetail:
    def __init__(
        self,
        defaultPriority: int,
        preemption: bool,
        k8Priorities: "t.Optional[t.Sequence[v1K8PriorityClass]]" = None,
    ):
        self.preemption = preemption
        self.defaultPriority = defaultPriority
        self.k8Priorities = k8Priorities

    @classmethod
    def from_json(cls, obj: Json) -> "v1ResourcePoolPrioritySchedulerDetail":
        return cls(
            preemption=obj["preemption"],
            defaultPriority=obj["defaultPriority"],
            k8Priorities=[v1K8PriorityClass.from_json(x) for x in obj["k8Priorities"]] if obj.get("k8Priorities", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "preemption": self.preemption,
            "defaultPriority": self.defaultPriority,
            "k8Priorities": [x.to_json() for x in self.k8Priorities] if self.k8Priorities is not None else None,
        }

class v1ResourcePoolType(enum.Enum):
    RESOURCE_POOL_TYPE_UNSPECIFIED = "RESOURCE_POOL_TYPE_UNSPECIFIED"
    RESOURCE_POOL_TYPE_AWS = "RESOURCE_POOL_TYPE_AWS"
    RESOURCE_POOL_TYPE_GCP = "RESOURCE_POOL_TYPE_GCP"
    RESOURCE_POOL_TYPE_STATIC = "RESOURCE_POOL_TYPE_STATIC"
    RESOURCE_POOL_TYPE_K8S = "RESOURCE_POOL_TYPE_K8S"

class v1RunnableOperation:
    def __init__(
        self,
        length: "t.Optional[v1TrainingLength]" = None,
        type: "t.Optional[v1RunnableType]" = None,
    ):
        self.type = type
        self.length = length

    @classmethod
    def from_json(cls, obj: Json) -> "v1RunnableOperation":
        return cls(
            type=obj["type"] if obj.get("type", None) is not None else None,
            length=v1TrainingLength.from_json(obj["length"]) if obj.get("length", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "type": self.type if self.type is not None else None,
            "length": self.length.to_json() if self.length is not None else None,
        }

class v1RunnableType(enum.Enum):
    RUNNABLE_TYPE_UNSPECIFIED = "RUNNABLE_TYPE_UNSPECIFIED"
    RUNNABLE_TYPE_TRAIN = "RUNNABLE_TYPE_TRAIN"
    RUNNABLE_TYPE_VALIDATE = "RUNNABLE_TYPE_VALIDATE"

class v1SSOProvider:
    def __init__(
        self,
        name: str,
        ssoUrl: str,
    ):
        self.name = name
        self.ssoUrl = ssoUrl

    @classmethod
    def from_json(cls, obj: Json) -> "v1SSOProvider":
        return cls(
            name=obj["name"],
            ssoUrl=obj["ssoUrl"],
        )

    def to_json(self) -> t.Any:
        return {
            "name": self.name,
            "ssoUrl": self.ssoUrl,
        }

class v1SchedulerType(enum.Enum):
    SCHEDULER_TYPE_UNSPECIFIED = "SCHEDULER_TYPE_UNSPECIFIED"
    SCHEDULER_TYPE_PRIORITY = "SCHEDULER_TYPE_PRIORITY"
    SCHEDULER_TYPE_FAIR_SHARE = "SCHEDULER_TYPE_FAIR_SHARE"
    SCHEDULER_TYPE_ROUND_ROBIN = "SCHEDULER_TYPE_ROUND_ROBIN"
    SCHEDULER_TYPE_KUBERNETES = "SCHEDULER_TYPE_KUBERNETES"

class v1SearcherOperation:
    def __init__(
        self,
        validateAfter: "t.Optional[v1ValidateAfterOperation]" = None,
    ):
        self.validateAfter = validateAfter

    @classmethod
    def from_json(cls, obj: Json) -> "v1SearcherOperation":
        return cls(
            validateAfter=v1ValidateAfterOperation.from_json(obj["validateAfter"]) if obj.get("validateAfter", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "validateAfter": self.validateAfter.to_json() if self.validateAfter is not None else None,
        }

class v1SetCommandPriorityRequest:
    def __init__(
        self,
        commandId: "t.Optional[str]" = None,
        priority: "t.Optional[int]" = None,
    ):
        self.commandId = commandId
        self.priority = priority

    @classmethod
    def from_json(cls, obj: Json) -> "v1SetCommandPriorityRequest":
        return cls(
            commandId=obj["commandId"] if obj.get("commandId", None) is not None else None,
            priority=obj["priority"] if obj.get("priority", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "commandId": self.commandId if self.commandId is not None else None,
            "priority": self.priority if self.priority is not None else None,
        }

class v1SetCommandPriorityResponse:
    def __init__(
        self,
        command: "t.Optional[v1Command]" = None,
    ):
        self.command = command

    @classmethod
    def from_json(cls, obj: Json) -> "v1SetCommandPriorityResponse":
        return cls(
            command=v1Command.from_json(obj["command"]) if obj.get("command", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "command": self.command.to_json() if self.command is not None else None,
        }

class v1SetNotebookPriorityRequest:
    def __init__(
        self,
        notebookId: "t.Optional[str]" = None,
        priority: "t.Optional[int]" = None,
    ):
        self.notebookId = notebookId
        self.priority = priority

    @classmethod
    def from_json(cls, obj: Json) -> "v1SetNotebookPriorityRequest":
        return cls(
            notebookId=obj["notebookId"] if obj.get("notebookId", None) is not None else None,
            priority=obj["priority"] if obj.get("priority", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "notebookId": self.notebookId if self.notebookId is not None else None,
            "priority": self.priority if self.priority is not None else None,
        }

class v1SetNotebookPriorityResponse:
    def __init__(
        self,
        notebook: "t.Optional[v1Notebook]" = None,
    ):
        self.notebook = notebook

    @classmethod
    def from_json(cls, obj: Json) -> "v1SetNotebookPriorityResponse":
        return cls(
            notebook=v1Notebook.from_json(obj["notebook"]) if obj.get("notebook", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "notebook": self.notebook.to_json() if self.notebook is not None else None,
        }

class v1SetShellPriorityRequest:
    def __init__(
        self,
        priority: "t.Optional[int]" = None,
        shellId: "t.Optional[str]" = None,
    ):
        self.shellId = shellId
        self.priority = priority

    @classmethod
    def from_json(cls, obj: Json) -> "v1SetShellPriorityRequest":
        return cls(
            shellId=obj["shellId"] if obj.get("shellId", None) is not None else None,
            priority=obj["priority"] if obj.get("priority", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "shellId": self.shellId if self.shellId is not None else None,
            "priority": self.priority if self.priority is not None else None,
        }

class v1SetShellPriorityResponse:
    def __init__(
        self,
        shell: "t.Optional[v1Shell]" = None,
    ):
        self.shell = shell

    @classmethod
    def from_json(cls, obj: Json) -> "v1SetShellPriorityResponse":
        return cls(
            shell=v1Shell.from_json(obj["shell"]) if obj.get("shell", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "shell": self.shell.to_json() if self.shell is not None else None,
        }

class v1SetTensorboardPriorityRequest:
    def __init__(
        self,
        priority: "t.Optional[int]" = None,
        tensorboardId: "t.Optional[str]" = None,
    ):
        self.tensorboardId = tensorboardId
        self.priority = priority

    @classmethod
    def from_json(cls, obj: Json) -> "v1SetTensorboardPriorityRequest":
        return cls(
            tensorboardId=obj["tensorboardId"] if obj.get("tensorboardId", None) is not None else None,
            priority=obj["priority"] if obj.get("priority", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "tensorboardId": self.tensorboardId if self.tensorboardId is not None else None,
            "priority": self.priority if self.priority is not None else None,
        }

class v1SetTensorboardPriorityResponse:
    def __init__(
        self,
        tensorboard: "t.Optional[v1Tensorboard]" = None,
    ):
        self.tensorboard = tensorboard

    @classmethod
    def from_json(cls, obj: Json) -> "v1SetTensorboardPriorityResponse":
        return cls(
            tensorboard=v1Tensorboard.from_json(obj["tensorboard"]) if obj.get("tensorboard", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "tensorboard": self.tensorboard.to_json() if self.tensorboard is not None else None,
        }

class v1SetUserPasswordResponse:
    def __init__(
        self,
        user: "t.Optional[v1User]" = None,
    ):
        self.user = user

    @classmethod
    def from_json(cls, obj: Json) -> "v1SetUserPasswordResponse":
        return cls(
            user=v1User.from_json(obj["user"]) if obj.get("user", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "user": self.user.to_json() if self.user is not None else None,
        }

class v1Shell:
    def __init__(
        self,
        description: str,
        id: str,
        jobId: str,
        resourcePool: str,
        startTime: str,
        state: "determinedtaskv1State",
        username: str,
        addresses: "t.Optional[t.Sequence[t.Dict[str, t.Any]]]" = None,
        agentUserGroup: "t.Optional[t.Dict[str, t.Any]]" = None,
        container: "t.Optional[v1Container]" = None,
        exitStatus: "t.Optional[str]" = None,
        privateKey: "t.Optional[str]" = None,
        publicKey: "t.Optional[str]" = None,
    ):
        self.id = id
        self.description = description
        self.state = state
        self.startTime = startTime
        self.container = container
        self.privateKey = privateKey
        self.publicKey = publicKey
        self.username = username
        self.resourcePool = resourcePool
        self.exitStatus = exitStatus
        self.addresses = addresses
        self.agentUserGroup = agentUserGroup
        self.jobId = jobId

    @classmethod
    def from_json(cls, obj: Json) -> "v1Shell":
        return cls(
            id=obj["id"],
            description=obj["description"],
            state=obj["state"],
            startTime=obj["startTime"],
            container=v1Container.from_json(obj["container"]) if obj.get("container", None) is not None else None,
            privateKey=obj["privateKey"] if obj.get("privateKey", None) is not None else None,
            publicKey=obj["publicKey"] if obj.get("publicKey", None) is not None else None,
            username=obj["username"],
            resourcePool=obj["resourcePool"],
            exitStatus=obj["exitStatus"] if obj.get("exitStatus", None) is not None else None,
            addresses=obj["addresses"] if obj.get("addresses", None) is not None else None,
            agentUserGroup=obj["agentUserGroup"] if obj.get("agentUserGroup", None) is not None else None,
            jobId=obj["jobId"],
        )

    def to_json(self) -> t.Any:
        return {
            "id": self.id,
            "description": self.description,
            "state": self.state,
            "startTime": self.startTime,
            "container": self.container.to_json() if self.container is not None else None,
            "privateKey": self.privateKey if self.privateKey is not None else None,
            "publicKey": self.publicKey if self.publicKey is not None else None,
            "username": self.username,
            "resourcePool": self.resourcePool,
            "exitStatus": self.exitStatus if self.exitStatus is not None else None,
            "addresses": self.addresses if self.addresses is not None else None,
            "agentUserGroup": self.agentUserGroup if self.agentUserGroup is not None else None,
            "jobId": self.jobId,
        }

class v1Slot:
    def __init__(
        self,
        container: "t.Optional[v1Container]" = None,
        device: "t.Optional[v1Device]" = None,
        draining: "t.Optional[bool]" = None,
        enabled: "t.Optional[bool]" = None,
        id: "t.Optional[str]" = None,
    ):
        self.id = id
        self.device = device
        self.enabled = enabled
        self.container = container
        self.draining = draining

    @classmethod
    def from_json(cls, obj: Json) -> "v1Slot":
        return cls(
            id=obj["id"] if obj.get("id", None) is not None else None,
            device=v1Device.from_json(obj["device"]) if obj.get("device", None) is not None else None,
            enabled=obj["enabled"] if obj.get("enabled", None) is not None else None,
            container=v1Container.from_json(obj["container"]) if obj.get("container", None) is not None else None,
            draining=obj["draining"] if obj.get("draining", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "id": self.id if self.id is not None else None,
            "device": self.device.to_json() if self.device is not None else None,
            "enabled": self.enabled if self.enabled is not None else None,
            "container": self.container.to_json() if self.container is not None else None,
            "draining": self.draining if self.draining is not None else None,
        }

class v1Template:
    def __init__(
        self,
        config: "t.Dict[str, t.Any]",
        name: str,
    ):
        self.name = name
        self.config = config

    @classmethod
    def from_json(cls, obj: Json) -> "v1Template":
        return cls(
            name=obj["name"],
            config=obj["config"],
        )

    def to_json(self) -> t.Any:
        return {
            "name": self.name,
            "config": self.config,
        }

class v1Tensorboard:
    def __init__(
        self,
        description: str,
        id: str,
        jobId: str,
        resourcePool: str,
        startTime: str,
        state: "determinedtaskv1State",
        username: str,
        container: "t.Optional[v1Container]" = None,
        exitStatus: "t.Optional[str]" = None,
        experimentIds: "t.Optional[t.Sequence[int]]" = None,
        serviceAddress: "t.Optional[str]" = None,
        trialIds: "t.Optional[t.Sequence[int]]" = None,
    ):
        self.id = id
        self.description = description
        self.state = state
        self.startTime = startTime
        self.container = container
        self.experimentIds = experimentIds
        self.trialIds = trialIds
        self.username = username
        self.serviceAddress = serviceAddress
        self.resourcePool = resourcePool
        self.exitStatus = exitStatus
        self.jobId = jobId

    @classmethod
    def from_json(cls, obj: Json) -> "v1Tensorboard":
        return cls(
            id=obj["id"],
            description=obj["description"],
            state=obj["state"],
            startTime=obj["startTime"],
            container=v1Container.from_json(obj["container"]) if obj.get("container", None) is not None else None,
            experimentIds=obj["experimentIds"] if obj.get("experimentIds", None) is not None else None,
            trialIds=obj["trialIds"] if obj.get("trialIds", None) is not None else None,
            username=obj["username"],
            serviceAddress=obj["serviceAddress"] if obj.get("serviceAddress", None) is not None else None,
            resourcePool=obj["resourcePool"],
            exitStatus=obj["exitStatus"] if obj.get("exitStatus", None) is not None else None,
            jobId=obj["jobId"],
        )

    def to_json(self) -> t.Any:
        return {
            "id": self.id,
            "description": self.description,
            "state": self.state,
            "startTime": self.startTime,
            "container": self.container.to_json() if self.container is not None else None,
            "experimentIds": self.experimentIds if self.experimentIds is not None else None,
            "trialIds": self.trialIds if self.trialIds is not None else None,
            "username": self.username,
            "serviceAddress": self.serviceAddress if self.serviceAddress is not None else None,
            "resourcePool": self.resourcePool,
            "exitStatus": self.exitStatus if self.exitStatus is not None else None,
            "jobId": self.jobId,
        }

class v1TrainingLength:
    def __init__(
        self,
        length: int,
        unit: "TrainingLengthUnit",
    ):
        self.unit = unit
        self.length = length

    @classmethod
    def from_json(cls, obj: Json) -> "v1TrainingLength":
        return cls(
            unit=obj["unit"],
            length=obj["length"],
        )

    def to_json(self) -> t.Any:
        return {
            "unit": self.unit,
            "length": self.length,
        }

class v1TrialEarlyExit:
    def __init__(
        self,
        reason: "TrialEarlyExitExitedReason",
    ):
        self.reason = reason

    @classmethod
    def from_json(cls, obj: Json) -> "v1TrialEarlyExit":
        return cls(
            reason=obj["reason"],
        )

    def to_json(self) -> t.Any:
        return {
            "reason": self.reason,
        }

class v1TrialLogsFieldsResponse:
    def __init__(
        self,
        agentIds: "t.Optional[t.Sequence[str]]" = None,
        containerIds: "t.Optional[t.Sequence[str]]" = None,
        rankIds: "t.Optional[t.Sequence[int]]" = None,
        sources: "t.Optional[t.Sequence[str]]" = None,
        stdtypes: "t.Optional[t.Sequence[str]]" = None,
    ):
        self.agentIds = agentIds
        self.containerIds = containerIds
        self.rankIds = rankIds
        self.stdtypes = stdtypes
        self.sources = sources

    @classmethod
    def from_json(cls, obj: Json) -> "v1TrialLogsFieldsResponse":
        return cls(
            agentIds=obj["agentIds"] if obj.get("agentIds", None) is not None else None,
            containerIds=obj["containerIds"] if obj.get("containerIds", None) is not None else None,
            rankIds=obj["rankIds"] if obj.get("rankIds", None) is not None else None,
            stdtypes=obj["stdtypes"] if obj.get("stdtypes", None) is not None else None,
            sources=obj["sources"] if obj.get("sources", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "agentIds": self.agentIds if self.agentIds is not None else None,
            "containerIds": self.containerIds if self.containerIds is not None else None,
            "rankIds": self.rankIds if self.rankIds is not None else None,
            "stdtypes": self.stdtypes if self.stdtypes is not None else None,
            "sources": self.sources if self.sources is not None else None,
        }

class v1TrialLogsResponse:
    def __init__(
        self,
        id: str,
        level: "v1LogLevel",
        message: str,
        timestamp: str,
    ):
        self.id = id
        self.timestamp = timestamp
        self.message = message
        self.level = level

    @classmethod
    def from_json(cls, obj: Json) -> "v1TrialLogsResponse":
        return cls(
            id=obj["id"],
            timestamp=obj["timestamp"],
            message=obj["message"],
            level=obj["level"],
        )

    def to_json(self) -> t.Any:
        return {
            "id": self.id,
            "timestamp": self.timestamp,
            "message": self.message,
            "level": self.level,
        }

class v1TrialMetrics:
    def __init__(
        self,
        latestBatch: int,
        metrics: "t.Dict[str, t.Any]",
        trialId: int,
        trialRunId: int,
        batchMetrics: "t.Optional[t.Sequence[t.Dict[str, t.Any]]]" = None,
    ):
        self.trialId = trialId
        self.trialRunId = trialRunId
        self.latestBatch = latestBatch
        self.metrics = metrics
        self.batchMetrics = batchMetrics

    @classmethod
    def from_json(cls, obj: Json) -> "v1TrialMetrics":
        return cls(
            trialId=obj["trialId"],
            trialRunId=obj["trialRunId"],
            latestBatch=obj["latestBatch"],
            metrics=obj["metrics"],
            batchMetrics=obj["batchMetrics"] if obj.get("batchMetrics", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "trialId": self.trialId,
            "trialRunId": self.trialRunId,
            "latestBatch": self.latestBatch,
            "metrics": self.metrics,
            "batchMetrics": self.batchMetrics if self.batchMetrics is not None else None,
        }

class v1TrialProfilerMetricLabels:
    def __init__(
        self,
        name: str,
        trialId: int,
        agentId: "t.Optional[str]" = None,
        gpuUuid: "t.Optional[str]" = None,
        metricType: "t.Optional[TrialProfilerMetricLabelsProfilerMetricType]" = None,
    ):
        self.trialId = trialId
        self.name = name
        self.agentId = agentId
        self.gpuUuid = gpuUuid
        self.metricType = metricType

    @classmethod
    def from_json(cls, obj: Json) -> "v1TrialProfilerMetricLabels":
        return cls(
            trialId=obj["trialId"],
            name=obj["name"],
            agentId=obj["agentId"] if obj.get("agentId", None) is not None else None,
            gpuUuid=obj["gpuUuid"] if obj.get("gpuUuid", None) is not None else None,
            metricType=obj["metricType"] if obj.get("metricType", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "trialId": self.trialId,
            "name": self.name,
            "agentId": self.agentId if self.agentId is not None else None,
            "gpuUuid": self.gpuUuid if self.gpuUuid is not None else None,
            "metricType": self.metricType if self.metricType is not None else None,
        }

class v1TrialProfilerMetricsBatch:
    def __init__(
        self,
        batches: "t.Sequence[int]",
        labels: "v1TrialProfilerMetricLabels",
        timestamps: "t.Sequence[str]",
        values: "t.Sequence[float]",
    ):
        self.values = values
        self.batches = batches
        self.timestamps = timestamps
        self.labels = labels

    @classmethod
    def from_json(cls, obj: Json) -> "v1TrialProfilerMetricsBatch":
        return cls(
            values=[float(x) for x in obj["values"]],
            batches=obj["batches"],
            timestamps=obj["timestamps"],
            labels=v1TrialProfilerMetricLabels.from_json(obj["labels"]),
        )

    def to_json(self) -> t.Any:
        return {
            "values": [dump_float(x) for x in self.values],
            "batches": self.batches,
            "timestamps": self.timestamps,
            "labels": self.labels.to_json(),
        }

class v1TrialRunnerMetadata:
    def __init__(
        self,
        state: str,
    ):
        self.state = state

    @classmethod
    def from_json(cls, obj: Json) -> "v1TrialRunnerMetadata":
        return cls(
            state=obj["state"],
        )

    def to_json(self) -> t.Any:
        return {
            "state": self.state,
        }

class v1TrialSimulation:
    def __init__(
        self,
        occurrences: "t.Optional[int]" = None,
        operations: "t.Optional[t.Sequence[v1RunnableOperation]]" = None,
    ):
        self.operations = operations
        self.occurrences = occurrences

    @classmethod
    def from_json(cls, obj: Json) -> "v1TrialSimulation":
        return cls(
            operations=[v1RunnableOperation.from_json(x) for x in obj["operations"]] if obj.get("operations", None) is not None else None,
            occurrences=obj["occurrences"] if obj.get("occurrences", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "operations": [x.to_json() for x in self.operations] if self.operations is not None else None,
            "occurrences": self.occurrences if self.occurrences is not None else None,
        }

class v1TrialsSampleResponse:
    def __init__(
        self,
        demotedTrials: "t.Sequence[int]",
        promotedTrials: "t.Sequence[int]",
        trials: "t.Sequence[v1TrialsSampleResponseTrial]",
    ):
        self.trials = trials
        self.promotedTrials = promotedTrials
        self.demotedTrials = demotedTrials

    @classmethod
    def from_json(cls, obj: Json) -> "v1TrialsSampleResponse":
        return cls(
            trials=[v1TrialsSampleResponseTrial.from_json(x) for x in obj["trials"]],
            promotedTrials=obj["promotedTrials"],
            demotedTrials=obj["demotedTrials"],
        )

    def to_json(self) -> t.Any:
        return {
            "trials": [x.to_json() for x in self.trials],
            "promotedTrials": self.promotedTrials,
            "demotedTrials": self.demotedTrials,
        }

class v1TrialsSampleResponseTrial:
    def __init__(
        self,
        data: "t.Sequence[TrialsSampleResponseDataPoint]",
        hparams: "t.Dict[str, t.Any]",
        trialId: int,
    ):
        self.trialId = trialId
        self.hparams = hparams
        self.data = data

    @classmethod
    def from_json(cls, obj: Json) -> "v1TrialsSampleResponseTrial":
        return cls(
            trialId=obj["trialId"],
            hparams=obj["hparams"],
            data=[TrialsSampleResponseDataPoint.from_json(x) for x in obj["data"]],
        )

    def to_json(self) -> t.Any:
        return {
            "trialId": self.trialId,
            "hparams": self.hparams,
            "data": [x.to_json() for x in self.data],
        }

class v1TrialsSnapshotResponse:
    def __init__(
        self,
        trials: "t.Sequence[v1TrialsSnapshotResponseTrial]",
    ):
        self.trials = trials

    @classmethod
    def from_json(cls, obj: Json) -> "v1TrialsSnapshotResponse":
        return cls(
            trials=[v1TrialsSnapshotResponseTrial.from_json(x) for x in obj["trials"]],
        )

    def to_json(self) -> t.Any:
        return {
            "trials": [x.to_json() for x in self.trials],
        }

class v1TrialsSnapshotResponseTrial:
    def __init__(
        self,
        batchesProcessed: int,
        hparams: "t.Dict[str, t.Any]",
        metric: float,
        trialId: int,
    ):
        self.trialId = trialId
        self.hparams = hparams
        self.metric = metric
        self.batchesProcessed = batchesProcessed

    @classmethod
    def from_json(cls, obj: Json) -> "v1TrialsSnapshotResponseTrial":
        return cls(
            trialId=obj["trialId"],
            hparams=obj["hparams"],
            metric=float(obj["metric"]),
            batchesProcessed=obj["batchesProcessed"],
        )

    def to_json(self) -> t.Any:
        return {
            "trialId": self.trialId,
            "hparams": self.hparams,
            "metric": dump_float(self.metric),
            "batchesProcessed": self.batchesProcessed,
        }

class v1UpdateJobQueueRequest:
    def __init__(
        self,
        updates: "t.Sequence[v1QueueControl]",
    ):
        self.updates = updates

    @classmethod
    def from_json(cls, obj: Json) -> "v1UpdateJobQueueRequest":
        return cls(
            updates=[v1QueueControl.from_json(x) for x in obj["updates"]],
        )

    def to_json(self) -> t.Any:
        return {
            "updates": [x.to_json() for x in self.updates],
        }

class v1User:
    def __init__(
        self,
        active: bool,
        admin: bool,
        id: int,
        username: str,
        agentUserGroup: "t.Optional[v1AgentUserGroup]" = None,
    ):
        self.id = id
        self.username = username
        self.admin = admin
        self.active = active
        self.agentUserGroup = agentUserGroup

    @classmethod
    def from_json(cls, obj: Json) -> "v1User":
        return cls(
            id=obj["id"],
            username=obj["username"],
            admin=obj["admin"],
            active=obj["active"],
            agentUserGroup=v1AgentUserGroup.from_json(obj["agentUserGroup"]) if obj.get("agentUserGroup", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "id": self.id,
            "username": self.username,
            "admin": self.admin,
            "active": self.active,
            "agentUserGroup": self.agentUserGroup.to_json() if self.agentUserGroup is not None else None,
        }

class v1ValidateAfterOperation:
    def __init__(
        self,
        length: "t.Optional[v1TrainingLength]" = None,
    ):
        self.length = length

    @classmethod
    def from_json(cls, obj: Json) -> "v1ValidateAfterOperation":
        return cls(
            length=v1TrainingLength.from_json(obj["length"]) if obj.get("length", None) is not None else None,
        )

    def to_json(self) -> t.Any:
        return {
            "length": self.length.to_json() if self.length is not None else None,
        }

class v1ValidationHistoryEntry:
    def __init__(
        self,
        endTime: str,
        searcherMetric: float,
        trialId: int,
    ):
        self.trialId = trialId
        self.endTime = endTime
        self.searcherMetric = searcherMetric

    @classmethod
    def from_json(cls, obj: Json) -> "v1ValidationHistoryEntry":
        return cls(
            trialId=obj["trialId"],
            endTime=obj["endTime"],
            searcherMetric=float(obj["searcherMetric"]),
        )

    def to_json(self) -> t.Any:
        return {
            "trialId": self.trialId,
            "endTime": self.endTime,
            "searcherMetric": dump_float(self.searcherMetric),
        }

def post_AckAllocationPreemptionSignal(
    do_request: Request,
    *,
    allocationId: str,
    body: "v1AckAllocationPreemptionSignalRequest",
) -> None:
    _params = None
    _req = do_request(
        "POST",
        f"/api/v1/allocations/{allocationId}/signals/ack_preemption",
        _params,
        body.to_json(),
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return
    raise ValueError(_req.status_code)

def post_ActivateExperiment(
    do_request: Request,
    *,
    id: int,
) -> None:
    _params = None
    _req = do_request(
        "POST",
        f"/api/v1/experiments/{id}/activate",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return
    raise ValueError(_req.status_code)

def get_AllocationPreemptionSignal(
    do_request: Request,
    *,
    allocationId: str,
    timeoutSeconds: "t.Optional[int]" = None,
) -> "v1AllocationPreemptionSignalResponse":
    _params = {
        "timeoutSeconds": timeoutSeconds,
    }
    _req = do_request(
        "GET",
        f"/api/v1/allocations/{allocationId}/signals/preemption",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1AllocationPreemptionSignalResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def get_AllocationRendezvousInfo(
    do_request: Request,
    *,
    allocationId: str,
    containerId: str,
) -> "v1AllocationRendezvousInfoResponse":
    _params = None
    _req = do_request(
        "GET",
        f"/api/v1/allocations/{allocationId}/rendezvous_info/{containerId}",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1AllocationRendezvousInfoResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def post_ArchiveExperiment(
    do_request: Request,
    *,
    id: int,
) -> None:
    _params = None
    _req = do_request(
        "POST",
        f"/api/v1/experiments/{id}/archive",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return
    raise ValueError(_req.status_code)

def post_ArchiveModel(
    do_request: Request,
    *,
    modelId: int,
) -> None:
    _params = None
    _req = do_request(
        "POST",
        f"/api/v1/models/{modelId}/archive",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return
    raise ValueError(_req.status_code)

def post_CancelExperiment(
    do_request: Request,
    *,
    id: int,
) -> None:
    _params = None
    _req = do_request(
        "POST",
        f"/api/v1/experiments/{id}/cancel",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return
    raise ValueError(_req.status_code)

def post_CompleteTrialSearcherValidation(
    do_request: Request,
    *,
    body: "v1CompleteValidateAfterOperation",
    trialId: int,
) -> None:
    _params = None
    _req = do_request(
        "POST",
        f"/api/v1/trials/{trialId}/searcher/completed_operation",
        _params,
        body.to_json(),
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return
    raise ValueError(_req.status_code)

def post_ComputeHPImportance(
    do_request: Request,
    *,
    experimentId: int,
) -> None:
    _params = None
    _req = do_request(
        "POST",
        f"/api/v1/experiments/{experimentId}/hyperparameter-importance",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return
    raise ValueError(_req.status_code)

def post_CreateExperiment(
    do_request: Request,
    *,
    body: "v1CreateExperimentRequest",
) -> "v1CreateExperimentResponse":
    _params = None
    _req = do_request(
        "POST",
        "/api/v1/experiments",
        _params,
        body.to_json(),
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1CreateExperimentResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def get_CurrentUser(
    do_request: Request,
) -> "v1CurrentUserResponse":
    _params = None
    _req = do_request(
        "GET",
        "/api/v1/auth/user",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1CurrentUserResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def delete_DeleteExperiment(
    do_request: Request,
    *,
    experimentId: int,
) -> None:
    _params = None
    _req = do_request(
        "DELETE",
        f"/api/v1/experiments/{experimentId}",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return
    raise ValueError(_req.status_code)

def delete_DeleteModel(
    do_request: Request,
    *,
    modelId: int,
) -> None:
    _params = None
    _req = do_request(
        "DELETE",
        f"/api/v1/models/{modelId}",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return
    raise ValueError(_req.status_code)

def delete_DeleteModelVersion(
    do_request: Request,
    *,
    modelId: int,
    modelVersionId: int,
) -> None:
    _params = None
    _req = do_request(
        "DELETE",
        f"/api/v1/models/{modelId}/versions/{modelVersionId}",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return
    raise ValueError(_req.status_code)

def delete_DeleteTemplate(
    do_request: Request,
    *,
    templateName: str,
) -> None:
    _params = None
    _req = do_request(
        "DELETE",
        f"/api/v1/templates/{templateName}",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return
    raise ValueError(_req.status_code)

def post_DisableAgent(
    do_request: Request,
    *,
    agentId: str,
    body: "v1DisableAgentRequest",
) -> "v1DisableAgentResponse":
    _params = None
    _req = do_request(
        "POST",
        f"/api/v1/agents/{agentId}/disable",
        _params,
        body.to_json(),
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1DisableAgentResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def post_DisableSlot(
    do_request: Request,
    *,
    agentId: str,
    slotId: str,
) -> "v1DisableSlotResponse":
    _params = None
    _req = do_request(
        "POST",
        f"/api/v1/agents/{agentId}/slots/{slotId}/disable",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1DisableSlotResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def post_EnableAgent(
    do_request: Request,
    *,
    agentId: str,
) -> "v1EnableAgentResponse":
    _params = None
    _req = do_request(
        "POST",
        f"/api/v1/agents/{agentId}/enable",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1EnableAgentResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def post_EnableSlot(
    do_request: Request,
    *,
    agentId: str,
    slotId: str,
) -> "v1EnableSlotResponse":
    _params = None
    _req = do_request(
        "POST",
        f"/api/v1/agents/{agentId}/slots/{slotId}/enable",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1EnableSlotResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def get_GetAgent(
    do_request: Request,
    *,
    agentId: str,
) -> "v1GetAgentResponse":
    _params = None
    _req = do_request(
        "GET",
        f"/api/v1/agents/{agentId}",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1GetAgentResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def get_GetAgents(
    do_request: Request,
    *,
    label: "t.Optional[str]" = None,
    limit: "t.Optional[int]" = None,
    offset: "t.Optional[int]" = None,
    orderBy: "t.Optional[str]" = None,
    sortBy: "t.Optional[str]" = None,
) -> "v1GetAgentsResponse":
    _params = {
        "label": label,
        "limit": limit,
        "offset": offset,
        "orderBy": orderBy,
        "sortBy": sortBy,
    }
    _req = do_request(
        "GET",
        "/api/v1/agents",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1GetAgentsResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def get_GetBestSearcherValidationMetric(
    do_request: Request,
    *,
    experimentId: int,
) -> "v1GetBestSearcherValidationMetricResponse":
    _params = None
    _req = do_request(
        "GET",
        f"/api/v1/experiments/{experimentId}/searcher/best_searcher_validation_metric",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1GetBestSearcherValidationMetricResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def get_GetCheckpoint(
    do_request: Request,
    *,
    checkpointUuid: str,
) -> "v1GetCheckpointResponse":
    _params = None
    _req = do_request(
        "GET",
        f"/api/v1/checkpoints/{checkpointUuid}",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1GetCheckpointResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def get_GetCommand(
    do_request: Request,
    *,
    commandId: str,
) -> "v1GetCommandResponse":
    _params = None
    _req = do_request(
        "GET",
        f"/api/v1/commands/{commandId}",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1GetCommandResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def get_GetCommands(
    do_request: Request,
    *,
    limit: "t.Optional[int]" = None,
    offset: "t.Optional[int]" = None,
    orderBy: "t.Optional[str]" = None,
    sortBy: "t.Optional[str]" = None,
    users: "t.Optional[t.Sequence[str]]" = None,
) -> "v1GetCommandsResponse":
    _params = {
        "limit": limit,
        "offset": offset,
        "orderBy": orderBy,
        "sortBy": sortBy,
        "users": users,
    }
    _req = do_request(
        "GET",
        "/api/v1/commands",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1GetCommandsResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def get_GetCurrentTrialSearcherOperation(
    do_request: Request,
    *,
    trialId: int,
) -> "v1GetCurrentTrialSearcherOperationResponse":
    _params = None
    _req = do_request(
        "GET",
        f"/api/v1/trials/{trialId}/searcher/operation",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1GetCurrentTrialSearcherOperationResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def get_GetExperiment(
    do_request: Request,
    *,
    experimentId: int,
) -> "v1GetExperimentResponse":
    _params = None
    _req = do_request(
        "GET",
        f"/api/v1/experiments/{experimentId}",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1GetExperimentResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def get_GetExperimentCheckpoints(
    do_request: Request,
    *,
    id: int,
    limit: "t.Optional[int]" = None,
    offset: "t.Optional[int]" = None,
    orderBy: "t.Optional[str]" = None,
    sortBy: "t.Optional[str]" = None,
    states: "t.Optional[t.Sequence[str]]" = None,
    validationStates: "t.Optional[t.Sequence[str]]" = None,
) -> "v1GetExperimentCheckpointsResponse":
    _params = {
        "limit": limit,
        "offset": offset,
        "orderBy": orderBy,
        "sortBy": sortBy,
        "states": states,
        "validationStates": validationStates,
    }
    _req = do_request(
        "GET",
        f"/api/v1/experiments/{id}/checkpoints",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1GetExperimentCheckpointsResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def get_GetExperimentLabels(
    do_request: Request,
) -> "v1GetExperimentLabelsResponse":
    _params = None
    _req = do_request(
        "GET",
        "/api/v1/experiment/labels",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1GetExperimentLabelsResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def get_GetExperimentTrials(
    do_request: Request,
    *,
    experimentId: int,
    limit: "t.Optional[int]" = None,
    offset: "t.Optional[int]" = None,
    orderBy: "t.Optional[str]" = None,
    sortBy: "t.Optional[str]" = None,
    states: "t.Optional[t.Sequence[str]]" = None,
) -> "v1GetExperimentTrialsResponse":
    _params = {
        "limit": limit,
        "offset": offset,
        "orderBy": orderBy,
        "sortBy": sortBy,
        "states": states,
    }
    _req = do_request(
        "GET",
        f"/api/v1/experiments/{experimentId}/trials",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1GetExperimentTrialsResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def get_GetExperimentValidationHistory(
    do_request: Request,
    *,
    experimentId: int,
) -> "v1GetExperimentValidationHistoryResponse":
    _params = None
    _req = do_request(
        "GET",
        f"/api/v1/experiments/{experimentId}/validation-history",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1GetExperimentValidationHistoryResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def get_GetExperiments(
    do_request: Request,
    *,
    archived: "t.Optional[bool]" = None,
    description: "t.Optional[str]" = None,
    labels: "t.Optional[t.Sequence[str]]" = None,
    limit: "t.Optional[int]" = None,
    name: "t.Optional[str]" = None,
    offset: "t.Optional[int]" = None,
    orderBy: "t.Optional[str]" = None,
    sortBy: "t.Optional[str]" = None,
    states: "t.Optional[t.Sequence[str]]" = None,
    users: "t.Optional[t.Sequence[str]]" = None,
) -> "v1GetExperimentsResponse":
    _params = {
        "archived": archived,
        "description": description,
        "labels": labels,
        "limit": limit,
        "name": name,
        "offset": offset,
        "orderBy": orderBy,
        "sortBy": sortBy,
        "states": states,
        "users": users,
    }
    _req = do_request(
        "GET",
        "/api/v1/experiments",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1GetExperimentsResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def get_GetJobQueueStats(
    do_request: Request,
    *,
    resourcePools: "t.Optional[t.Sequence[str]]" = None,
) -> "v1GetJobQueueStatsResponse":
    _params = {
        "resourcePools": resourcePools,
    }
    _req = do_request(
        "GET",
        "/api/v1/job-queues/stats",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1GetJobQueueStatsResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def get_GetJobs(
    do_request: Request,
    *,
    orderBy: "t.Optional[str]" = None,
    pagination_limit: "t.Optional[int]" = None,
    pagination_offset: "t.Optional[int]" = None,
    resourcePool: "t.Optional[str]" = None,
) -> "v1GetJobsResponse":
    _params = {
        "orderBy": orderBy,
        "pagination.limit": pagination_limit,
        "pagination.offset": pagination_offset,
        "resourcePool": resourcePool,
    }
    _req = do_request(
        "GET",
        "/api/v1/job-queues",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1GetJobsResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def get_GetMaster(
    do_request: Request,
) -> "v1GetMasterResponse":
    _params = None
    _req = do_request(
        "GET",
        "/api/v1/master",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1GetMasterResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def get_GetMasterConfig(
    do_request: Request,
) -> "v1GetMasterConfigResponse":
    _params = None
    _req = do_request(
        "GET",
        "/api/v1/master/config",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1GetMasterConfigResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def get_GetModel(
    do_request: Request,
    *,
    modelId: int,
) -> "v1GetModelResponse":
    _params = None
    _req = do_request(
        "GET",
        f"/api/v1/models/{modelId}",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1GetModelResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def get_GetModelDef(
    do_request: Request,
    *,
    experimentId: int,
) -> "v1GetModelDefResponse":
    _params = None
    _req = do_request(
        "GET",
        f"/api/v1/experiments/{experimentId}/model_def",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1GetModelDefResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def get_GetModelLabels(
    do_request: Request,
) -> "v1GetModelLabelsResponse":
    _params = None
    _req = do_request(
        "GET",
        "/api/v1/model/labels",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1GetModelLabelsResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def get_GetModelVersion(
    do_request: Request,
    *,
    modelId: int,
    modelVersion: int,
) -> "v1GetModelVersionResponse":
    _params = None
    _req = do_request(
        "GET",
        f"/api/v1/models/{modelId}/versions/{modelVersion}",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1GetModelVersionResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def get_GetModelVersions(
    do_request: Request,
    *,
    modelId: int,
    limit: "t.Optional[int]" = None,
    offset: "t.Optional[int]" = None,
    orderBy: "t.Optional[str]" = None,
    sortBy: "t.Optional[str]" = None,
) -> "v1GetModelVersionsResponse":
    _params = {
        "limit": limit,
        "offset": offset,
        "orderBy": orderBy,
        "sortBy": sortBy,
    }
    _req = do_request(
        "GET",
        f"/api/v1/models/{modelId}/versions",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1GetModelVersionsResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def get_GetModels(
    do_request: Request,
    *,
    archived: "t.Optional[bool]" = None,
    description: "t.Optional[str]" = None,
    labels: "t.Optional[t.Sequence[str]]" = None,
    limit: "t.Optional[int]" = None,
    name: "t.Optional[str]" = None,
    offset: "t.Optional[int]" = None,
    orderBy: "t.Optional[str]" = None,
    sortBy: "t.Optional[str]" = None,
    users: "t.Optional[t.Sequence[str]]" = None,
) -> "v1GetModelsResponse":
    _params = {
        "archived": archived,
        "description": description,
        "labels": labels,
        "limit": limit,
        "name": name,
        "offset": offset,
        "orderBy": orderBy,
        "sortBy": sortBy,
        "users": users,
    }
    _req = do_request(
        "GET",
        "/api/v1/models",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1GetModelsResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def get_GetNotebook(
    do_request: Request,
    *,
    notebookId: str,
) -> "v1GetNotebookResponse":
    _params = None
    _req = do_request(
        "GET",
        f"/api/v1/notebooks/{notebookId}",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1GetNotebookResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def get_GetNotebooks(
    do_request: Request,
    *,
    limit: "t.Optional[int]" = None,
    offset: "t.Optional[int]" = None,
    orderBy: "t.Optional[str]" = None,
    sortBy: "t.Optional[str]" = None,
    users: "t.Optional[t.Sequence[str]]" = None,
) -> "v1GetNotebooksResponse":
    _params = {
        "limit": limit,
        "offset": offset,
        "orderBy": orderBy,
        "sortBy": sortBy,
        "users": users,
    }
    _req = do_request(
        "GET",
        "/api/v1/notebooks",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1GetNotebooksResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def get_GetResourcePools(
    do_request: Request,
    *,
    limit: "t.Optional[int]" = None,
    offset: "t.Optional[int]" = None,
) -> "v1GetResourcePoolsResponse":
    _params = {
        "limit": limit,
        "offset": offset,
    }
    _req = do_request(
        "GET",
        "/api/v1/resource-pools",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1GetResourcePoolsResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def get_GetShell(
    do_request: Request,
    *,
    shellId: str,
) -> "v1GetShellResponse":
    _params = None
    _req = do_request(
        "GET",
        f"/api/v1/shells/{shellId}",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1GetShellResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def get_GetShells(
    do_request: Request,
    *,
    limit: "t.Optional[int]" = None,
    offset: "t.Optional[int]" = None,
    orderBy: "t.Optional[str]" = None,
    sortBy: "t.Optional[str]" = None,
    users: "t.Optional[t.Sequence[str]]" = None,
) -> "v1GetShellsResponse":
    _params = {
        "limit": limit,
        "offset": offset,
        "orderBy": orderBy,
        "sortBy": sortBy,
        "users": users,
    }
    _req = do_request(
        "GET",
        "/api/v1/shells",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1GetShellsResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def get_GetSlot(
    do_request: Request,
    *,
    agentId: str,
    slotId: str,
) -> "v1GetSlotResponse":
    _params = None
    _req = do_request(
        "GET",
        f"/api/v1/agents/{agentId}/slots/{slotId}",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1GetSlotResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def get_GetSlots(
    do_request: Request,
    *,
    agentId: str,
) -> "v1GetSlotsResponse":
    _params = None
    _req = do_request(
        "GET",
        f"/api/v1/agents/{agentId}/slots",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1GetSlotsResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def get_GetTelemetry(
    do_request: Request,
) -> "v1GetTelemetryResponse":
    _params = None
    _req = do_request(
        "GET",
        "/api/v1/master/telemetry",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1GetTelemetryResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def get_GetTemplate(
    do_request: Request,
    *,
    templateName: str,
) -> "v1GetTemplateResponse":
    _params = None
    _req = do_request(
        "GET",
        f"/api/v1/templates/{templateName}",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1GetTemplateResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def get_GetTemplates(
    do_request: Request,
    *,
    limit: "t.Optional[int]" = None,
    name: "t.Optional[str]" = None,
    offset: "t.Optional[int]" = None,
    orderBy: "t.Optional[str]" = None,
    sortBy: "t.Optional[str]" = None,
) -> "v1GetTemplatesResponse":
    _params = {
        "limit": limit,
        "name": name,
        "offset": offset,
        "orderBy": orderBy,
        "sortBy": sortBy,
    }
    _req = do_request(
        "GET",
        "/api/v1/templates",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1GetTemplatesResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def get_GetTensorboard(
    do_request: Request,
    *,
    tensorboardId: str,
) -> "v1GetTensorboardResponse":
    _params = None
    _req = do_request(
        "GET",
        f"/api/v1/tensorboards/{tensorboardId}",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1GetTensorboardResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def get_GetTensorboards(
    do_request: Request,
    *,
    limit: "t.Optional[int]" = None,
    offset: "t.Optional[int]" = None,
    orderBy: "t.Optional[str]" = None,
    sortBy: "t.Optional[str]" = None,
    users: "t.Optional[t.Sequence[str]]" = None,
) -> "v1GetTensorboardsResponse":
    _params = {
        "limit": limit,
        "offset": offset,
        "orderBy": orderBy,
        "sortBy": sortBy,
        "users": users,
    }
    _req = do_request(
        "GET",
        "/api/v1/tensorboards",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1GetTensorboardsResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def get_GetTrial(
    do_request: Request,
    *,
    trialId: int,
) -> "v1GetTrialResponse":
    _params = None
    _req = do_request(
        "GET",
        f"/api/v1/trials/{trialId}",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1GetTrialResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def get_GetTrialCheckpoints(
    do_request: Request,
    *,
    id: int,
    limit: "t.Optional[int]" = None,
    offset: "t.Optional[int]" = None,
    orderBy: "t.Optional[str]" = None,
    sortBy: "t.Optional[str]" = None,
    states: "t.Optional[t.Sequence[str]]" = None,
    validationStates: "t.Optional[t.Sequence[str]]" = None,
) -> "v1GetTrialCheckpointsResponse":
    _params = {
        "limit": limit,
        "offset": offset,
        "orderBy": orderBy,
        "sortBy": sortBy,
        "states": states,
        "validationStates": validationStates,
    }
    _req = do_request(
        "GET",
        f"/api/v1/trials/{id}/checkpoints",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1GetTrialCheckpointsResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def get_GetUser(
    do_request: Request,
    *,
    username: str,
) -> "v1GetUserResponse":
    _params = None
    _req = do_request(
        "GET",
        f"/api/v1/users/{username}",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1GetUserResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def get_GetUsers(
    do_request: Request,
) -> "v1GetUsersResponse":
    _params = None
    _req = do_request(
        "GET",
        "/api/v1/users",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1GetUsersResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def put_IdleNotebook(
    do_request: Request,
    *,
    body: "v1IdleNotebookRequest",
    notebookId: str,
) -> None:
    _params = None
    _req = do_request(
        "PUT",
        f"/api/v1/notebooks/{notebookId}/report_idle",
        _params,
        body.to_json(),
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return
    raise ValueError(_req.status_code)

def post_KillCommand(
    do_request: Request,
    *,
    commandId: str,
) -> "v1KillCommandResponse":
    _params = None
    _req = do_request(
        "POST",
        f"/api/v1/commands/{commandId}/kill",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1KillCommandResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def post_KillExperiment(
    do_request: Request,
    *,
    id: int,
) -> None:
    _params = None
    _req = do_request(
        "POST",
        f"/api/v1/experiments/{id}/kill",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return
    raise ValueError(_req.status_code)

def post_KillNotebook(
    do_request: Request,
    *,
    notebookId: str,
) -> "v1KillNotebookResponse":
    _params = None
    _req = do_request(
        "POST",
        f"/api/v1/notebooks/{notebookId}/kill",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1KillNotebookResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def post_KillShell(
    do_request: Request,
    *,
    shellId: str,
) -> "v1KillShellResponse":
    _params = None
    _req = do_request(
        "POST",
        f"/api/v1/shells/{shellId}/kill",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1KillShellResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def post_KillTensorboard(
    do_request: Request,
    *,
    tensorboardId: str,
) -> "v1KillTensorboardResponse":
    _params = None
    _req = do_request(
        "POST",
        f"/api/v1/tensorboards/{tensorboardId}/kill",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1KillTensorboardResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def post_KillTrial(
    do_request: Request,
    *,
    id: int,
) -> None:
    _params = None
    _req = do_request(
        "POST",
        f"/api/v1/trials/{id}/kill",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return
    raise ValueError(_req.status_code)

def post_LaunchCommand(
    do_request: Request,
    *,
    body: "v1LaunchCommandRequest",
) -> "v1LaunchCommandResponse":
    _params = None
    _req = do_request(
        "POST",
        "/api/v1/commands",
        _params,
        body.to_json(),
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1LaunchCommandResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def post_LaunchNotebook(
    do_request: Request,
    *,
    body: "v1LaunchNotebookRequest",
) -> "v1LaunchNotebookResponse":
    _params = None
    _req = do_request(
        "POST",
        "/api/v1/notebooks",
        _params,
        body.to_json(),
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1LaunchNotebookResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def post_LaunchShell(
    do_request: Request,
    *,
    body: "v1LaunchShellRequest",
) -> "v1LaunchShellResponse":
    _params = None
    _req = do_request(
        "POST",
        "/api/v1/shells",
        _params,
        body.to_json(),
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1LaunchShellResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def post_LaunchTensorboard(
    do_request: Request,
    *,
    body: "v1LaunchTensorboardRequest",
) -> "v1LaunchTensorboardResponse":
    _params = None
    _req = do_request(
        "POST",
        "/api/v1/tensorboards",
        _params,
        body.to_json(),
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1LaunchTensorboardResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def post_Login(
    do_request: Request,
    *,
    body: "v1LoginRequest",
) -> "v1LoginResponse":
    _params = None
    _req = do_request(
        "POST",
        "/api/v1/auth/login",
        _params,
        body.to_json(),
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1LoginResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def post_Logout(
    do_request: Request,
) -> None:
    _params = None
    _req = do_request(
        "POST",
        "/api/v1/auth/logout",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return
    raise ValueError(_req.status_code)

def post_MarkAllocationReservationDaemon(
    do_request: Request,
    *,
    allocationId: str,
    body: "v1MarkAllocationReservationDaemonRequest",
    containerId: str,
) -> None:
    _params = None
    _req = do_request(
        "POST",
        f"/api/v1/allocations/{allocationId}/containers/{containerId}/daemon",
        _params,
        body.to_json(),
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return
    raise ValueError(_req.status_code)

def patch_PatchExperiment(
    do_request: Request,
    *,
    body: "v1Experiment",
    experiment_id: int,
) -> "v1PatchExperimentResponse":
    _params = None
    _req = do_request(
        "PATCH",
        f"/api/v1/experiments/{experiment_id}",
        _params,
        body.to_json(),
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1PatchExperimentResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def patch_PatchModel(
    do_request: Request,
    *,
    body: "v1PatchModelRequest",
    modelId: int,
) -> "v1PatchModelResponse":
    _params = None
    _req = do_request(
        "PATCH",
        f"/api/v1/models/{modelId}",
        _params,
        body.to_json(),
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1PatchModelResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def patch_PatchModelVersion(
    do_request: Request,
    *,
    body: "v1PatchModelVersionRequest",
    modelId: int,
    modelVersionId: int,
) -> "v1PatchModelVersionResponse":
    _params = None
    _req = do_request(
        "PATCH",
        f"/api/v1/models/{modelId}/versions/{modelVersionId}",
        _params,
        body.to_json(),
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1PatchModelVersionResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def post_PauseExperiment(
    do_request: Request,
    *,
    id: int,
) -> None:
    _params = None
    _req = do_request(
        "POST",
        f"/api/v1/experiments/{id}/pause",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return
    raise ValueError(_req.status_code)

def post_PostCheckpointMetadata(
    do_request: Request,
    *,
    body: "v1PostCheckpointMetadataRequest",
    checkpoint_uuid: str,
) -> "v1PostCheckpointMetadataResponse":
    _params = None
    _req = do_request(
        "POST",
        f"/api/v1/checkpoints/{checkpoint_uuid}/metadata",
        _params,
        body.to_json(),
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1PostCheckpointMetadataResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def post_PostModel(
    do_request: Request,
    *,
    body: "v1PostModelRequest",
) -> "v1PostModelResponse":
    _params = None
    _req = do_request(
        "POST",
        "/api/v1/models",
        _params,
        body.to_json(),
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1PostModelResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def post_PostModelVersion(
    do_request: Request,
    *,
    body: "v1PostModelVersionRequest",
    modelId: int,
) -> "v1PostModelVersionResponse":
    _params = None
    _req = do_request(
        "POST",
        f"/api/v1/models/{modelId}/versions",
        _params,
        body.to_json(),
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1PostModelVersionResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def post_PostTrialProfilerMetricsBatch(
    do_request: Request,
    *,
    body: "v1PostTrialProfilerMetricsBatchRequest",
) -> None:
    _params = None
    _req = do_request(
        "POST",
        "/api/v1/trials/profiler/metrics",
        _params,
        body.to_json(),
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return
    raise ValueError(_req.status_code)

def post_PostTrialRunnerMetadata(
    do_request: Request,
    *,
    body: "v1TrialRunnerMetadata",
    trialId: int,
) -> None:
    _params = None
    _req = do_request(
        "POST",
        f"/api/v1/trials/{trialId}/runner/metadata",
        _params,
        body.to_json(),
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return
    raise ValueError(_req.status_code)

def post_PostUser(
    do_request: Request,
    *,
    body: "v1PostUserRequest",
) -> "v1PostUserResponse":
    _params = None
    _req = do_request(
        "POST",
        "/api/v1/users",
        _params,
        body.to_json(),
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1PostUserResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def post_PreviewHPSearch(
    do_request: Request,
    *,
    body: "v1PreviewHPSearchRequest",
) -> "v1PreviewHPSearchResponse":
    _params = None
    _req = do_request(
        "POST",
        "/api/v1/preview-hp-search",
        _params,
        body.to_json(),
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1PreviewHPSearchResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def put_PutTemplate(
    do_request: Request,
    *,
    body: "v1Template",
    template_name: str,
) -> "v1PutTemplateResponse":
    _params = None
    _req = do_request(
        "PUT",
        f"/api/v1/templates/{template_name}",
        _params,
        body.to_json(),
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1PutTemplateResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def post_ReportTrialCheckpointMetadata(
    do_request: Request,
    *,
    body: "v1CheckpointMetadata",
    checkpointMetadata_trialId: int,
) -> None:
    _params = None
    _req = do_request(
        "POST",
        f"/api/v1/trials/{checkpointMetadata_trialId}/checkpoint_metadata",
        _params,
        body.to_json(),
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return
    raise ValueError(_req.status_code)

def post_ReportTrialProgress(
    do_request: Request,
    *,
    body: float,
    trialId: int,
) -> None:
    _params = None
    _req = do_request(
        "POST",
        f"/api/v1/trials/{trialId}/progress",
        _params,
        dump_float(body),
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return
    raise ValueError(_req.status_code)

def post_ReportTrialSearcherEarlyExit(
    do_request: Request,
    *,
    body: "v1TrialEarlyExit",
    trialId: int,
) -> None:
    _params = None
    _req = do_request(
        "POST",
        f"/api/v1/trials/{trialId}/early_exit",
        _params,
        body.to_json(),
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return
    raise ValueError(_req.status_code)

def post_ReportTrialTrainingMetrics(
    do_request: Request,
    *,
    body: "v1TrialMetrics",
    trainingMetrics_trialId: int,
) -> None:
    _params = None
    _req = do_request(
        "POST",
        f"/api/v1/trials/{trainingMetrics_trialId}/training_metrics",
        _params,
        body.to_json(),
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return
    raise ValueError(_req.status_code)

def post_ReportTrialValidationMetrics(
    do_request: Request,
    *,
    body: "v1TrialMetrics",
    validationMetrics_trialId: int,
) -> None:
    _params = None
    _req = do_request(
        "POST",
        f"/api/v1/trials/{validationMetrics_trialId}/validation_metrics",
        _params,
        body.to_json(),
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return
    raise ValueError(_req.status_code)

def get_ResourceAllocationAggregated(
    do_request: Request,
    *,
    endDate: "t.Optional[str]" = None,
    period: "t.Optional[str]" = None,
    startDate: "t.Optional[str]" = None,
) -> "v1ResourceAllocationAggregatedResponse":
    _params = {
        "endDate": endDate,
        "period": period,
        "startDate": startDate,
    }
    _req = do_request(
        "GET",
        "/api/v1/resources/allocation/aggregated",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1ResourceAllocationAggregatedResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def get_ResourceAllocationRaw(
    do_request: Request,
    *,
    timestampAfter: "t.Optional[str]" = None,
    timestampBefore: "t.Optional[str]" = None,
) -> "v1ResourceAllocationRawResponse":
    _params = {
        "timestampAfter": timestampAfter,
        "timestampBefore": timestampBefore,
    }
    _req = do_request(
        "GET",
        "/api/v1/resources/allocation/raw",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1ResourceAllocationRawResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def post_SetCommandPriority(
    do_request: Request,
    *,
    body: "v1SetCommandPriorityRequest",
    commandId: str,
) -> "v1SetCommandPriorityResponse":
    _params = None
    _req = do_request(
        "POST",
        f"/api/v1/commands/{commandId}/set_priority",
        _params,
        body.to_json(),
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1SetCommandPriorityResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def post_SetNotebookPriority(
    do_request: Request,
    *,
    body: "v1SetNotebookPriorityRequest",
    notebookId: str,
) -> "v1SetNotebookPriorityResponse":
    _params = None
    _req = do_request(
        "POST",
        f"/api/v1/notebooks/{notebookId}/set_priority",
        _params,
        body.to_json(),
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1SetNotebookPriorityResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def post_SetShellPriority(
    do_request: Request,
    *,
    body: "v1SetShellPriorityRequest",
    shellId: str,
) -> "v1SetShellPriorityResponse":
    _params = None
    _req = do_request(
        "POST",
        f"/api/v1/shells/{shellId}/set_priority",
        _params,
        body.to_json(),
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1SetShellPriorityResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def post_SetTensorboardPriority(
    do_request: Request,
    *,
    body: "v1SetTensorboardPriorityRequest",
    tensorboardId: str,
) -> "v1SetTensorboardPriorityResponse":
    _params = None
    _req = do_request(
        "POST",
        f"/api/v1/tensorboards/{tensorboardId}/set_priority",
        _params,
        body.to_json(),
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1SetTensorboardPriorityResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def post_SetUserPassword(
    do_request: Request,
    *,
    body: str,
    username: str,
) -> "v1SetUserPasswordResponse":
    _params = None
    _req = do_request(
        "POST",
        f"/api/v1/users/{username}/password",
        _params,
        body,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return v1SetUserPasswordResponse.from_json(_req.json())
    raise ValueError(_req.status_code)

def post_UnarchiveExperiment(
    do_request: Request,
    *,
    id: int,
) -> None:
    _params = None
    _req = do_request(
        "POST",
        f"/api/v1/experiments/{id}/unarchive",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return
    raise ValueError(_req.status_code)

def post_UnarchiveModel(
    do_request: Request,
    *,
    modelId: int,
) -> None:
    _params = None
    _req = do_request(
        "POST",
        f"/api/v1/models/{modelId}/unarchive",
        _params,
        None,
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return
    raise ValueError(_req.status_code)

def post_UpdateJobQueue(
    do_request: Request,
    *,
    body: "v1UpdateJobQueueRequest",
) -> None:
    _params = None
    _req = do_request(
        "POST",
        "/api/v1/job-queues",
        _params,
        body.to_json(),
        None,
        None,
        None,
    )
    if _req.status_code == 200:
        return
    raise ValueError(_req.status_code)
