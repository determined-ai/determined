# The contents of this file are programatically generated.
import enum
import math
import typing

import requests

if typing.TYPE_CHECKING:
    from determined.experimental import client

# flake8: noqa
Json = typing.Any


Request = typing.Callable[
    [
        str,  # method
        str,  # path
        typing.Optional[typing.Dict[str, typing.Any]],  # params
        typing.Any,  # json body
    ],
    requests.Response,
]


def dump_float(val: typing.Any) -> typing.Any:
    if math.isnan(val):
        return "Nan"
    if math.isinf(val):
        return "Infinity" if val > 0 else "-Infinity"
    return val


class APIHttpError(Exception):
    # APIHttpError is used if an HTTP(s) API request fails.
    def __init__(self, operation_name: str, response: requests.Response) -> None:
        self.response = response
        self.operation_name = operation_name
        self.message = (
            f"API Error: {operation_name} failed."
        )

    def __str__(self) -> str:
        return self.message


class GetHPImportanceResponseMetricHPImportance:
    def __init__(
        self,
        error: "typing.Optional[str]" = None,
        experimentProgress: "typing.Optional[float]" = None,
        hpImportance: "typing.Optional[typing.Dict[str, float]]" = None,
        inProgress: "typing.Optional[bool]" = None,
        pending: "typing.Optional[bool]" = None,
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
            error=obj.get("error", None),
            pending=obj.get("pending", None),
            inProgress=obj.get("inProgress", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "hpImportance": {k: dump_float(v) for k, v in self.hpImportance.items()} if self.hpImportance is not None else None,
            "experimentProgress": dump_float(self.experimentProgress) if self.experimentProgress is not None else None,
            "error": self.error if self.error is not None else None,
            "pending": self.pending if self.pending is not None else None,
            "inProgress": self.inProgress if self.inProgress is not None else None,
        }

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

    def to_json(self) -> typing.Any:
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
    TYPE_CUDA = "TYPE_CUDA"
    TYPE_ROCM = "TYPE_ROCM"

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
        typeUrl: "typing.Optional[str]" = None,
        value: "typing.Optional[str]" = None,
    ):
        self.typeUrl = typeUrl
        self.value = value

    @classmethod
    def from_json(cls, obj: Json) -> "protobufAny":
        return cls(
            typeUrl=obj.get("typeUrl", None),
            value=obj.get("value", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "typeUrl": self.typeUrl if self.typeUrl is not None else None,
            "value": self.value if self.value is not None else None,
        }

class protobufNullValue(enum.Enum):
    NULL_VALUE = "NULL_VALUE"

class runtimeError:
    def __init__(
        self,
        code: "typing.Optional[int]" = None,
        details: "typing.Optional[typing.Sequence[protobufAny]]" = None,
        error: "typing.Optional[str]" = None,
        message: "typing.Optional[str]" = None,
    ):
        self.error = error
        self.code = code
        self.message = message
        self.details = details

    @classmethod
    def from_json(cls, obj: Json) -> "runtimeError":
        return cls(
            error=obj.get("error", None),
            code=obj.get("code", None),
            message=obj.get("message", None),
            details=[protobufAny.from_json(x) for x in obj["details"]] if obj.get("details", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "error": self.error if self.error is not None else None,
            "code": self.code if self.code is not None else None,
            "message": self.message if self.message is not None else None,
            "details": [x.to_json() for x in self.details] if self.details is not None else None,
        }

class runtimeStreamError:
    def __init__(
        self,
        details: "typing.Optional[typing.Sequence[protobufAny]]" = None,
        grpcCode: "typing.Optional[int]" = None,
        httpCode: "typing.Optional[int]" = None,
        httpStatus: "typing.Optional[str]" = None,
        message: "typing.Optional[str]" = None,
    ):
        self.grpcCode = grpcCode
        self.httpCode = httpCode
        self.message = message
        self.httpStatus = httpStatus
        self.details = details

    @classmethod
    def from_json(cls, obj: Json) -> "runtimeStreamError":
        return cls(
            grpcCode=obj.get("grpcCode", None),
            httpCode=obj.get("httpCode", None),
            message=obj.get("message", None),
            httpStatus=obj.get("httpStatus", None),
            details=[protobufAny.from_json(x) for x in obj["details"]] if obj.get("details", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
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
        hparams: "typing.Dict[str, typing.Any]",
        id: int,
        restarts: int,
        startTime: str,
        state: "determinedexperimentv1State",
        totalBatchesProcessed: int,
        bestCheckpoint: "typing.Optional[v1CheckpointWorkload]" = None,
        bestValidation: "typing.Optional[v1MetricsWorkload]" = None,
        endTime: "typing.Optional[str]" = None,
        latestTraining: "typing.Optional[v1MetricsWorkload]" = None,
        latestValidation: "typing.Optional[v1MetricsWorkload]" = None,
        runnerState: "typing.Optional[str]" = None,
        taskId: "typing.Optional[str]" = None,
        totalCheckpointSize: "typing.Optional[str]" = None,
        wallClockTime: "typing.Optional[float]" = None,
        warmStartCheckpointUuid: "typing.Optional[str]" = None,
    ):
        self.id = id
        self.experimentId = experimentId
        self.startTime = startTime
        self.endTime = endTime
        self.state = state
        self.restarts = restarts
        self.hparams = hparams
        self.totalBatchesProcessed = totalBatchesProcessed
        self.bestValidation = bestValidation
        self.latestValidation = latestValidation
        self.bestCheckpoint = bestCheckpoint
        self.latestTraining = latestTraining
        self.runnerState = runnerState
        self.wallClockTime = wallClockTime
        self.warmStartCheckpointUuid = warmStartCheckpointUuid
        self.taskId = taskId
        self.totalCheckpointSize = totalCheckpointSize

    @classmethod
    def from_json(cls, obj: Json) -> "trialv1Trial":
        return cls(
            id=obj["id"],
            experimentId=obj["experimentId"],
            startTime=obj["startTime"],
            endTime=obj.get("endTime", None),
            state=determinedexperimentv1State(obj["state"]),
            restarts=obj["restarts"],
            hparams=obj["hparams"],
            totalBatchesProcessed=obj["totalBatchesProcessed"],
            bestValidation=v1MetricsWorkload.from_json(obj["bestValidation"]) if obj.get("bestValidation", None) is not None else None,
            latestValidation=v1MetricsWorkload.from_json(obj["latestValidation"]) if obj.get("latestValidation", None) is not None else None,
            bestCheckpoint=v1CheckpointWorkload.from_json(obj["bestCheckpoint"]) if obj.get("bestCheckpoint", None) is not None else None,
            latestTraining=v1MetricsWorkload.from_json(obj["latestTraining"]) if obj.get("latestTraining", None) is not None else None,
            runnerState=obj.get("runnerState", None),
            wallClockTime=float(obj["wallClockTime"]) if obj.get("wallClockTime", None) is not None else None,
            warmStartCheckpointUuid=obj.get("warmStartCheckpointUuid", None),
            taskId=obj.get("taskId", None),
            totalCheckpointSize=obj.get("totalCheckpointSize", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "id": self.id,
            "experimentId": self.experimentId,
            "startTime": self.startTime,
            "endTime": self.endTime if self.endTime is not None else None,
            "state": self.state.value,
            "restarts": self.restarts,
            "hparams": self.hparams,
            "totalBatchesProcessed": self.totalBatchesProcessed,
            "bestValidation": self.bestValidation.to_json() if self.bestValidation is not None else None,
            "latestValidation": self.latestValidation.to_json() if self.latestValidation is not None else None,
            "bestCheckpoint": self.bestCheckpoint.to_json() if self.bestCheckpoint is not None else None,
            "latestTraining": self.latestTraining.to_json() if self.latestTraining is not None else None,
            "runnerState": self.runnerState if self.runnerState is not None else None,
            "wallClockTime": dump_float(self.wallClockTime) if self.wallClockTime is not None else None,
            "warmStartCheckpointUuid": self.warmStartCheckpointUuid if self.warmStartCheckpointUuid is not None else None,
            "taskId": self.taskId if self.taskId is not None else None,
            "totalCheckpointSize": self.totalCheckpointSize if self.totalCheckpointSize is not None else None,
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

    def to_json(self) -> typing.Any:
        return {
            "allocationId": self.allocationId,
        }

class v1Agent:
    def __init__(
        self,
        addresses: "typing.Optional[typing.Sequence[str]]" = None,
        containers: "typing.Optional[typing.Dict[str, v1Container]]" = None,
        draining: "typing.Optional[bool]" = None,
        enabled: "typing.Optional[bool]" = None,
        id: "typing.Optional[str]" = None,
        label: "typing.Optional[str]" = None,
        registeredTime: "typing.Optional[str]" = None,
        resourcePool: "typing.Optional[str]" = None,
        slots: "typing.Optional[typing.Dict[str, v1Slot]]" = None,
        version: "typing.Optional[str]" = None,
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
        self.version = version

    @classmethod
    def from_json(cls, obj: Json) -> "v1Agent":
        return cls(
            id=obj.get("id", None),
            registeredTime=obj.get("registeredTime", None),
            slots={k: v1Slot.from_json(v) for k, v in obj["slots"].items()} if obj.get("slots", None) is not None else None,
            containers={k: v1Container.from_json(v) for k, v in obj["containers"].items()} if obj.get("containers", None) is not None else None,
            label=obj.get("label", None),
            resourcePool=obj.get("resourcePool", None),
            addresses=obj.get("addresses", None),
            enabled=obj.get("enabled", None),
            draining=obj.get("draining", None),
            version=obj.get("version", None),
        )

    def to_json(self) -> typing.Any:
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
            "version": self.version if self.version is not None else None,
        }

class v1AgentUserGroup:
    def __init__(
        self,
        agentGid: "typing.Optional[int]" = None,
        agentUid: "typing.Optional[int]" = None,
    ):
        self.agentUid = agentUid
        self.agentGid = agentGid

    @classmethod
    def from_json(cls, obj: Json) -> "v1AgentUserGroup":
        return cls(
            agentUid=obj.get("agentUid", None),
            agentGid=obj.get("agentGid", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "agentUid": self.agentUid if self.agentUid is not None else None,
            "agentGid": self.agentGid if self.agentGid is not None else None,
        }

class v1AggregateQueueStats:
    def __init__(
        self,
        periodStart: str,
        seconds: float,
    ):
        self.periodStart = periodStart
        self.seconds = seconds

    @classmethod
    def from_json(cls, obj: Json) -> "v1AggregateQueueStats":
        return cls(
            periodStart=obj["periodStart"],
            seconds=float(obj["seconds"]),
        )

    def to_json(self) -> typing.Any:
        return {
            "periodStart": self.periodStart,
            "seconds": dump_float(self.seconds),
        }

class v1Allocation:
    def __init__(
        self,
        allocationId: "typing.Optional[str]" = None,
        endTime: "typing.Optional[str]" = None,
        isReady: "typing.Optional[bool]" = None,
        startTime: "typing.Optional[str]" = None,
        state: "typing.Optional[determinedtaskv1State]" = None,
        taskId: "typing.Optional[str]" = None,
    ):
        self.taskId = taskId
        self.state = state
        self.isReady = isReady
        self.startTime = startTime
        self.endTime = endTime
        self.allocationId = allocationId

    @classmethod
    def from_json(cls, obj: Json) -> "v1Allocation":
        return cls(
            taskId=obj.get("taskId", None),
            state=determinedtaskv1State(obj["state"]) if obj.get("state", None) is not None else None,
            isReady=obj.get("isReady", None),
            startTime=obj.get("startTime", None),
            endTime=obj.get("endTime", None),
            allocationId=obj.get("allocationId", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "taskId": self.taskId if self.taskId is not None else None,
            "state": self.state.value if self.state is not None else None,
            "isReady": self.isReady if self.isReady is not None else None,
            "startTime": self.startTime if self.startTime is not None else None,
            "endTime": self.endTime if self.endTime is not None else None,
            "allocationId": self.allocationId if self.allocationId is not None else None,
        }

class v1AllocationAllGatherRequest:
    def __init__(
        self,
        allocationId: str,
        data: "typing.Dict[str, typing.Any]",
        numPeers: "typing.Optional[int]" = None,
        requestUuid: "typing.Optional[str]" = None,
    ):
        self.allocationId = allocationId
        self.requestUuid = requestUuid
        self.numPeers = numPeers
        self.data = data

    @classmethod
    def from_json(cls, obj: Json) -> "v1AllocationAllGatherRequest":
        return cls(
            allocationId=obj["allocationId"],
            requestUuid=obj.get("requestUuid", None),
            numPeers=obj.get("numPeers", None),
            data=obj["data"],
        )

    def to_json(self) -> typing.Any:
        return {
            "allocationId": self.allocationId,
            "requestUuid": self.requestUuid if self.requestUuid is not None else None,
            "numPeers": self.numPeers if self.numPeers is not None else None,
            "data": self.data,
        }

class v1AllocationAllGatherResponse:
    def __init__(
        self,
        data: "typing.Sequence[typing.Dict[str, typing.Any]]",
    ):
        self.data = data

    @classmethod
    def from_json(cls, obj: Json) -> "v1AllocationAllGatherResponse":
        return cls(
            data=obj["data"],
        )

    def to_json(self) -> typing.Any:
        return {
            "data": self.data,
        }

class v1AllocationPendingPreemptionSignalRequest:
    def __init__(
        self,
        allocationId: str,
    ):
        self.allocationId = allocationId

    @classmethod
    def from_json(cls, obj: Json) -> "v1AllocationPendingPreemptionSignalRequest":
        return cls(
            allocationId=obj["allocationId"],
        )

    def to_json(self) -> typing.Any:
        return {
            "allocationId": self.allocationId,
        }

class v1AllocationPreemptionSignalResponse:
    def __init__(
        self,
        preempt: "typing.Optional[bool]" = None,
    ):
        self.preempt = preempt

    @classmethod
    def from_json(cls, obj: Json) -> "v1AllocationPreemptionSignalResponse":
        return cls(
            preempt=obj.get("preempt", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "preempt": self.preempt if self.preempt is not None else None,
        }

class v1AllocationReadyRequest:
    def __init__(
        self,
        allocationId: "typing.Optional[str]" = None,
    ):
        self.allocationId = allocationId

    @classmethod
    def from_json(cls, obj: Json) -> "v1AllocationReadyRequest":
        return cls(
            allocationId=obj.get("allocationId", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "allocationId": self.allocationId if self.allocationId is not None else None,
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

    def to_json(self) -> typing.Any:
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

    def to_json(self) -> typing.Any:
        return {
            "key": self.key,
            "value": self.value,
        }

class v1Checkpoint:
    def __init__(
        self,
        metadata: "typing.Dict[str, typing.Any]",
        resources: "typing.Dict[str, str]",
        training: "v1CheckpointTrainingMetadata",
        uuid: str,
        allocationId: "typing.Optional[str]" = None,
        reportTime: "typing.Optional[str]" = None,
        state: "typing.Optional[determinedcheckpointv1State]" = None,
        taskId: "typing.Optional[str]" = None,
    ):
        self.taskId = taskId
        self.allocationId = allocationId
        self.uuid = uuid
        self.reportTime = reportTime
        self.resources = resources
        self.metadata = metadata
        self.state = state
        self.training = training

    @classmethod
    def from_json(cls, obj: Json) -> "v1Checkpoint":
        return cls(
            taskId=obj.get("taskId", None),
            allocationId=obj.get("allocationId", None),
            uuid=obj["uuid"],
            reportTime=obj.get("reportTime", None),
            resources=obj["resources"],
            metadata=obj["metadata"],
            state=determinedcheckpointv1State(obj["state"]) if obj.get("state", None) is not None else None,
            training=v1CheckpointTrainingMetadata.from_json(obj["training"]),
        )

    def to_json(self) -> typing.Any:
        return {
            "taskId": self.taskId if self.taskId is not None else None,
            "allocationId": self.allocationId if self.allocationId is not None else None,
            "uuid": self.uuid,
            "reportTime": self.reportTime if self.reportTime is not None else None,
            "resources": self.resources,
            "metadata": self.metadata,
            "state": self.state.value if self.state is not None else None,
            "training": self.training.to_json(),
        }

class v1CheckpointTrainingMetadata:
    def __init__(
        self,
        experimentConfig: "typing.Optional[typing.Dict[str, typing.Any]]" = None,
        experimentId: "typing.Optional[int]" = None,
        hparams: "typing.Optional[typing.Dict[str, typing.Any]]" = None,
        searcherMetric: "typing.Optional[float]" = None,
        trainingMetrics: "typing.Optional[v1Metrics]" = None,
        trialId: "typing.Optional[int]" = None,
        validationMetrics: "typing.Optional[v1Metrics]" = None,
    ):
        self.trialId = trialId
        self.experimentId = experimentId
        self.experimentConfig = experimentConfig
        self.hparams = hparams
        self.trainingMetrics = trainingMetrics
        self.validationMetrics = validationMetrics
        self.searcherMetric = searcherMetric

    @classmethod
    def from_json(cls, obj: Json) -> "v1CheckpointTrainingMetadata":
        return cls(
            trialId=obj.get("trialId", None),
            experimentId=obj.get("experimentId", None),
            experimentConfig=obj.get("experimentConfig", None),
            hparams=obj.get("hparams", None),
            trainingMetrics=v1Metrics.from_json(obj["trainingMetrics"]) if obj.get("trainingMetrics", None) is not None else None,
            validationMetrics=v1Metrics.from_json(obj["validationMetrics"]) if obj.get("validationMetrics", None) is not None else None,
            searcherMetric=float(obj["searcherMetric"]) if obj.get("searcherMetric", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "trialId": self.trialId if self.trialId is not None else None,
            "experimentId": self.experimentId if self.experimentId is not None else None,
            "experimentConfig": self.experimentConfig if self.experimentConfig is not None else None,
            "hparams": self.hparams if self.hparams is not None else None,
            "trainingMetrics": self.trainingMetrics.to_json() if self.trainingMetrics is not None else None,
            "validationMetrics": self.validationMetrics.to_json() if self.validationMetrics is not None else None,
            "searcherMetric": dump_float(self.searcherMetric) if self.searcherMetric is not None else None,
        }

class v1CheckpointWorkload:
    def __init__(
        self,
        state: "determinedcheckpointv1State",
        totalBatches: int,
        endTime: "typing.Optional[str]" = None,
        resources: "typing.Optional[typing.Dict[str, str]]" = None,
        uuid: "typing.Optional[str]" = None,
    ):
        self.uuid = uuid
        self.endTime = endTime
        self.state = state
        self.resources = resources
        self.totalBatches = totalBatches

    @classmethod
    def from_json(cls, obj: Json) -> "v1CheckpointWorkload":
        return cls(
            uuid=obj.get("uuid", None),
            endTime=obj.get("endTime", None),
            state=determinedcheckpointv1State(obj["state"]),
            resources=obj.get("resources", None),
            totalBatches=obj["totalBatches"],
        )

    def to_json(self) -> typing.Any:
        return {
            "uuid": self.uuid if self.uuid is not None else None,
            "endTime": self.endTime if self.endTime is not None else None,
            "state": self.state.value,
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
        container: "typing.Optional[v1Container]" = None,
        displayName: "typing.Optional[str]" = None,
        exitStatus: "typing.Optional[str]" = None,
        userId: "typing.Optional[int]" = None,
    ):
        self.id = id
        self.description = description
        self.state = state
        self.startTime = startTime
        self.container = container
        self.displayName = displayName
        self.userId = userId
        self.username = username
        self.resourcePool = resourcePool
        self.exitStatus = exitStatus
        self.jobId = jobId

    @classmethod
    def from_json(cls, obj: Json) -> "v1Command":
        return cls(
            id=obj["id"],
            description=obj["description"],
            state=determinedtaskv1State(obj["state"]),
            startTime=obj["startTime"],
            container=v1Container.from_json(obj["container"]) if obj.get("container", None) is not None else None,
            displayName=obj.get("displayName", None),
            userId=obj.get("userId", None),
            username=obj["username"],
            resourcePool=obj["resourcePool"],
            exitStatus=obj.get("exitStatus", None),
            jobId=obj["jobId"],
        )

    def to_json(self) -> typing.Any:
        return {
            "id": self.id,
            "description": self.description,
            "state": self.state.value,
            "startTime": self.startTime,
            "container": self.container.to_json() if self.container is not None else None,
            "displayName": self.displayName if self.displayName is not None else None,
            "userId": self.userId if self.userId is not None else None,
            "username": self.username,
            "resourcePool": self.resourcePool,
            "exitStatus": self.exitStatus if self.exitStatus is not None else None,
            "jobId": self.jobId,
        }

class v1CompleteValidateAfterOperation:
    def __init__(
        self,
        op: "typing.Optional[v1ValidateAfterOperation]" = None,
        searcherMetric: "typing.Optional[float]" = None,
    ):
        self.op = op
        self.searcherMetric = searcherMetric

    @classmethod
    def from_json(cls, obj: Json) -> "v1CompleteValidateAfterOperation":
        return cls(
            op=v1ValidateAfterOperation.from_json(obj["op"]) if obj.get("op", None) is not None else None,
            searcherMetric=float(obj["searcherMetric"]) if obj.get("searcherMetric", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "op": self.op.to_json() if self.op is not None else None,
            "searcherMetric": dump_float(self.searcherMetric) if self.searcherMetric is not None else None,
        }

class v1Container:
    def __init__(
        self,
        id: str,
        state: "determinedcontainerv1State",
        devices: "typing.Optional[typing.Sequence[v1Device]]" = None,
        parent: "typing.Optional[str]" = None,
    ):
        self.parent = parent
        self.id = id
        self.state = state
        self.devices = devices

    @classmethod
    def from_json(cls, obj: Json) -> "v1Container":
        return cls(
            parent=obj.get("parent", None),
            id=obj["id"],
            state=determinedcontainerv1State(obj["state"]),
            devices=[v1Device.from_json(x) for x in obj["devices"]] if obj.get("devices", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "parent": self.parent if self.parent is not None else None,
            "id": self.id,
            "state": self.state.value,
            "devices": [x.to_json() for x in self.devices] if self.devices is not None else None,
        }

class v1CreateExperimentRequest:
    def __init__(
        self,
        activate: "typing.Optional[bool]" = None,
        config: "typing.Optional[str]" = None,
        modelDefinition: "typing.Optional[typing.Sequence[v1File]]" = None,
        parentId: "typing.Optional[int]" = None,
        validateOnly: "typing.Optional[bool]" = None,
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
            config=obj.get("config", None),
            validateOnly=obj.get("validateOnly", None),
            parentId=obj.get("parentId", None),
            activate=obj.get("activate", None),
        )

    def to_json(self) -> typing.Any:
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
        config: "typing.Dict[str, typing.Any]",
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

    def to_json(self) -> typing.Any:
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

    def to_json(self) -> typing.Any:
        return {
            "user": self.user.to_json(),
        }

class v1Device:
    def __init__(
        self,
        brand: "typing.Optional[str]" = None,
        id: "typing.Optional[int]" = None,
        type: "typing.Optional[determineddevicev1Type]" = None,
        uuid: "typing.Optional[str]" = None,
    ):
        self.id = id
        self.brand = brand
        self.uuid = uuid
        self.type = type

    @classmethod
    def from_json(cls, obj: Json) -> "v1Device":
        return cls(
            id=obj.get("id", None),
            brand=obj.get("brand", None),
            uuid=obj.get("uuid", None),
            type=determineddevicev1Type(obj["type"]) if obj.get("type", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "id": self.id if self.id is not None else None,
            "brand": self.brand if self.brand is not None else None,
            "uuid": self.uuid if self.uuid is not None else None,
            "type": self.type.value if self.type is not None else None,
        }

class v1DisableAgentRequest:
    def __init__(
        self,
        agentId: "typing.Optional[str]" = None,
        drain: "typing.Optional[bool]" = None,
    ):
        self.agentId = agentId
        self.drain = drain

    @classmethod
    def from_json(cls, obj: Json) -> "v1DisableAgentRequest":
        return cls(
            agentId=obj.get("agentId", None),
            drain=obj.get("drain", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "agentId": self.agentId if self.agentId is not None else None,
            "drain": self.drain if self.drain is not None else None,
        }

class v1DisableAgentResponse:
    def __init__(
        self,
        agent: "typing.Optional[v1Agent]" = None,
    ):
        self.agent = agent

    @classmethod
    def from_json(cls, obj: Json) -> "v1DisableAgentResponse":
        return cls(
            agent=v1Agent.from_json(obj["agent"]) if obj.get("agent", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "agent": self.agent.to_json() if self.agent is not None else None,
        }

class v1DisableSlotResponse:
    def __init__(
        self,
        slot: "typing.Optional[v1Slot]" = None,
    ):
        self.slot = slot

    @classmethod
    def from_json(cls, obj: Json) -> "v1DisableSlotResponse":
        return cls(
            slot=v1Slot.from_json(obj["slot"]) if obj.get("slot", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "slot": self.slot.to_json() if self.slot is not None else None,
        }

class v1EnableAgentResponse:
    def __init__(
        self,
        agent: "typing.Optional[v1Agent]" = None,
    ):
        self.agent = agent

    @classmethod
    def from_json(cls, obj: Json) -> "v1EnableAgentResponse":
        return cls(
            agent=v1Agent.from_json(obj["agent"]) if obj.get("agent", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "agent": self.agent.to_json() if self.agent is not None else None,
        }

class v1EnableSlotResponse:
    def __init__(
        self,
        slot: "typing.Optional[v1Slot]" = None,
    ):
        self.slot = slot

    @classmethod
    def from_json(cls, obj: Json) -> "v1EnableSlotResponse":
        return cls(
            slot=v1Slot.from_json(obj["slot"]) if obj.get("slot", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
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
        description: "typing.Optional[str]" = None,
        displayName: "typing.Optional[str]" = None,
        endTime: "typing.Optional[str]" = None,
        forkedFrom: "typing.Optional[int]" = None,
        labels: "typing.Optional[typing.Sequence[str]]" = None,
        notes: "typing.Optional[str]" = None,
        progress: "typing.Optional[float]" = None,
        resourcePool: "typing.Optional[str]" = None,
        trialIds: "typing.Optional[typing.Sequence[int]]" = None,
        userId: "typing.Optional[int]" = None,
    ):
        self.id = id
        self.description = description
        self.labels = labels
        self.startTime = startTime
        self.endTime = endTime
        self.state = state
        self.archived = archived
        self.numTrials = numTrials
        self.trialIds = trialIds
        self.displayName = displayName
        self.userId = userId
        self.username = username
        self.resourcePool = resourcePool
        self.searcherType = searcherType
        self.name = name
        self.notes = notes
        self.jobId = jobId
        self.forkedFrom = forkedFrom
        self.progress = progress

    @classmethod
    def from_json(cls, obj: Json) -> "v1Experiment":
        return cls(
            id=obj["id"],
            description=obj.get("description", None),
            labels=obj.get("labels", None),
            startTime=obj["startTime"],
            endTime=obj.get("endTime", None),
            state=determinedexperimentv1State(obj["state"]),
            archived=obj["archived"],
            numTrials=obj["numTrials"],
            trialIds=obj.get("trialIds", None),
            displayName=obj.get("displayName", None),
            userId=obj.get("userId", None),
            username=obj["username"],
            resourcePool=obj.get("resourcePool", None),
            searcherType=obj["searcherType"],
            name=obj["name"],
            notes=obj.get("notes", None),
            jobId=obj["jobId"],
            forkedFrom=obj.get("forkedFrom", None),
            progress=float(obj["progress"]) if obj.get("progress", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "id": self.id,
            "description": self.description if self.description is not None else None,
            "labels": self.labels if self.labels is not None else None,
            "startTime": self.startTime,
            "endTime": self.endTime if self.endTime is not None else None,
            "state": self.state.value,
            "archived": self.archived,
            "numTrials": self.numTrials,
            "trialIds": self.trialIds if self.trialIds is not None else None,
            "displayName": self.displayName if self.displayName is not None else None,
            "userId": self.userId if self.userId is not None else None,
            "username": self.username,
            "resourcePool": self.resourcePool if self.resourcePool is not None else None,
            "searcherType": self.searcherType,
            "name": self.name,
            "notes": self.notes if self.notes is not None else None,
            "jobId": self.jobId,
            "forkedFrom": self.forkedFrom if self.forkedFrom is not None else None,
            "progress": dump_float(self.progress) if self.progress is not None else None,
        }

class v1ExperimentSimulation:
    def __init__(
        self,
        config: "typing.Optional[typing.Dict[str, typing.Any]]" = None,
        seed: "typing.Optional[int]" = None,
        trials: "typing.Optional[typing.Sequence[v1TrialSimulation]]" = None,
    ):
        self.config = config
        self.seed = seed
        self.trials = trials

    @classmethod
    def from_json(cls, obj: Json) -> "v1ExperimentSimulation":
        return cls(
            config=obj.get("config", None),
            seed=obj.get("seed", None),
            trials=[v1TrialSimulation.from_json(x) for x in obj["trials"]] if obj.get("trials", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
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

    def to_json(self) -> typing.Any:
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
        agent: "typing.Optional[v1Agent]" = None,
    ):
        self.agent = agent

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetAgentResponse":
        return cls(
            agent=v1Agent.from_json(obj["agent"]) if obj.get("agent", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
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
        agents: "typing.Optional[typing.Sequence[v1Agent]]" = None,
        pagination: "typing.Optional[v1Pagination]" = None,
    ):
        self.agents = agents
        self.pagination = pagination

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetAgentsResponse":
        return cls(
            agents=[v1Agent.from_json(x) for x in obj["agents"]] if obj.get("agents", None) is not None else None,
            pagination=v1Pagination.from_json(obj["pagination"]) if obj.get("pagination", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "agents": [x.to_json() for x in self.agents] if self.agents is not None else None,
            "pagination": self.pagination.to_json() if self.pagination is not None else None,
        }

class v1GetBestSearcherValidationMetricResponse:
    def __init__(
        self,
        metric: "typing.Optional[float]" = None,
    ):
        self.metric = metric

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetBestSearcherValidationMetricResponse":
        return cls(
            metric=float(obj["metric"]) if obj.get("metric", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "metric": dump_float(self.metric) if self.metric is not None else None,
        }

class v1GetCheckpointResponse:
    def __init__(
        self,
        checkpoint: "typing.Optional[v1Checkpoint]" = None,
    ):
        self.checkpoint = checkpoint

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetCheckpointResponse":
        return cls(
            checkpoint=v1Checkpoint.from_json(obj["checkpoint"]) if obj.get("checkpoint", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "checkpoint": self.checkpoint.to_json() if self.checkpoint is not None else None,
        }

class v1GetCommandResponse:
    def __init__(
        self,
        command: "typing.Optional[v1Command]" = None,
        config: "typing.Optional[typing.Dict[str, typing.Any]]" = None,
    ):
        self.command = command
        self.config = config

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetCommandResponse":
        return cls(
            command=v1Command.from_json(obj["command"]) if obj.get("command", None) is not None else None,
            config=obj.get("config", None),
        )

    def to_json(self) -> typing.Any:
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
        commands: "typing.Optional[typing.Sequence[v1Command]]" = None,
        pagination: "typing.Optional[v1Pagination]" = None,
    ):
        self.commands = commands
        self.pagination = pagination

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetCommandsResponse":
        return cls(
            commands=[v1Command.from_json(x) for x in obj["commands"]] if obj.get("commands", None) is not None else None,
            pagination=v1Pagination.from_json(obj["pagination"]) if obj.get("pagination", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "commands": [x.to_json() for x in self.commands] if self.commands is not None else None,
            "pagination": self.pagination.to_json() if self.pagination is not None else None,
        }

class v1GetCurrentTrialSearcherOperationResponse:
    def __init__(
        self,
        completed: "typing.Optional[bool]" = None,
        op: "typing.Optional[v1SearcherOperation]" = None,
    ):
        self.op = op
        self.completed = completed

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetCurrentTrialSearcherOperationResponse":
        return cls(
            op=v1SearcherOperation.from_json(obj["op"]) if obj.get("op", None) is not None else None,
            completed=obj.get("completed", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "op": self.op.to_json() if self.op is not None else None,
            "completed": self.completed if self.completed is not None else None,
        }

class v1GetExperimentCheckpointsRequestSortBy(enum.Enum):
    SORT_BY_UNSPECIFIED = "SORT_BY_UNSPECIFIED"
    SORT_BY_UUID = "SORT_BY_UUID"
    SORT_BY_TRIAL_ID = "SORT_BY_TRIAL_ID"
    SORT_BY_BATCH_NUMBER = "SORT_BY_BATCH_NUMBER"
    SORT_BY_END_TIME = "SORT_BY_END_TIME"
    SORT_BY_STATE = "SORT_BY_STATE"
    SORT_BY_SEARCHER_METRIC = "SORT_BY_SEARCHER_METRIC"

class v1GetExperimentCheckpointsResponse:
    def __init__(
        self,
        checkpoints: "typing.Sequence[v1Checkpoint]",
        pagination: "v1Pagination",
    ):
        self.checkpoints = checkpoints
        self.pagination = pagination

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetExperimentCheckpointsResponse":
        return cls(
            checkpoints=[v1Checkpoint.from_json(x) for x in obj["checkpoints"]],
            pagination=v1Pagination.from_json(obj["pagination"]),
        )

    def to_json(self) -> typing.Any:
        return {
            "checkpoints": [x.to_json() for x in self.checkpoints],
            "pagination": self.pagination.to_json(),
        }

class v1GetExperimentLabelsResponse:
    def __init__(
        self,
        labels: "typing.Optional[typing.Sequence[str]]" = None,
    ):
        self.labels = labels

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetExperimentLabelsResponse":
        return cls(
            labels=obj.get("labels", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "labels": self.labels if self.labels is not None else None,
        }

class v1GetExperimentResponse:
    def __init__(
        self,
        config: "typing.Dict[str, typing.Any]",
        experiment: "v1Experiment",
        jobSummary: "typing.Optional[v1JobSummary]" = None,
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

    def to_json(self) -> typing.Any:
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
    SORT_BY_RESTARTS = "SORT_BY_RESTARTS"

class v1GetExperimentTrialsResponse:
    def __init__(
        self,
        pagination: "v1Pagination",
        trials: "typing.Sequence[trialv1Trial]",
    ):
        self.trials = trials
        self.pagination = pagination

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetExperimentTrialsResponse":
        return cls(
            trials=[trialv1Trial.from_json(x) for x in obj["trials"]],
            pagination=v1Pagination.from_json(obj["pagination"]),
        )

    def to_json(self) -> typing.Any:
        return {
            "trials": [x.to_json() for x in self.trials],
            "pagination": self.pagination.to_json(),
        }

class v1GetExperimentValidationHistoryResponse:
    def __init__(
        self,
        validationHistory: "typing.Optional[typing.Sequence[v1ValidationHistoryEntry]]" = None,
    ):
        self.validationHistory = validationHistory

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetExperimentValidationHistoryResponse":
        return cls(
            validationHistory=[v1ValidationHistoryEntry.from_json(x) for x in obj["validationHistory"]] if obj.get("validationHistory", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
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
    SORT_BY_FORKED_FROM = "SORT_BY_FORKED_FROM"
    SORT_BY_RESOURCE_POOL = "SORT_BY_RESOURCE_POOL"

class v1GetExperimentsResponse:
    def __init__(
        self,
        experiments: "typing.Sequence[v1Experiment]",
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

    def to_json(self) -> typing.Any:
        return {
            "experiments": [x.to_json() for x in self.experiments],
            "pagination": self.pagination.to_json(),
        }

class v1GetHPImportanceResponse:
    def __init__(
        self,
        trainingMetrics: "typing.Dict[str, GetHPImportanceResponseMetricHPImportance]",
        validationMetrics: "typing.Dict[str, GetHPImportanceResponseMetricHPImportance]",
    ):
        self.trainingMetrics = trainingMetrics
        self.validationMetrics = validationMetrics

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetHPImportanceResponse":
        return cls(
            trainingMetrics={k: GetHPImportanceResponseMetricHPImportance.from_json(v) for k, v in obj["trainingMetrics"].items()},
            validationMetrics={k: GetHPImportanceResponseMetricHPImportance.from_json(v) for k, v in obj["validationMetrics"].items()},
        )

    def to_json(self) -> typing.Any:
        return {
            "trainingMetrics": {k: v.to_json() for k, v in self.trainingMetrics.items()},
            "validationMetrics": {k: v.to_json() for k, v in self.validationMetrics.items()},
        }

class v1GetJobQueueStatsResponse:
    def __init__(
        self,
        results: "typing.Sequence[v1RPQueueStat]",
    ):
        self.results = results

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetJobQueueStatsResponse":
        return cls(
            results=[v1RPQueueStat.from_json(x) for x in obj["results"]],
        )

    def to_json(self) -> typing.Any:
        return {
            "results": [x.to_json() for x in self.results],
        }

class v1GetJobsResponse:
    def __init__(
        self,
        jobs: "typing.Sequence[v1Job]",
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

    def to_json(self) -> typing.Any:
        return {
            "pagination": self.pagination.to_json(),
            "jobs": [x.to_json() for x in self.jobs],
        }

class v1GetMasterConfigResponse:
    def __init__(
        self,
        config: "typing.Dict[str, typing.Any]",
    ):
        self.config = config

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetMasterConfigResponse":
        return cls(
            config=obj["config"],
        )

    def to_json(self) -> typing.Any:
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
        branding: "typing.Optional[str]" = None,
        externalLoginUri: "typing.Optional[str]" = None,
        externalLogoutUri: "typing.Optional[str]" = None,
        ssoProviders: "typing.Optional[typing.Sequence[v1SSOProvider]]" = None,
        telemetryEnabled: "typing.Optional[bool]" = None,
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
            telemetryEnabled=obj.get("telemetryEnabled", None),
            ssoProviders=[v1SSOProvider.from_json(x) for x in obj["ssoProviders"]] if obj.get("ssoProviders", None) is not None else None,
            externalLoginUri=obj.get("externalLoginUri", None),
            externalLogoutUri=obj.get("externalLogoutUri", None),
            branding=obj.get("branding", None),
        )

    def to_json(self) -> typing.Any:
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

    def to_json(self) -> typing.Any:
        return {
            "b64Tgz": self.b64Tgz,
        }

class v1GetModelLabelsResponse:
    def __init__(
        self,
        labels: "typing.Sequence[str]",
    ):
        self.labels = labels

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetModelLabelsResponse":
        return cls(
            labels=obj["labels"],
        )

    def to_json(self) -> typing.Any:
        return {
            "labels": self.labels,
        }

class v1GetModelResponse:
    def __init__(
        self,
        model: "v1Model",
    ):
        self.model = model

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetModelResponse":
        return cls(
            model=v1Model.from_json(obj["model"]),
        )

    def to_json(self) -> typing.Any:
        return {
            "model": self.model.to_json(),
        }

class v1GetModelVersionResponse:
    def __init__(
        self,
        modelVersion: "v1ModelVersion",
    ):
        self.modelVersion = modelVersion

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetModelVersionResponse":
        return cls(
            modelVersion=v1ModelVersion.from_json(obj["modelVersion"]),
        )

    def to_json(self) -> typing.Any:
        return {
            "modelVersion": self.modelVersion.to_json(),
        }

class v1GetModelVersionsRequestSortBy(enum.Enum):
    SORT_BY_UNSPECIFIED = "SORT_BY_UNSPECIFIED"
    SORT_BY_VERSION = "SORT_BY_VERSION"
    SORT_BY_CREATION_TIME = "SORT_BY_CREATION_TIME"

class v1GetModelVersionsResponse:
    def __init__(
        self,
        model: "v1Model",
        modelVersions: "typing.Sequence[v1ModelVersion]",
        pagination: "v1Pagination",
    ):
        self.model = model
        self.modelVersions = modelVersions
        self.pagination = pagination

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetModelVersionsResponse":
        return cls(
            model=v1Model.from_json(obj["model"]),
            modelVersions=[v1ModelVersion.from_json(x) for x in obj["modelVersions"]],
            pagination=v1Pagination.from_json(obj["pagination"]),
        )

    def to_json(self) -> typing.Any:
        return {
            "model": self.model.to_json(),
            "modelVersions": [x.to_json() for x in self.modelVersions],
            "pagination": self.pagination.to_json(),
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
        models: "typing.Sequence[v1Model]",
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

    def to_json(self) -> typing.Any:
        return {
            "models": [x.to_json() for x in self.models],
            "pagination": self.pagination.to_json(),
        }

class v1GetNotebookResponse:
    def __init__(
        self,
        config: "typing.Optional[typing.Dict[str, typing.Any]]" = None,
        notebook: "typing.Optional[v1Notebook]" = None,
    ):
        self.notebook = notebook
        self.config = config

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetNotebookResponse":
        return cls(
            notebook=v1Notebook.from_json(obj["notebook"]) if obj.get("notebook", None) is not None else None,
            config=obj.get("config", None),
        )

    def to_json(self) -> typing.Any:
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
        notebooks: "typing.Optional[typing.Sequence[v1Notebook]]" = None,
        pagination: "typing.Optional[v1Pagination]" = None,
    ):
        self.notebooks = notebooks
        self.pagination = pagination

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetNotebooksResponse":
        return cls(
            notebooks=[v1Notebook.from_json(x) for x in obj["notebooks"]] if obj.get("notebooks", None) is not None else None,
            pagination=v1Pagination.from_json(obj["pagination"]) if obj.get("pagination", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "notebooks": [x.to_json() for x in self.notebooks] if self.notebooks is not None else None,
            "pagination": self.pagination.to_json() if self.pagination is not None else None,
        }

class v1GetResourcePoolsResponse:
    def __init__(
        self,
        pagination: "typing.Optional[v1Pagination]" = None,
        resourcePools: "typing.Optional[typing.Sequence[v1ResourcePool]]" = None,
    ):
        self.resourcePools = resourcePools
        self.pagination = pagination

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetResourcePoolsResponse":
        return cls(
            resourcePools=[v1ResourcePool.from_json(x) for x in obj["resourcePools"]] if obj.get("resourcePools", None) is not None else None,
            pagination=v1Pagination.from_json(obj["pagination"]) if obj.get("pagination", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "resourcePools": [x.to_json() for x in self.resourcePools] if self.resourcePools is not None else None,
            "pagination": self.pagination.to_json() if self.pagination is not None else None,
        }

class v1GetShellResponse:
    def __init__(
        self,
        config: "typing.Optional[typing.Dict[str, typing.Any]]" = None,
        shell: "typing.Optional[v1Shell]" = None,
    ):
        self.shell = shell
        self.config = config

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetShellResponse":
        return cls(
            shell=v1Shell.from_json(obj["shell"]) if obj.get("shell", None) is not None else None,
            config=obj.get("config", None),
        )

    def to_json(self) -> typing.Any:
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
        pagination: "typing.Optional[v1Pagination]" = None,
        shells: "typing.Optional[typing.Sequence[v1Shell]]" = None,
    ):
        self.shells = shells
        self.pagination = pagination

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetShellsResponse":
        return cls(
            shells=[v1Shell.from_json(x) for x in obj["shells"]] if obj.get("shells", None) is not None else None,
            pagination=v1Pagination.from_json(obj["pagination"]) if obj.get("pagination", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "shells": [x.to_json() for x in self.shells] if self.shells is not None else None,
            "pagination": self.pagination.to_json() if self.pagination is not None else None,
        }

class v1GetSlotResponse:
    def __init__(
        self,
        slot: "typing.Optional[v1Slot]" = None,
    ):
        self.slot = slot

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetSlotResponse":
        return cls(
            slot=v1Slot.from_json(obj["slot"]) if obj.get("slot", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "slot": self.slot.to_json() if self.slot is not None else None,
        }

class v1GetSlotsResponse:
    def __init__(
        self,
        slots: "typing.Optional[typing.Sequence[v1Slot]]" = None,
    ):
        self.slots = slots

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetSlotsResponse":
        return cls(
            slots=[v1Slot.from_json(x) for x in obj["slots"]] if obj.get("slots", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "slots": [x.to_json() for x in self.slots] if self.slots is not None else None,
        }

class v1GetTaskResponse:
    def __init__(
        self,
        task: "typing.Optional[v1Task]" = None,
    ):
        self.task = task

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetTaskResponse":
        return cls(
            task=v1Task.from_json(obj["task"]) if obj.get("task", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "task": self.task.to_json() if self.task is not None else None,
        }

class v1GetTelemetryResponse:
    def __init__(
        self,
        enabled: bool,
        segmentKey: "typing.Optional[str]" = None,
    ):
        self.enabled = enabled
        self.segmentKey = segmentKey

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetTelemetryResponse":
        return cls(
            enabled=obj["enabled"],
            segmentKey=obj.get("segmentKey", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "enabled": self.enabled,
            "segmentKey": self.segmentKey if self.segmentKey is not None else None,
        }

class v1GetTemplateResponse:
    def __init__(
        self,
        template: "typing.Optional[v1Template]" = None,
    ):
        self.template = template

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetTemplateResponse":
        return cls(
            template=v1Template.from_json(obj["template"]) if obj.get("template", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "template": self.template.to_json() if self.template is not None else None,
        }

class v1GetTemplatesRequestSortBy(enum.Enum):
    SORT_BY_UNSPECIFIED = "SORT_BY_UNSPECIFIED"
    SORT_BY_NAME = "SORT_BY_NAME"

class v1GetTemplatesResponse:
    def __init__(
        self,
        pagination: "typing.Optional[v1Pagination]" = None,
        templates: "typing.Optional[typing.Sequence[v1Template]]" = None,
    ):
        self.templates = templates
        self.pagination = pagination

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetTemplatesResponse":
        return cls(
            templates=[v1Template.from_json(x) for x in obj["templates"]] if obj.get("templates", None) is not None else None,
            pagination=v1Pagination.from_json(obj["pagination"]) if obj.get("pagination", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "templates": [x.to_json() for x in self.templates] if self.templates is not None else None,
            "pagination": self.pagination.to_json() if self.pagination is not None else None,
        }

class v1GetTensorboardResponse:
    def __init__(
        self,
        config: "typing.Optional[typing.Dict[str, typing.Any]]" = None,
        tensorboard: "typing.Optional[v1Tensorboard]" = None,
    ):
        self.tensorboard = tensorboard
        self.config = config

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetTensorboardResponse":
        return cls(
            tensorboard=v1Tensorboard.from_json(obj["tensorboard"]) if obj.get("tensorboard", None) is not None else None,
            config=obj.get("config", None),
        )

    def to_json(self) -> typing.Any:
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
        pagination: "typing.Optional[v1Pagination]" = None,
        tensorboards: "typing.Optional[typing.Sequence[v1Tensorboard]]" = None,
    ):
        self.tensorboards = tensorboards
        self.pagination = pagination

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetTensorboardsResponse":
        return cls(
            tensorboards=[v1Tensorboard.from_json(x) for x in obj["tensorboards"]] if obj.get("tensorboards", None) is not None else None,
            pagination=v1Pagination.from_json(obj["pagination"]) if obj.get("pagination", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "tensorboards": [x.to_json() for x in self.tensorboards] if self.tensorboards is not None else None,
            "pagination": self.pagination.to_json() if self.pagination is not None else None,
        }

class v1GetTrialCheckpointsRequestSortBy(enum.Enum):
    SORT_BY_UNSPECIFIED = "SORT_BY_UNSPECIFIED"
    SORT_BY_UUID = "SORT_BY_UUID"
    SORT_BY_BATCH_NUMBER = "SORT_BY_BATCH_NUMBER"
    SORT_BY_END_TIME = "SORT_BY_END_TIME"
    SORT_BY_STATE = "SORT_BY_STATE"

class v1GetTrialCheckpointsResponse:
    def __init__(
        self,
        checkpoints: "typing.Optional[typing.Sequence[v1Checkpoint]]" = None,
        pagination: "typing.Optional[v1Pagination]" = None,
    ):
        self.checkpoints = checkpoints
        self.pagination = pagination

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetTrialCheckpointsResponse":
        return cls(
            checkpoints=[v1Checkpoint.from_json(x) for x in obj["checkpoints"]] if obj.get("checkpoints", None) is not None else None,
            pagination=v1Pagination.from_json(obj["pagination"]) if obj.get("pagination", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "checkpoints": [x.to_json() for x in self.checkpoints] if self.checkpoints is not None else None,
            "pagination": self.pagination.to_json() if self.pagination is not None else None,
        }

class v1GetTrialProfilerAvailableSeriesResponse:
    def __init__(
        self,
        labels: "typing.Sequence[v1TrialProfilerMetricLabels]",
    ):
        self.labels = labels

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetTrialProfilerAvailableSeriesResponse":
        return cls(
            labels=[v1TrialProfilerMetricLabels.from_json(x) for x in obj["labels"]],
        )

    def to_json(self) -> typing.Any:
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

    def to_json(self) -> typing.Any:
        return {
            "batch": self.batch.to_json(),
        }

class v1GetTrialResponse:
    def __init__(
        self,
        trial: "trialv1Trial",
        workloads: "typing.Sequence[v1WorkloadContainer]",
    ):
        self.trial = trial
        self.workloads = workloads

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetTrialResponse":
        return cls(
            trial=trialv1Trial.from_json(obj["trial"]),
            workloads=[v1WorkloadContainer.from_json(x) for x in obj["workloads"]],
        )

    def to_json(self) -> typing.Any:
        return {
            "trial": self.trial.to_json(),
            "workloads": [x.to_json() for x in self.workloads],
        }

class v1GetTrialWorkloadsResponse:
    def __init__(
        self,
        pagination: "v1Pagination",
        workloads: "typing.Sequence[v1WorkloadContainer]",
    ):
        self.workloads = workloads
        self.pagination = pagination

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetTrialWorkloadsResponse":
        return cls(
            workloads=[v1WorkloadContainer.from_json(x) for x in obj["workloads"]],
            pagination=v1Pagination.from_json(obj["pagination"]),
        )

    def to_json(self) -> typing.Any:
        return {
            "workloads": [x.to_json() for x in self.workloads],
            "pagination": self.pagination.to_json(),
        }

class v1GetUserResponse:
    def __init__(
        self,
        user: "typing.Optional[v1User]" = None,
    ):
        self.user = user

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetUserResponse":
        return cls(
            user=v1User.from_json(obj["user"]) if obj.get("user", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "user": self.user.to_json() if self.user is not None else None,
        }

class v1GetUsersResponse:
    def __init__(
        self,
        users: "typing.Optional[typing.Sequence[v1User]]" = None,
    ):
        self.users = users

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetUsersResponse":
        return cls(
            users=[v1User.from_json(x) for x in obj["users"]] if obj.get("users", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "users": [x.to_json() for x in self.users] if self.users is not None else None,
        }

class v1IdleNotebookRequest:
    def __init__(
        self,
        idle: "typing.Optional[bool]" = None,
        notebookId: "typing.Optional[str]" = None,
    ):
        self.notebookId = notebookId
        self.idle = idle

    @classmethod
    def from_json(cls, obj: Json) -> "v1IdleNotebookRequest":
        return cls(
            notebookId=obj.get("notebookId", None),
            idle=obj.get("idle", None),
        )

    def to_json(self) -> typing.Any:
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
        priority: "typing.Optional[int]" = None,
        progress: "typing.Optional[float]" = None,
        summary: "typing.Optional[v1JobSummary]" = None,
        userId: "typing.Optional[int]" = None,
        weight: "typing.Optional[float]" = None,
    ):
        self.summary = summary
        self.type = type
        self.submissionTime = submissionTime
        self.username = username
        self.userId = userId
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
            type=determinedjobv1Type(obj["type"]),
            submissionTime=obj["submissionTime"],
            username=obj["username"],
            userId=obj.get("userId", None),
            resourcePool=obj["resourcePool"],
            isPreemptible=obj["isPreemptible"],
            priority=obj.get("priority", None),
            weight=float(obj["weight"]) if obj.get("weight", None) is not None else None,
            entityId=obj["entityId"],
            jobId=obj["jobId"],
            requestedSlots=obj["requestedSlots"],
            allocatedSlots=obj["allocatedSlots"],
            name=obj["name"],
            progress=float(obj["progress"]) if obj.get("progress", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "summary": self.summary.to_json() if self.summary is not None else None,
            "type": self.type.value,
            "submissionTime": self.submissionTime,
            "username": self.username,
            "userId": self.userId if self.userId is not None else None,
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
            state=determinedjobv1State(obj["state"]),
            jobsAhead=obj["jobsAhead"],
        )

    def to_json(self) -> typing.Any:
        return {
            "state": self.state.value,
            "jobsAhead": self.jobsAhead,
        }

class v1K8PriorityClass:
    def __init__(
        self,
        priorityClass: "typing.Optional[str]" = None,
        priorityValue: "typing.Optional[int]" = None,
    ):
        self.priorityClass = priorityClass
        self.priorityValue = priorityValue

    @classmethod
    def from_json(cls, obj: Json) -> "v1K8PriorityClass":
        return cls(
            priorityClass=obj.get("priorityClass", None),
            priorityValue=obj.get("priorityValue", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "priorityClass": self.priorityClass if self.priorityClass is not None else None,
            "priorityValue": self.priorityValue if self.priorityValue is not None else None,
        }

class v1KillCommandResponse:
    def __init__(
        self,
        command: "typing.Optional[v1Command]" = None,
    ):
        self.command = command

    @classmethod
    def from_json(cls, obj: Json) -> "v1KillCommandResponse":
        return cls(
            command=v1Command.from_json(obj["command"]) if obj.get("command", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "command": self.command.to_json() if self.command is not None else None,
        }

class v1KillNotebookResponse:
    def __init__(
        self,
        notebook: "typing.Optional[v1Notebook]" = None,
    ):
        self.notebook = notebook

    @classmethod
    def from_json(cls, obj: Json) -> "v1KillNotebookResponse":
        return cls(
            notebook=v1Notebook.from_json(obj["notebook"]) if obj.get("notebook", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "notebook": self.notebook.to_json() if self.notebook is not None else None,
        }

class v1KillShellResponse:
    def __init__(
        self,
        shell: "typing.Optional[v1Shell]" = None,
    ):
        self.shell = shell

    @classmethod
    def from_json(cls, obj: Json) -> "v1KillShellResponse":
        return cls(
            shell=v1Shell.from_json(obj["shell"]) if obj.get("shell", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "shell": self.shell.to_json() if self.shell is not None else None,
        }

class v1KillTensorboardResponse:
    def __init__(
        self,
        tensorboard: "typing.Optional[v1Tensorboard]" = None,
    ):
        self.tensorboard = tensorboard

    @classmethod
    def from_json(cls, obj: Json) -> "v1KillTensorboardResponse":
        return cls(
            tensorboard=v1Tensorboard.from_json(obj["tensorboard"]) if obj.get("tensorboard", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "tensorboard": self.tensorboard.to_json() if self.tensorboard is not None else None,
        }

class v1LaunchCommandRequest:
    def __init__(
        self,
        config: "typing.Optional[typing.Dict[str, typing.Any]]" = None,
        data: "typing.Optional[str]" = None,
        files: "typing.Optional[typing.Sequence[v1File]]" = None,
        templateName: "typing.Optional[str]" = None,
    ):
        self.config = config
        self.templateName = templateName
        self.files = files
        self.data = data

    @classmethod
    def from_json(cls, obj: Json) -> "v1LaunchCommandRequest":
        return cls(
            config=obj.get("config", None),
            templateName=obj.get("templateName", None),
            files=[v1File.from_json(x) for x in obj["files"]] if obj.get("files", None) is not None else None,
            data=obj.get("data", None),
        )

    def to_json(self) -> typing.Any:
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
        config: "typing.Dict[str, typing.Any]",
    ):
        self.command = command
        self.config = config

    @classmethod
    def from_json(cls, obj: Json) -> "v1LaunchCommandResponse":
        return cls(
            command=v1Command.from_json(obj["command"]),
            config=obj["config"],
        )

    def to_json(self) -> typing.Any:
        return {
            "command": self.command.to_json(),
            "config": self.config,
        }

class v1LaunchNotebookRequest:
    def __init__(
        self,
        config: "typing.Optional[typing.Dict[str, typing.Any]]" = None,
        files: "typing.Optional[typing.Sequence[v1File]]" = None,
        preview: "typing.Optional[bool]" = None,
        templateName: "typing.Optional[str]" = None,
    ):
        self.config = config
        self.templateName = templateName
        self.files = files
        self.preview = preview

    @classmethod
    def from_json(cls, obj: Json) -> "v1LaunchNotebookRequest":
        return cls(
            config=obj.get("config", None),
            templateName=obj.get("templateName", None),
            files=[v1File.from_json(x) for x in obj["files"]] if obj.get("files", None) is not None else None,
            preview=obj.get("preview", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "config": self.config if self.config is not None else None,
            "templateName": self.templateName if self.templateName is not None else None,
            "files": [x.to_json() for x in self.files] if self.files is not None else None,
            "preview": self.preview if self.preview is not None else None,
        }

class v1LaunchNotebookResponse:
    def __init__(
        self,
        config: "typing.Dict[str, typing.Any]",
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

    def to_json(self) -> typing.Any:
        return {
            "notebook": self.notebook.to_json(),
            "config": self.config,
        }

class v1LaunchShellRequest:
    def __init__(
        self,
        config: "typing.Optional[typing.Dict[str, typing.Any]]" = None,
        data: "typing.Optional[str]" = None,
        files: "typing.Optional[typing.Sequence[v1File]]" = None,
        templateName: "typing.Optional[str]" = None,
    ):
        self.config = config
        self.templateName = templateName
        self.files = files
        self.data = data

    @classmethod
    def from_json(cls, obj: Json) -> "v1LaunchShellRequest":
        return cls(
            config=obj.get("config", None),
            templateName=obj.get("templateName", None),
            files=[v1File.from_json(x) for x in obj["files"]] if obj.get("files", None) is not None else None,
            data=obj.get("data", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "config": self.config if self.config is not None else None,
            "templateName": self.templateName if self.templateName is not None else None,
            "files": [x.to_json() for x in self.files] if self.files is not None else None,
            "data": self.data if self.data is not None else None,
        }

class v1LaunchShellResponse:
    def __init__(
        self,
        config: "typing.Dict[str, typing.Any]",
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

    def to_json(self) -> typing.Any:
        return {
            "shell": self.shell.to_json(),
            "config": self.config,
        }

class v1LaunchTensorboardRequest:
    def __init__(
        self,
        config: "typing.Optional[typing.Dict[str, typing.Any]]" = None,
        experimentIds: "typing.Optional[typing.Sequence[int]]" = None,
        files: "typing.Optional[typing.Sequence[v1File]]" = None,
        templateName: "typing.Optional[str]" = None,
        trialIds: "typing.Optional[typing.Sequence[int]]" = None,
    ):
        self.experimentIds = experimentIds
        self.trialIds = trialIds
        self.config = config
        self.templateName = templateName
        self.files = files

    @classmethod
    def from_json(cls, obj: Json) -> "v1LaunchTensorboardRequest":
        return cls(
            experimentIds=obj.get("experimentIds", None),
            trialIds=obj.get("trialIds", None),
            config=obj.get("config", None),
            templateName=obj.get("templateName", None),
            files=[v1File.from_json(x) for x in obj["files"]] if obj.get("files", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
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
        config: "typing.Dict[str, typing.Any]",
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

    def to_json(self) -> typing.Any:
        return {
            "tensorboard": self.tensorboard.to_json(),
            "config": self.config,
        }

class v1LogEntry:
    def __init__(
        self,
        id: int,
        level: "typing.Optional[v1LogLevel]" = None,
        message: "typing.Optional[str]" = None,
        timestamp: "typing.Optional[str]" = None,
    ):
        self.id = id
        self.message = message
        self.timestamp = timestamp
        self.level = level

    @classmethod
    def from_json(cls, obj: Json) -> "v1LogEntry":
        return cls(
            id=obj["id"],
            message=obj.get("message", None),
            timestamp=obj.get("timestamp", None),
            level=v1LogLevel(obj["level"]) if obj.get("level", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "id": self.id,
            "message": self.message if self.message is not None else None,
            "timestamp": self.timestamp if self.timestamp is not None else None,
            "level": self.level.value if self.level is not None else None,
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
        isHashed: "typing.Optional[bool]" = None,
    ):
        self.username = username
        self.password = password
        self.isHashed = isHashed

    @classmethod
    def from_json(cls, obj: Json) -> "v1LoginRequest":
        return cls(
            username=obj["username"],
            password=obj["password"],
            isHashed=obj.get("isHashed", None),
        )

    def to_json(self) -> typing.Any:
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

    def to_json(self) -> typing.Any:
        return {
            "token": self.token,
            "user": self.user.to_json(),
        }

class v1MarkAllocationResourcesDaemonRequest:
    def __init__(
        self,
        allocationId: str,
        resourcesId: "typing.Optional[str]" = None,
    ):
        self.allocationId = allocationId
        self.resourcesId = resourcesId

    @classmethod
    def from_json(cls, obj: Json) -> "v1MarkAllocationResourcesDaemonRequest":
        return cls(
            allocationId=obj["allocationId"],
            resourcesId=obj.get("resourcesId", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "allocationId": self.allocationId,
            "resourcesId": self.resourcesId if self.resourcesId is not None else None,
        }

class v1MasterLogsResponse:
    def __init__(
        self,
        logEntry: "typing.Optional[v1LogEntry]" = None,
    ):
        self.logEntry = logEntry

    @classmethod
    def from_json(cls, obj: Json) -> "v1MasterLogsResponse":
        return cls(
            logEntry=v1LogEntry.from_json(obj["logEntry"]) if obj.get("logEntry", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "logEntry": self.logEntry.to_json() if self.logEntry is not None else None,
        }

class v1MetricBatchesResponse:
    def __init__(
        self,
        batches: "typing.Optional[typing.Sequence[int]]" = None,
    ):
        self.batches = batches

    @classmethod
    def from_json(cls, obj: Json) -> "v1MetricBatchesResponse":
        return cls(
            batches=obj.get("batches", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "batches": self.batches if self.batches is not None else None,
        }

class v1MetricNamesResponse:
    def __init__(
        self,
        searcherMetric: "typing.Optional[str]" = None,
        trainingMetrics: "typing.Optional[typing.Sequence[str]]" = None,
        validationMetrics: "typing.Optional[typing.Sequence[str]]" = None,
    ):
        self.searcherMetric = searcherMetric
        self.trainingMetrics = trainingMetrics
        self.validationMetrics = validationMetrics

    @classmethod
    def from_json(cls, obj: Json) -> "v1MetricNamesResponse":
        return cls(
            searcherMetric=obj.get("searcherMetric", None),
            trainingMetrics=obj.get("trainingMetrics", None),
            validationMetrics=obj.get("validationMetrics", None),
        )

    def to_json(self) -> typing.Any:
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
        avgMetrics: "typing.Dict[str, typing.Any]",
        batchMetrics: "typing.Optional[typing.Sequence[typing.Dict[str, typing.Any]]]" = None,
    ):
        self.avgMetrics = avgMetrics
        self.batchMetrics = batchMetrics

    @classmethod
    def from_json(cls, obj: Json) -> "v1Metrics":
        return cls(
            avgMetrics=obj["avgMetrics"],
            batchMetrics=obj.get("batchMetrics", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "avgMetrics": self.avgMetrics,
            "batchMetrics": self.batchMetrics if self.batchMetrics is not None else None,
        }

class v1MetricsWorkload:
    def __init__(
        self,
        metrics: "typing.Dict[str, typing.Any]",
        numInputs: int,
        state: "determinedexperimentv1State",
        totalBatches: int,
        endTime: "typing.Optional[str]" = None,
    ):
        self.endTime = endTime
        self.state = state
        self.metrics = metrics
        self.numInputs = numInputs
        self.totalBatches = totalBatches

    @classmethod
    def from_json(cls, obj: Json) -> "v1MetricsWorkload":
        return cls(
            endTime=obj.get("endTime", None),
            state=determinedexperimentv1State(obj["state"]),
            metrics=obj["metrics"],
            numInputs=obj["numInputs"],
            totalBatches=obj["totalBatches"],
        )

    def to_json(self) -> typing.Any:
        return {
            "endTime": self.endTime if self.endTime is not None else None,
            "state": self.state.value,
            "metrics": self.metrics,
            "numInputs": self.numInputs,
            "totalBatches": self.totalBatches,
        }

class v1Model:
    def __init__(
        self,
        creationTime: str,
        id: int,
        lastUpdatedTime: str,
        metadata: "typing.Dict[str, typing.Any]",
        name: str,
        numVersions: int,
        username: str,
        archived: "typing.Optional[bool]" = None,
        description: "typing.Optional[str]" = None,
        labels: "typing.Optional[typing.Sequence[str]]" = None,
        notes: "typing.Optional[str]" = None,
        userId: "typing.Optional[int]" = None,
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
        self.userId = userId
        self.archived = archived
        self.notes = notes

    @classmethod
    def from_json(cls, obj: Json) -> "v1Model":
        return cls(
            name=obj["name"],
            description=obj.get("description", None),
            metadata=obj["metadata"],
            creationTime=obj["creationTime"],
            lastUpdatedTime=obj["lastUpdatedTime"],
            id=obj["id"],
            numVersions=obj["numVersions"],
            labels=obj.get("labels", None),
            username=obj["username"],
            userId=obj.get("userId", None),
            archived=obj.get("archived", None),
            notes=obj.get("notes", None),
        )

    def to_json(self) -> typing.Any:
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
            "userId": self.userId if self.userId is not None else None,
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
        comment: "typing.Optional[str]" = None,
        labels: "typing.Optional[typing.Sequence[str]]" = None,
        lastUpdatedTime: "typing.Optional[str]" = None,
        metadata: "typing.Optional[typing.Dict[str, typing.Any]]" = None,
        name: "typing.Optional[str]" = None,
        notes: "typing.Optional[str]" = None,
        userId: "typing.Optional[int]" = None,
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
        self.userId = userId
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
            name=obj.get("name", None),
            metadata=obj.get("metadata", None),
            lastUpdatedTime=obj.get("lastUpdatedTime", None),
            comment=obj.get("comment", None),
            username=obj["username"],
            userId=obj.get("userId", None),
            labels=obj.get("labels", None),
            notes=obj.get("notes", None),
        )

    def to_json(self) -> typing.Any:
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
            "userId": self.userId if self.userId is not None else None,
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
        container: "typing.Optional[v1Container]" = None,
        displayName: "typing.Optional[str]" = None,
        exitStatus: "typing.Optional[str]" = None,
        serviceAddress: "typing.Optional[str]" = None,
        userId: "typing.Optional[int]" = None,
    ):
        self.id = id
        self.description = description
        self.state = state
        self.startTime = startTime
        self.container = container
        self.displayName = displayName
        self.userId = userId
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
            state=determinedtaskv1State(obj["state"]),
            startTime=obj["startTime"],
            container=v1Container.from_json(obj["container"]) if obj.get("container", None) is not None else None,
            displayName=obj.get("displayName", None),
            userId=obj.get("userId", None),
            username=obj["username"],
            serviceAddress=obj.get("serviceAddress", None),
            resourcePool=obj["resourcePool"],
            exitStatus=obj.get("exitStatus", None),
            jobId=obj["jobId"],
        )

    def to_json(self) -> typing.Any:
        return {
            "id": self.id,
            "description": self.description,
            "state": self.state.value,
            "startTime": self.startTime,
            "container": self.container.to_json() if self.container is not None else None,
            "displayName": self.displayName if self.displayName is not None else None,
            "userId": self.userId if self.userId is not None else None,
            "username": self.username,
            "serviceAddress": self.serviceAddress if self.serviceAddress is not None else None,
            "resourcePool": self.resourcePool,
            "exitStatus": self.exitStatus if self.exitStatus is not None else None,
            "jobId": self.jobId,
        }

class v1OrderBy(enum.Enum):
    ORDER_BY_UNSPECIFIED = "ORDER_BY_UNSPECIFIED"
    ORDER_BY_ASC = "ORDER_BY_ASC"
    ORDER_BY_DESC = "ORDER_BY_DESC"

class v1Pagination:
    def __init__(
        self,
        endIndex: "typing.Optional[int]" = None,
        limit: "typing.Optional[int]" = None,
        offset: "typing.Optional[int]" = None,
        startIndex: "typing.Optional[int]" = None,
        total: "typing.Optional[int]" = None,
    ):
        self.offset = offset
        self.limit = limit
        self.startIndex = startIndex
        self.endIndex = endIndex
        self.total = total

    @classmethod
    def from_json(cls, obj: Json) -> "v1Pagination":
        return cls(
            offset=obj.get("offset", None),
            limit=obj.get("limit", None),
            startIndex=obj.get("startIndex", None),
            endIndex=obj.get("endIndex", None),
            total=obj.get("total", None),
        )

    def to_json(self) -> typing.Any:
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
        limit: "typing.Optional[int]" = None,
        offset: "typing.Optional[int]" = None,
    ):
        self.offset = offset
        self.limit = limit

    @classmethod
    def from_json(cls, obj: Json) -> "v1PaginationRequest":
        return cls(
            offset=obj.get("offset", None),
            limit=obj.get("limit", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "offset": self.offset if self.offset is not None else None,
            "limit": self.limit if self.limit is not None else None,
        }

class v1PatchExperiment:
    def __init__(
        self,
        id: int,
        description: "typing.Optional[str]" = None,
        labels: "typing.Optional[typing.Sequence[typing.Dict[str, typing.Any]]]" = None,
        name: "typing.Optional[str]" = None,
        notes: "typing.Optional[str]" = None,
    ):
        self.id = id
        self.description = description
        self.labels = labels
        self.name = name
        self.notes = notes

    @classmethod
    def from_json(cls, obj: Json) -> "v1PatchExperiment":
        return cls(
            id=obj["id"],
            description=obj.get("description", None),
            labels=obj.get("labels", None),
            name=obj.get("name", None),
            notes=obj.get("notes", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "id": self.id,
            "description": self.description if self.description is not None else None,
            "labels": self.labels if self.labels is not None else None,
            "name": self.name if self.name is not None else None,
            "notes": self.notes if self.notes is not None else None,
        }

class v1PatchExperimentResponse:
    def __init__(
        self,
        experiment: "typing.Optional[v1Experiment]" = None,
    ):
        self.experiment = experiment

    @classmethod
    def from_json(cls, obj: Json) -> "v1PatchExperimentResponse":
        return cls(
            experiment=v1Experiment.from_json(obj["experiment"]) if obj.get("experiment", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "experiment": self.experiment.to_json() if self.experiment is not None else None,
        }

class v1PatchModel:
    def __init__(
        self,
        description: "typing.Optional[str]" = None,
        labels: "typing.Optional[typing.Sequence[typing.Dict[str, typing.Any]]]" = None,
        metadata: "typing.Optional[typing.Dict[str, typing.Any]]" = None,
        name: "typing.Optional[str]" = None,
        notes: "typing.Optional[str]" = None,
    ):
        self.name = name
        self.description = description
        self.metadata = metadata
        self.labels = labels
        self.notes = notes

    @classmethod
    def from_json(cls, obj: Json) -> "v1PatchModel":
        return cls(
            name=obj.get("name", None),
            description=obj.get("description", None),
            metadata=obj.get("metadata", None),
            labels=obj.get("labels", None),
            notes=obj.get("notes", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "name": self.name if self.name is not None else None,
            "description": self.description if self.description is not None else None,
            "metadata": self.metadata if self.metadata is not None else None,
            "labels": self.labels if self.labels is not None else None,
            "notes": self.notes if self.notes is not None else None,
        }

class v1PatchModelResponse:
    def __init__(
        self,
        model: "v1Model",
    ):
        self.model = model

    @classmethod
    def from_json(cls, obj: Json) -> "v1PatchModelResponse":
        return cls(
            model=v1Model.from_json(obj["model"]),
        )

    def to_json(self) -> typing.Any:
        return {
            "model": self.model.to_json(),
        }

class v1PatchModelVersion:
    def __init__(
        self,
        checkpoint: "typing.Optional[v1Checkpoint]" = None,
        comment: "typing.Optional[str]" = None,
        labels: "typing.Optional[typing.Sequence[typing.Dict[str, typing.Any]]]" = None,
        metadata: "typing.Optional[typing.Dict[str, typing.Any]]" = None,
        name: "typing.Optional[str]" = None,
        notes: "typing.Optional[str]" = None,
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
            name=obj.get("name", None),
            metadata=obj.get("metadata", None),
            comment=obj.get("comment", None),
            labels=obj.get("labels", None),
            notes=obj.get("notes", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "checkpoint": self.checkpoint.to_json() if self.checkpoint is not None else None,
            "name": self.name if self.name is not None else None,
            "metadata": self.metadata if self.metadata is not None else None,
            "comment": self.comment if self.comment is not None else None,
            "labels": self.labels if self.labels is not None else None,
            "notes": self.notes if self.notes is not None else None,
        }

class v1PatchModelVersionResponse:
    def __init__(
        self,
        modelVersion: "v1ModelVersion",
    ):
        self.modelVersion = modelVersion

    @classmethod
    def from_json(cls, obj: Json) -> "v1PatchModelVersionResponse":
        return cls(
            modelVersion=v1ModelVersion.from_json(obj["modelVersion"]),
        )

    def to_json(self) -> typing.Any:
        return {
            "modelVersion": self.modelVersion.to_json(),
        }

class v1PatchUser:
    def __init__(
        self,
        displayName: "typing.Optional[str]" = None,
    ):
        self.displayName = displayName

    @classmethod
    def from_json(cls, obj: Json) -> "v1PatchUser":
        return cls(
            displayName=obj.get("displayName", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "displayName": self.displayName if self.displayName is not None else None,
        }

class v1PatchUserResponse:
    def __init__(
        self,
        user: "v1User",
    ):
        self.user = user

    @classmethod
    def from_json(cls, obj: Json) -> "v1PatchUserResponse":
        return cls(
            user=v1User.from_json(obj["user"]),
        )

    def to_json(self) -> typing.Any:
        return {
            "user": self.user.to_json(),
        }

class v1PostAllocationProxyAddressRequest:
    def __init__(
        self,
        allocationId: "typing.Optional[str]" = None,
        proxyAddress: "typing.Optional[str]" = None,
    ):
        self.allocationId = allocationId
        self.proxyAddress = proxyAddress

    @classmethod
    def from_json(cls, obj: Json) -> "v1PostAllocationProxyAddressRequest":
        return cls(
            allocationId=obj.get("allocationId", None),
            proxyAddress=obj.get("proxyAddress", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "allocationId": self.allocationId if self.allocationId is not None else None,
            "proxyAddress": self.proxyAddress if self.proxyAddress is not None else None,
        }

class v1PostCheckpointMetadataRequest:
    def __init__(
        self,
        checkpoint: "typing.Optional[v1Checkpoint]" = None,
    ):
        self.checkpoint = checkpoint

    @classmethod
    def from_json(cls, obj: Json) -> "v1PostCheckpointMetadataRequest":
        return cls(
            checkpoint=v1Checkpoint.from_json(obj["checkpoint"]) if obj.get("checkpoint", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "checkpoint": self.checkpoint.to_json() if self.checkpoint is not None else None,
        }

class v1PostCheckpointMetadataResponse:
    def __init__(
        self,
        checkpoint: "typing.Optional[v1Checkpoint]" = None,
    ):
        self.checkpoint = checkpoint

    @classmethod
    def from_json(cls, obj: Json) -> "v1PostCheckpointMetadataResponse":
        return cls(
            checkpoint=v1Checkpoint.from_json(obj["checkpoint"]) if obj.get("checkpoint", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "checkpoint": self.checkpoint.to_json() if self.checkpoint is not None else None,
        }

class v1PostModelRequest:
    def __init__(
        self,
        name: str,
        description: "typing.Optional[str]" = None,
        labels: "typing.Optional[typing.Sequence[str]]" = None,
        metadata: "typing.Optional[typing.Dict[str, typing.Any]]" = None,
        notes: "typing.Optional[str]" = None,
    ):
        self.name = name
        self.description = description
        self.metadata = metadata
        self.labels = labels
        self.notes = notes

    @classmethod
    def from_json(cls, obj: Json) -> "v1PostModelRequest":
        return cls(
            name=obj["name"],
            description=obj.get("description", None),
            metadata=obj.get("metadata", None),
            labels=obj.get("labels", None),
            notes=obj.get("notes", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "name": self.name,
            "description": self.description if self.description is not None else None,
            "metadata": self.metadata if self.metadata is not None else None,
            "labels": self.labels if self.labels is not None else None,
            "notes": self.notes if self.notes is not None else None,
        }

class v1PostModelResponse:
    def __init__(
        self,
        model: "v1Model",
    ):
        self.model = model

    @classmethod
    def from_json(cls, obj: Json) -> "v1PostModelResponse":
        return cls(
            model=v1Model.from_json(obj["model"]),
        )

    def to_json(self) -> typing.Any:
        return {
            "model": self.model.to_json(),
        }

class v1PostModelVersionRequest:
    def __init__(
        self,
        checkpointUuid: str,
        modelName: str,
        comment: "typing.Optional[str]" = None,
        labels: "typing.Optional[typing.Sequence[str]]" = None,
        metadata: "typing.Optional[typing.Dict[str, typing.Any]]" = None,
        name: "typing.Optional[str]" = None,
        notes: "typing.Optional[str]" = None,
    ):
        self.modelName = modelName
        self.checkpointUuid = checkpointUuid
        self.name = name
        self.comment = comment
        self.metadata = metadata
        self.labels = labels
        self.notes = notes

    @classmethod
    def from_json(cls, obj: Json) -> "v1PostModelVersionRequest":
        return cls(
            modelName=obj["modelName"],
            checkpointUuid=obj["checkpointUuid"],
            name=obj.get("name", None),
            comment=obj.get("comment", None),
            metadata=obj.get("metadata", None),
            labels=obj.get("labels", None),
            notes=obj.get("notes", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "modelName": self.modelName,
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
        modelVersion: "v1ModelVersion",
    ):
        self.modelVersion = modelVersion

    @classmethod
    def from_json(cls, obj: Json) -> "v1PostModelVersionResponse":
        return cls(
            modelVersion=v1ModelVersion.from_json(obj["modelVersion"]),
        )

    def to_json(self) -> typing.Any:
        return {
            "modelVersion": self.modelVersion.to_json(),
        }

class v1PostTrialProfilerMetricsBatchRequest:
    def __init__(
        self,
        batches: "typing.Optional[typing.Sequence[v1TrialProfilerMetricsBatch]]" = None,
    ):
        self.batches = batches

    @classmethod
    def from_json(cls, obj: Json) -> "v1PostTrialProfilerMetricsBatchRequest":
        return cls(
            batches=[v1TrialProfilerMetricsBatch.from_json(x) for x in obj["batches"]] if obj.get("batches", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "batches": [x.to_json() for x in self.batches] if self.batches is not None else None,
        }

class v1PostUserRequest:
    def __init__(
        self,
        password: "typing.Optional[str]" = None,
        user: "typing.Optional[v1User]" = None,
    ):
        self.user = user
        self.password = password

    @classmethod
    def from_json(cls, obj: Json) -> "v1PostUserRequest":
        return cls(
            user=v1User.from_json(obj["user"]) if obj.get("user", None) is not None else None,
            password=obj.get("password", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "user": self.user.to_json() if self.user is not None else None,
            "password": self.password if self.password is not None else None,
        }

class v1PostUserResponse:
    def __init__(
        self,
        user: "typing.Optional[v1User]" = None,
    ):
        self.user = user

    @classmethod
    def from_json(cls, obj: Json) -> "v1PostUserResponse":
        return cls(
            user=v1User.from_json(obj["user"]) if obj.get("user", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "user": self.user.to_json() if self.user is not None else None,
        }

class v1PreviewHPSearchRequest:
    def __init__(
        self,
        config: "typing.Optional[typing.Dict[str, typing.Any]]" = None,
        seed: "typing.Optional[int]" = None,
    ):
        self.config = config
        self.seed = seed

    @classmethod
    def from_json(cls, obj: Json) -> "v1PreviewHPSearchRequest":
        return cls(
            config=obj.get("config", None),
            seed=obj.get("seed", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "config": self.config if self.config is not None else None,
            "seed": self.seed if self.seed is not None else None,
        }

class v1PreviewHPSearchResponse:
    def __init__(
        self,
        simulation: "typing.Optional[v1ExperimentSimulation]" = None,
    ):
        self.simulation = simulation

    @classmethod
    def from_json(cls, obj: Json) -> "v1PreviewHPSearchResponse":
        return cls(
            simulation=v1ExperimentSimulation.from_json(obj["simulation"]) if obj.get("simulation", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "simulation": self.simulation.to_json() if self.simulation is not None else None,
        }

class v1PutTemplateResponse:
    def __init__(
        self,
        template: "typing.Optional[v1Template]" = None,
    ):
        self.template = template

    @classmethod
    def from_json(cls, obj: Json) -> "v1PutTemplateResponse":
        return cls(
            template=v1Template.from_json(obj["template"]) if obj.get("template", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "template": self.template.to_json() if self.template is not None else None,
        }

class v1QueueControl:
    def __init__(
        self,
        jobId: str,
        aheadOf: "typing.Optional[str]" = None,
        behindOf: "typing.Optional[str]" = None,
        priority: "typing.Optional[int]" = None,
        resourcePool: "typing.Optional[str]" = None,
        weight: "typing.Optional[float]" = None,
    ):
        self.jobId = jobId
        self.aheadOf = aheadOf
        self.behindOf = behindOf
        self.resourcePool = resourcePool
        self.priority = priority
        self.weight = weight

    @classmethod
    def from_json(cls, obj: Json) -> "v1QueueControl":
        return cls(
            jobId=obj["jobId"],
            aheadOf=obj.get("aheadOf", None),
            behindOf=obj.get("behindOf", None),
            resourcePool=obj.get("resourcePool", None),
            priority=obj.get("priority", None),
            weight=float(obj["weight"]) if obj.get("weight", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "jobId": self.jobId,
            "aheadOf": self.aheadOf if self.aheadOf is not None else None,
            "behindOf": self.behindOf if self.behindOf is not None else None,
            "resourcePool": self.resourcePool if self.resourcePool is not None else None,
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

    def to_json(self) -> typing.Any:
        return {
            "queuedCount": self.queuedCount,
            "scheduledCount": self.scheduledCount,
        }

class v1RPQueueStat:
    def __init__(
        self,
        resourcePool: str,
        stats: "v1QueueStats",
        aggregates: "typing.Optional[typing.Sequence[v1AggregateQueueStats]]" = None,
    ):
        self.stats = stats
        self.resourcePool = resourcePool
        self.aggregates = aggregates

    @classmethod
    def from_json(cls, obj: Json) -> "v1RPQueueStat":
        return cls(
            stats=v1QueueStats.from_json(obj["stats"]),
            resourcePool=obj["resourcePool"],
            aggregates=[v1AggregateQueueStats.from_json(x) for x in obj["aggregates"]] if obj.get("aggregates", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "stats": self.stats.to_json(),
            "resourcePool": self.resourcePool,
            "aggregates": [x.to_json() for x in self.aggregates] if self.aggregates is not None else None,
        }

class v1RendezvousInfo:
    def __init__(
        self,
        addresses: "typing.Sequence[str]",
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

    def to_json(self) -> typing.Any:
        return {
            "addresses": self.addresses,
            "rank": self.rank,
        }

class v1ResourceAllocationAggregatedEntry:
    def __init__(
        self,
        byAgentLabel: "typing.Dict[str, float]",
        byExperimentLabel: "typing.Dict[str, float]",
        byResourcePool: "typing.Dict[str, float]",
        byUsername: "typing.Dict[str, float]",
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
            period=v1ResourceAllocationAggregationPeriod(obj["period"]),
            seconds=float(obj["seconds"]),
            byUsername={k: float(v) for k, v in obj["byUsername"].items()},
            byExperimentLabel={k: float(v) for k, v in obj["byExperimentLabel"].items()},
            byResourcePool={k: float(v) for k, v in obj["byResourcePool"].items()},
            byAgentLabel={k: float(v) for k, v in obj["byAgentLabel"].items()},
        )

    def to_json(self) -> typing.Any:
        return {
            "periodStart": self.periodStart,
            "period": self.period.value,
            "seconds": dump_float(self.seconds),
            "byUsername": {k: dump_float(v) for k, v in self.byUsername.items()},
            "byExperimentLabel": {k: dump_float(v) for k, v in self.byExperimentLabel.items()},
            "byResourcePool": {k: dump_float(v) for k, v in self.byResourcePool.items()},
            "byAgentLabel": {k: dump_float(v) for k, v in self.byAgentLabel.items()},
        }

class v1ResourceAllocationAggregatedResponse:
    def __init__(
        self,
        resourceEntries: "typing.Sequence[v1ResourceAllocationAggregatedEntry]",
    ):
        self.resourceEntries = resourceEntries

    @classmethod
    def from_json(cls, obj: Json) -> "v1ResourceAllocationAggregatedResponse":
        return cls(
            resourceEntries=[v1ResourceAllocationAggregatedEntry.from_json(x) for x in obj["resourceEntries"]],
        )

    def to_json(self) -> typing.Any:
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
        endTime: "typing.Optional[str]" = None,
        experimentId: "typing.Optional[int]" = None,
        kind: "typing.Optional[str]" = None,
        labels: "typing.Optional[typing.Sequence[str]]" = None,
        seconds: "typing.Optional[float]" = None,
        slots: "typing.Optional[int]" = None,
        startTime: "typing.Optional[str]" = None,
        userId: "typing.Optional[int]" = None,
        username: "typing.Optional[str]" = None,
    ):
        self.kind = kind
        self.startTime = startTime
        self.endTime = endTime
        self.experimentId = experimentId
        self.username = username
        self.userId = userId
        self.labels = labels
        self.seconds = seconds
        self.slots = slots

    @classmethod
    def from_json(cls, obj: Json) -> "v1ResourceAllocationRawEntry":
        return cls(
            kind=obj.get("kind", None),
            startTime=obj.get("startTime", None),
            endTime=obj.get("endTime", None),
            experimentId=obj.get("experimentId", None),
            username=obj.get("username", None),
            userId=obj.get("userId", None),
            labels=obj.get("labels", None),
            seconds=float(obj["seconds"]) if obj.get("seconds", None) is not None else None,
            slots=obj.get("slots", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "kind": self.kind if self.kind is not None else None,
            "startTime": self.startTime if self.startTime is not None else None,
            "endTime": self.endTime if self.endTime is not None else None,
            "experimentId": self.experimentId if self.experimentId is not None else None,
            "username": self.username if self.username is not None else None,
            "userId": self.userId if self.userId is not None else None,
            "labels": self.labels if self.labels is not None else None,
            "seconds": dump_float(self.seconds) if self.seconds is not None else None,
            "slots": self.slots if self.slots is not None else None,
        }

class v1ResourceAllocationRawResponse:
    def __init__(
        self,
        resourceEntries: "typing.Optional[typing.Sequence[v1ResourceAllocationRawEntry]]" = None,
    ):
        self.resourceEntries = resourceEntries

    @classmethod
    def from_json(cls, obj: Json) -> "v1ResourceAllocationRawResponse":
        return cls(
            resourceEntries=[v1ResourceAllocationRawEntry.from_json(x) for x in obj["resourceEntries"]] if obj.get("resourceEntries", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
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
        accelerator: "typing.Optional[str]" = None,
        slotsPerAgent: "typing.Optional[int]" = None,
        stats: "typing.Optional[v1QueueStats]" = None,
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
        self.accelerator = accelerator
        self.stats = stats

    @classmethod
    def from_json(cls, obj: Json) -> "v1ResourcePool":
        return cls(
            name=obj["name"],
            description=obj["description"],
            type=v1ResourcePoolType(obj["type"]),
            numAgents=obj["numAgents"],
            slotsAvailable=obj["slotsAvailable"],
            slotsUsed=obj["slotsUsed"],
            slotType=determineddevicev1Type(obj["slotType"]),
            auxContainerCapacity=obj["auxContainerCapacity"],
            auxContainersRunning=obj["auxContainersRunning"],
            defaultComputePool=obj["defaultComputePool"],
            defaultAuxPool=obj["defaultAuxPool"],
            preemptible=obj["preemptible"],
            minAgents=obj["minAgents"],
            maxAgents=obj["maxAgents"],
            slotsPerAgent=obj.get("slotsPerAgent", None),
            auxContainerCapacityPerAgent=obj["auxContainerCapacityPerAgent"],
            schedulerType=v1SchedulerType(obj["schedulerType"]),
            schedulerFittingPolicy=v1FittingPolicy(obj["schedulerFittingPolicy"]),
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
            accelerator=obj.get("accelerator", None),
            stats=v1QueueStats.from_json(obj["stats"]) if obj.get("stats", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "name": self.name,
            "description": self.description,
            "type": self.type.value,
            "numAgents": self.numAgents,
            "slotsAvailable": self.slotsAvailable,
            "slotsUsed": self.slotsUsed,
            "slotType": self.slotType.value,
            "auxContainerCapacity": self.auxContainerCapacity,
            "auxContainersRunning": self.auxContainersRunning,
            "defaultComputePool": self.defaultComputePool,
            "defaultAuxPool": self.defaultAuxPool,
            "preemptible": self.preemptible,
            "minAgents": self.minAgents,
            "maxAgents": self.maxAgents,
            "slotsPerAgent": self.slotsPerAgent if self.slotsPerAgent is not None else None,
            "auxContainerCapacityPerAgent": self.auxContainerCapacityPerAgent,
            "schedulerType": self.schedulerType.value,
            "schedulerFittingPolicy": self.schedulerFittingPolicy.value,
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
            "accelerator": self.accelerator if self.accelerator is not None else None,
            "stats": self.stats.to_json() if self.stats is not None else None,
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
        customTags: "typing.Optional[typing.Sequence[v1AwsCustomTag]]" = None,
        instanceType: "typing.Optional[str]" = None,
        logGroup: "typing.Optional[str]" = None,
        logStream: "typing.Optional[str]" = None,
        spotMaxPrice: "typing.Optional[str]" = None,
        subnetId: "typing.Optional[str]" = None,
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
            subnetId=obj.get("subnetId", None),
            securityGroupId=obj["securityGroupId"],
            iamInstanceProfileArn=obj["iamInstanceProfileArn"],
            instanceType=obj.get("instanceType", None),
            logGroup=obj.get("logGroup", None),
            logStream=obj.get("logStream", None),
            spotEnabled=obj["spotEnabled"],
            spotMaxPrice=obj.get("spotMaxPrice", None),
            customTags=[v1AwsCustomTag.from_json(x) for x in obj["customTags"]] if obj.get("customTags", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
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
        aws: "typing.Optional[v1ResourcePoolAwsDetail]" = None,
        gcp: "typing.Optional[v1ResourcePoolGcpDetail]" = None,
        priorityScheduler: "typing.Optional[v1ResourcePoolPrioritySchedulerDetail]" = None,
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

    def to_json(self) -> typing.Any:
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
        serviceAccountScopes: "typing.Sequence[str]",
        zone: str,
        networkTags: "typing.Optional[typing.Sequence[str]]" = None,
        subnetwork: "typing.Optional[str]" = None,
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
            subnetwork=obj.get("subnetwork", None),
            externalIp=obj["externalIp"],
            networkTags=obj.get("networkTags", None),
            serviceAccountEmail=obj["serviceAccountEmail"],
            serviceAccountScopes=obj["serviceAccountScopes"],
            machineType=obj["machineType"],
            gpuType=obj["gpuType"],
            gpuNum=obj["gpuNum"],
            preemptible=obj["preemptible"],
            operationTimeoutPeriod=float(obj["operationTimeoutPeriod"]),
        )

    def to_json(self) -> typing.Any:
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
        k8Priorities: "typing.Optional[typing.Sequence[v1K8PriorityClass]]" = None,
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

    def to_json(self) -> typing.Any:
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
        length: "typing.Optional[str]" = None,
        type: "typing.Optional[v1RunnableType]" = None,
    ):
        self.type = type
        self.length = length

    @classmethod
    def from_json(cls, obj: Json) -> "v1RunnableOperation":
        return cls(
            type=v1RunnableType(obj["type"]) if obj.get("type", None) is not None else None,
            length=obj.get("length", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "type": self.type.value if self.type is not None else None,
            "length": self.length if self.length is not None else None,
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

    def to_json(self) -> typing.Any:
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
        validateAfter: "typing.Optional[v1ValidateAfterOperation]" = None,
    ):
        self.validateAfter = validateAfter

    @classmethod
    def from_json(cls, obj: Json) -> "v1SearcherOperation":
        return cls(
            validateAfter=v1ValidateAfterOperation.from_json(obj["validateAfter"]) if obj.get("validateAfter", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "validateAfter": self.validateAfter.to_json() if self.validateAfter is not None else None,
        }

class v1SetCommandPriorityRequest:
    def __init__(
        self,
        commandId: "typing.Optional[str]" = None,
        priority: "typing.Optional[int]" = None,
    ):
        self.commandId = commandId
        self.priority = priority

    @classmethod
    def from_json(cls, obj: Json) -> "v1SetCommandPriorityRequest":
        return cls(
            commandId=obj.get("commandId", None),
            priority=obj.get("priority", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "commandId": self.commandId if self.commandId is not None else None,
            "priority": self.priority if self.priority is not None else None,
        }

class v1SetCommandPriorityResponse:
    def __init__(
        self,
        command: "typing.Optional[v1Command]" = None,
    ):
        self.command = command

    @classmethod
    def from_json(cls, obj: Json) -> "v1SetCommandPriorityResponse":
        return cls(
            command=v1Command.from_json(obj["command"]) if obj.get("command", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "command": self.command.to_json() if self.command is not None else None,
        }

class v1SetNotebookPriorityRequest:
    def __init__(
        self,
        notebookId: "typing.Optional[str]" = None,
        priority: "typing.Optional[int]" = None,
    ):
        self.notebookId = notebookId
        self.priority = priority

    @classmethod
    def from_json(cls, obj: Json) -> "v1SetNotebookPriorityRequest":
        return cls(
            notebookId=obj.get("notebookId", None),
            priority=obj.get("priority", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "notebookId": self.notebookId if self.notebookId is not None else None,
            "priority": self.priority if self.priority is not None else None,
        }

class v1SetNotebookPriorityResponse:
    def __init__(
        self,
        notebook: "typing.Optional[v1Notebook]" = None,
    ):
        self.notebook = notebook

    @classmethod
    def from_json(cls, obj: Json) -> "v1SetNotebookPriorityResponse":
        return cls(
            notebook=v1Notebook.from_json(obj["notebook"]) if obj.get("notebook", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "notebook": self.notebook.to_json() if self.notebook is not None else None,
        }

class v1SetShellPriorityRequest:
    def __init__(
        self,
        priority: "typing.Optional[int]" = None,
        shellId: "typing.Optional[str]" = None,
    ):
        self.shellId = shellId
        self.priority = priority

    @classmethod
    def from_json(cls, obj: Json) -> "v1SetShellPriorityRequest":
        return cls(
            shellId=obj.get("shellId", None),
            priority=obj.get("priority", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "shellId": self.shellId if self.shellId is not None else None,
            "priority": self.priority if self.priority is not None else None,
        }

class v1SetShellPriorityResponse:
    def __init__(
        self,
        shell: "typing.Optional[v1Shell]" = None,
    ):
        self.shell = shell

    @classmethod
    def from_json(cls, obj: Json) -> "v1SetShellPriorityResponse":
        return cls(
            shell=v1Shell.from_json(obj["shell"]) if obj.get("shell", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "shell": self.shell.to_json() if self.shell is not None else None,
        }

class v1SetTensorboardPriorityRequest:
    def __init__(
        self,
        priority: "typing.Optional[int]" = None,
        tensorboardId: "typing.Optional[str]" = None,
    ):
        self.tensorboardId = tensorboardId
        self.priority = priority

    @classmethod
    def from_json(cls, obj: Json) -> "v1SetTensorboardPriorityRequest":
        return cls(
            tensorboardId=obj.get("tensorboardId", None),
            priority=obj.get("priority", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "tensorboardId": self.tensorboardId if self.tensorboardId is not None else None,
            "priority": self.priority if self.priority is not None else None,
        }

class v1SetTensorboardPriorityResponse:
    def __init__(
        self,
        tensorboard: "typing.Optional[v1Tensorboard]" = None,
    ):
        self.tensorboard = tensorboard

    @classmethod
    def from_json(cls, obj: Json) -> "v1SetTensorboardPriorityResponse":
        return cls(
            tensorboard=v1Tensorboard.from_json(obj["tensorboard"]) if obj.get("tensorboard", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "tensorboard": self.tensorboard.to_json() if self.tensorboard is not None else None,
        }

class v1SetUserPasswordResponse:
    def __init__(
        self,
        user: "typing.Optional[v1User]" = None,
    ):
        self.user = user

    @classmethod
    def from_json(cls, obj: Json) -> "v1SetUserPasswordResponse":
        return cls(
            user=v1User.from_json(obj["user"]) if obj.get("user", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
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
        addresses: "typing.Optional[typing.Sequence[typing.Dict[str, typing.Any]]]" = None,
        agentUserGroup: "typing.Optional[typing.Dict[str, typing.Any]]" = None,
        container: "typing.Optional[v1Container]" = None,
        displayName: "typing.Optional[str]" = None,
        exitStatus: "typing.Optional[str]" = None,
        privateKey: "typing.Optional[str]" = None,
        publicKey: "typing.Optional[str]" = None,
        userId: "typing.Optional[int]" = None,
    ):
        self.id = id
        self.description = description
        self.state = state
        self.startTime = startTime
        self.container = container
        self.privateKey = privateKey
        self.publicKey = publicKey
        self.displayName = displayName
        self.userId = userId
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
            state=determinedtaskv1State(obj["state"]),
            startTime=obj["startTime"],
            container=v1Container.from_json(obj["container"]) if obj.get("container", None) is not None else None,
            privateKey=obj.get("privateKey", None),
            publicKey=obj.get("publicKey", None),
            displayName=obj.get("displayName", None),
            userId=obj.get("userId", None),
            username=obj["username"],
            resourcePool=obj["resourcePool"],
            exitStatus=obj.get("exitStatus", None),
            addresses=obj.get("addresses", None),
            agentUserGroup=obj.get("agentUserGroup", None),
            jobId=obj["jobId"],
        )

    def to_json(self) -> typing.Any:
        return {
            "id": self.id,
            "description": self.description,
            "state": self.state.value,
            "startTime": self.startTime,
            "container": self.container.to_json() if self.container is not None else None,
            "privateKey": self.privateKey if self.privateKey is not None else None,
            "publicKey": self.publicKey if self.publicKey is not None else None,
            "displayName": self.displayName if self.displayName is not None else None,
            "userId": self.userId if self.userId is not None else None,
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
        container: "typing.Optional[v1Container]" = None,
        device: "typing.Optional[v1Device]" = None,
        draining: "typing.Optional[bool]" = None,
        enabled: "typing.Optional[bool]" = None,
        id: "typing.Optional[str]" = None,
    ):
        self.id = id
        self.device = device
        self.enabled = enabled
        self.container = container
        self.draining = draining

    @classmethod
    def from_json(cls, obj: Json) -> "v1Slot":
        return cls(
            id=obj.get("id", None),
            device=v1Device.from_json(obj["device"]) if obj.get("device", None) is not None else None,
            enabled=obj.get("enabled", None),
            container=v1Container.from_json(obj["container"]) if obj.get("container", None) is not None else None,
            draining=obj.get("draining", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "id": self.id if self.id is not None else None,
            "device": self.device.to_json() if self.device is not None else None,
            "enabled": self.enabled if self.enabled is not None else None,
            "container": self.container.to_json() if self.container is not None else None,
            "draining": self.draining if self.draining is not None else None,
        }

class v1Task:
    def __init__(
        self,
        allocations: "typing.Optional[typing.Sequence[v1Allocation]]" = None,
        taskId: "typing.Optional[str]" = None,
    ):
        self.taskId = taskId
        self.allocations = allocations

    @classmethod
    def from_json(cls, obj: Json) -> "v1Task":
        return cls(
            taskId=obj.get("taskId", None),
            allocations=[v1Allocation.from_json(x) for x in obj["allocations"]] if obj.get("allocations", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "taskId": self.taskId if self.taskId is not None else None,
            "allocations": [x.to_json() for x in self.allocations] if self.allocations is not None else None,
        }

class v1TaskLogsFieldsResponse:
    def __init__(
        self,
        agentIds: "typing.Optional[typing.Sequence[str]]" = None,
        allocationIds: "typing.Optional[typing.Sequence[str]]" = None,
        containerIds: "typing.Optional[typing.Sequence[str]]" = None,
        rankIds: "typing.Optional[typing.Sequence[int]]" = None,
        sources: "typing.Optional[typing.Sequence[str]]" = None,
        stdtypes: "typing.Optional[typing.Sequence[str]]" = None,
    ):
        self.allocationIds = allocationIds
        self.agentIds = agentIds
        self.containerIds = containerIds
        self.rankIds = rankIds
        self.stdtypes = stdtypes
        self.sources = sources

    @classmethod
    def from_json(cls, obj: Json) -> "v1TaskLogsFieldsResponse":
        return cls(
            allocationIds=obj.get("allocationIds", None),
            agentIds=obj.get("agentIds", None),
            containerIds=obj.get("containerIds", None),
            rankIds=obj.get("rankIds", None),
            stdtypes=obj.get("stdtypes", None),
            sources=obj.get("sources", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "allocationIds": self.allocationIds if self.allocationIds is not None else None,
            "agentIds": self.agentIds if self.agentIds is not None else None,
            "containerIds": self.containerIds if self.containerIds is not None else None,
            "rankIds": self.rankIds if self.rankIds is not None else None,
            "stdtypes": self.stdtypes if self.stdtypes is not None else None,
            "sources": self.sources if self.sources is not None else None,
        }

class v1TaskLogsResponse:
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
    def from_json(cls, obj: Json) -> "v1TaskLogsResponse":
        return cls(
            id=obj["id"],
            timestamp=obj["timestamp"],
            message=obj["message"],
            level=v1LogLevel(obj["level"]),
        )

    def to_json(self) -> typing.Any:
        return {
            "id": self.id,
            "timestamp": self.timestamp,
            "message": self.message,
            "level": self.level.value,
        }

class v1Template:
    def __init__(
        self,
        config: "typing.Dict[str, typing.Any]",
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

    def to_json(self) -> typing.Any:
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
        container: "typing.Optional[v1Container]" = None,
        displayName: "typing.Optional[str]" = None,
        exitStatus: "typing.Optional[str]" = None,
        experimentIds: "typing.Optional[typing.Sequence[int]]" = None,
        serviceAddress: "typing.Optional[str]" = None,
        trialIds: "typing.Optional[typing.Sequence[int]]" = None,
        userId: "typing.Optional[int]" = None,
    ):
        self.id = id
        self.description = description
        self.state = state
        self.startTime = startTime
        self.container = container
        self.experimentIds = experimentIds
        self.trialIds = trialIds
        self.displayName = displayName
        self.userId = userId
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
            state=determinedtaskv1State(obj["state"]),
            startTime=obj["startTime"],
            container=v1Container.from_json(obj["container"]) if obj.get("container", None) is not None else None,
            experimentIds=obj.get("experimentIds", None),
            trialIds=obj.get("trialIds", None),
            displayName=obj.get("displayName", None),
            userId=obj.get("userId", None),
            username=obj["username"],
            serviceAddress=obj.get("serviceAddress", None),
            resourcePool=obj["resourcePool"],
            exitStatus=obj.get("exitStatus", None),
            jobId=obj["jobId"],
        )

    def to_json(self) -> typing.Any:
        return {
            "id": self.id,
            "description": self.description,
            "state": self.state.value,
            "startTime": self.startTime,
            "container": self.container.to_json() if self.container is not None else None,
            "experimentIds": self.experimentIds if self.experimentIds is not None else None,
            "trialIds": self.trialIds if self.trialIds is not None else None,
            "displayName": self.displayName if self.displayName is not None else None,
            "userId": self.userId if self.userId is not None else None,
            "username": self.username,
            "serviceAddress": self.serviceAddress if self.serviceAddress is not None else None,
            "resourcePool": self.resourcePool,
            "exitStatus": self.exitStatus if self.exitStatus is not None else None,
            "jobId": self.jobId,
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
            reason=TrialEarlyExitExitedReason(obj["reason"]),
        )

    def to_json(self) -> typing.Any:
        return {
            "reason": self.reason.value,
        }

class v1TrialLogsFieldsResponse:
    def __init__(
        self,
        agentIds: "typing.Optional[typing.Sequence[str]]" = None,
        containerIds: "typing.Optional[typing.Sequence[str]]" = None,
        rankIds: "typing.Optional[typing.Sequence[int]]" = None,
        sources: "typing.Optional[typing.Sequence[str]]" = None,
        stdtypes: "typing.Optional[typing.Sequence[str]]" = None,
    ):
        self.agentIds = agentIds
        self.containerIds = containerIds
        self.rankIds = rankIds
        self.stdtypes = stdtypes
        self.sources = sources

    @classmethod
    def from_json(cls, obj: Json) -> "v1TrialLogsFieldsResponse":
        return cls(
            agentIds=obj.get("agentIds", None),
            containerIds=obj.get("containerIds", None),
            rankIds=obj.get("rankIds", None),
            stdtypes=obj.get("stdtypes", None),
            sources=obj.get("sources", None),
        )

    def to_json(self) -> typing.Any:
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
            level=v1LogLevel(obj["level"]),
        )

    def to_json(self) -> typing.Any:
        return {
            "id": self.id,
            "timestamp": self.timestamp,
            "message": self.message,
            "level": self.level.value,
        }

class v1TrialMetrics:
    def __init__(
        self,
        metrics: "typing.Dict[str, typing.Any]",
        stepsCompleted: int,
        trialId: int,
        trialRunId: int,
        batchMetrics: "typing.Optional[typing.Sequence[typing.Dict[str, typing.Any]]]" = None,
    ):
        self.trialId = trialId
        self.trialRunId = trialRunId
        self.stepsCompleted = stepsCompleted
        self.metrics = metrics
        self.batchMetrics = batchMetrics

    @classmethod
    def from_json(cls, obj: Json) -> "v1TrialMetrics":
        return cls(
            trialId=obj["trialId"],
            trialRunId=obj["trialRunId"],
            stepsCompleted=obj["stepsCompleted"],
            metrics=obj["metrics"],
            batchMetrics=obj.get("batchMetrics", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "trialId": self.trialId,
            "trialRunId": self.trialRunId,
            "stepsCompleted": self.stepsCompleted,
            "metrics": self.metrics,
            "batchMetrics": self.batchMetrics if self.batchMetrics is not None else None,
        }

class v1TrialProfilerMetricLabels:
    def __init__(
        self,
        name: str,
        trialId: int,
        agentId: "typing.Optional[str]" = None,
        gpuUuid: "typing.Optional[str]" = None,
        metricType: "typing.Optional[TrialProfilerMetricLabelsProfilerMetricType]" = None,
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
            agentId=obj.get("agentId", None),
            gpuUuid=obj.get("gpuUuid", None),
            metricType=TrialProfilerMetricLabelsProfilerMetricType(obj["metricType"]) if obj.get("metricType", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "trialId": self.trialId,
            "name": self.name,
            "agentId": self.agentId if self.agentId is not None else None,
            "gpuUuid": self.gpuUuid if self.gpuUuid is not None else None,
            "metricType": self.metricType.value if self.metricType is not None else None,
        }

class v1TrialProfilerMetricsBatch:
    def __init__(
        self,
        batches: "typing.Sequence[int]",
        labels: "v1TrialProfilerMetricLabels",
        timestamps: "typing.Sequence[str]",
        values: "typing.Sequence[float]",
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

    def to_json(self) -> typing.Any:
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

    def to_json(self) -> typing.Any:
        return {
            "state": self.state,
        }

class v1TrialSimulation:
    def __init__(
        self,
        occurrences: "typing.Optional[int]" = None,
        operations: "typing.Optional[typing.Sequence[v1RunnableOperation]]" = None,
    ):
        self.operations = operations
        self.occurrences = occurrences

    @classmethod
    def from_json(cls, obj: Json) -> "v1TrialSimulation":
        return cls(
            operations=[v1RunnableOperation.from_json(x) for x in obj["operations"]] if obj.get("operations", None) is not None else None,
            occurrences=obj.get("occurrences", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "operations": [x.to_json() for x in self.operations] if self.operations is not None else None,
            "occurrences": self.occurrences if self.occurrences is not None else None,
        }

class v1TrialsSampleResponse:
    def __init__(
        self,
        demotedTrials: "typing.Sequence[int]",
        promotedTrials: "typing.Sequence[int]",
        trials: "typing.Sequence[v1TrialsSampleResponseTrial]",
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

    def to_json(self) -> typing.Any:
        return {
            "trials": [x.to_json() for x in self.trials],
            "promotedTrials": self.promotedTrials,
            "demotedTrials": self.demotedTrials,
        }

class v1TrialsSampleResponseTrial:
    def __init__(
        self,
        data: "typing.Sequence[TrialsSampleResponseDataPoint]",
        hparams: "typing.Dict[str, typing.Any]",
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

    def to_json(self) -> typing.Any:
        return {
            "trialId": self.trialId,
            "hparams": self.hparams,
            "data": [x.to_json() for x in self.data],
        }

class v1TrialsSnapshotResponse:
    def __init__(
        self,
        trials: "typing.Sequence[v1TrialsSnapshotResponseTrial]",
    ):
        self.trials = trials

    @classmethod
    def from_json(cls, obj: Json) -> "v1TrialsSnapshotResponse":
        return cls(
            trials=[v1TrialsSnapshotResponseTrial.from_json(x) for x in obj["trials"]],
        )

    def to_json(self) -> typing.Any:
        return {
            "trials": [x.to_json() for x in self.trials],
        }

class v1TrialsSnapshotResponseTrial:
    def __init__(
        self,
        batchesProcessed: int,
        hparams: "typing.Dict[str, typing.Any]",
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

    def to_json(self) -> typing.Any:
        return {
            "trialId": self.trialId,
            "hparams": self.hparams,
            "metric": dump_float(self.metric),
            "batchesProcessed": self.batchesProcessed,
        }

class v1UpdateJobQueueRequest:
    def __init__(
        self,
        updates: "typing.Sequence[v1QueueControl]",
    ):
        self.updates = updates

    @classmethod
    def from_json(cls, obj: Json) -> "v1UpdateJobQueueRequest":
        return cls(
            updates=[v1QueueControl.from_json(x) for x in obj["updates"]],
        )

    def to_json(self) -> typing.Any:
        return {
            "updates": [x.to_json() for x in self.updates],
        }

class v1User:
    def __init__(
        self,
        active: bool,
        admin: bool,
        username: str,
        agentUserGroup: "typing.Optional[v1AgentUserGroup]" = None,
        displayName: "typing.Optional[str]" = None,
        id: "typing.Optional[int]" = None,
        modifiedAt: "typing.Optional[str]" = None,
    ):
        self.id = id
        self.username = username
        self.admin = admin
        self.active = active
        self.agentUserGroup = agentUserGroup
        self.displayName = displayName
        self.modifiedAt = modifiedAt

    @classmethod
    def from_json(cls, obj: Json) -> "v1User":
        return cls(
            id=obj.get("id", None),
            username=obj["username"],
            admin=obj["admin"],
            active=obj["active"],
            agentUserGroup=v1AgentUserGroup.from_json(obj["agentUserGroup"]) if obj.get("agentUserGroup", None) is not None else None,
            displayName=obj.get("displayName", None),
            modifiedAt=obj.get("modifiedAt", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "id": self.id if self.id is not None else None,
            "username": self.username,
            "admin": self.admin,
            "active": self.active,
            "agentUserGroup": self.agentUserGroup.to_json() if self.agentUserGroup is not None else None,
            "displayName": self.displayName if self.displayName is not None else None,
            "modifiedAt": self.modifiedAt if self.modifiedAt is not None else None,
        }

class v1ValidateAfterOperation:
    def __init__(
        self,
        length: "typing.Optional[str]" = None,
    ):
        self.length = length

    @classmethod
    def from_json(cls, obj: Json) -> "v1ValidateAfterOperation":
        return cls(
            length=obj.get("length", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "length": self.length if self.length is not None else None,
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

    def to_json(self) -> typing.Any:
        return {
            "trialId": self.trialId,
            "endTime": self.endTime,
            "searcherMetric": dump_float(self.searcherMetric),
        }

class v1WorkloadContainer:
    def __init__(
        self,
        checkpoint: "typing.Optional[v1CheckpointWorkload]" = None,
        training: "typing.Optional[v1MetricsWorkload]" = None,
        validation: "typing.Optional[v1MetricsWorkload]" = None,
    ):
        self.training = training
        self.validation = validation
        self.checkpoint = checkpoint

    @classmethod
    def from_json(cls, obj: Json) -> "v1WorkloadContainer":
        return cls(
            training=v1MetricsWorkload.from_json(obj["training"]) if obj.get("training", None) is not None else None,
            validation=v1MetricsWorkload.from_json(obj["validation"]) if obj.get("validation", None) is not None else None,
            checkpoint=v1CheckpointWorkload.from_json(obj["checkpoint"]) if obj.get("checkpoint", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "training": self.training.to_json() if self.training is not None else None,
            "validation": self.validation.to_json() if self.validation is not None else None,
            "checkpoint": self.checkpoint.to_json() if self.checkpoint is not None else None,
        }

def post_AckAllocationPreemptionSignal(
    session: "client.Session",
    *,
    allocationId: str,
    body: "v1AckAllocationPreemptionSignalRequest",
) -> None:
    _params = None
    _resp = session._do_request(
        method="POST",
        path=f"/api/v1/allocations/{allocationId}/signals/ack_preemption",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_AckAllocationPreemptionSignal", _resp)

def post_ActivateExperiment(
    session: "client.Session",
    *,
    id: int,
) -> None:
    _params = None
    _resp = session._do_request(
        method="POST",
        path=f"/api/v1/experiments/{id}/activate",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_ActivateExperiment", _resp)

def post_AllocationAllGather(
    session: "client.Session",
    *,
    allocationId: str,
    body: "v1AllocationAllGatherRequest",
) -> "v1AllocationAllGatherResponse":
    _params = None
    _resp = session._do_request(
        method="POST",
        path=f"/api/v1/allocations/{allocationId}/all_gather",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1AllocationAllGatherResponse.from_json(_resp.json())
    raise APIHttpError("post_AllocationAllGather", _resp)

def post_AllocationPendingPreemptionSignal(
    session: "client.Session",
    *,
    allocationId: str,
    body: "v1AllocationPendingPreemptionSignalRequest",
) -> None:
    _params = None
    _resp = session._do_request(
        method="POST",
        path=f"/api/v1/allocations/{allocationId}/signals/pending_preemption",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_AllocationPendingPreemptionSignal", _resp)

def get_AllocationPreemptionSignal(
    session: "client.Session",
    *,
    allocationId: str,
    timeoutSeconds: "typing.Optional[int]" = None,
) -> "v1AllocationPreemptionSignalResponse":
    _params = {
        "timeoutSeconds": timeoutSeconds,
    }
    _resp = session._do_request(
        method="GET",
        path=f"/api/v1/allocations/{allocationId}/signals/preemption",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1AllocationPreemptionSignalResponse.from_json(_resp.json())
    raise APIHttpError("get_AllocationPreemptionSignal", _resp)

def post_AllocationReady(
    session: "client.Session",
    *,
    allocationId: str,
    body: "v1AllocationReadyRequest",
) -> None:
    _params = None
    _resp = session._do_request(
        method="POST",
        path=f"/api/v1/allocations/{allocationId}/ready",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_AllocationReady", _resp)

def get_AllocationRendezvousInfo(
    session: "client.Session",
    *,
    allocationId: str,
    resourcesId: str,
) -> "v1AllocationRendezvousInfoResponse":
    _params = None
    _resp = session._do_request(
        method="GET",
        path=f"/api/v1/allocations/{allocationId}/resources/{resourcesId}/rendezvous",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1AllocationRendezvousInfoResponse.from_json(_resp.json())
    raise APIHttpError("get_AllocationRendezvousInfo", _resp)

def post_ArchiveExperiment(
    session: "client.Session",
    *,
    id: int,
) -> None:
    _params = None
    _resp = session._do_request(
        method="POST",
        path=f"/api/v1/experiments/{id}/archive",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_ArchiveExperiment", _resp)

def post_ArchiveModel(
    session: "client.Session",
    *,
    modelName: str,
) -> None:
    _params = None
    _resp = session._do_request(
        method="POST",
        path=f"/api/v1/models/{modelName}/archive",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_ArchiveModel", _resp)

def post_CancelExperiment(
    session: "client.Session",
    *,
    id: int,
) -> None:
    _params = None
    _resp = session._do_request(
        method="POST",
        path=f"/api/v1/experiments/{id}/cancel",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_CancelExperiment", _resp)

def post_CompleteTrialSearcherValidation(
    session: "client.Session",
    *,
    body: "v1CompleteValidateAfterOperation",
    trialId: int,
) -> None:
    _params = None
    _resp = session._do_request(
        method="POST",
        path=f"/api/v1/trials/{trialId}/searcher/completed_operation",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_CompleteTrialSearcherValidation", _resp)

def post_ComputeHPImportance(
    session: "client.Session",
    *,
    experimentId: int,
) -> None:
    _params = None
    _resp = session._do_request(
        method="POST",
        path=f"/api/v1/experiments/{experimentId}/hyperparameter-importance",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_ComputeHPImportance", _resp)

def post_CreateExperiment(
    session: "client.Session",
    *,
    body: "v1CreateExperimentRequest",
) -> "v1CreateExperimentResponse":
    _params = None
    _resp = session._do_request(
        method="POST",
        path="/api/v1/experiments",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1CreateExperimentResponse.from_json(_resp.json())
    raise APIHttpError("post_CreateExperiment", _resp)

def get_CurrentUser(
    session: "client.Session",
) -> "v1CurrentUserResponse":
    _params = None
    _resp = session._do_request(
        method="GET",
        path="/api/v1/auth/user",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1CurrentUserResponse.from_json(_resp.json())
    raise APIHttpError("get_CurrentUser", _resp)

def delete_DeleteExperiment(
    session: "client.Session",
    *,
    experimentId: int,
) -> None:
    _params = None
    _resp = session._do_request(
        method="DELETE",
        path=f"/api/v1/experiments/{experimentId}",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("delete_DeleteExperiment", _resp)

def delete_DeleteModel(
    session: "client.Session",
    *,
    modelName: str,
) -> None:
    _params = None
    _resp = session._do_request(
        method="DELETE",
        path=f"/api/v1/models/{modelName}",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("delete_DeleteModel", _resp)

def delete_DeleteModelVersion(
    session: "client.Session",
    *,
    modelName: str,
    modelVersionId: int,
) -> None:
    _params = None
    _resp = session._do_request(
        method="DELETE",
        path=f"/api/v1/models/{modelName}/versions/{modelVersionId}",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("delete_DeleteModelVersion", _resp)

def delete_DeleteTemplate(
    session: "client.Session",
    *,
    templateName: str,
) -> None:
    _params = None
    _resp = session._do_request(
        method="DELETE",
        path=f"/api/v1/templates/{templateName}",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("delete_DeleteTemplate", _resp)

def post_DisableAgent(
    session: "client.Session",
    *,
    agentId: str,
    body: "v1DisableAgentRequest",
) -> "v1DisableAgentResponse":
    _params = None
    _resp = session._do_request(
        method="POST",
        path=f"/api/v1/agents/{agentId}/disable",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1DisableAgentResponse.from_json(_resp.json())
    raise APIHttpError("post_DisableAgent", _resp)

def post_DisableSlot(
    session: "client.Session",
    *,
    agentId: str,
    slotId: str,
) -> "v1DisableSlotResponse":
    _params = None
    _resp = session._do_request(
        method="POST",
        path=f"/api/v1/agents/{agentId}/slots/{slotId}/disable",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1DisableSlotResponse.from_json(_resp.json())
    raise APIHttpError("post_DisableSlot", _resp)

def post_EnableAgent(
    session: "client.Session",
    *,
    agentId: str,
) -> "v1EnableAgentResponse":
    _params = None
    _resp = session._do_request(
        method="POST",
        path=f"/api/v1/agents/{agentId}/enable",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1EnableAgentResponse.from_json(_resp.json())
    raise APIHttpError("post_EnableAgent", _resp)

def post_EnableSlot(
    session: "client.Session",
    *,
    agentId: str,
    slotId: str,
) -> "v1EnableSlotResponse":
    _params = None
    _resp = session._do_request(
        method="POST",
        path=f"/api/v1/agents/{agentId}/slots/{slotId}/enable",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1EnableSlotResponse.from_json(_resp.json())
    raise APIHttpError("post_EnableSlot", _resp)

def get_GetAgent(
    session: "client.Session",
    *,
    agentId: str,
) -> "v1GetAgentResponse":
    _params = None
    _resp = session._do_request(
        method="GET",
        path=f"/api/v1/agents/{agentId}",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1GetAgentResponse.from_json(_resp.json())
    raise APIHttpError("get_GetAgent", _resp)

def get_GetAgents(
    session: "client.Session",
    *,
    label: "typing.Optional[str]" = None,
    limit: "typing.Optional[int]" = None,
    offset: "typing.Optional[int]" = None,
    orderBy: "typing.Optional[v1OrderBy]" = None,
    sortBy: "typing.Optional[v1GetAgentsRequestSortBy]" = None,
) -> "v1GetAgentsResponse":
    _params = {
        "label": label,
        "limit": limit,
        "offset": offset,
        "orderBy": orderBy.value if orderBy is not None else None,
        "sortBy": sortBy.value if sortBy is not None else None,
    }
    _resp = session._do_request(
        method="GET",
        path="/api/v1/agents",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1GetAgentsResponse.from_json(_resp.json())
    raise APIHttpError("get_GetAgents", _resp)

def get_GetBestSearcherValidationMetric(
    session: "client.Session",
    *,
    experimentId: int,
) -> "v1GetBestSearcherValidationMetricResponse":
    _params = None
    _resp = session._do_request(
        method="GET",
        path=f"/api/v1/experiments/{experimentId}/searcher/best_searcher_validation_metric",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1GetBestSearcherValidationMetricResponse.from_json(_resp.json())
    raise APIHttpError("get_GetBestSearcherValidationMetric", _resp)

def get_GetCheckpoint(
    session: "client.Session",
    *,
    checkpointUuid: str,
) -> "v1GetCheckpointResponse":
    _params = None
    _resp = session._do_request(
        method="GET",
        path=f"/api/v1/checkpoints/{checkpointUuid}",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1GetCheckpointResponse.from_json(_resp.json())
    raise APIHttpError("get_GetCheckpoint", _resp)

def get_GetCommand(
    session: "client.Session",
    *,
    commandId: str,
) -> "v1GetCommandResponse":
    _params = None
    _resp = session._do_request(
        method="GET",
        path=f"/api/v1/commands/{commandId}",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1GetCommandResponse.from_json(_resp.json())
    raise APIHttpError("get_GetCommand", _resp)

def get_GetCommands(
    session: "client.Session",
    *,
    limit: "typing.Optional[int]" = None,
    offset: "typing.Optional[int]" = None,
    orderBy: "typing.Optional[v1OrderBy]" = None,
    sortBy: "typing.Optional[v1GetTensorboardsRequestSortBy]" = None,
    userIds: "typing.Optional[typing.Sequence[int]]" = None,
    users: "typing.Optional[typing.Sequence[str]]" = None,
) -> "v1GetCommandsResponse":
    _params = {
        "limit": limit,
        "offset": offset,
        "orderBy": orderBy.value if orderBy is not None else None,
        "sortBy": sortBy.value if sortBy is not None else None,
        "userIds": userIds,
        "users": users,
    }
    _resp = session._do_request(
        method="GET",
        path="/api/v1/commands",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1GetCommandsResponse.from_json(_resp.json())
    raise APIHttpError("get_GetCommands", _resp)

def get_GetCurrentTrialSearcherOperation(
    session: "client.Session",
    *,
    trialId: int,
) -> "v1GetCurrentTrialSearcherOperationResponse":
    _params = None
    _resp = session._do_request(
        method="GET",
        path=f"/api/v1/trials/{trialId}/searcher/operation",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1GetCurrentTrialSearcherOperationResponse.from_json(_resp.json())
    raise APIHttpError("get_GetCurrentTrialSearcherOperation", _resp)

def get_GetExperiment(
    session: "client.Session",
    *,
    experimentId: int,
) -> "v1GetExperimentResponse":
    _params = None
    _resp = session._do_request(
        method="GET",
        path=f"/api/v1/experiments/{experimentId}",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1GetExperimentResponse.from_json(_resp.json())
    raise APIHttpError("get_GetExperiment", _resp)

def get_GetExperimentCheckpoints(
    session: "client.Session",
    *,
    id: int,
    limit: "typing.Optional[int]" = None,
    offset: "typing.Optional[int]" = None,
    orderBy: "typing.Optional[v1OrderBy]" = None,
    sortBy: "typing.Optional[v1GetExperimentCheckpointsRequestSortBy]" = None,
    states: "typing.Optional[typing.Sequence[determinedcheckpointv1State]]" = None,
) -> "v1GetExperimentCheckpointsResponse":
    _params = {
        "limit": limit,
        "offset": offset,
        "orderBy": orderBy.value if orderBy is not None else None,
        "sortBy": sortBy.value if sortBy is not None else None,
        "states": [x.value for x in states] if states is not None else None,
    }
    _resp = session._do_request(
        method="GET",
        path=f"/api/v1/experiments/{id}/checkpoints",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1GetExperimentCheckpointsResponse.from_json(_resp.json())
    raise APIHttpError("get_GetExperimentCheckpoints", _resp)

def get_GetExperimentLabels(
    session: "client.Session",
) -> "v1GetExperimentLabelsResponse":
    _params = None
    _resp = session._do_request(
        method="GET",
        path="/api/v1/experiment/labels",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1GetExperimentLabelsResponse.from_json(_resp.json())
    raise APIHttpError("get_GetExperimentLabels", _resp)

def get_GetExperimentTrials(
    session: "client.Session",
    *,
    experimentId: int,
    limit: "typing.Optional[int]" = None,
    offset: "typing.Optional[int]" = None,
    orderBy: "typing.Optional[v1OrderBy]" = None,
    sortBy: "typing.Optional[v1GetExperimentTrialsRequestSortBy]" = None,
    states: "typing.Optional[typing.Sequence[determinedexperimentv1State]]" = None,
) -> "v1GetExperimentTrialsResponse":
    _params = {
        "limit": limit,
        "offset": offset,
        "orderBy": orderBy.value if orderBy is not None else None,
        "sortBy": sortBy.value if sortBy is not None else None,
        "states": [x.value for x in states] if states is not None else None,
    }
    _resp = session._do_request(
        method="GET",
        path=f"/api/v1/experiments/{experimentId}/trials",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1GetExperimentTrialsResponse.from_json(_resp.json())
    raise APIHttpError("get_GetExperimentTrials", _resp)

def get_GetExperimentValidationHistory(
    session: "client.Session",
    *,
    experimentId: int,
) -> "v1GetExperimentValidationHistoryResponse":
    _params = None
    _resp = session._do_request(
        method="GET",
        path=f"/api/v1/experiments/{experimentId}/validation-history",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1GetExperimentValidationHistoryResponse.from_json(_resp.json())
    raise APIHttpError("get_GetExperimentValidationHistory", _resp)

def get_GetExperiments(
    session: "client.Session",
    *,
    archived: "typing.Optional[bool]" = None,
    description: "typing.Optional[str]" = None,
    labels: "typing.Optional[typing.Sequence[str]]" = None,
    limit: "typing.Optional[int]" = None,
    name: "typing.Optional[str]" = None,
    offset: "typing.Optional[int]" = None,
    orderBy: "typing.Optional[v1OrderBy]" = None,
    sortBy: "typing.Optional[v1GetExperimentsRequestSortBy]" = None,
    states: "typing.Optional[typing.Sequence[determinedexperimentv1State]]" = None,
    userIds: "typing.Optional[typing.Sequence[int]]" = None,
    users: "typing.Optional[typing.Sequence[str]]" = None,
) -> "v1GetExperimentsResponse":
    _params = {
        "archived": str(archived).lower() if archived is not None else None,
        "description": description,
        "labels": labels,
        "limit": limit,
        "name": name,
        "offset": offset,
        "orderBy": orderBy.value if orderBy is not None else None,
        "sortBy": sortBy.value if sortBy is not None else None,
        "states": [x.value for x in states] if states is not None else None,
        "userIds": userIds,
        "users": users,
    }
    _resp = session._do_request(
        method="GET",
        path="/api/v1/experiments",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1GetExperimentsResponse.from_json(_resp.json())
    raise APIHttpError("get_GetExperiments", _resp)

def get_GetJobQueueStats(
    session: "client.Session",
    *,
    resourcePools: "typing.Optional[typing.Sequence[str]]" = None,
) -> "v1GetJobQueueStatsResponse":
    _params = {
        "resourcePools": resourcePools,
    }
    _resp = session._do_request(
        method="GET",
        path="/api/v1/job-queues/stats",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1GetJobQueueStatsResponse.from_json(_resp.json())
    raise APIHttpError("get_GetJobQueueStats", _resp)

def get_GetJobs(
    session: "client.Session",
    *,
    orderBy: "typing.Optional[v1OrderBy]" = None,
    pagination_limit: "typing.Optional[int]" = None,
    pagination_offset: "typing.Optional[int]" = None,
    resourcePool: "typing.Optional[str]" = None,
) -> "v1GetJobsResponse":
    _params = {
        "orderBy": orderBy.value if orderBy is not None else None,
        "pagination.limit": pagination_limit,
        "pagination.offset": pagination_offset,
        "resourcePool": resourcePool,
    }
    _resp = session._do_request(
        method="GET",
        path="/api/v1/job-queues",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1GetJobsResponse.from_json(_resp.json())
    raise APIHttpError("get_GetJobs", _resp)

def get_GetMaster(
    session: "client.Session",
) -> "v1GetMasterResponse":
    _params = None
    _resp = session._do_request(
        method="GET",
        path="/api/v1/master",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1GetMasterResponse.from_json(_resp.json())
    raise APIHttpError("get_GetMaster", _resp)

def get_GetMasterConfig(
    session: "client.Session",
) -> "v1GetMasterConfigResponse":
    _params = None
    _resp = session._do_request(
        method="GET",
        path="/api/v1/master/config",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1GetMasterConfigResponse.from_json(_resp.json())
    raise APIHttpError("get_GetMasterConfig", _resp)

def get_GetModel(
    session: "client.Session",
    *,
    modelName: str,
) -> "v1GetModelResponse":
    _params = None
    _resp = session._do_request(
        method="GET",
        path=f"/api/v1/models/{modelName}",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1GetModelResponse.from_json(_resp.json())
    raise APIHttpError("get_GetModel", _resp)

def get_GetModelDef(
    session: "client.Session",
    *,
    experimentId: int,
) -> "v1GetModelDefResponse":
    _params = None
    _resp = session._do_request(
        method="GET",
        path=f"/api/v1/experiments/{experimentId}/model_def",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1GetModelDefResponse.from_json(_resp.json())
    raise APIHttpError("get_GetModelDef", _resp)

def get_GetModelLabels(
    session: "client.Session",
) -> "v1GetModelLabelsResponse":
    _params = None
    _resp = session._do_request(
        method="GET",
        path="/api/v1/model/labels",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1GetModelLabelsResponse.from_json(_resp.json())
    raise APIHttpError("get_GetModelLabels", _resp)

def get_GetModelVersion(
    session: "client.Session",
    *,
    modelName: str,
    modelVersion: int,
) -> "v1GetModelVersionResponse":
    _params = None
    _resp = session._do_request(
        method="GET",
        path=f"/api/v1/models/{modelName}/versions/{modelVersion}",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1GetModelVersionResponse.from_json(_resp.json())
    raise APIHttpError("get_GetModelVersion", _resp)

def get_GetModelVersions(
    session: "client.Session",
    *,
    modelName: str,
    limit: "typing.Optional[int]" = None,
    offset: "typing.Optional[int]" = None,
    orderBy: "typing.Optional[v1OrderBy]" = None,
    sortBy: "typing.Optional[v1GetModelVersionsRequestSortBy]" = None,
) -> "v1GetModelVersionsResponse":
    _params = {
        "limit": limit,
        "offset": offset,
        "orderBy": orderBy.value if orderBy is not None else None,
        "sortBy": sortBy.value if sortBy is not None else None,
    }
    _resp = session._do_request(
        method="GET",
        path=f"/api/v1/models/{modelName}/versions",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1GetModelVersionsResponse.from_json(_resp.json())
    raise APIHttpError("get_GetModelVersions", _resp)

def get_GetModels(
    session: "client.Session",
    *,
    archived: "typing.Optional[bool]" = None,
    description: "typing.Optional[str]" = None,
    id: "typing.Optional[int]" = None,
    labels: "typing.Optional[typing.Sequence[str]]" = None,
    limit: "typing.Optional[int]" = None,
    name: "typing.Optional[str]" = None,
    offset: "typing.Optional[int]" = None,
    orderBy: "typing.Optional[v1OrderBy]" = None,
    sortBy: "typing.Optional[v1GetModelsRequestSortBy]" = None,
    userIds: "typing.Optional[typing.Sequence[int]]" = None,
    users: "typing.Optional[typing.Sequence[str]]" = None,
) -> "v1GetModelsResponse":
    _params = {
        "archived": str(archived).lower() if archived is not None else None,
        "description": description,
        "id": id,
        "labels": labels,
        "limit": limit,
        "name": name,
        "offset": offset,
        "orderBy": orderBy.value if orderBy is not None else None,
        "sortBy": sortBy.value if sortBy is not None else None,
        "userIds": userIds,
        "users": users,
    }
    _resp = session._do_request(
        method="GET",
        path="/api/v1/models",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1GetModelsResponse.from_json(_resp.json())
    raise APIHttpError("get_GetModels", _resp)

def get_GetNotebook(
    session: "client.Session",
    *,
    notebookId: str,
) -> "v1GetNotebookResponse":
    _params = None
    _resp = session._do_request(
        method="GET",
        path=f"/api/v1/notebooks/{notebookId}",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1GetNotebookResponse.from_json(_resp.json())
    raise APIHttpError("get_GetNotebook", _resp)

def get_GetNotebooks(
    session: "client.Session",
    *,
    limit: "typing.Optional[int]" = None,
    offset: "typing.Optional[int]" = None,
    orderBy: "typing.Optional[v1OrderBy]" = None,
    sortBy: "typing.Optional[v1GetTensorboardsRequestSortBy]" = None,
    userIds: "typing.Optional[typing.Sequence[int]]" = None,
    users: "typing.Optional[typing.Sequence[str]]" = None,
) -> "v1GetNotebooksResponse":
    _params = {
        "limit": limit,
        "offset": offset,
        "orderBy": orderBy.value if orderBy is not None else None,
        "sortBy": sortBy.value if sortBy is not None else None,
        "userIds": userIds,
        "users": users,
    }
    _resp = session._do_request(
        method="GET",
        path="/api/v1/notebooks",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1GetNotebooksResponse.from_json(_resp.json())
    raise APIHttpError("get_GetNotebooks", _resp)

def get_GetResourcePools(
    session: "client.Session",
    *,
    limit: "typing.Optional[int]" = None,
    offset: "typing.Optional[int]" = None,
) -> "v1GetResourcePoolsResponse":
    _params = {
        "limit": limit,
        "offset": offset,
    }
    _resp = session._do_request(
        method="GET",
        path="/api/v1/resource-pools",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1GetResourcePoolsResponse.from_json(_resp.json())
    raise APIHttpError("get_GetResourcePools", _resp)

def get_GetShell(
    session: "client.Session",
    *,
    shellId: str,
) -> "v1GetShellResponse":
    _params = None
    _resp = session._do_request(
        method="GET",
        path=f"/api/v1/shells/{shellId}",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1GetShellResponse.from_json(_resp.json())
    raise APIHttpError("get_GetShell", _resp)

def get_GetShells(
    session: "client.Session",
    *,
    limit: "typing.Optional[int]" = None,
    offset: "typing.Optional[int]" = None,
    orderBy: "typing.Optional[v1OrderBy]" = None,
    sortBy: "typing.Optional[v1GetTensorboardsRequestSortBy]" = None,
    userIds: "typing.Optional[typing.Sequence[int]]" = None,
    users: "typing.Optional[typing.Sequence[str]]" = None,
) -> "v1GetShellsResponse":
    _params = {
        "limit": limit,
        "offset": offset,
        "orderBy": orderBy.value if orderBy is not None else None,
        "sortBy": sortBy.value if sortBy is not None else None,
        "userIds": userIds,
        "users": users,
    }
    _resp = session._do_request(
        method="GET",
        path="/api/v1/shells",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1GetShellsResponse.from_json(_resp.json())
    raise APIHttpError("get_GetShells", _resp)

def get_GetSlot(
    session: "client.Session",
    *,
    agentId: str,
    slotId: str,
) -> "v1GetSlotResponse":
    _params = None
    _resp = session._do_request(
        method="GET",
        path=f"/api/v1/agents/{agentId}/slots/{slotId}",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1GetSlotResponse.from_json(_resp.json())
    raise APIHttpError("get_GetSlot", _resp)

def get_GetSlots(
    session: "client.Session",
    *,
    agentId: str,
) -> "v1GetSlotsResponse":
    _params = None
    _resp = session._do_request(
        method="GET",
        path=f"/api/v1/agents/{agentId}/slots",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1GetSlotsResponse.from_json(_resp.json())
    raise APIHttpError("get_GetSlots", _resp)

def get_GetTask(
    session: "client.Session",
    *,
    taskId: str,
) -> "v1GetTaskResponse":
    _params = None
    _resp = session._do_request(
        method="GET",
        path=f"/api/v1/tasks/{taskId}",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1GetTaskResponse.from_json(_resp.json())
    raise APIHttpError("get_GetTask", _resp)

def get_GetTelemetry(
    session: "client.Session",
) -> "v1GetTelemetryResponse":
    _params = None
    _resp = session._do_request(
        method="GET",
        path="/api/v1/master/telemetry",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1GetTelemetryResponse.from_json(_resp.json())
    raise APIHttpError("get_GetTelemetry", _resp)

def get_GetTemplate(
    session: "client.Session",
    *,
    templateName: str,
) -> "v1GetTemplateResponse":
    _params = None
    _resp = session._do_request(
        method="GET",
        path=f"/api/v1/templates/{templateName}",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1GetTemplateResponse.from_json(_resp.json())
    raise APIHttpError("get_GetTemplate", _resp)

def get_GetTemplates(
    session: "client.Session",
    *,
    limit: "typing.Optional[int]" = None,
    name: "typing.Optional[str]" = None,
    offset: "typing.Optional[int]" = None,
    orderBy: "typing.Optional[v1OrderBy]" = None,
    sortBy: "typing.Optional[v1GetTemplatesRequestSortBy]" = None,
) -> "v1GetTemplatesResponse":
    _params = {
        "limit": limit,
        "name": name,
        "offset": offset,
        "orderBy": orderBy.value if orderBy is not None else None,
        "sortBy": sortBy.value if sortBy is not None else None,
    }
    _resp = session._do_request(
        method="GET",
        path="/api/v1/templates",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1GetTemplatesResponse.from_json(_resp.json())
    raise APIHttpError("get_GetTemplates", _resp)

def get_GetTensorboard(
    session: "client.Session",
    *,
    tensorboardId: str,
) -> "v1GetTensorboardResponse":
    _params = None
    _resp = session._do_request(
        method="GET",
        path=f"/api/v1/tensorboards/{tensorboardId}",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1GetTensorboardResponse.from_json(_resp.json())
    raise APIHttpError("get_GetTensorboard", _resp)

def get_GetTensorboards(
    session: "client.Session",
    *,
    limit: "typing.Optional[int]" = None,
    offset: "typing.Optional[int]" = None,
    orderBy: "typing.Optional[v1OrderBy]" = None,
    sortBy: "typing.Optional[v1GetTensorboardsRequestSortBy]" = None,
    userIds: "typing.Optional[typing.Sequence[int]]" = None,
    users: "typing.Optional[typing.Sequence[str]]" = None,
) -> "v1GetTensorboardsResponse":
    _params = {
        "limit": limit,
        "offset": offset,
        "orderBy": orderBy.value if orderBy is not None else None,
        "sortBy": sortBy.value if sortBy is not None else None,
        "userIds": userIds,
        "users": users,
    }
    _resp = session._do_request(
        method="GET",
        path="/api/v1/tensorboards",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1GetTensorboardsResponse.from_json(_resp.json())
    raise APIHttpError("get_GetTensorboards", _resp)

def get_GetTrial(
    session: "client.Session",
    *,
    trialId: int,
) -> "v1GetTrialResponse":
    _params = None
    _resp = session._do_request(
        method="GET",
        path=f"/api/v1/trials/{trialId}",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1GetTrialResponse.from_json(_resp.json())
    raise APIHttpError("get_GetTrial", _resp)

def get_GetTrialCheckpoints(
    session: "client.Session",
    *,
    id: int,
    limit: "typing.Optional[int]" = None,
    offset: "typing.Optional[int]" = None,
    orderBy: "typing.Optional[v1OrderBy]" = None,
    sortBy: "typing.Optional[v1GetTrialCheckpointsRequestSortBy]" = None,
    states: "typing.Optional[typing.Sequence[determinedcheckpointv1State]]" = None,
) -> "v1GetTrialCheckpointsResponse":
    _params = {
        "limit": limit,
        "offset": offset,
        "orderBy": orderBy.value if orderBy is not None else None,
        "sortBy": sortBy.value if sortBy is not None else None,
        "states": [x.value for x in states] if states is not None else None,
    }
    _resp = session._do_request(
        method="GET",
        path=f"/api/v1/trials/{id}/checkpoints",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1GetTrialCheckpointsResponse.from_json(_resp.json())
    raise APIHttpError("get_GetTrialCheckpoints", _resp)

def get_GetTrialWorkloads(
    session: "client.Session",
    *,
    trialId: int,
    limit: "typing.Optional[int]" = None,
    offset: "typing.Optional[int]" = None,
    orderBy: "typing.Optional[v1OrderBy]" = None,
) -> "v1GetTrialWorkloadsResponse":
    _params = {
        "limit": limit,
        "offset": offset,
        "orderBy": orderBy.value if orderBy is not None else None,
    }
    _resp = session._do_request(
        method="GET",
        path=f"/api/v1/trials/{trialId}/workloads",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1GetTrialWorkloadsResponse.from_json(_resp.json())
    raise APIHttpError("get_GetTrialWorkloads", _resp)

def get_GetUser(
    session: "client.Session",
    *,
    userId: int,
) -> "v1GetUserResponse":
    _params = None
    _resp = session._do_request(
        method="GET",
        path=f"/api/v1/users/{userId}",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1GetUserResponse.from_json(_resp.json())
    raise APIHttpError("get_GetUser", _resp)

def get_GetUsers(
    session: "client.Session",
) -> "v1GetUsersResponse":
    _params = None
    _resp = session._do_request(
        method="GET",
        path="/api/v1/users",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1GetUsersResponse.from_json(_resp.json())
    raise APIHttpError("get_GetUsers", _resp)

def put_IdleNotebook(
    session: "client.Session",
    *,
    body: "v1IdleNotebookRequest",
    notebookId: str,
) -> None:
    _params = None
    _resp = session._do_request(
        method="PUT",
        path=f"/api/v1/notebooks/{notebookId}/report_idle",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("put_IdleNotebook", _resp)

def post_KillCommand(
    session: "client.Session",
    *,
    commandId: str,
) -> "v1KillCommandResponse":
    _params = None
    _resp = session._do_request(
        method="POST",
        path=f"/api/v1/commands/{commandId}/kill",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1KillCommandResponse.from_json(_resp.json())
    raise APIHttpError("post_KillCommand", _resp)

def post_KillExperiment(
    session: "client.Session",
    *,
    id: int,
) -> None:
    _params = None
    _resp = session._do_request(
        method="POST",
        path=f"/api/v1/experiments/{id}/kill",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_KillExperiment", _resp)

def post_KillNotebook(
    session: "client.Session",
    *,
    notebookId: str,
) -> "v1KillNotebookResponse":
    _params = None
    _resp = session._do_request(
        method="POST",
        path=f"/api/v1/notebooks/{notebookId}/kill",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1KillNotebookResponse.from_json(_resp.json())
    raise APIHttpError("post_KillNotebook", _resp)

def post_KillShell(
    session: "client.Session",
    *,
    shellId: str,
) -> "v1KillShellResponse":
    _params = None
    _resp = session._do_request(
        method="POST",
        path=f"/api/v1/shells/{shellId}/kill",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1KillShellResponse.from_json(_resp.json())
    raise APIHttpError("post_KillShell", _resp)

def post_KillTensorboard(
    session: "client.Session",
    *,
    tensorboardId: str,
) -> "v1KillTensorboardResponse":
    _params = None
    _resp = session._do_request(
        method="POST",
        path=f"/api/v1/tensorboards/{tensorboardId}/kill",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1KillTensorboardResponse.from_json(_resp.json())
    raise APIHttpError("post_KillTensorboard", _resp)

def post_KillTrial(
    session: "client.Session",
    *,
    id: int,
) -> None:
    _params = None
    _resp = session._do_request(
        method="POST",
        path=f"/api/v1/trials/{id}/kill",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_KillTrial", _resp)

def post_LaunchCommand(
    session: "client.Session",
    *,
    body: "v1LaunchCommandRequest",
) -> "v1LaunchCommandResponse":
    _params = None
    _resp = session._do_request(
        method="POST",
        path="/api/v1/commands",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1LaunchCommandResponse.from_json(_resp.json())
    raise APIHttpError("post_LaunchCommand", _resp)

def post_LaunchNotebook(
    session: "client.Session",
    *,
    body: "v1LaunchNotebookRequest",
) -> "v1LaunchNotebookResponse":
    _params = None
    _resp = session._do_request(
        method="POST",
        path="/api/v1/notebooks",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1LaunchNotebookResponse.from_json(_resp.json())
    raise APIHttpError("post_LaunchNotebook", _resp)

def post_LaunchShell(
    session: "client.Session",
    *,
    body: "v1LaunchShellRequest",
) -> "v1LaunchShellResponse":
    _params = None
    _resp = session._do_request(
        method="POST",
        path="/api/v1/shells",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1LaunchShellResponse.from_json(_resp.json())
    raise APIHttpError("post_LaunchShell", _resp)

def post_LaunchTensorboard(
    session: "client.Session",
    *,
    body: "v1LaunchTensorboardRequest",
) -> "v1LaunchTensorboardResponse":
    _params = None
    _resp = session._do_request(
        method="POST",
        path="/api/v1/tensorboards",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1LaunchTensorboardResponse.from_json(_resp.json())
    raise APIHttpError("post_LaunchTensorboard", _resp)

def post_Login(
    session: "client.Session",
    *,
    body: "v1LoginRequest",
) -> "v1LoginResponse":
    _params = None
    _resp = session._do_request(
        method="POST",
        path="/api/v1/auth/login",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1LoginResponse.from_json(_resp.json())
    raise APIHttpError("post_Login", _resp)

def post_Logout(
    session: "client.Session",
) -> None:
    _params = None
    _resp = session._do_request(
        method="POST",
        path="/api/v1/auth/logout",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_Logout", _resp)

def post_MarkAllocationResourcesDaemon(
    session: "client.Session",
    *,
    allocationId: str,
    body: "v1MarkAllocationResourcesDaemonRequest",
    resourcesId: str,
) -> None:
    _params = None
    _resp = session._do_request(
        method="POST",
        path=f"/api/v1/allocations/{allocationId}/resources/{resourcesId}/daemon",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_MarkAllocationResourcesDaemon", _resp)

def patch_PatchExperiment(
    session: "client.Session",
    *,
    body: "v1PatchExperiment",
    experiment_id: int,
) -> "v1PatchExperimentResponse":
    _params = None
    _resp = session._do_request(
        method="PATCH",
        path=f"/api/v1/experiments/{experiment_id}",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1PatchExperimentResponse.from_json(_resp.json())
    raise APIHttpError("patch_PatchExperiment", _resp)

def patch_PatchModel(
    session: "client.Session",
    *,
    body: "v1PatchModel",
    modelName: str,
) -> "v1PatchModelResponse":
    _params = None
    _resp = session._do_request(
        method="PATCH",
        path=f"/api/v1/models/{modelName}",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1PatchModelResponse.from_json(_resp.json())
    raise APIHttpError("patch_PatchModel", _resp)

def patch_PatchModelVersion(
    session: "client.Session",
    *,
    body: "v1PatchModelVersion",
    modelName: str,
    modelVersionId: int,
) -> "v1PatchModelVersionResponse":
    _params = None
    _resp = session._do_request(
        method="PATCH",
        path=f"/api/v1/models/{modelName}/versions/{modelVersionId}",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1PatchModelVersionResponse.from_json(_resp.json())
    raise APIHttpError("patch_PatchModelVersion", _resp)

def patch_PatchUser(
    session: "client.Session",
    *,
    body: "v1PatchUser",
    userId: int,
) -> "v1PatchUserResponse":
    _params = None
    _resp = session._do_request(
        method="PATCH",
        path=f"/api/v1/users/{userId}",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1PatchUserResponse.from_json(_resp.json())
    raise APIHttpError("patch_PatchUser", _resp)

def post_PauseExperiment(
    session: "client.Session",
    *,
    id: int,
) -> None:
    _params = None
    _resp = session._do_request(
        method="POST",
        path=f"/api/v1/experiments/{id}/pause",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_PauseExperiment", _resp)

def post_PostAllocationProxyAddress(
    session: "client.Session",
    *,
    allocationId: str,
    body: "v1PostAllocationProxyAddressRequest",
) -> None:
    _params = None
    _resp = session._do_request(
        method="POST",
        path=f"/api/v1/allocations/{allocationId}/proxy_address",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_PostAllocationProxyAddress", _resp)

def post_PostCheckpointMetadata(
    session: "client.Session",
    *,
    body: "v1PostCheckpointMetadataRequest",
    checkpoint_uuid: str,
) -> "v1PostCheckpointMetadataResponse":
    _params = None
    _resp = session._do_request(
        method="POST",
        path=f"/api/v1/checkpoints/{checkpoint_uuid}/metadata",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1PostCheckpointMetadataResponse.from_json(_resp.json())
    raise APIHttpError("post_PostCheckpointMetadata", _resp)

def post_PostModel(
    session: "client.Session",
    *,
    body: "v1PostModelRequest",
) -> "v1PostModelResponse":
    _params = None
    _resp = session._do_request(
        method="POST",
        path="/api/v1/models",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1PostModelResponse.from_json(_resp.json())
    raise APIHttpError("post_PostModel", _resp)

def post_PostModelVersion(
    session: "client.Session",
    *,
    body: "v1PostModelVersionRequest",
    modelName: str,
) -> "v1PostModelVersionResponse":
    _params = None
    _resp = session._do_request(
        method="POST",
        path=f"/api/v1/models/{modelName}/versions",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1PostModelVersionResponse.from_json(_resp.json())
    raise APIHttpError("post_PostModelVersion", _resp)

def post_PostTrialProfilerMetricsBatch(
    session: "client.Session",
    *,
    body: "v1PostTrialProfilerMetricsBatchRequest",
) -> None:
    _params = None
    _resp = session._do_request(
        method="POST",
        path="/api/v1/trials/profiler/metrics",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_PostTrialProfilerMetricsBatch", _resp)

def post_PostTrialRunnerMetadata(
    session: "client.Session",
    *,
    body: "v1TrialRunnerMetadata",
    trialId: int,
) -> None:
    _params = None
    _resp = session._do_request(
        method="POST",
        path=f"/api/v1/trials/{trialId}/runner/metadata",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_PostTrialRunnerMetadata", _resp)

def post_PostUser(
    session: "client.Session",
    *,
    body: "v1PostUserRequest",
) -> "v1PostUserResponse":
    _params = None
    _resp = session._do_request(
        method="POST",
        path="/api/v1/users",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1PostUserResponse.from_json(_resp.json())
    raise APIHttpError("post_PostUser", _resp)

def post_PreviewHPSearch(
    session: "client.Session",
    *,
    body: "v1PreviewHPSearchRequest",
) -> "v1PreviewHPSearchResponse":
    _params = None
    _resp = session._do_request(
        method="POST",
        path="/api/v1/preview-hp-search",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1PreviewHPSearchResponse.from_json(_resp.json())
    raise APIHttpError("post_PreviewHPSearch", _resp)

def put_PutTemplate(
    session: "client.Session",
    *,
    body: "v1Template",
    template_name: str,
) -> "v1PutTemplateResponse":
    _params = None
    _resp = session._do_request(
        method="PUT",
        path=f"/api/v1/templates/{template_name}",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1PutTemplateResponse.from_json(_resp.json())
    raise APIHttpError("put_PutTemplate", _resp)

def post_ReportCheckpoint(
    session: "client.Session",
    *,
    body: "v1Checkpoint",
) -> None:
    _params = None
    _resp = session._do_request(
        method="POST",
        path="/api/v1/checkpoints",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_ReportCheckpoint", _resp)

def post_ReportTrialProgress(
    session: "client.Session",
    *,
    body: float,
    trialId: int,
) -> None:
    _params = None
    _resp = session._do_request(
        method="POST",
        path=f"/api/v1/trials/{trialId}/progress",
        params=_params,
        json=dump_float(body),
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_ReportTrialProgress", _resp)

def post_ReportTrialSearcherEarlyExit(
    session: "client.Session",
    *,
    body: "v1TrialEarlyExit",
    trialId: int,
) -> None:
    _params = None
    _resp = session._do_request(
        method="POST",
        path=f"/api/v1/trials/{trialId}/early_exit",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_ReportTrialSearcherEarlyExit", _resp)

def post_ReportTrialTrainingMetrics(
    session: "client.Session",
    *,
    body: "v1TrialMetrics",
    trainingMetrics_trialId: int,
) -> None:
    _params = None
    _resp = session._do_request(
        method="POST",
        path=f"/api/v1/trials/{trainingMetrics_trialId}/training_metrics",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_ReportTrialTrainingMetrics", _resp)

def post_ReportTrialValidationMetrics(
    session: "client.Session",
    *,
    body: "v1TrialMetrics",
    validationMetrics_trialId: int,
) -> None:
    _params = None
    _resp = session._do_request(
        method="POST",
        path=f"/api/v1/trials/{validationMetrics_trialId}/validation_metrics",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_ReportTrialValidationMetrics", _resp)

def get_ResourceAllocationAggregated(
    session: "client.Session",
    *,
    endDate: "typing.Optional[str]" = None,
    period: "typing.Optional[v1ResourceAllocationAggregationPeriod]" = None,
    startDate: "typing.Optional[str]" = None,
) -> "v1ResourceAllocationAggregatedResponse":
    _params = {
        "endDate": endDate,
        "period": period.value if period is not None else None,
        "startDate": startDate,
    }
    _resp = session._do_request(
        method="GET",
        path="/api/v1/resources/allocation/aggregated",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1ResourceAllocationAggregatedResponse.from_json(_resp.json())
    raise APIHttpError("get_ResourceAllocationAggregated", _resp)

def get_ResourceAllocationRaw(
    session: "client.Session",
    *,
    timestampAfter: "typing.Optional[str]" = None,
    timestampBefore: "typing.Optional[str]" = None,
) -> "v1ResourceAllocationRawResponse":
    _params = {
        "timestampAfter": timestampAfter,
        "timestampBefore": timestampBefore,
    }
    _resp = session._do_request(
        method="GET",
        path="/api/v1/resources/allocation/raw",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1ResourceAllocationRawResponse.from_json(_resp.json())
    raise APIHttpError("get_ResourceAllocationRaw", _resp)

def post_SetCommandPriority(
    session: "client.Session",
    *,
    body: "v1SetCommandPriorityRequest",
    commandId: str,
) -> "v1SetCommandPriorityResponse":
    _params = None
    _resp = session._do_request(
        method="POST",
        path=f"/api/v1/commands/{commandId}/set_priority",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1SetCommandPriorityResponse.from_json(_resp.json())
    raise APIHttpError("post_SetCommandPriority", _resp)

def post_SetNotebookPriority(
    session: "client.Session",
    *,
    body: "v1SetNotebookPriorityRequest",
    notebookId: str,
) -> "v1SetNotebookPriorityResponse":
    _params = None
    _resp = session._do_request(
        method="POST",
        path=f"/api/v1/notebooks/{notebookId}/set_priority",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1SetNotebookPriorityResponse.from_json(_resp.json())
    raise APIHttpError("post_SetNotebookPriority", _resp)

def post_SetShellPriority(
    session: "client.Session",
    *,
    body: "v1SetShellPriorityRequest",
    shellId: str,
) -> "v1SetShellPriorityResponse":
    _params = None
    _resp = session._do_request(
        method="POST",
        path=f"/api/v1/shells/{shellId}/set_priority",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1SetShellPriorityResponse.from_json(_resp.json())
    raise APIHttpError("post_SetShellPriority", _resp)

def post_SetTensorboardPriority(
    session: "client.Session",
    *,
    body: "v1SetTensorboardPriorityRequest",
    tensorboardId: str,
) -> "v1SetTensorboardPriorityResponse":
    _params = None
    _resp = session._do_request(
        method="POST",
        path=f"/api/v1/tensorboards/{tensorboardId}/set_priority",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1SetTensorboardPriorityResponse.from_json(_resp.json())
    raise APIHttpError("post_SetTensorboardPriority", _resp)

def post_SetUserPassword(
    session: "client.Session",
    *,
    body: str,
    userId: int,
) -> "v1SetUserPasswordResponse":
    _params = None
    _resp = session._do_request(
        method="POST",
        path=f"/api/v1/users/{userId}/password",
        params=_params,
        json=body,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return v1SetUserPasswordResponse.from_json(_resp.json())
    raise APIHttpError("post_SetUserPassword", _resp)

def post_UnarchiveExperiment(
    session: "client.Session",
    *,
    id: int,
) -> None:
    _params = None
    _resp = session._do_request(
        method="POST",
        path=f"/api/v1/experiments/{id}/unarchive",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_UnarchiveExperiment", _resp)

def post_UnarchiveModel(
    session: "client.Session",
    *,
    modelName: str,
) -> None:
    _params = None
    _resp = session._do_request(
        method="POST",
        path=f"/api/v1/models/{modelName}/unarchive",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_UnarchiveModel", _resp)

def post_UpdateJobQueue(
    session: "client.Session",
    *,
    body: "v1UpdateJobQueueRequest",
) -> None:
    _params = None
    _resp = session._do_request(
        method="POST",
        path="/api/v1/job-queues",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_UpdateJobQueue", _resp)
