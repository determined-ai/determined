# The contents of this file are programmatically generated.
import enum
import json
import math
import typing

import requests

if typing.TYPE_CHECKING:
    from determined.common import api

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
            f"API Error: {operation_name} failed: {response.reason}."
        )

    def __str__(self) -> str:
        return self.message


class APIHttpStreamError(APIHttpError):
    # APIHttpStreamError is used if an streaming API request fails mid-stream.
    def __init__(self, operation_name: str, error: "runtimeStreamError") -> None:
        self.operation_name = operation_name
        self.error = error
        self.message = (
            f"Stream Error during {operation_name}: {error.message}"
        )

    def __str__(self) -> str:
        return self.message


class ExpCompareTrialsSampleResponseExpTrial:
    def __init__(
        self,
        *,
        data: "typing.Sequence[v1DataPoint]",
        experimentId: int,
        hparams: "typing.Dict[str, typing.Any]",
        trialId: int,
    ):
        self.trialId = trialId
        self.hparams = hparams
        self.data = data
        self.experimentId = experimentId

    @classmethod
    def from_json(cls, obj: Json) -> "ExpCompareTrialsSampleResponseExpTrial":
        return cls(
            trialId=obj["trialId"],
            hparams=obj["hparams"],
            data=[v1DataPoint.from_json(x) for x in obj["data"]],
            experimentId=obj["experimentId"],
        )

    def to_json(self) -> typing.Any:
        return {
            "trialId": self.trialId,
            "hparams": self.hparams,
            "data": [x.to_json() for x in self.data],
            "experimentId": self.experimentId,
        }

class GetHPImportanceResponseMetricHPImportance:
    def __init__(
        self,
        *,
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

class GetTrialWorkloadsRequestFilterOption(enum.Enum):
    FILTER_OPTION_UNSPECIFIED = "FILTER_OPTION_UNSPECIFIED"
    FILTER_OPTION_CHECKPOINT = "FILTER_OPTION_CHECKPOINT"
    FILTER_OPTION_VALIDATION = "FILTER_OPTION_VALIDATION"
    FILTER_OPTION_CHECKPOINT_OR_VALIDATION = "FILTER_OPTION_CHECKPOINT_OR_VALIDATION"

class TrialFiltersRankWithinExp:
    def __init__(
        self,
        *,
        rank: "typing.Optional[int]" = None,
        sorter: "typing.Optional[v1TrialSorter]" = None,
    ):
        self.sorter = sorter
        self.rank = rank

    @classmethod
    def from_json(cls, obj: Json) -> "TrialFiltersRankWithinExp":
        return cls(
            sorter=v1TrialSorter.from_json(obj["sorter"]) if obj.get("sorter", None) is not None else None,
            rank=obj.get("rank", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "sorter": self.sorter.to_json() if self.sorter is not None else None,
            "rank": self.rank if self.rank is not None else None,
        }

class TrialProfilerMetricLabelsProfilerMetricType(enum.Enum):
    PROFILER_METRIC_TYPE_UNSPECIFIED = "PROFILER_METRIC_TYPE_UNSPECIFIED"
    PROFILER_METRIC_TYPE_SYSTEM = "PROFILER_METRIC_TYPE_SYSTEM"
    PROFILER_METRIC_TYPE_TIMING = "PROFILER_METRIC_TYPE_TIMING"
    PROFILER_METRIC_TYPE_MISC = "PROFILER_METRIC_TYPE_MISC"

class TrialSorterNamespace(enum.Enum):
    NAMESPACE_UNSPECIFIED = "NAMESPACE_UNSPECIFIED"
    NAMESPACE_HPARAMS = "NAMESPACE_HPARAMS"
    NAMESPACE_TRAINING_METRICS = "NAMESPACE_TRAINING_METRICS"
    NAMESPACE_VALIDATION_METRICS = "NAMESPACE_VALIDATION_METRICS"

class UpdateTrialTagsRequestIds:
    def __init__(
        self,
        *,
        ids: "typing.Optional[typing.Sequence[int]]" = None,
    ):
        self.ids = ids

    @classmethod
    def from_json(cls, obj: Json) -> "UpdateTrialTagsRequestIds":
        return cls(
            ids=obj.get("ids", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "ids": self.ids if self.ids is not None else None,
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
    STATE_QUEUED = "STATE_QUEUED"
    STATE_PULLING = "STATE_PULLING"
    STATE_STARTING = "STATE_STARTING"
    STATE_RUNNING = "STATE_RUNNING"

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
    TYPE_CHECKPOINT_GC = "TYPE_CHECKPOINT_GC"

class determinedtaskv1State(enum.Enum):
    STATE_UNSPECIFIED = "STATE_UNSPECIFIED"
    STATE_PULLING = "STATE_PULLING"
    STATE_STARTING = "STATE_STARTING"
    STATE_RUNNING = "STATE_RUNNING"
    STATE_TERMINATED = "STATE_TERMINATED"
    STATE_TERMINATING = "STATE_TERMINATING"
    STATE_WAITING = "STATE_WAITING"
    STATE_QUEUED = "STATE_QUEUED"

class determinedtrialv1State(enum.Enum):
    STATE_UNSPECIFIED = "STATE_UNSPECIFIED"
    STATE_ACTIVE = "STATE_ACTIVE"
    STATE_PAUSED = "STATE_PAUSED"
    STATE_STOPPING_CANCELED = "STATE_STOPPING_CANCELED"
    STATE_STOPPING_KILLED = "STATE_STOPPING_KILLED"
    STATE_STOPPING_COMPLETED = "STATE_STOPPING_COMPLETED"
    STATE_STOPPING_ERROR = "STATE_STOPPING_ERROR"
    STATE_CANCELED = "STATE_CANCELED"
    STATE_COMPLETED = "STATE_COMPLETED"
    STATE_ERROR = "STATE_ERROR"

class protobufAny:
    def __init__(
        self,
        *,
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
        *,
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
        *,
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
        *,
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
        *,
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

class v1AddProjectNoteResponse:
    def __init__(
        self,
        *,
        notes: "typing.Sequence[v1Note]",
    ):
        self.notes = notes

    @classmethod
    def from_json(cls, obj: Json) -> "v1AddProjectNoteResponse":
        return cls(
            notes=[v1Note.from_json(x) for x in obj["notes"]],
        )

    def to_json(self) -> typing.Any:
        return {
            "notes": [x.to_json() for x in self.notes],
        }

class v1Agent:
    def __init__(
        self,
        *,
        id: str,
        addresses: "typing.Optional[typing.Sequence[str]]" = None,
        containers: "typing.Optional[typing.Dict[str, v1Container]]" = None,
        draining: "typing.Optional[bool]" = None,
        enabled: "typing.Optional[bool]" = None,
        label: "typing.Optional[str]" = None,
        registeredTime: "typing.Optional[str]" = None,
        resourcePools: "typing.Optional[typing.Sequence[str]]" = None,
        slots: "typing.Optional[typing.Dict[str, v1Slot]]" = None,
        version: "typing.Optional[str]" = None,
    ):
        self.id = id
        self.registeredTime = registeredTime
        self.slots = slots
        self.containers = containers
        self.label = label
        self.addresses = addresses
        self.enabled = enabled
        self.draining = draining
        self.version = version
        self.resourcePools = resourcePools

    @classmethod
    def from_json(cls, obj: Json) -> "v1Agent":
        return cls(
            id=obj["id"],
            registeredTime=obj.get("registeredTime", None),
            slots={k: v1Slot.from_json(v) for k, v in obj["slots"].items()} if obj.get("slots", None) is not None else None,
            containers={k: v1Container.from_json(v) for k, v in obj["containers"].items()} if obj.get("containers", None) is not None else None,
            label=obj.get("label", None),
            addresses=obj.get("addresses", None),
            enabled=obj.get("enabled", None),
            draining=obj.get("draining", None),
            version=obj.get("version", None),
            resourcePools=obj.get("resourcePools", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "id": self.id,
            "registeredTime": self.registeredTime if self.registeredTime is not None else None,
            "slots": {k: v.to_json() for k, v in self.slots.items()} if self.slots is not None else None,
            "containers": {k: v.to_json() for k, v in self.containers.items()} if self.containers is not None else None,
            "label": self.label if self.label is not None else None,
            "addresses": self.addresses if self.addresses is not None else None,
            "enabled": self.enabled if self.enabled is not None else None,
            "draining": self.draining if self.draining is not None else None,
            "version": self.version if self.version is not None else None,
            "resourcePools": self.resourcePools if self.resourcePools is not None else None,
        }

class v1AgentUserGroup:
    def __init__(
        self,
        *,
        agentGid: "typing.Optional[int]" = None,
        agentGroup: "typing.Optional[str]" = None,
        agentUid: "typing.Optional[int]" = None,
        agentUser: "typing.Optional[str]" = None,
    ):
        self.agentUid = agentUid
        self.agentGid = agentGid
        self.agentUser = agentUser
        self.agentGroup = agentGroup

    @classmethod
    def from_json(cls, obj: Json) -> "v1AgentUserGroup":
        return cls(
            agentUid=obj.get("agentUid", None),
            agentGid=obj.get("agentGid", None),
            agentUser=obj.get("agentUser", None),
            agentGroup=obj.get("agentGroup", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "agentUid": self.agentUid if self.agentUid is not None else None,
            "agentGid": self.agentGid if self.agentGid is not None else None,
            "agentUser": self.agentUser if self.agentUser is not None else None,
            "agentGroup": self.agentGroup if self.agentGroup is not None else None,
        }

class v1AggregateQueueStats:
    def __init__(
        self,
        *,
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
        *,
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
        *,
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
        *,
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
        *,
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
        *,
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
        *,
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
        *,
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

class v1AllocationWaitingRequest:
    def __init__(
        self,
        *,
        allocationId: "typing.Optional[str]" = None,
    ):
        self.allocationId = allocationId

    @classmethod
    def from_json(cls, obj: Json) -> "v1AllocationWaitingRequest":
        return cls(
            allocationId=obj.get("allocationId", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "allocationId": self.allocationId if self.allocationId is not None else None,
        }

class v1AssignRolesRequest:
    def __init__(
        self,
        *,
        groupRoleAssignments: "typing.Optional[typing.Sequence[v1GroupRoleAssignment]]" = None,
        userRoleAssignments: "typing.Optional[typing.Sequence[v1UserRoleAssignment]]" = None,
    ):
        self.groupRoleAssignments = groupRoleAssignments
        self.userRoleAssignments = userRoleAssignments

    @classmethod
    def from_json(cls, obj: Json) -> "v1AssignRolesRequest":
        return cls(
            groupRoleAssignments=[v1GroupRoleAssignment.from_json(x) for x in obj["groupRoleAssignments"]] if obj.get("groupRoleAssignments", None) is not None else None,
            userRoleAssignments=[v1UserRoleAssignment.from_json(x) for x in obj["userRoleAssignments"]] if obj.get("userRoleAssignments", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "groupRoleAssignments": [x.to_json() for x in self.groupRoleAssignments] if self.groupRoleAssignments is not None else None,
            "userRoleAssignments": [x.to_json() for x in self.userRoleAssignments] if self.userRoleAssignments is not None else None,
        }

class v1AugmentedTrial:
    def __init__(
        self,
        *,
        endTime: str,
        experimentDescription: str,
        experimentId: int,
        experimentLabels: "typing.Sequence[str]",
        experimentName: str,
        hparams: "typing.Dict[str, typing.Any]",
        projectId: int,
        searcherType: str,
        startTime: str,
        state: "determinedtrialv1State",
        tags: "typing.Dict[str, typing.Any]",
        totalBatches: int,
        trainingMetrics: "typing.Dict[str, typing.Any]",
        trialId: int,
        userId: int,
        validationMetrics: "typing.Dict[str, typing.Any]",
        workspaceId: int,
        rankWithinExp: "typing.Optional[int]" = None,
        searcherMetric: "typing.Optional[str]" = None,
        searcherMetricLoss: "typing.Optional[float]" = None,
        searcherMetricValue: "typing.Optional[float]" = None,
    ):
        self.trialId = trialId
        self.state = state
        self.hparams = hparams
        self.trainingMetrics = trainingMetrics
        self.validationMetrics = validationMetrics
        self.tags = tags
        self.startTime = startTime
        self.endTime = endTime
        self.searcherType = searcherType
        self.rankWithinExp = rankWithinExp
        self.experimentId = experimentId
        self.experimentName = experimentName
        self.experimentDescription = experimentDescription
        self.experimentLabels = experimentLabels
        self.userId = userId
        self.projectId = projectId
        self.workspaceId = workspaceId
        self.totalBatches = totalBatches
        self.searcherMetric = searcherMetric
        self.searcherMetricValue = searcherMetricValue
        self.searcherMetricLoss = searcherMetricLoss

    @classmethod
    def from_json(cls, obj: Json) -> "v1AugmentedTrial":
        return cls(
            trialId=obj["trialId"],
            state=determinedtrialv1State(obj["state"]),
            hparams=obj["hparams"],
            trainingMetrics=obj["trainingMetrics"],
            validationMetrics=obj["validationMetrics"],
            tags=obj["tags"],
            startTime=obj["startTime"],
            endTime=obj["endTime"],
            searcherType=obj["searcherType"],
            rankWithinExp=obj.get("rankWithinExp", None),
            experimentId=obj["experimentId"],
            experimentName=obj["experimentName"],
            experimentDescription=obj["experimentDescription"],
            experimentLabels=obj["experimentLabels"],
            userId=obj["userId"],
            projectId=obj["projectId"],
            workspaceId=obj["workspaceId"],
            totalBatches=obj["totalBatches"],
            searcherMetric=obj.get("searcherMetric", None),
            searcherMetricValue=float(obj["searcherMetricValue"]) if obj.get("searcherMetricValue", None) is not None else None,
            searcherMetricLoss=float(obj["searcherMetricLoss"]) if obj.get("searcherMetricLoss", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "trialId": self.trialId,
            "state": self.state.value,
            "hparams": self.hparams,
            "trainingMetrics": self.trainingMetrics,
            "validationMetrics": self.validationMetrics,
            "tags": self.tags,
            "startTime": self.startTime,
            "endTime": self.endTime,
            "searcherType": self.searcherType,
            "rankWithinExp": self.rankWithinExp if self.rankWithinExp is not None else None,
            "experimentId": self.experimentId,
            "experimentName": self.experimentName,
            "experimentDescription": self.experimentDescription,
            "experimentLabels": self.experimentLabels,
            "userId": self.userId,
            "projectId": self.projectId,
            "workspaceId": self.workspaceId,
            "totalBatches": self.totalBatches,
            "searcherMetric": self.searcherMetric if self.searcherMetric is not None else None,
            "searcherMetricValue": dump_float(self.searcherMetricValue) if self.searcherMetricValue is not None else None,
            "searcherMetricLoss": dump_float(self.searcherMetricLoss) if self.searcherMetricLoss is not None else None,
        }

class v1AwsCustomTag:
    def __init__(
        self,
        *,
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
        *,
        metadata: "typing.Dict[str, typing.Any]",
        resources: "typing.Dict[str, str]",
        state: "determinedcheckpointv1State",
        training: "v1CheckpointTrainingMetadata",
        uuid: str,
        allocationId: "typing.Optional[str]" = None,
        reportTime: "typing.Optional[str]" = None,
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
            state=determinedcheckpointv1State(obj["state"]),
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
            "state": self.state.value,
            "training": self.training.to_json(),
        }

class v1CheckpointTrainingMetadata:
    def __init__(
        self,
        *,
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
        *,
        state: "determinedcheckpointv1State",
        totalBatches: int,
        endTime: "typing.Optional[str]" = None,
        metadata: "typing.Optional[typing.Dict[str, typing.Any]]" = None,
        resources: "typing.Optional[typing.Dict[str, str]]" = None,
        uuid: "typing.Optional[str]" = None,
    ):
        self.uuid = uuid
        self.endTime = endTime
        self.state = state
        self.resources = resources
        self.totalBatches = totalBatches
        self.metadata = metadata

    @classmethod
    def from_json(cls, obj: Json) -> "v1CheckpointWorkload":
        return cls(
            uuid=obj.get("uuid", None),
            endTime=obj.get("endTime", None),
            state=determinedcheckpointv1State(obj["state"]),
            resources=obj.get("resources", None),
            totalBatches=obj["totalBatches"],
            metadata=obj.get("metadata", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "uuid": self.uuid if self.uuid is not None else None,
            "endTime": self.endTime if self.endTime is not None else None,
            "state": self.state.value,
            "resources": self.resources if self.resources is not None else None,
            "totalBatches": self.totalBatches,
            "metadata": self.metadata if self.metadata is not None else None,
        }

class v1CloseTrialOperation:
    def __init__(
        self,
        *,
        requestId: "typing.Optional[str]" = None,
    ):
        self.requestId = requestId

    @classmethod
    def from_json(cls, obj: Json) -> "v1CloseTrialOperation":
        return cls(
            requestId=obj.get("requestId", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "requestId": self.requestId if self.requestId is not None else None,
        }

class v1ColumnFilter:
    def __init__(
        self,
        *,
        filter: "typing.Optional[v1DoubleFieldFilter]" = None,
        name: "typing.Optional[str]" = None,
    ):
        self.name = name
        self.filter = filter

    @classmethod
    def from_json(cls, obj: Json) -> "v1ColumnFilter":
        return cls(
            name=obj.get("name", None),
            filter=v1DoubleFieldFilter.from_json(obj["filter"]) if obj.get("filter", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "name": self.name if self.name is not None else None,
            "filter": self.filter.to_json() if self.filter is not None else None,
        }

class v1Command:
    def __init__(
        self,
        *,
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

class v1ComparableTrial:
    def __init__(
        self,
        *,
        metrics: "typing.Sequence[v1SummarizedMetric]",
        trial: "trialv1Trial",
    ):
        self.trial = trial
        self.metrics = metrics

    @classmethod
    def from_json(cls, obj: Json) -> "v1ComparableTrial":
        return cls(
            trial=trialv1Trial.from_json(obj["trial"]),
            metrics=[v1SummarizedMetric.from_json(x) for x in obj["metrics"]],
        )

    def to_json(self) -> typing.Any:
        return {
            "trial": self.trial.to_json(),
            "metrics": [x.to_json() for x in self.metrics],
        }

class v1CompareTrialsResponse:
    def __init__(
        self,
        *,
        trials: "typing.Sequence[v1ComparableTrial]",
    ):
        self.trials = trials

    @classmethod
    def from_json(cls, obj: Json) -> "v1CompareTrialsResponse":
        return cls(
            trials=[v1ComparableTrial.from_json(x) for x in obj["trials"]],
        )

    def to_json(self) -> typing.Any:
        return {
            "trials": [x.to_json() for x in self.trials],
        }

class v1CompleteValidateAfterOperation:
    def __init__(
        self,
        *,
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
        *,
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
        *,
        activate: "typing.Optional[bool]" = None,
        config: "typing.Optional[str]" = None,
        modelDefinition: "typing.Optional[typing.Sequence[v1File]]" = None,
        parentId: "typing.Optional[int]" = None,
        projectId: "typing.Optional[int]" = None,
        validateOnly: "typing.Optional[bool]" = None,
    ):
        self.modelDefinition = modelDefinition
        self.config = config
        self.validateOnly = validateOnly
        self.parentId = parentId
        self.activate = activate
        self.projectId = projectId

    @classmethod
    def from_json(cls, obj: Json) -> "v1CreateExperimentRequest":
        return cls(
            modelDefinition=[v1File.from_json(x) for x in obj["modelDefinition"]] if obj.get("modelDefinition", None) is not None else None,
            config=obj.get("config", None),
            validateOnly=obj.get("validateOnly", None),
            parentId=obj.get("parentId", None),
            activate=obj.get("activate", None),
            projectId=obj.get("projectId", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "modelDefinition": [x.to_json() for x in self.modelDefinition] if self.modelDefinition is not None else None,
            "config": self.config if self.config is not None else None,
            "validateOnly": self.validateOnly if self.validateOnly is not None else None,
            "parentId": self.parentId if self.parentId is not None else None,
            "activate": self.activate if self.activate is not None else None,
            "projectId": self.projectId if self.projectId is not None else None,
        }

class v1CreateExperimentResponse:
    def __init__(
        self,
        *,
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

class v1CreateGroupRequest:
    def __init__(
        self,
        *,
        name: str,
        addUsers: "typing.Optional[typing.Sequence[int]]" = None,
    ):
        self.name = name
        self.addUsers = addUsers

    @classmethod
    def from_json(cls, obj: Json) -> "v1CreateGroupRequest":
        return cls(
            name=obj["name"],
            addUsers=obj.get("addUsers", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "name": self.name,
            "addUsers": self.addUsers if self.addUsers is not None else None,
        }

class v1CreateGroupResponse:
    def __init__(
        self,
        *,
        group: "v1GroupDetails",
    ):
        self.group = group

    @classmethod
    def from_json(cls, obj: Json) -> "v1CreateGroupResponse":
        return cls(
            group=v1GroupDetails.from_json(obj["group"]),
        )

    def to_json(self) -> typing.Any:
        return {
            "group": self.group.to_json(),
        }

class v1CreateTrialOperation:
    def __init__(
        self,
        *,
        hyperparams: "typing.Optional[str]" = None,
        requestId: "typing.Optional[str]" = None,
    ):
        self.requestId = requestId
        self.hyperparams = hyperparams

    @classmethod
    def from_json(cls, obj: Json) -> "v1CreateTrialOperation":
        return cls(
            requestId=obj.get("requestId", None),
            hyperparams=obj.get("hyperparams", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "requestId": self.requestId if self.requestId is not None else None,
            "hyperparams": self.hyperparams if self.hyperparams is not None else None,
        }

class v1CreateTrialsCollectionRequest:
    def __init__(
        self,
        *,
        filters: "v1TrialFilters",
        name: str,
        projectId: int,
        sorter: "v1TrialSorter",
    ):
        self.name = name
        self.projectId = projectId
        self.filters = filters
        self.sorter = sorter

    @classmethod
    def from_json(cls, obj: Json) -> "v1CreateTrialsCollectionRequest":
        return cls(
            name=obj["name"],
            projectId=obj["projectId"],
            filters=v1TrialFilters.from_json(obj["filters"]),
            sorter=v1TrialSorter.from_json(obj["sorter"]),
        )

    def to_json(self) -> typing.Any:
        return {
            "name": self.name,
            "projectId": self.projectId,
            "filters": self.filters.to_json(),
            "sorter": self.sorter.to_json(),
        }

class v1CreateTrialsCollectionResponse:
    def __init__(
        self,
        *,
        collection: "typing.Optional[v1TrialsCollection]" = None,
    ):
        self.collection = collection

    @classmethod
    def from_json(cls, obj: Json) -> "v1CreateTrialsCollectionResponse":
        return cls(
            collection=v1TrialsCollection.from_json(obj["collection"]) if obj.get("collection", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "collection": self.collection.to_json() if self.collection is not None else None,
        }

class v1CurrentUserResponse:
    def __init__(
        self,
        *,
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

class v1DataPoint:
    def __init__(
        self,
        *,
        batches: int,
        value: float,
    ):
        self.batches = batches
        self.value = value

    @classmethod
    def from_json(cls, obj: Json) -> "v1DataPoint":
        return cls(
            batches=obj["batches"],
            value=float(obj["value"]),
        )

    def to_json(self) -> typing.Any:
        return {
            "batches": self.batches,
            "value": dump_float(self.value),
        }

class v1DeleteCheckpointsRequest:
    def __init__(
        self,
        *,
        checkpointUuids: "typing.Sequence[str]",
    ):
        self.checkpointUuids = checkpointUuids

    @classmethod
    def from_json(cls, obj: Json) -> "v1DeleteCheckpointsRequest":
        return cls(
            checkpointUuids=obj["checkpointUuids"],
        )

    def to_json(self) -> typing.Any:
        return {
            "checkpointUuids": self.checkpointUuids,
        }

class v1DeleteProjectResponse:
    def __init__(
        self,
        *,
        completed: bool,
    ):
        self.completed = completed

    @classmethod
    def from_json(cls, obj: Json) -> "v1DeleteProjectResponse":
        return cls(
            completed=obj["completed"],
        )

    def to_json(self) -> typing.Any:
        return {
            "completed": self.completed,
        }

class v1DeleteWorkspaceResponse:
    def __init__(
        self,
        *,
        completed: bool,
    ):
        self.completed = completed

    @classmethod
    def from_json(cls, obj: Json) -> "v1DeleteWorkspaceResponse":
        return cls(
            completed=obj["completed"],
        )

    def to_json(self) -> typing.Any:
        return {
            "completed": self.completed,
        }

class v1Device:
    def __init__(
        self,
        *,
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
        *,
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
        *,
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
        *,
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

class v1DoubleFieldFilter:
    def __init__(
        self,
        *,
        gt: "typing.Optional[float]" = None,
        gte: "typing.Optional[float]" = None,
        lt: "typing.Optional[float]" = None,
        lte: "typing.Optional[float]" = None,
    ):
        self.lt = lt
        self.lte = lte
        self.gt = gt
        self.gte = gte

    @classmethod
    def from_json(cls, obj: Json) -> "v1DoubleFieldFilter":
        return cls(
            lt=float(obj["lt"]) if obj.get("lt", None) is not None else None,
            lte=float(obj["lte"]) if obj.get("lte", None) is not None else None,
            gt=float(obj["gt"]) if obj.get("gt", None) is not None else None,
            gte=float(obj["gte"]) if obj.get("gte", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "lt": dump_float(self.lt) if self.lt is not None else None,
            "lte": dump_float(self.lte) if self.lte is not None else None,
            "gt": dump_float(self.gt) if self.gt is not None else None,
            "gte": dump_float(self.gte) if self.gte is not None else None,
        }

class v1EnableAgentResponse:
    def __init__(
        self,
        *,
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
        *,
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

class v1ExpCompareMetricNamesResponse:
    def __init__(
        self,
        *,
        trainingMetrics: "typing.Optional[typing.Sequence[str]]" = None,
        validationMetrics: "typing.Optional[typing.Sequence[str]]" = None,
    ):
        self.trainingMetrics = trainingMetrics
        self.validationMetrics = validationMetrics

    @classmethod
    def from_json(cls, obj: Json) -> "v1ExpCompareMetricNamesResponse":
        return cls(
            trainingMetrics=obj.get("trainingMetrics", None),
            validationMetrics=obj.get("validationMetrics", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "trainingMetrics": self.trainingMetrics if self.trainingMetrics is not None else None,
            "validationMetrics": self.validationMetrics if self.validationMetrics is not None else None,
        }

class v1ExpCompareTrialsSampleResponse:
    def __init__(
        self,
        *,
        demotedTrials: "typing.Sequence[int]",
        promotedTrials: "typing.Sequence[int]",
        trials: "typing.Sequence[ExpCompareTrialsSampleResponseExpTrial]",
    ):
        self.trials = trials
        self.promotedTrials = promotedTrials
        self.demotedTrials = demotedTrials

    @classmethod
    def from_json(cls, obj: Json) -> "v1ExpCompareTrialsSampleResponse":
        return cls(
            trials=[ExpCompareTrialsSampleResponseExpTrial.from_json(x) for x in obj["trials"]],
            promotedTrials=obj["promotedTrials"],
            demotedTrials=obj["demotedTrials"],
        )

    def to_json(self) -> typing.Any:
        return {
            "trials": [x.to_json() for x in self.trials],
            "promotedTrials": self.promotedTrials,
            "demotedTrials": self.demotedTrials,
        }

class v1Experiment:
    def __init__(
        self,
        *,
        archived: bool,
        config: "typing.Dict[str, typing.Any]",
        id: int,
        jobId: str,
        name: str,
        numTrials: int,
        originalConfig: str,
        projectId: int,
        projectOwnerId: int,
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
        parentArchived: "typing.Optional[bool]" = None,
        progress: "typing.Optional[float]" = None,
        projectName: "typing.Optional[str]" = None,
        resourcePool: "typing.Optional[str]" = None,
        trialIds: "typing.Optional[typing.Sequence[int]]" = None,
        userId: "typing.Optional[int]" = None,
        workspaceId: "typing.Optional[int]" = None,
        workspaceName: "typing.Optional[str]" = None,
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
        self.projectId = projectId
        self.projectName = projectName
        self.workspaceId = workspaceId
        self.workspaceName = workspaceName
        self.parentArchived = parentArchived
        self.config = config
        self.originalConfig = originalConfig
        self.projectOwnerId = projectOwnerId

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
            projectId=obj["projectId"],
            projectName=obj.get("projectName", None),
            workspaceId=obj.get("workspaceId", None),
            workspaceName=obj.get("workspaceName", None),
            parentArchived=obj.get("parentArchived", None),
            config=obj["config"],
            originalConfig=obj["originalConfig"],
            projectOwnerId=obj["projectOwnerId"],
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
            "projectId": self.projectId,
            "projectName": self.projectName if self.projectName is not None else None,
            "workspaceId": self.workspaceId if self.workspaceId is not None else None,
            "workspaceName": self.workspaceName if self.workspaceName is not None else None,
            "parentArchived": self.parentArchived if self.parentArchived is not None else None,
            "config": self.config,
            "originalConfig": self.originalConfig,
            "projectOwnerId": self.projectOwnerId,
        }

class v1ExperimentInactive:
    def __init__(
        self,
        *,
        experimentState: "determinedexperimentv1State",
    ):
        self.experimentState = experimentState

    @classmethod
    def from_json(cls, obj: Json) -> "v1ExperimentInactive":
        return cls(
            experimentState=determinedexperimentv1State(obj["experimentState"]),
        )

    def to_json(self) -> typing.Any:
        return {
            "experimentState": self.experimentState.value,
        }

class v1ExperimentSimulation:
    def __init__(
        self,
        *,
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
        *,
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

class v1FileNode:
    def __init__(
        self,
        *,
        contentLength: "typing.Optional[int]" = None,
        contentType: "typing.Optional[str]" = None,
        files: "typing.Optional[typing.Sequence[v1FileNode]]" = None,
        isDir: "typing.Optional[bool]" = None,
        modifiedTime: "typing.Optional[str]" = None,
        name: "typing.Optional[str]" = None,
        path: "typing.Optional[str]" = None,
    ):
        self.path = path
        self.name = name
        self.modifiedTime = modifiedTime
        self.contentLength = contentLength
        self.isDir = isDir
        self.contentType = contentType
        self.files = files

    @classmethod
    def from_json(cls, obj: Json) -> "v1FileNode":
        return cls(
            path=obj.get("path", None),
            name=obj.get("name", None),
            modifiedTime=obj.get("modifiedTime", None),
            contentLength=obj.get("contentLength", None),
            isDir=obj.get("isDir", None),
            contentType=obj.get("contentType", None),
            files=[v1FileNode.from_json(x) for x in obj["files"]] if obj.get("files", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "path": self.path if self.path is not None else None,
            "name": self.name if self.name is not None else None,
            "modifiedTime": self.modifiedTime if self.modifiedTime is not None else None,
            "contentLength": self.contentLength if self.contentLength is not None else None,
            "isDir": self.isDir if self.isDir is not None else None,
            "contentType": self.contentType if self.contentType is not None else None,
            "files": [x.to_json() for x in self.files] if self.files is not None else None,
        }

class v1FittingPolicy(enum.Enum):
    FITTING_POLICY_UNSPECIFIED = "FITTING_POLICY_UNSPECIFIED"
    FITTING_POLICY_BEST = "FITTING_POLICY_BEST"
    FITTING_POLICY_WORST = "FITTING_POLICY_WORST"
    FITTING_POLICY_KUBERNETES = "FITTING_POLICY_KUBERNETES"
    FITTING_POLICY_SLURM = "FITTING_POLICY_SLURM"
    FITTING_POLICY_PBS = "FITTING_POLICY_PBS"

class v1GetActiveTasksCountResponse:
    def __init__(
        self,
        *,
        commands: int,
        notebooks: int,
        shells: int,
        tensorboards: int,
    ):
        self.commands = commands
        self.notebooks = notebooks
        self.shells = shells
        self.tensorboards = tensorboards

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetActiveTasksCountResponse":
        return cls(
            commands=obj["commands"],
            notebooks=obj["notebooks"],
            shells=obj["shells"],
            tensorboards=obj["tensorboards"],
        )

    def to_json(self) -> typing.Any:
        return {
            "commands": self.commands,
            "notebooks": self.notebooks,
            "shells": self.shells,
            "tensorboards": self.tensorboards,
        }

class v1GetAgentResponse:
    def __init__(
        self,
        *,
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
        *,
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
        *,
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
        *,
        checkpoint: "v1Checkpoint",
    ):
        self.checkpoint = checkpoint

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetCheckpointResponse":
        return cls(
            checkpoint=v1Checkpoint.from_json(obj["checkpoint"]),
        )

    def to_json(self) -> typing.Any:
        return {
            "checkpoint": self.checkpoint.to_json(),
        }

class v1GetCommandResponse:
    def __init__(
        self,
        *,
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
        *,
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
        *,
        completed: "typing.Optional[bool]" = None,
        op: "typing.Optional[v1TrialOperation]" = None,
    ):
        self.op = op
        self.completed = completed

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetCurrentTrialSearcherOperationResponse":
        return cls(
            op=v1TrialOperation.from_json(obj["op"]) if obj.get("op", None) is not None else None,
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
        *,
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
        *,
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
        *,
        experiment: "v1Experiment",
        jobSummary: "typing.Optional[v1JobSummary]" = None,
    ):
        self.experiment = experiment
        self.jobSummary = jobSummary

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetExperimentResponse":
        return cls(
            experiment=v1Experiment.from_json(obj["experiment"]),
            jobSummary=v1JobSummary.from_json(obj["jobSummary"]) if obj.get("jobSummary", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "experiment": self.experiment.to_json(),
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
        *,
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
        *,
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
    SORT_BY_PROJECT_ID = "SORT_BY_PROJECT_ID"

class v1GetExperimentsResponse:
    def __init__(
        self,
        *,
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

class v1GetGroupResponse:
    def __init__(
        self,
        *,
        group: "v1GroupDetails",
    ):
        self.group = group

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetGroupResponse":
        return cls(
            group=v1GroupDetails.from_json(obj["group"]),
        )

    def to_json(self) -> typing.Any:
        return {
            "group": self.group.to_json(),
        }

class v1GetGroupsAndUsersAssignedToWorkspaceResponse:
    def __init__(
        self,
        *,
        assignments: "typing.Sequence[v1RoleWithAssignments]",
        groups: "typing.Sequence[v1GroupDetails]",
        usersAssignedDirectly: "typing.Sequence[v1User]",
    ):
        self.groups = groups
        self.usersAssignedDirectly = usersAssignedDirectly
        self.assignments = assignments

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetGroupsAndUsersAssignedToWorkspaceResponse":
        return cls(
            groups=[v1GroupDetails.from_json(x) for x in obj["groups"]],
            usersAssignedDirectly=[v1User.from_json(x) for x in obj["usersAssignedDirectly"]],
            assignments=[v1RoleWithAssignments.from_json(x) for x in obj["assignments"]],
        )

    def to_json(self) -> typing.Any:
        return {
            "groups": [x.to_json() for x in self.groups],
            "usersAssignedDirectly": [x.to_json() for x in self.usersAssignedDirectly],
            "assignments": [x.to_json() for x in self.assignments],
        }

class v1GetGroupsRequest:
    def __init__(
        self,
        *,
        limit: int,
        name: "typing.Optional[str]" = None,
        offset: "typing.Optional[int]" = None,
        userId: "typing.Optional[int]" = None,
    ):
        self.userId = userId
        self.name = name
        self.offset = offset
        self.limit = limit

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetGroupsRequest":
        return cls(
            userId=obj.get("userId", None),
            name=obj.get("name", None),
            offset=obj.get("offset", None),
            limit=obj["limit"],
        )

    def to_json(self) -> typing.Any:
        return {
            "userId": self.userId if self.userId is not None else None,
            "name": self.name if self.name is not None else None,
            "offset": self.offset if self.offset is not None else None,
            "limit": self.limit,
        }

class v1GetGroupsResponse:
    def __init__(
        self,
        *,
        groups: "typing.Optional[typing.Sequence[v1GroupSearchResult]]" = None,
        pagination: "typing.Optional[v1Pagination]" = None,
    ):
        self.groups = groups
        self.pagination = pagination

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetGroupsResponse":
        return cls(
            groups=[v1GroupSearchResult.from_json(x) for x in obj["groups"]] if obj.get("groups", None) is not None else None,
            pagination=v1Pagination.from_json(obj["pagination"]) if obj.get("pagination", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "groups": [x.to_json() for x in self.groups] if self.groups is not None else None,
            "pagination": self.pagination.to_json() if self.pagination is not None else None,
        }

class v1GetHPImportanceResponse:
    def __init__(
        self,
        *,
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
        *,
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
        *,
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
        *,
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
        *,
        clusterId: str,
        clusterName: str,
        masterId: str,
        version: str,
        branding: "typing.Optional[str]" = None,
        externalLoginUri: "typing.Optional[str]" = None,
        externalLogoutUri: "typing.Optional[str]" = None,
        rbacEnabled: "typing.Optional[bool]" = None,
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
        self.rbacEnabled = rbacEnabled

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
            rbacEnabled=obj.get("rbacEnabled", None),
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
            "rbacEnabled": self.rbacEnabled if self.rbacEnabled is not None else None,
        }

class v1GetModelDefFileRequest:
    def __init__(
        self,
        *,
        experimentId: "typing.Optional[int]" = None,
        path: "typing.Optional[str]" = None,
    ):
        self.experimentId = experimentId
        self.path = path

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetModelDefFileRequest":
        return cls(
            experimentId=obj.get("experimentId", None),
            path=obj.get("path", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "experimentId": self.experimentId if self.experimentId is not None else None,
            "path": self.path if self.path is not None else None,
        }

class v1GetModelDefFileResponse:
    def __init__(
        self,
        *,
        file: "typing.Optional[str]" = None,
    ):
        self.file = file

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetModelDefFileResponse":
        return cls(
            file=obj.get("file", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "file": self.file if self.file is not None else None,
        }

class v1GetModelDefResponse:
    def __init__(
        self,
        *,
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

class v1GetModelDefTreeResponse:
    def __init__(
        self,
        *,
        files: "typing.Optional[typing.Sequence[v1FileNode]]" = None,
    ):
        self.files = files

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetModelDefTreeResponse":
        return cls(
            files=[v1FileNode.from_json(x) for x in obj["files"]] if obj.get("files", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "files": [x.to_json() for x in self.files] if self.files is not None else None,
        }

class v1GetModelLabelsResponse:
    def __init__(
        self,
        *,
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
        *,
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
        *,
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
        *,
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
        *,
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
        *,
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
        *,
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

class v1GetPermissionsSummaryResponse:
    def __init__(
        self,
        *,
        assignments: "typing.Sequence[v1RoleAssignmentSummary]",
        roles: "typing.Sequence[v1Role]",
    ):
        self.roles = roles
        self.assignments = assignments

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetPermissionsSummaryResponse":
        return cls(
            roles=[v1Role.from_json(x) for x in obj["roles"]],
            assignments=[v1RoleAssignmentSummary.from_json(x) for x in obj["assignments"]],
        )

    def to_json(self) -> typing.Any:
        return {
            "roles": [x.to_json() for x in self.roles],
            "assignments": [x.to_json() for x in self.assignments],
        }

class v1GetProjectResponse:
    def __init__(
        self,
        *,
        project: "v1Project",
    ):
        self.project = project

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetProjectResponse":
        return cls(
            project=v1Project.from_json(obj["project"]),
        )

    def to_json(self) -> typing.Any:
        return {
            "project": self.project.to_json(),
        }

class v1GetResourcePoolsResponse:
    def __init__(
        self,
        *,
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

class v1GetRolesAssignedToGroupResponse:
    def __init__(
        self,
        *,
        roles: "typing.Optional[typing.Sequence[v1Role]]" = None,
    ):
        self.roles = roles

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetRolesAssignedToGroupResponse":
        return cls(
            roles=[v1Role.from_json(x) for x in obj["roles"]] if obj.get("roles", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "roles": [x.to_json() for x in self.roles] if self.roles is not None else None,
        }

class v1GetRolesAssignedToUserResponse:
    def __init__(
        self,
        *,
        roles: "typing.Optional[typing.Sequence[v1Role]]" = None,
    ):
        self.roles = roles

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetRolesAssignedToUserResponse":
        return cls(
            roles=[v1Role.from_json(x) for x in obj["roles"]] if obj.get("roles", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "roles": [x.to_json() for x in self.roles] if self.roles is not None else None,
        }

class v1GetRolesByIDRequest:
    def __init__(
        self,
        *,
        roleIds: "typing.Optional[typing.Sequence[int]]" = None,
    ):
        self.roleIds = roleIds

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetRolesByIDRequest":
        return cls(
            roleIds=obj.get("roleIds", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "roleIds": self.roleIds if self.roleIds is not None else None,
        }

class v1GetRolesByIDResponse:
    def __init__(
        self,
        *,
        roles: "typing.Optional[typing.Sequence[v1RoleWithAssignments]]" = None,
    ):
        self.roles = roles

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetRolesByIDResponse":
        return cls(
            roles=[v1RoleWithAssignments.from_json(x) for x in obj["roles"]] if obj.get("roles", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "roles": [x.to_json() for x in self.roles] if self.roles is not None else None,
        }

class v1GetSearcherEventsResponse:
    def __init__(
        self,
        *,
        searcherEvents: "typing.Optional[typing.Sequence[v1SearcherEvent]]" = None,
    ):
        self.searcherEvents = searcherEvents

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetSearcherEventsResponse":
        return cls(
            searcherEvents=[v1SearcherEvent.from_json(x) for x in obj["searcherEvents"]] if obj.get("searcherEvents", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "searcherEvents": [x.to_json() for x in self.searcherEvents] if self.searcherEvents is not None else None,
        }

class v1GetShellResponse:
    def __init__(
        self,
        *,
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
        *,
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
        *,
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
        *,
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
        *,
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
        *,
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
        *,
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
        *,
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
        *,
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
        *,
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
        *,
        checkpoints: "typing.Sequence[v1Checkpoint]",
        pagination: "v1Pagination",
    ):
        self.checkpoints = checkpoints
        self.pagination = pagination

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetTrialCheckpointsResponse":
        return cls(
            checkpoints=[v1Checkpoint.from_json(x) for x in obj["checkpoints"]],
            pagination=v1Pagination.from_json(obj["pagination"]),
        )

    def to_json(self) -> typing.Any:
        return {
            "checkpoints": [x.to_json() for x in self.checkpoints],
            "pagination": self.pagination.to_json(),
        }

class v1GetTrialProfilerAvailableSeriesResponse:
    def __init__(
        self,
        *,
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
        *,
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
        *,
        trial: "trialv1Trial",
    ):
        self.trial = trial

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetTrialResponse":
        return cls(
            trial=trialv1Trial.from_json(obj["trial"]),
        )

    def to_json(self) -> typing.Any:
        return {
            "trial": self.trial.to_json(),
        }

class v1GetTrialWorkloadsResponse:
    def __init__(
        self,
        *,
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

class v1GetTrialsCollectionsResponse:
    def __init__(
        self,
        *,
        collections: "typing.Optional[typing.Sequence[v1TrialsCollection]]" = None,
    ):
        self.collections = collections

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetTrialsCollectionsResponse":
        return cls(
            collections=[v1TrialsCollection.from_json(x) for x in obj["collections"]] if obj.get("collections", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "collections": [x.to_json() for x in self.collections] if self.collections is not None else None,
        }

class v1GetUserResponse:
    def __init__(
        self,
        *,
        user: "v1User",
    ):
        self.user = user

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetUserResponse":
        return cls(
            user=v1User.from_json(obj["user"]),
        )

    def to_json(self) -> typing.Any:
        return {
            "user": self.user.to_json(),
        }

class v1GetUserSettingResponse:
    def __init__(
        self,
        *,
        settings: "typing.Sequence[v1UserWebSetting]",
    ):
        self.settings = settings

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetUserSettingResponse":
        return cls(
            settings=[v1UserWebSetting.from_json(x) for x in obj["settings"]],
        )

    def to_json(self) -> typing.Any:
        return {
            "settings": [x.to_json() for x in self.settings],
        }

class v1GetUsersRequestSortBy(enum.Enum):
    SORT_BY_UNSPECIFIED = "SORT_BY_UNSPECIFIED"
    SORT_BY_DISPLAY_NAME = "SORT_BY_DISPLAY_NAME"
    SORT_BY_USER_NAME = "SORT_BY_USER_NAME"
    SORT_BY_ADMIN = "SORT_BY_ADMIN"
    SORT_BY_ACTIVE = "SORT_BY_ACTIVE"
    SORT_BY_MODIFIED_TIME = "SORT_BY_MODIFIED_TIME"

class v1GetUsersResponse:
    def __init__(
        self,
        *,
        pagination: "typing.Optional[v1Pagination]" = None,
        users: "typing.Optional[typing.Sequence[v1User]]" = None,
    ):
        self.users = users
        self.pagination = pagination

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetUsersResponse":
        return cls(
            users=[v1User.from_json(x) for x in obj["users"]] if obj.get("users", None) is not None else None,
            pagination=v1Pagination.from_json(obj["pagination"]) if obj.get("pagination", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "users": [x.to_json() for x in self.users] if self.users is not None else None,
            "pagination": self.pagination.to_json() if self.pagination is not None else None,
        }

class v1GetWebhooksResponse:
    def __init__(
        self,
        *,
        webhooks: "typing.Sequence[v1Webhook]",
    ):
        self.webhooks = webhooks

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetWebhooksResponse":
        return cls(
            webhooks=[v1Webhook.from_json(x) for x in obj["webhooks"]],
        )

    def to_json(self) -> typing.Any:
        return {
            "webhooks": [x.to_json() for x in self.webhooks],
        }

class v1GetWorkspaceProjectsRequestSortBy(enum.Enum):
    SORT_BY_UNSPECIFIED = "SORT_BY_UNSPECIFIED"
    SORT_BY_CREATION_TIME = "SORT_BY_CREATION_TIME"
    SORT_BY_LAST_EXPERIMENT_START_TIME = "SORT_BY_LAST_EXPERIMENT_START_TIME"
    SORT_BY_NAME = "SORT_BY_NAME"
    SORT_BY_DESCRIPTION = "SORT_BY_DESCRIPTION"
    SORT_BY_ID = "SORT_BY_ID"

class v1GetWorkspaceProjectsResponse:
    def __init__(
        self,
        *,
        pagination: "v1Pagination",
        projects: "typing.Sequence[v1Project]",
    ):
        self.projects = projects
        self.pagination = pagination

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetWorkspaceProjectsResponse":
        return cls(
            projects=[v1Project.from_json(x) for x in obj["projects"]],
            pagination=v1Pagination.from_json(obj["pagination"]),
        )

    def to_json(self) -> typing.Any:
        return {
            "projects": [x.to_json() for x in self.projects],
            "pagination": self.pagination.to_json(),
        }

class v1GetWorkspaceResponse:
    def __init__(
        self,
        *,
        workspace: "v1Workspace",
    ):
        self.workspace = workspace

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetWorkspaceResponse":
        return cls(
            workspace=v1Workspace.from_json(obj["workspace"]),
        )

    def to_json(self) -> typing.Any:
        return {
            "workspace": self.workspace.to_json(),
        }

class v1GetWorkspacesRequestSortBy(enum.Enum):
    SORT_BY_UNSPECIFIED = "SORT_BY_UNSPECIFIED"
    SORT_BY_ID = "SORT_BY_ID"
    SORT_BY_NAME = "SORT_BY_NAME"

class v1GetWorkspacesResponse:
    def __init__(
        self,
        *,
        pagination: "v1Pagination",
        workspaces: "typing.Sequence[v1Workspace]",
    ):
        self.workspaces = workspaces
        self.pagination = pagination

    @classmethod
    def from_json(cls, obj: Json) -> "v1GetWorkspacesResponse":
        return cls(
            workspaces=[v1Workspace.from_json(x) for x in obj["workspaces"]],
            pagination=v1Pagination.from_json(obj["pagination"]),
        )

    def to_json(self) -> typing.Any:
        return {
            "workspaces": [x.to_json() for x in self.workspaces],
            "pagination": self.pagination.to_json(),
        }

class v1Group:
    def __init__(
        self,
        *,
        groupId: "typing.Optional[int]" = None,
        name: "typing.Optional[str]" = None,
    ):
        self.groupId = groupId
        self.name = name

    @classmethod
    def from_json(cls, obj: Json) -> "v1Group":
        return cls(
            groupId=obj.get("groupId", None),
            name=obj.get("name", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "groupId": self.groupId if self.groupId is not None else None,
            "name": self.name if self.name is not None else None,
        }

class v1GroupDetails:
    def __init__(
        self,
        *,
        groupId: "typing.Optional[int]" = None,
        name: "typing.Optional[str]" = None,
        users: "typing.Optional[typing.Sequence[v1User]]" = None,
    ):
        self.groupId = groupId
        self.name = name
        self.users = users

    @classmethod
    def from_json(cls, obj: Json) -> "v1GroupDetails":
        return cls(
            groupId=obj.get("groupId", None),
            name=obj.get("name", None),
            users=[v1User.from_json(x) for x in obj["users"]] if obj.get("users", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "groupId": self.groupId if self.groupId is not None else None,
            "name": self.name if self.name is not None else None,
            "users": [x.to_json() for x in self.users] if self.users is not None else None,
        }

class v1GroupRoleAssignment:
    def __init__(
        self,
        *,
        groupId: int,
        roleAssignment: "v1RoleAssignment",
    ):
        self.groupId = groupId
        self.roleAssignment = roleAssignment

    @classmethod
    def from_json(cls, obj: Json) -> "v1GroupRoleAssignment":
        return cls(
            groupId=obj["groupId"],
            roleAssignment=v1RoleAssignment.from_json(obj["roleAssignment"]),
        )

    def to_json(self) -> typing.Any:
        return {
            "groupId": self.groupId,
            "roleAssignment": self.roleAssignment.to_json(),
        }

class v1GroupSearchResult:
    def __init__(
        self,
        *,
        group: "v1Group",
        numMembers: int,
    ):
        self.group = group
        self.numMembers = numMembers

    @classmethod
    def from_json(cls, obj: Json) -> "v1GroupSearchResult":
        return cls(
            group=v1Group.from_json(obj["group"]),
            numMembers=obj["numMembers"],
        )

    def to_json(self) -> typing.Any:
        return {
            "group": self.group.to_json(),
            "numMembers": self.numMembers,
        }

class v1IdleNotebookRequest:
    def __init__(
        self,
        *,
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

class v1InitialOperations:
    def __init__(
        self,
        *,
        placeholder: "typing.Optional[int]" = None,
    ):
        self.placeholder = placeholder

    @classmethod
    def from_json(cls, obj: Json) -> "v1InitialOperations":
        return cls(
            placeholder=obj.get("placeholder", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "placeholder": self.placeholder if self.placeholder is not None else None,
        }

class v1Int32FieldFilter:
    def __init__(
        self,
        *,
        gt: "typing.Optional[int]" = None,
        gte: "typing.Optional[int]" = None,
        incl: "typing.Optional[typing.Sequence[int]]" = None,
        lt: "typing.Optional[int]" = None,
        lte: "typing.Optional[int]" = None,
        notIn: "typing.Optional[typing.Sequence[int]]" = None,
    ):
        self.lt = lt
        self.lte = lte
        self.gt = gt
        self.gte = gte
        self.incl = incl
        self.notIn = notIn

    @classmethod
    def from_json(cls, obj: Json) -> "v1Int32FieldFilter":
        return cls(
            lt=obj.get("lt", None),
            lte=obj.get("lte", None),
            gt=obj.get("gt", None),
            gte=obj.get("gte", None),
            incl=obj.get("incl", None),
            notIn=obj.get("notIn", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "lt": self.lt if self.lt is not None else None,
            "lte": self.lte if self.lte is not None else None,
            "gt": self.gt if self.gt is not None else None,
            "gte": self.gte if self.gte is not None else None,
            "incl": self.incl if self.incl is not None else None,
            "notIn": self.notIn if self.notIn is not None else None,
        }

class v1Job:
    def __init__(
        self,
        *,
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
        *,
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
        *,
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
        *,
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
        *,
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
        *,
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
        *,
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
        *,
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
        *,
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
        *,
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
        *,
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
        *,
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
        *,
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
        *,
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
        *,
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

class v1ListRolesRequest:
    def __init__(
        self,
        *,
        limit: int,
        offset: "typing.Optional[int]" = None,
    ):
        self.offset = offset
        self.limit = limit

    @classmethod
    def from_json(cls, obj: Json) -> "v1ListRolesRequest":
        return cls(
            offset=obj.get("offset", None),
            limit=obj["limit"],
        )

    def to_json(self) -> typing.Any:
        return {
            "offset": self.offset if self.offset is not None else None,
            "limit": self.limit,
        }

class v1ListRolesResponse:
    def __init__(
        self,
        *,
        pagination: "v1Pagination",
        roles: "typing.Sequence[v1Role]",
    ):
        self.roles = roles
        self.pagination = pagination

    @classmethod
    def from_json(cls, obj: Json) -> "v1ListRolesResponse":
        return cls(
            roles=[v1Role.from_json(x) for x in obj["roles"]],
            pagination=v1Pagination.from_json(obj["pagination"]),
        )

    def to_json(self) -> typing.Any:
        return {
            "roles": [x.to_json() for x in self.roles],
            "pagination": self.pagination.to_json(),
        }

class v1LogEntry:
    def __init__(
        self,
        *,
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
        *,
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
        *,
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
        *,
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
        *,
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
        *,
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
        *,
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
        *,
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
        *,
        metrics: "v1Metrics",
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
            metrics=v1Metrics.from_json(obj["metrics"]),
            numInputs=obj["numInputs"],
            totalBatches=obj["totalBatches"],
        )

    def to_json(self) -> typing.Any:
        return {
            "endTime": self.endTime if self.endTime is not None else None,
            "state": self.state.value,
            "metrics": self.metrics.to_json(),
            "numInputs": self.numInputs,
            "totalBatches": self.totalBatches,
        }

class v1Model:
    def __init__(
        self,
        *,
        archived: bool,
        creationTime: str,
        id: int,
        lastUpdatedTime: str,
        metadata: "typing.Dict[str, typing.Any]",
        name: str,
        numVersions: int,
        userId: int,
        username: str,
        description: "typing.Optional[str]" = None,
        labels: "typing.Optional[typing.Sequence[str]]" = None,
        notes: "typing.Optional[str]" = None,
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
            userId=obj["userId"],
            archived=obj["archived"],
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
            "userId": self.userId,
            "archived": self.archived,
            "notes": self.notes if self.notes is not None else None,
        }

class v1ModelVersion:
    def __init__(
        self,
        *,
        checkpoint: "v1Checkpoint",
        creationTime: str,
        id: int,
        lastUpdatedTime: str,
        model: "v1Model",
        version: int,
        comment: "typing.Optional[str]" = None,
        labels: "typing.Optional[typing.Sequence[str]]" = None,
        metadata: "typing.Optional[typing.Dict[str, typing.Any]]" = None,
        name: "typing.Optional[str]" = None,
        notes: "typing.Optional[str]" = None,
        userId: "typing.Optional[int]" = None,
        username: "typing.Optional[str]" = None,
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
            lastUpdatedTime=obj["lastUpdatedTime"],
            comment=obj.get("comment", None),
            username=obj.get("username", None),
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
            "lastUpdatedTime": self.lastUpdatedTime,
            "comment": self.comment if self.comment is not None else None,
            "username": self.username if self.username is not None else None,
            "userId": self.userId if self.userId is not None else None,
            "labels": self.labels if self.labels is not None else None,
            "notes": self.notes if self.notes is not None else None,
        }

class v1MoveExperimentRequest:
    def __init__(
        self,
        *,
        destinationProjectId: int,
        experimentId: int,
    ):
        self.experimentId = experimentId
        self.destinationProjectId = destinationProjectId

    @classmethod
    def from_json(cls, obj: Json) -> "v1MoveExperimentRequest":
        return cls(
            experimentId=obj["experimentId"],
            destinationProjectId=obj["destinationProjectId"],
        )

    def to_json(self) -> typing.Any:
        return {
            "experimentId": self.experimentId,
            "destinationProjectId": self.destinationProjectId,
        }

class v1MoveProjectRequest:
    def __init__(
        self,
        *,
        destinationWorkspaceId: int,
        projectId: int,
    ):
        self.projectId = projectId
        self.destinationWorkspaceId = destinationWorkspaceId

    @classmethod
    def from_json(cls, obj: Json) -> "v1MoveProjectRequest":
        return cls(
            projectId=obj["projectId"],
            destinationWorkspaceId=obj["destinationWorkspaceId"],
        )

    def to_json(self) -> typing.Any:
        return {
            "projectId": self.projectId,
            "destinationWorkspaceId": self.destinationWorkspaceId,
        }

class v1Note:
    def __init__(
        self,
        *,
        contents: str,
        name: str,
    ):
        self.name = name
        self.contents = contents

    @classmethod
    def from_json(cls, obj: Json) -> "v1Note":
        return cls(
            name=obj["name"],
            contents=obj["contents"],
        )

    def to_json(self) -> typing.Any:
        return {
            "name": self.name,
            "contents": self.contents,
        }

class v1Notebook:
    def __init__(
        self,
        *,
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
        *,
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

class v1PatchExperiment:
    def __init__(
        self,
        *,
        id: int,
        description: "typing.Optional[str]" = None,
        labels: "typing.Optional[typing.Sequence[str]]" = None,
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
        *,
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
        *,
        description: "typing.Optional[str]" = None,
        labels: "typing.Optional[typing.Sequence[str]]" = None,
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
        *,
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
        *,
        checkpoint: "typing.Optional[v1Checkpoint]" = None,
        comment: "typing.Optional[str]" = None,
        labels: "typing.Optional[typing.Sequence[str]]" = None,
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
        *,
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

class v1PatchProject:
    def __init__(
        self,
        *,
        description: "typing.Optional[str]" = None,
        name: "typing.Optional[str]" = None,
    ):
        self.name = name
        self.description = description

    @classmethod
    def from_json(cls, obj: Json) -> "v1PatchProject":
        return cls(
            name=obj.get("name", None),
            description=obj.get("description", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "name": self.name if self.name is not None else None,
            "description": self.description if self.description is not None else None,
        }

class v1PatchProjectResponse:
    def __init__(
        self,
        *,
        project: "v1Project",
    ):
        self.project = project

    @classmethod
    def from_json(cls, obj: Json) -> "v1PatchProjectResponse":
        return cls(
            project=v1Project.from_json(obj["project"]),
        )

    def to_json(self) -> typing.Any:
        return {
            "project": self.project.to_json(),
        }

class v1PatchTrialsCollectionRequest:
    def __init__(
        self,
        *,
        id: int,
        filters: "typing.Optional[v1TrialFilters]" = None,
        name: "typing.Optional[str]" = None,
        sorter: "typing.Optional[v1TrialSorter]" = None,
    ):
        self.id = id
        self.name = name
        self.filters = filters
        self.sorter = sorter

    @classmethod
    def from_json(cls, obj: Json) -> "v1PatchTrialsCollectionRequest":
        return cls(
            id=obj["id"],
            name=obj.get("name", None),
            filters=v1TrialFilters.from_json(obj["filters"]) if obj.get("filters", None) is not None else None,
            sorter=v1TrialSorter.from_json(obj["sorter"]) if obj.get("sorter", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "id": self.id,
            "name": self.name if self.name is not None else None,
            "filters": self.filters.to_json() if self.filters is not None else None,
            "sorter": self.sorter.to_json() if self.sorter is not None else None,
        }

class v1PatchTrialsCollectionResponse:
    def __init__(
        self,
        *,
        collection: "typing.Optional[v1TrialsCollection]" = None,
    ):
        self.collection = collection

    @classmethod
    def from_json(cls, obj: Json) -> "v1PatchTrialsCollectionResponse":
        return cls(
            collection=v1TrialsCollection.from_json(obj["collection"]) if obj.get("collection", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "collection": self.collection.to_json() if self.collection is not None else None,
        }

class v1PatchUser:
    def __init__(
        self,
        *,
        active: "typing.Optional[bool]" = None,
        admin: "typing.Optional[bool]" = None,
        agentUserGroup: "typing.Optional[v1AgentUserGroup]" = None,
        displayName: "typing.Optional[str]" = None,
    ):
        self.admin = admin
        self.active = active
        self.displayName = displayName
        self.agentUserGroup = agentUserGroup

    @classmethod
    def from_json(cls, obj: Json) -> "v1PatchUser":
        return cls(
            admin=obj.get("admin", None),
            active=obj.get("active", None),
            displayName=obj.get("displayName", None),
            agentUserGroup=v1AgentUserGroup.from_json(obj["agentUserGroup"]) if obj.get("agentUserGroup", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "admin": self.admin if self.admin is not None else None,
            "active": self.active if self.active is not None else None,
            "displayName": self.displayName if self.displayName is not None else None,
            "agentUserGroup": self.agentUserGroup.to_json() if self.agentUserGroup is not None else None,
        }

class v1PatchUserResponse:
    def __init__(
        self,
        *,
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

class v1PatchWorkspace:
    def __init__(
        self,
        *,
        agentUserGroup: "typing.Optional[v1AgentUserGroup]" = None,
        name: "typing.Optional[str]" = None,
    ):
        self.name = name
        self.agentUserGroup = agentUserGroup

    @classmethod
    def from_json(cls, obj: Json) -> "v1PatchWorkspace":
        return cls(
            name=obj.get("name", None),
            agentUserGroup=v1AgentUserGroup.from_json(obj["agentUserGroup"]) if obj.get("agentUserGroup", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "name": self.name if self.name is not None else None,
            "agentUserGroup": self.agentUserGroup.to_json() if self.agentUserGroup is not None else None,
        }

class v1PatchWorkspaceResponse:
    def __init__(
        self,
        *,
        workspace: "v1Workspace",
    ):
        self.workspace = workspace

    @classmethod
    def from_json(cls, obj: Json) -> "v1PatchWorkspaceResponse":
        return cls(
            workspace=v1Workspace.from_json(obj["workspace"]),
        )

    def to_json(self) -> typing.Any:
        return {
            "workspace": self.workspace.to_json(),
        }

class v1Permission:
    def __init__(
        self,
        *,
        id: "v1PermissionType",
        isGlobal: "typing.Optional[bool]" = None,
        name: "typing.Optional[str]" = None,
    ):
        self.id = id
        self.name = name
        self.isGlobal = isGlobal

    @classmethod
    def from_json(cls, obj: Json) -> "v1Permission":
        return cls(
            id=v1PermissionType(obj["id"]),
            name=obj.get("name", None),
            isGlobal=obj.get("isGlobal", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "id": self.id.value,
            "name": self.name if self.name is not None else None,
            "isGlobal": self.isGlobal if self.isGlobal is not None else None,
        }

class v1PermissionType(enum.Enum):
    PERMISSION_TYPE_UNSPECIFIED = "PERMISSION_TYPE_UNSPECIFIED"
    PERMISSION_TYPE_ADMINISTRATE_USER = "PERMISSION_TYPE_ADMINISTRATE_USER"
    PERMISSION_TYPE_CREATE_EXPERIMENT = "PERMISSION_TYPE_CREATE_EXPERIMENT"
    PERMISSION_TYPE_VIEW_EXPERIMENT_ARTIFACTS = "PERMISSION_TYPE_VIEW_EXPERIMENT_ARTIFACTS"
    PERMISSION_TYPE_VIEW_EXPERIMENT_METADATA = "PERMISSION_TYPE_VIEW_EXPERIMENT_METADATA"
    PERMISSION_TYPE_UPDATE_EXPERIMENT = "PERMISSION_TYPE_UPDATE_EXPERIMENT"
    PERMISSION_TYPE_UPDATE_EXPERIMENT_METADATA = "PERMISSION_TYPE_UPDATE_EXPERIMENT_METADATA"
    PERMISSION_TYPE_DELETE_EXPERIMENT = "PERMISSION_TYPE_DELETE_EXPERIMENT"
    PERMISSION_TYPE_UPDATE_GROUP = "PERMISSION_TYPE_UPDATE_GROUP"
    PERMISSION_TYPE_CREATE_WORKSPACE = "PERMISSION_TYPE_CREATE_WORKSPACE"
    PERMISSION_TYPE_VIEW_WORKSPACE = "PERMISSION_TYPE_VIEW_WORKSPACE"
    PERMISSION_TYPE_UPDATE_WORKSPACE = "PERMISSION_TYPE_UPDATE_WORKSPACE"
    PERMISSION_TYPE_DELETE_WORKSPACE = "PERMISSION_TYPE_DELETE_WORKSPACE"
    PERMISSION_TYPE_SET_WORKSPACE_AGENT_USER_GROUP = "PERMISSION_TYPE_SET_WORKSPACE_AGENT_USER_GROUP"
    PERMISSION_TYPE_CREATE_PROJECT = "PERMISSION_TYPE_CREATE_PROJECT"
    PERMISSION_TYPE_VIEW_PROJECT = "PERMISSION_TYPE_VIEW_PROJECT"
    PERMISSION_TYPE_UPDATE_PROJECT = "PERMISSION_TYPE_UPDATE_PROJECT"
    PERMISSION_TYPE_DELETE_PROJECT = "PERMISSION_TYPE_DELETE_PROJECT"
    PERMISSION_TYPE_UPDATE_ROLES = "PERMISSION_TYPE_UPDATE_ROLES"
    PERMISSION_TYPE_ASSIGN_ROLES = "PERMISSION_TYPE_ASSIGN_ROLES"
    PERMISSION_TYPE_EDIT_WEBHOOKS = "PERMISSION_TYPE_EDIT_WEBHOOKS"

class v1PostAllocationProxyAddressRequest:
    def __init__(
        self,
        *,
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
        *,
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
        *,
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
        *,
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
        *,
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
        *,
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
        *,
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

class v1PostProjectRequest:
    def __init__(
        self,
        *,
        name: str,
        workspaceId: int,
        description: "typing.Optional[str]" = None,
    ):
        self.name = name
        self.description = description
        self.workspaceId = workspaceId

    @classmethod
    def from_json(cls, obj: Json) -> "v1PostProjectRequest":
        return cls(
            name=obj["name"],
            description=obj.get("description", None),
            workspaceId=obj["workspaceId"],
        )

    def to_json(self) -> typing.Any:
        return {
            "name": self.name,
            "description": self.description if self.description is not None else None,
            "workspaceId": self.workspaceId,
        }

class v1PostProjectResponse:
    def __init__(
        self,
        *,
        project: "v1Project",
    ):
        self.project = project

    @classmethod
    def from_json(cls, obj: Json) -> "v1PostProjectResponse":
        return cls(
            project=v1Project.from_json(obj["project"]),
        )

    def to_json(self) -> typing.Any:
        return {
            "project": self.project.to_json(),
        }

class v1PostSearcherOperationsRequest:
    def __init__(
        self,
        *,
        experimentId: "typing.Optional[int]" = None,
        searcherOperations: "typing.Optional[typing.Sequence[v1SearcherOperation]]" = None,
        triggeredByEvent: "typing.Optional[v1SearcherEvent]" = None,
    ):
        self.experimentId = experimentId
        self.searcherOperations = searcherOperations
        self.triggeredByEvent = triggeredByEvent

    @classmethod
    def from_json(cls, obj: Json) -> "v1PostSearcherOperationsRequest":
        return cls(
            experimentId=obj.get("experimentId", None),
            searcherOperations=[v1SearcherOperation.from_json(x) for x in obj["searcherOperations"]] if obj.get("searcherOperations", None) is not None else None,
            triggeredByEvent=v1SearcherEvent.from_json(obj["triggeredByEvent"]) if obj.get("triggeredByEvent", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "experimentId": self.experimentId if self.experimentId is not None else None,
            "searcherOperations": [x.to_json() for x in self.searcherOperations] if self.searcherOperations is not None else None,
            "triggeredByEvent": self.triggeredByEvent.to_json() if self.triggeredByEvent is not None else None,
        }

class v1PostTrialProfilerMetricsBatchRequest:
    def __init__(
        self,
        *,
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
        *,
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
        *,
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

class v1PostUserSettingRequest:
    def __init__(
        self,
        *,
        setting: "v1UserWebSetting",
        storagePath: str,
    ):
        self.storagePath = storagePath
        self.setting = setting

    @classmethod
    def from_json(cls, obj: Json) -> "v1PostUserSettingRequest":
        return cls(
            storagePath=obj["storagePath"],
            setting=v1UserWebSetting.from_json(obj["setting"]),
        )

    def to_json(self) -> typing.Any:
        return {
            "storagePath": self.storagePath,
            "setting": self.setting.to_json(),
        }

class v1PostWebhookResponse:
    def __init__(
        self,
        *,
        webhook: "v1Webhook",
    ):
        self.webhook = webhook

    @classmethod
    def from_json(cls, obj: Json) -> "v1PostWebhookResponse":
        return cls(
            webhook=v1Webhook.from_json(obj["webhook"]),
        )

    def to_json(self) -> typing.Any:
        return {
            "webhook": self.webhook.to_json(),
        }

class v1PostWorkspaceRequest:
    def __init__(
        self,
        *,
        name: str,
        agentUserGroup: "typing.Optional[v1AgentUserGroup]" = None,
    ):
        self.name = name
        self.agentUserGroup = agentUserGroup

    @classmethod
    def from_json(cls, obj: Json) -> "v1PostWorkspaceRequest":
        return cls(
            name=obj["name"],
            agentUserGroup=v1AgentUserGroup.from_json(obj["agentUserGroup"]) if obj.get("agentUserGroup", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "name": self.name,
            "agentUserGroup": self.agentUserGroup.to_json() if self.agentUserGroup is not None else None,
        }

class v1PostWorkspaceResponse:
    def __init__(
        self,
        *,
        workspace: "v1Workspace",
    ):
        self.workspace = workspace

    @classmethod
    def from_json(cls, obj: Json) -> "v1PostWorkspaceResponse":
        return cls(
            workspace=v1Workspace.from_json(obj["workspace"]),
        )

    def to_json(self) -> typing.Any:
        return {
            "workspace": self.workspace.to_json(),
        }

class v1PreviewHPSearchRequest:
    def __init__(
        self,
        *,
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
        *,
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

class v1Project:
    def __init__(
        self,
        *,
        archived: bool,
        errorMessage: str,
        id: int,
        immutable: bool,
        name: str,
        notes: "typing.Sequence[v1Note]",
        numActiveExperiments: int,
        numExperiments: int,
        state: "v1WorkspaceState",
        userId: int,
        username: str,
        workspaceId: int,
        description: "typing.Optional[str]" = None,
        lastExperimentStartedAt: "typing.Optional[str]" = None,
        workspaceName: "typing.Optional[str]" = None,
    ):
        self.id = id
        self.name = name
        self.workspaceId = workspaceId
        self.description = description
        self.lastExperimentStartedAt = lastExperimentStartedAt
        self.notes = notes
        self.numExperiments = numExperiments
        self.numActiveExperiments = numActiveExperiments
        self.archived = archived
        self.username = username
        self.immutable = immutable
        self.userId = userId
        self.workspaceName = workspaceName
        self.state = state
        self.errorMessage = errorMessage

    @classmethod
    def from_json(cls, obj: Json) -> "v1Project":
        return cls(
            id=obj["id"],
            name=obj["name"],
            workspaceId=obj["workspaceId"],
            description=obj.get("description", None),
            lastExperimentStartedAt=obj.get("lastExperimentStartedAt", None),
            notes=[v1Note.from_json(x) for x in obj["notes"]],
            numExperiments=obj["numExperiments"],
            numActiveExperiments=obj["numActiveExperiments"],
            archived=obj["archived"],
            username=obj["username"],
            immutable=obj["immutable"],
            userId=obj["userId"],
            workspaceName=obj.get("workspaceName", None),
            state=v1WorkspaceState(obj["state"]),
            errorMessage=obj["errorMessage"],
        )

    def to_json(self) -> typing.Any:
        return {
            "id": self.id,
            "name": self.name,
            "workspaceId": self.workspaceId,
            "description": self.description if self.description is not None else None,
            "lastExperimentStartedAt": self.lastExperimentStartedAt if self.lastExperimentStartedAt is not None else None,
            "notes": [x.to_json() for x in self.notes],
            "numExperiments": self.numExperiments,
            "numActiveExperiments": self.numActiveExperiments,
            "archived": self.archived,
            "username": self.username,
            "immutable": self.immutable,
            "userId": self.userId,
            "workspaceName": self.workspaceName if self.workspaceName is not None else None,
            "state": self.state.value,
            "errorMessage": self.errorMessage,
        }

class v1PutProjectNotesRequest:
    def __init__(
        self,
        *,
        notes: "typing.Sequence[v1Note]",
        projectId: int,
    ):
        self.notes = notes
        self.projectId = projectId

    @classmethod
    def from_json(cls, obj: Json) -> "v1PutProjectNotesRequest":
        return cls(
            notes=[v1Note.from_json(x) for x in obj["notes"]],
            projectId=obj["projectId"],
        )

    def to_json(self) -> typing.Any:
        return {
            "notes": [x.to_json() for x in self.notes],
            "projectId": self.projectId,
        }

class v1PutProjectNotesResponse:
    def __init__(
        self,
        *,
        notes: "typing.Sequence[v1Note]",
    ):
        self.notes = notes

    @classmethod
    def from_json(cls, obj: Json) -> "v1PutProjectNotesResponse":
        return cls(
            notes=[v1Note.from_json(x) for x in obj["notes"]],
        )

    def to_json(self) -> typing.Any:
        return {
            "notes": [x.to_json() for x in self.notes],
        }

class v1PutTemplateResponse:
    def __init__(
        self,
        *,
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

class v1QueryTrialsRequest:
    def __init__(
        self,
        *,
        filters: "v1TrialFilters",
        limit: "typing.Optional[int]" = None,
        offset: "typing.Optional[int]" = None,
        sorter: "typing.Optional[v1TrialSorter]" = None,
    ):
        self.filters = filters
        self.sorter = sorter
        self.offset = offset
        self.limit = limit

    @classmethod
    def from_json(cls, obj: Json) -> "v1QueryTrialsRequest":
        return cls(
            filters=v1TrialFilters.from_json(obj["filters"]),
            sorter=v1TrialSorter.from_json(obj["sorter"]) if obj.get("sorter", None) is not None else None,
            offset=obj.get("offset", None),
            limit=obj.get("limit", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "filters": self.filters.to_json(),
            "sorter": self.sorter.to_json() if self.sorter is not None else None,
            "offset": self.offset if self.offset is not None else None,
            "limit": self.limit if self.limit is not None else None,
        }

class v1QueryTrialsResponse:
    def __init__(
        self,
        *,
        trials: "typing.Sequence[v1AugmentedTrial]",
    ):
        self.trials = trials

    @classmethod
    def from_json(cls, obj: Json) -> "v1QueryTrialsResponse":
        return cls(
            trials=[v1AugmentedTrial.from_json(x) for x in obj["trials"]],
        )

    def to_json(self) -> typing.Any:
        return {
            "trials": [x.to_json() for x in self.trials],
        }

class v1QueueControl:
    def __init__(
        self,
        *,
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
        *,
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
        *,
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

class v1RemoveAssignmentsRequest:
    def __init__(
        self,
        *,
        groupRoleAssignments: "typing.Optional[typing.Sequence[v1GroupRoleAssignment]]" = None,
        userRoleAssignments: "typing.Optional[typing.Sequence[v1UserRoleAssignment]]" = None,
    ):
        self.groupRoleAssignments = groupRoleAssignments
        self.userRoleAssignments = userRoleAssignments

    @classmethod
    def from_json(cls, obj: Json) -> "v1RemoveAssignmentsRequest":
        return cls(
            groupRoleAssignments=[v1GroupRoleAssignment.from_json(x) for x in obj["groupRoleAssignments"]] if obj.get("groupRoleAssignments", None) is not None else None,
            userRoleAssignments=[v1UserRoleAssignment.from_json(x) for x in obj["userRoleAssignments"]] if obj.get("userRoleAssignments", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "groupRoleAssignments": [x.to_json() for x in self.groupRoleAssignments] if self.groupRoleAssignments is not None else None,
            "userRoleAssignments": [x.to_json() for x in self.userRoleAssignments] if self.userRoleAssignments is not None else None,
        }

class v1RendezvousInfo:
    def __init__(
        self,
        *,
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
        *,
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
        *,
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
        *,
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
        *,
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
        *,
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
        *,
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
        *,
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
        *,
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
        *,
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

class v1Role:
    def __init__(
        self,
        *,
        roleId: int,
        name: "typing.Optional[str]" = None,
        permissions: "typing.Optional[typing.Sequence[v1Permission]]" = None,
    ):
        self.roleId = roleId
        self.name = name
        self.permissions = permissions

    @classmethod
    def from_json(cls, obj: Json) -> "v1Role":
        return cls(
            roleId=obj["roleId"],
            name=obj.get("name", None),
            permissions=[v1Permission.from_json(x) for x in obj["permissions"]] if obj.get("permissions", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "roleId": self.roleId,
            "name": self.name if self.name is not None else None,
            "permissions": [x.to_json() for x in self.permissions] if self.permissions is not None else None,
        }

class v1RoleAssignment:
    def __init__(
        self,
        *,
        role: "v1Role",
        scopeWorkspaceId: "typing.Optional[int]" = None,
    ):
        self.role = role
        self.scopeWorkspaceId = scopeWorkspaceId

    @classmethod
    def from_json(cls, obj: Json) -> "v1RoleAssignment":
        return cls(
            role=v1Role.from_json(obj["role"]),
            scopeWorkspaceId=obj.get("scopeWorkspaceId", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "role": self.role.to_json(),
            "scopeWorkspaceId": self.scopeWorkspaceId if self.scopeWorkspaceId is not None else None,
        }

class v1RoleAssignmentSummary:
    def __init__(
        self,
        *,
        roleId: int,
        isGlobal: "typing.Optional[bool]" = None,
        scopeWorkspaceIds: "typing.Optional[typing.Sequence[int]]" = None,
    ):
        self.roleId = roleId
        self.scopeWorkspaceIds = scopeWorkspaceIds
        self.isGlobal = isGlobal

    @classmethod
    def from_json(cls, obj: Json) -> "v1RoleAssignmentSummary":
        return cls(
            roleId=obj["roleId"],
            scopeWorkspaceIds=obj.get("scopeWorkspaceIds", None),
            isGlobal=obj.get("isGlobal", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "roleId": self.roleId,
            "scopeWorkspaceIds": self.scopeWorkspaceIds if self.scopeWorkspaceIds is not None else None,
            "isGlobal": self.isGlobal if self.isGlobal is not None else None,
        }

class v1RoleWithAssignments:
    def __init__(
        self,
        *,
        groupRoleAssignments: "typing.Optional[typing.Sequence[v1GroupRoleAssignment]]" = None,
        role: "typing.Optional[v1Role]" = None,
        userRoleAssignments: "typing.Optional[typing.Sequence[v1UserRoleAssignment]]" = None,
    ):
        self.role = role
        self.groupRoleAssignments = groupRoleAssignments
        self.userRoleAssignments = userRoleAssignments

    @classmethod
    def from_json(cls, obj: Json) -> "v1RoleWithAssignments":
        return cls(
            role=v1Role.from_json(obj["role"]) if obj.get("role", None) is not None else None,
            groupRoleAssignments=[v1GroupRoleAssignment.from_json(x) for x in obj["groupRoleAssignments"]] if obj.get("groupRoleAssignments", None) is not None else None,
            userRoleAssignments=[v1UserRoleAssignment.from_json(x) for x in obj["userRoleAssignments"]] if obj.get("userRoleAssignments", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "role": self.role.to_json() if self.role is not None else None,
            "groupRoleAssignments": [x.to_json() for x in self.groupRoleAssignments] if self.groupRoleAssignments is not None else None,
            "userRoleAssignments": [x.to_json() for x in self.userRoleAssignments] if self.userRoleAssignments is not None else None,
        }

class v1RunnableOperation:
    def __init__(
        self,
        *,
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
        *,
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

class v1Scale(enum.Enum):
    SCALE_UNSPECIFIED = "SCALE_UNSPECIFIED"
    SCALE_LINEAR = "SCALE_LINEAR"
    SCALE_LOG = "SCALE_LOG"

class v1SchedulerType(enum.Enum):
    SCHEDULER_TYPE_UNSPECIFIED = "SCHEDULER_TYPE_UNSPECIFIED"
    SCHEDULER_TYPE_PRIORITY = "SCHEDULER_TYPE_PRIORITY"
    SCHEDULER_TYPE_FAIR_SHARE = "SCHEDULER_TYPE_FAIR_SHARE"
    SCHEDULER_TYPE_ROUND_ROBIN = "SCHEDULER_TYPE_ROUND_ROBIN"
    SCHEDULER_TYPE_KUBERNETES = "SCHEDULER_TYPE_KUBERNETES"
    SCHEDULER_TYPE_SLURM = "SCHEDULER_TYPE_SLURM"
    SCHEDULER_TYPE_PBS = "SCHEDULER_TYPE_PBS"

class v1SearchRolesAssignableToScopeRequest:
    def __init__(
        self,
        *,
        limit: int,
        offset: "typing.Optional[int]" = None,
        workspaceId: "typing.Optional[int]" = None,
    ):
        self.limit = limit
        self.offset = offset
        self.workspaceId = workspaceId

    @classmethod
    def from_json(cls, obj: Json) -> "v1SearchRolesAssignableToScopeRequest":
        return cls(
            limit=obj["limit"],
            offset=obj.get("offset", None),
            workspaceId=obj.get("workspaceId", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "limit": self.limit,
            "offset": self.offset if self.offset is not None else None,
            "workspaceId": self.workspaceId if self.workspaceId is not None else None,
        }

class v1SearchRolesAssignableToScopeResponse:
    def __init__(
        self,
        *,
        pagination: "typing.Optional[v1Pagination]" = None,
        roles: "typing.Optional[typing.Sequence[v1Role]]" = None,
    ):
        self.pagination = pagination
        self.roles = roles

    @classmethod
    def from_json(cls, obj: Json) -> "v1SearchRolesAssignableToScopeResponse":
        return cls(
            pagination=v1Pagination.from_json(obj["pagination"]) if obj.get("pagination", None) is not None else None,
            roles=[v1Role.from_json(x) for x in obj["roles"]] if obj.get("roles", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "pagination": self.pagination.to_json() if self.pagination is not None else None,
            "roles": [x.to_json() for x in self.roles] if self.roles is not None else None,
        }

class v1SearcherEvent:
    def __init__(
        self,
        *,
        id: int,
        experimentInactive: "typing.Optional[v1ExperimentInactive]" = None,
        initialOperations: "typing.Optional[v1InitialOperations]" = None,
        trialClosed: "typing.Optional[v1TrialClosed]" = None,
        trialCreated: "typing.Optional[v1TrialCreated]" = None,
        trialExitedEarly: "typing.Optional[v1TrialExitedEarly]" = None,
        trialProgress: "typing.Optional[v1TrialProgress]" = None,
        validationCompleted: "typing.Optional[v1ValidationCompleted]" = None,
    ):
        self.id = id
        self.initialOperations = initialOperations
        self.trialCreated = trialCreated
        self.validationCompleted = validationCompleted
        self.trialClosed = trialClosed
        self.trialExitedEarly = trialExitedEarly
        self.trialProgress = trialProgress
        self.experimentInactive = experimentInactive

    @classmethod
    def from_json(cls, obj: Json) -> "v1SearcherEvent":
        return cls(
            id=obj["id"],
            initialOperations=v1InitialOperations.from_json(obj["initialOperations"]) if obj.get("initialOperations", None) is not None else None,
            trialCreated=v1TrialCreated.from_json(obj["trialCreated"]) if obj.get("trialCreated", None) is not None else None,
            validationCompleted=v1ValidationCompleted.from_json(obj["validationCompleted"]) if obj.get("validationCompleted", None) is not None else None,
            trialClosed=v1TrialClosed.from_json(obj["trialClosed"]) if obj.get("trialClosed", None) is not None else None,
            trialExitedEarly=v1TrialExitedEarly.from_json(obj["trialExitedEarly"]) if obj.get("trialExitedEarly", None) is not None else None,
            trialProgress=v1TrialProgress.from_json(obj["trialProgress"]) if obj.get("trialProgress", None) is not None else None,
            experimentInactive=v1ExperimentInactive.from_json(obj["experimentInactive"]) if obj.get("experimentInactive", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "id": self.id,
            "initialOperations": self.initialOperations.to_json() if self.initialOperations is not None else None,
            "trialCreated": self.trialCreated.to_json() if self.trialCreated is not None else None,
            "validationCompleted": self.validationCompleted.to_json() if self.validationCompleted is not None else None,
            "trialClosed": self.trialClosed.to_json() if self.trialClosed is not None else None,
            "trialExitedEarly": self.trialExitedEarly.to_json() if self.trialExitedEarly is not None else None,
            "trialProgress": self.trialProgress.to_json() if self.trialProgress is not None else None,
            "experimentInactive": self.experimentInactive.to_json() if self.experimentInactive is not None else None,
        }

class v1SearcherOperation:
    def __init__(
        self,
        *,
        closeTrial: "typing.Optional[v1CloseTrialOperation]" = None,
        createTrial: "typing.Optional[v1CreateTrialOperation]" = None,
        setSearcherProgress: "typing.Optional[v1SetSearcherProgressOperation]" = None,
        shutDown: "typing.Optional[v1ShutDownOperation]" = None,
        trialOperation: "typing.Optional[v1TrialOperation]" = None,
    ):
        self.trialOperation = trialOperation
        self.createTrial = createTrial
        self.closeTrial = closeTrial
        self.shutDown = shutDown
        self.setSearcherProgress = setSearcherProgress

    @classmethod
    def from_json(cls, obj: Json) -> "v1SearcherOperation":
        return cls(
            trialOperation=v1TrialOperation.from_json(obj["trialOperation"]) if obj.get("trialOperation", None) is not None else None,
            createTrial=v1CreateTrialOperation.from_json(obj["createTrial"]) if obj.get("createTrial", None) is not None else None,
            closeTrial=v1CloseTrialOperation.from_json(obj["closeTrial"]) if obj.get("closeTrial", None) is not None else None,
            shutDown=v1ShutDownOperation.from_json(obj["shutDown"]) if obj.get("shutDown", None) is not None else None,
            setSearcherProgress=v1SetSearcherProgressOperation.from_json(obj["setSearcherProgress"]) if obj.get("setSearcherProgress", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "trialOperation": self.trialOperation.to_json() if self.trialOperation is not None else None,
            "createTrial": self.createTrial.to_json() if self.createTrial is not None else None,
            "closeTrial": self.closeTrial.to_json() if self.closeTrial is not None else None,
            "shutDown": self.shutDown.to_json() if self.shutDown is not None else None,
            "setSearcherProgress": self.setSearcherProgress.to_json() if self.setSearcherProgress is not None else None,
        }

class v1SetCommandPriorityRequest:
    def __init__(
        self,
        *,
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
        *,
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
        *,
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
        *,
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

class v1SetSearcherProgressOperation:
    def __init__(
        self,
        *,
        progress: "typing.Optional[float]" = None,
    ):
        self.progress = progress

    @classmethod
    def from_json(cls, obj: Json) -> "v1SetSearcherProgressOperation":
        return cls(
            progress=float(obj["progress"]) if obj.get("progress", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "progress": dump_float(self.progress) if self.progress is not None else None,
        }

class v1SetShellPriorityRequest:
    def __init__(
        self,
        *,
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
        *,
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
        *,
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
        *,
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
        *,
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
        *,
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

class v1ShutDownOperation:
    def __init__(
        self,
        *,
        placeholder: "typing.Optional[int]" = None,
    ):
        self.placeholder = placeholder

    @classmethod
    def from_json(cls, obj: Json) -> "v1ShutDownOperation":
        return cls(
            placeholder=obj.get("placeholder", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "placeholder": self.placeholder if self.placeholder is not None else None,
        }

class v1Slot:
    def __init__(
        self,
        *,
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

class v1SummarizeTrialResponse:
    def __init__(
        self,
        *,
        metrics: "typing.Sequence[v1SummarizedMetric]",
        trial: "trialv1Trial",
    ):
        self.trial = trial
        self.metrics = metrics

    @classmethod
    def from_json(cls, obj: Json) -> "v1SummarizeTrialResponse":
        return cls(
            trial=trialv1Trial.from_json(obj["trial"]),
            metrics=[v1SummarizedMetric.from_json(x) for x in obj["metrics"]],
        )

    def to_json(self) -> typing.Any:
        return {
            "trial": self.trial.to_json(),
            "metrics": [x.to_json() for x in self.metrics],
        }

class v1SummarizedMetric:
    def __init__(
        self,
        *,
        data: "typing.Sequence[v1DataPoint]",
        name: str,
        type: "v1MetricType",
    ):
        self.name = name
        self.data = data
        self.type = type

    @classmethod
    def from_json(cls, obj: Json) -> "v1SummarizedMetric":
        return cls(
            name=obj["name"],
            data=[v1DataPoint.from_json(x) for x in obj["data"]],
            type=v1MetricType(obj["type"]),
        )

    def to_json(self) -> typing.Any:
        return {
            "name": self.name,
            "data": [x.to_json() for x in self.data],
            "type": self.type.value,
        }

class v1Task:
    def __init__(
        self,
        *,
        allocations: "typing.Optional[typing.Sequence[v1Allocation]]" = None,
        taskId: "typing.Optional[str]" = None,
        taskType: "typing.Optional[str]" = None,
    ):
        self.taskId = taskId
        self.taskType = taskType
        self.allocations = allocations

    @classmethod
    def from_json(cls, obj: Json) -> "v1Task":
        return cls(
            taskId=obj.get("taskId", None),
            taskType=obj.get("taskType", None),
            allocations=[v1Allocation.from_json(x) for x in obj["allocations"]] if obj.get("allocations", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "taskId": self.taskId if self.taskId is not None else None,
            "taskType": self.taskType if self.taskType is not None else None,
            "allocations": [x.to_json() for x in self.allocations] if self.allocations is not None else None,
        }

class v1TaskLogsFieldsResponse:
    def __init__(
        self,
        *,
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
        *,
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
        *,
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
        *,
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

class v1TestWebhookResponse:
    def __init__(
        self,
        *,
        completed: bool,
    ):
        self.completed = completed

    @classmethod
    def from_json(cls, obj: Json) -> "v1TestWebhookResponse":
        return cls(
            completed=obj["completed"],
        )

    def to_json(self) -> typing.Any:
        return {
            "completed": self.completed,
        }

class v1TimestampFieldFilter:
    def __init__(
        self,
        *,
        gt: "typing.Optional[str]" = None,
        gte: "typing.Optional[str]" = None,
        lt: "typing.Optional[str]" = None,
        lte: "typing.Optional[str]" = None,
    ):
        self.lt = lt
        self.lte = lte
        self.gt = gt
        self.gte = gte

    @classmethod
    def from_json(cls, obj: Json) -> "v1TimestampFieldFilter":
        return cls(
            lt=obj.get("lt", None),
            lte=obj.get("lte", None),
            gt=obj.get("gt", None),
            gte=obj.get("gte", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "lt": self.lt if self.lt is not None else None,
            "lte": self.lte if self.lte is not None else None,
            "gt": self.gt if self.gt is not None else None,
            "gte": self.gte if self.gte is not None else None,
        }

class v1TrialClosed:
    def __init__(
        self,
        *,
        requestId: str,
    ):
        self.requestId = requestId

    @classmethod
    def from_json(cls, obj: Json) -> "v1TrialClosed":
        return cls(
            requestId=obj["requestId"],
        )

    def to_json(self) -> typing.Any:
        return {
            "requestId": self.requestId,
        }

class v1TrialCreated:
    def __init__(
        self,
        *,
        requestId: str,
    ):
        self.requestId = requestId

    @classmethod
    def from_json(cls, obj: Json) -> "v1TrialCreated":
        return cls(
            requestId=obj["requestId"],
        )

    def to_json(self) -> typing.Any:
        return {
            "requestId": self.requestId,
        }

class v1TrialEarlyExit:
    def __init__(
        self,
        *,
        reason: "v1TrialEarlyExitExitedReason",
    ):
        self.reason = reason

    @classmethod
    def from_json(cls, obj: Json) -> "v1TrialEarlyExit":
        return cls(
            reason=v1TrialEarlyExitExitedReason(obj["reason"]),
        )

    def to_json(self) -> typing.Any:
        return {
            "reason": self.reason.value,
        }

class v1TrialEarlyExitExitedReason(enum.Enum):
    EXITED_REASON_UNSPECIFIED = "EXITED_REASON_UNSPECIFIED"
    EXITED_REASON_INVALID_HP = "EXITED_REASON_INVALID_HP"
    EXITED_REASON_INIT_INVALID_HP = "EXITED_REASON_INIT_INVALID_HP"

class v1TrialExitedEarly:
    def __init__(
        self,
        *,
        exitedReason: "v1TrialExitedEarlyExitedReason",
        requestId: str,
    ):
        self.requestId = requestId
        self.exitedReason = exitedReason

    @classmethod
    def from_json(cls, obj: Json) -> "v1TrialExitedEarly":
        return cls(
            requestId=obj["requestId"],
            exitedReason=v1TrialExitedEarlyExitedReason(obj["exitedReason"]),
        )

    def to_json(self) -> typing.Any:
        return {
            "requestId": self.requestId,
            "exitedReason": self.exitedReason.value,
        }

class v1TrialExitedEarlyExitedReason(enum.Enum):
    EXITED_REASON_UNSPECIFIED = "EXITED_REASON_UNSPECIFIED"
    EXITED_REASON_INVALID_HP = "EXITED_REASON_INVALID_HP"
    EXITED_REASON_USER_REQUESTED_STOP = "EXITED_REASON_USER_REQUESTED_STOP"

class v1TrialFilters:
    def __init__(
        self,
        *,
        endTime: "typing.Optional[v1TimestampFieldFilter]" = None,
        experimentIds: "typing.Optional[typing.Sequence[int]]" = None,
        hparams: "typing.Optional[typing.Sequence[v1ColumnFilter]]" = None,
        projectIds: "typing.Optional[typing.Sequence[int]]" = None,
        rankWithinExp: "typing.Optional[TrialFiltersRankWithinExp]" = None,
        searcher: "typing.Optional[str]" = None,
        searcherMetric: "typing.Optional[str]" = None,
        searcherMetricValue: "typing.Optional[v1DoubleFieldFilter]" = None,
        startTime: "typing.Optional[v1TimestampFieldFilter]" = None,
        states: "typing.Optional[typing.Sequence[determinedtrialv1State]]" = None,
        tags: "typing.Optional[typing.Sequence[v1TrialTag]]" = None,
        trainingMetrics: "typing.Optional[typing.Sequence[v1ColumnFilter]]" = None,
        trialIds: "typing.Optional[typing.Sequence[int]]" = None,
        userIds: "typing.Optional[typing.Sequence[int]]" = None,
        validationMetrics: "typing.Optional[typing.Sequence[v1ColumnFilter]]" = None,
        workspaceIds: "typing.Optional[typing.Sequence[int]]" = None,
    ):
        self.experimentIds = experimentIds
        self.projectIds = projectIds
        self.workspaceIds = workspaceIds
        self.validationMetrics = validationMetrics
        self.trainingMetrics = trainingMetrics
        self.hparams = hparams
        self.userIds = userIds
        self.searcher = searcher
        self.tags = tags
        self.rankWithinExp = rankWithinExp
        self.startTime = startTime
        self.endTime = endTime
        self.states = states
        self.searcherMetric = searcherMetric
        self.searcherMetricValue = searcherMetricValue
        self.trialIds = trialIds

    @classmethod
    def from_json(cls, obj: Json) -> "v1TrialFilters":
        return cls(
            experimentIds=obj.get("experimentIds", None),
            projectIds=obj.get("projectIds", None),
            workspaceIds=obj.get("workspaceIds", None),
            validationMetrics=[v1ColumnFilter.from_json(x) for x in obj["validationMetrics"]] if obj.get("validationMetrics", None) is not None else None,
            trainingMetrics=[v1ColumnFilter.from_json(x) for x in obj["trainingMetrics"]] if obj.get("trainingMetrics", None) is not None else None,
            hparams=[v1ColumnFilter.from_json(x) for x in obj["hparams"]] if obj.get("hparams", None) is not None else None,
            userIds=obj.get("userIds", None),
            searcher=obj.get("searcher", None),
            tags=[v1TrialTag.from_json(x) for x in obj["tags"]] if obj.get("tags", None) is not None else None,
            rankWithinExp=TrialFiltersRankWithinExp.from_json(obj["rankWithinExp"]) if obj.get("rankWithinExp", None) is not None else None,
            startTime=v1TimestampFieldFilter.from_json(obj["startTime"]) if obj.get("startTime", None) is not None else None,
            endTime=v1TimestampFieldFilter.from_json(obj["endTime"]) if obj.get("endTime", None) is not None else None,
            states=[determinedtrialv1State(x) for x in obj["states"]] if obj.get("states", None) is not None else None,
            searcherMetric=obj.get("searcherMetric", None),
            searcherMetricValue=v1DoubleFieldFilter.from_json(obj["searcherMetricValue"]) if obj.get("searcherMetricValue", None) is not None else None,
            trialIds=obj.get("trialIds", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "experimentIds": self.experimentIds if self.experimentIds is not None else None,
            "projectIds": self.projectIds if self.projectIds is not None else None,
            "workspaceIds": self.workspaceIds if self.workspaceIds is not None else None,
            "validationMetrics": [x.to_json() for x in self.validationMetrics] if self.validationMetrics is not None else None,
            "trainingMetrics": [x.to_json() for x in self.trainingMetrics] if self.trainingMetrics is not None else None,
            "hparams": [x.to_json() for x in self.hparams] if self.hparams is not None else None,
            "userIds": self.userIds if self.userIds is not None else None,
            "searcher": self.searcher if self.searcher is not None else None,
            "tags": [x.to_json() for x in self.tags] if self.tags is not None else None,
            "rankWithinExp": self.rankWithinExp.to_json() if self.rankWithinExp is not None else None,
            "startTime": self.startTime.to_json() if self.startTime is not None else None,
            "endTime": self.endTime.to_json() if self.endTime is not None else None,
            "states": [x.value for x in self.states] if self.states is not None else None,
            "searcherMetric": self.searcherMetric if self.searcherMetric is not None else None,
            "searcherMetricValue": self.searcherMetricValue.to_json() if self.searcherMetricValue is not None else None,
            "trialIds": self.trialIds if self.trialIds is not None else None,
        }

class v1TrialLogsFieldsResponse:
    def __init__(
        self,
        *,
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
        *,
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
        *,
        metrics: "v1Metrics",
        stepsCompleted: int,
        trialId: int,
        trialRunId: int,
    ):
        self.trialId = trialId
        self.trialRunId = trialRunId
        self.stepsCompleted = stepsCompleted
        self.metrics = metrics

    @classmethod
    def from_json(cls, obj: Json) -> "v1TrialMetrics":
        return cls(
            trialId=obj["trialId"],
            trialRunId=obj["trialRunId"],
            stepsCompleted=obj["stepsCompleted"],
            metrics=v1Metrics.from_json(obj["metrics"]),
        )

    def to_json(self) -> typing.Any:
        return {
            "trialId": self.trialId,
            "trialRunId": self.trialRunId,
            "stepsCompleted": self.stepsCompleted,
            "metrics": self.metrics.to_json(),
        }

class v1TrialOperation:
    def __init__(
        self,
        *,
        validateAfter: "typing.Optional[v1ValidateAfterOperation]" = None,
    ):
        self.validateAfter = validateAfter

    @classmethod
    def from_json(cls, obj: Json) -> "v1TrialOperation":
        return cls(
            validateAfter=v1ValidateAfterOperation.from_json(obj["validateAfter"]) if obj.get("validateAfter", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "validateAfter": self.validateAfter.to_json() if self.validateAfter is not None else None,
        }

class v1TrialPatch:
    def __init__(
        self,
        *,
        addTag: "typing.Optional[typing.Sequence[v1TrialTag]]" = None,
        removeTag: "typing.Optional[typing.Sequence[v1TrialTag]]" = None,
    ):
        self.addTag = addTag
        self.removeTag = removeTag

    @classmethod
    def from_json(cls, obj: Json) -> "v1TrialPatch":
        return cls(
            addTag=[v1TrialTag.from_json(x) for x in obj["addTag"]] if obj.get("addTag", None) is not None else None,
            removeTag=[v1TrialTag.from_json(x) for x in obj["removeTag"]] if obj.get("removeTag", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "addTag": [x.to_json() for x in self.addTag] if self.addTag is not None else None,
            "removeTag": [x.to_json() for x in self.removeTag] if self.removeTag is not None else None,
        }

class v1TrialProfilerMetricLabels:
    def __init__(
        self,
        *,
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
        *,
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

class v1TrialProgress:
    def __init__(
        self,
        *,
        partialUnits: float,
        requestId: str,
    ):
        self.requestId = requestId
        self.partialUnits = partialUnits

    @classmethod
    def from_json(cls, obj: Json) -> "v1TrialProgress":
        return cls(
            requestId=obj["requestId"],
            partialUnits=float(obj["partialUnits"]),
        )

    def to_json(self) -> typing.Any:
        return {
            "requestId": self.requestId,
            "partialUnits": dump_float(self.partialUnits),
        }

class v1TrialRunnerMetadata:
    def __init__(
        self,
        *,
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
        *,
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

class v1TrialSorter:
    def __init__(
        self,
        *,
        field: str,
        namespace: "TrialSorterNamespace",
        orderBy: "typing.Optional[v1OrderBy]" = None,
    ):
        self.namespace = namespace
        self.field = field
        self.orderBy = orderBy

    @classmethod
    def from_json(cls, obj: Json) -> "v1TrialSorter":
        return cls(
            namespace=TrialSorterNamespace(obj["namespace"]),
            field=obj["field"],
            orderBy=v1OrderBy(obj["orderBy"]) if obj.get("orderBy", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "namespace": self.namespace.value,
            "field": self.field,
            "orderBy": self.orderBy.value if self.orderBy is not None else None,
        }

class v1TrialTag:
    def __init__(
        self,
        *,
        key: str,
    ):
        self.key = key

    @classmethod
    def from_json(cls, obj: Json) -> "v1TrialTag":
        return cls(
            key=obj["key"],
        )

    def to_json(self) -> typing.Any:
        return {
            "key": self.key,
        }

class v1TrialsCollection:
    def __init__(
        self,
        *,
        filters: "v1TrialFilters",
        id: int,
        name: str,
        projectId: int,
        sorter: "v1TrialSorter",
        userId: int,
    ):
        self.id = id
        self.userId = userId
        self.projectId = projectId
        self.name = name
        self.filters = filters
        self.sorter = sorter

    @classmethod
    def from_json(cls, obj: Json) -> "v1TrialsCollection":
        return cls(
            id=obj["id"],
            userId=obj["userId"],
            projectId=obj["projectId"],
            name=obj["name"],
            filters=v1TrialFilters.from_json(obj["filters"]),
            sorter=v1TrialSorter.from_json(obj["sorter"]),
        )

    def to_json(self) -> typing.Any:
        return {
            "id": self.id,
            "userId": self.userId,
            "projectId": self.projectId,
            "name": self.name,
            "filters": self.filters.to_json(),
            "sorter": self.sorter.to_json(),
        }

class v1TrialsSampleResponse:
    def __init__(
        self,
        *,
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
        *,
        data: "typing.Sequence[v1DataPoint]",
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
            data=[v1DataPoint.from_json(x) for x in obj["data"]],
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
        *,
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
        *,
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

class v1Trigger:
    def __init__(
        self,
        *,
        condition: "typing.Optional[typing.Dict[str, typing.Any]]" = None,
        id: "typing.Optional[int]" = None,
        triggerType: "typing.Optional[v1TriggerType]" = None,
        webhookId: "typing.Optional[int]" = None,
    ):
        self.id = id
        self.triggerType = triggerType
        self.condition = condition
        self.webhookId = webhookId

    @classmethod
    def from_json(cls, obj: Json) -> "v1Trigger":
        return cls(
            id=obj.get("id", None),
            triggerType=v1TriggerType(obj["triggerType"]) if obj.get("triggerType", None) is not None else None,
            condition=obj.get("condition", None),
            webhookId=obj.get("webhookId", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "id": self.id if self.id is not None else None,
            "triggerType": self.triggerType.value if self.triggerType is not None else None,
            "condition": self.condition if self.condition is not None else None,
            "webhookId": self.webhookId if self.webhookId is not None else None,
        }

class v1TriggerType(enum.Enum):
    TRIGGER_TYPE_UNSPECIFIED = "TRIGGER_TYPE_UNSPECIFIED"
    TRIGGER_TYPE_EXPERIMENT_STATE_CHANGE = "TRIGGER_TYPE_EXPERIMENT_STATE_CHANGE"
    TRIGGER_TYPE_METRIC_THRESHOLD_EXCEEDED = "TRIGGER_TYPE_METRIC_THRESHOLD_EXCEEDED"

class v1UpdateGroupRequest:
    def __init__(
        self,
        *,
        groupId: int,
        addUsers: "typing.Optional[typing.Sequence[int]]" = None,
        name: "typing.Optional[str]" = None,
        removeUsers: "typing.Optional[typing.Sequence[int]]" = None,
    ):
        self.groupId = groupId
        self.name = name
        self.addUsers = addUsers
        self.removeUsers = removeUsers

    @classmethod
    def from_json(cls, obj: Json) -> "v1UpdateGroupRequest":
        return cls(
            groupId=obj["groupId"],
            name=obj.get("name", None),
            addUsers=obj.get("addUsers", None),
            removeUsers=obj.get("removeUsers", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "groupId": self.groupId,
            "name": self.name if self.name is not None else None,
            "addUsers": self.addUsers if self.addUsers is not None else None,
            "removeUsers": self.removeUsers if self.removeUsers is not None else None,
        }

class v1UpdateGroupResponse:
    def __init__(
        self,
        *,
        group: "v1GroupDetails",
    ):
        self.group = group

    @classmethod
    def from_json(cls, obj: Json) -> "v1UpdateGroupResponse":
        return cls(
            group=v1GroupDetails.from_json(obj["group"]),
        )

    def to_json(self) -> typing.Any:
        return {
            "group": self.group.to_json(),
        }

class v1UpdateJobQueueRequest:
    def __init__(
        self,
        *,
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

class v1UpdateTrialTagsRequest:
    def __init__(
        self,
        *,
        patch: "v1TrialPatch",
        filters: "typing.Optional[v1TrialFilters]" = None,
        trial: "typing.Optional[UpdateTrialTagsRequestIds]" = None,
    ):
        self.filters = filters
        self.trial = trial
        self.patch = patch

    @classmethod
    def from_json(cls, obj: Json) -> "v1UpdateTrialTagsRequest":
        return cls(
            filters=v1TrialFilters.from_json(obj["filters"]) if obj.get("filters", None) is not None else None,
            trial=UpdateTrialTagsRequestIds.from_json(obj["trial"]) if obj.get("trial", None) is not None else None,
            patch=v1TrialPatch.from_json(obj["patch"]),
        )

    def to_json(self) -> typing.Any:
        return {
            "filters": self.filters.to_json() if self.filters is not None else None,
            "trial": self.trial.to_json() if self.trial is not None else None,
            "patch": self.patch.to_json(),
        }

class v1UpdateTrialTagsResponse:
    def __init__(
        self,
        *,
        rowsAffected: "typing.Optional[int]" = None,
    ):
        self.rowsAffected = rowsAffected

    @classmethod
    def from_json(cls, obj: Json) -> "v1UpdateTrialTagsResponse":
        return cls(
            rowsAffected=obj.get("rowsAffected", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "rowsAffected": self.rowsAffected if self.rowsAffected is not None else None,
        }

class v1User:
    def __init__(
        self,
        *,
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

class v1UserRoleAssignment:
    def __init__(
        self,
        *,
        roleAssignment: "v1RoleAssignment",
        userId: int,
    ):
        self.userId = userId
        self.roleAssignment = roleAssignment

    @classmethod
    def from_json(cls, obj: Json) -> "v1UserRoleAssignment":
        return cls(
            userId=obj["userId"],
            roleAssignment=v1RoleAssignment.from_json(obj["roleAssignment"]),
        )

    def to_json(self) -> typing.Any:
        return {
            "userId": self.userId,
            "roleAssignment": self.roleAssignment.to_json(),
        }

class v1UserWebSetting:
    def __init__(
        self,
        *,
        key: str,
        storagePath: "typing.Optional[str]" = None,
        value: "typing.Optional[str]" = None,
    ):
        self.key = key
        self.storagePath = storagePath
        self.value = value

    @classmethod
    def from_json(cls, obj: Json) -> "v1UserWebSetting":
        return cls(
            key=obj["key"],
            storagePath=obj.get("storagePath", None),
            value=obj.get("value", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "key": self.key,
            "storagePath": self.storagePath if self.storagePath is not None else None,
            "value": self.value if self.value is not None else None,
        }

class v1ValidateAfterOperation:
    def __init__(
        self,
        *,
        length: "typing.Optional[str]" = None,
        requestId: "typing.Optional[str]" = None,
    ):
        self.requestId = requestId
        self.length = length

    @classmethod
    def from_json(cls, obj: Json) -> "v1ValidateAfterOperation":
        return cls(
            requestId=obj.get("requestId", None),
            length=obj.get("length", None),
        )

    def to_json(self) -> typing.Any:
        return {
            "requestId": self.requestId if self.requestId is not None else None,
            "length": self.length if self.length is not None else None,
        }

class v1ValidationCompleted:
    def __init__(
        self,
        *,
        metric: float,
        requestId: str,
        validateAfterLength: str,
    ):
        self.requestId = requestId
        self.metric = metric
        self.validateAfterLength = validateAfterLength

    @classmethod
    def from_json(cls, obj: Json) -> "v1ValidationCompleted":
        return cls(
            requestId=obj["requestId"],
            metric=float(obj["metric"]),
            validateAfterLength=obj["validateAfterLength"],
        )

    def to_json(self) -> typing.Any:
        return {
            "requestId": self.requestId,
            "metric": dump_float(self.metric),
            "validateAfterLength": self.validateAfterLength,
        }

class v1ValidationHistoryEntry:
    def __init__(
        self,
        *,
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

class v1Webhook:
    def __init__(
        self,
        *,
        url: str,
        webhookType: "v1WebhookType",
        id: "typing.Optional[int]" = None,
        triggers: "typing.Optional[typing.Sequence[v1Trigger]]" = None,
    ):
        self.id = id
        self.url = url
        self.triggers = triggers
        self.webhookType = webhookType

    @classmethod
    def from_json(cls, obj: Json) -> "v1Webhook":
        return cls(
            id=obj.get("id", None),
            url=obj["url"],
            triggers=[v1Trigger.from_json(x) for x in obj["triggers"]] if obj.get("triggers", None) is not None else None,
            webhookType=v1WebhookType(obj["webhookType"]),
        )

    def to_json(self) -> typing.Any:
        return {
            "id": self.id if self.id is not None else None,
            "url": self.url,
            "triggers": [x.to_json() for x in self.triggers] if self.triggers is not None else None,
            "webhookType": self.webhookType.value,
        }

class v1WebhookType(enum.Enum):
    WEBHOOK_TYPE_UNSPECIFIED = "WEBHOOK_TYPE_UNSPECIFIED"
    WEBHOOK_TYPE_DEFAULT = "WEBHOOK_TYPE_DEFAULT"
    WEBHOOK_TYPE_SLACK = "WEBHOOK_TYPE_SLACK"

class v1WorkloadContainer:
    def __init__(
        self,
        *,
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

class v1Workspace:
    def __init__(
        self,
        *,
        archived: bool,
        errorMessage: str,
        id: int,
        immutable: bool,
        name: str,
        numExperiments: int,
        numProjects: int,
        pinned: bool,
        state: "v1WorkspaceState",
        userId: int,
        username: str,
        agentUserGroup: "typing.Optional[v1AgentUserGroup]" = None,
    ):
        self.id = id
        self.name = name
        self.archived = archived
        self.username = username
        self.immutable = immutable
        self.numProjects = numProjects
        self.pinned = pinned
        self.userId = userId
        self.numExperiments = numExperiments
        self.state = state
        self.errorMessage = errorMessage
        self.agentUserGroup = agentUserGroup

    @classmethod
    def from_json(cls, obj: Json) -> "v1Workspace":
        return cls(
            id=obj["id"],
            name=obj["name"],
            archived=obj["archived"],
            username=obj["username"],
            immutable=obj["immutable"],
            numProjects=obj["numProjects"],
            pinned=obj["pinned"],
            userId=obj["userId"],
            numExperiments=obj["numExperiments"],
            state=v1WorkspaceState(obj["state"]),
            errorMessage=obj["errorMessage"],
            agentUserGroup=v1AgentUserGroup.from_json(obj["agentUserGroup"]) if obj.get("agentUserGroup", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "id": self.id,
            "name": self.name,
            "archived": self.archived,
            "username": self.username,
            "immutable": self.immutable,
            "numProjects": self.numProjects,
            "pinned": self.pinned,
            "userId": self.userId,
            "numExperiments": self.numExperiments,
            "state": self.state.value,
            "errorMessage": self.errorMessage,
            "agentUserGroup": self.agentUserGroup.to_json() if self.agentUserGroup is not None else None,
        }

class v1WorkspaceState(enum.Enum):
    WORKSPACE_STATE_UNSPECIFIED = "WORKSPACE_STATE_UNSPECIFIED"
    WORKSPACE_STATE_DELETING = "WORKSPACE_STATE_DELETING"
    WORKSPACE_STATE_DELETE_FAILED = "WORKSPACE_STATE_DELETE_FAILED"
    WORKSPACE_STATE_DELETED = "WORKSPACE_STATE_DELETED"

def post_AckAllocationPreemptionSignal(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_AckAllocationPreemptionSignal", _resp)

def post_ActivateExperiment(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_ActivateExperiment", _resp)

def post_AddProjectNote(
    session: "api.Session",
    *,
    body: "v1Note",
    projectId: int,
) -> "v1AddProjectNoteResponse":
    _params = None
    _resp = session._do_request(
        method="POST",
        path=f"/api/v1/projects/{projectId}/notes",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return v1AddProjectNoteResponse.from_json(_resp.json())
    raise APIHttpError("post_AddProjectNote", _resp)

def post_AllocationAllGather(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1AllocationAllGatherResponse.from_json(_resp.json())
    raise APIHttpError("post_AllocationAllGather", _resp)

def post_AllocationPendingPreemptionSignal(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_AllocationPendingPreemptionSignal", _resp)

def get_AllocationPreemptionSignal(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1AllocationPreemptionSignalResponse.from_json(_resp.json())
    raise APIHttpError("get_AllocationPreemptionSignal", _resp)

def post_AllocationReady(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_AllocationReady", _resp)

def get_AllocationRendezvousInfo(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1AllocationRendezvousInfoResponse.from_json(_resp.json())
    raise APIHttpError("get_AllocationRendezvousInfo", _resp)

def post_AllocationWaiting(
    session: "api.Session",
    *,
    allocationId: str,
    body: "v1AllocationWaitingRequest",
) -> None:
    _params = None
    _resp = session._do_request(
        method="POST",
        path=f"/api/v1/allocations/{allocationId}/waiting",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_AllocationWaiting", _resp)

def post_ArchiveExperiment(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_ArchiveExperiment", _resp)

def post_ArchiveModel(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_ArchiveModel", _resp)

def post_ArchiveProject(
    session: "api.Session",
    *,
    id: int,
) -> None:
    _params = None
    _resp = session._do_request(
        method="POST",
        path=f"/api/v1/projects/{id}/archive",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_ArchiveProject", _resp)

def post_ArchiveWorkspace(
    session: "api.Session",
    *,
    id: int,
) -> None:
    _params = None
    _resp = session._do_request(
        method="POST",
        path=f"/api/v1/workspaces/{id}/archive",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_ArchiveWorkspace", _resp)

def post_AssignRoles(
    session: "api.Session",
    *,
    body: "v1AssignRolesRequest",
) -> None:
    _params = None
    _resp = session._do_request(
        method="POST",
        path="/api/v1/roles/add-assignments",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_AssignRoles", _resp)

def post_CancelExperiment(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_CancelExperiment", _resp)

def get_CompareTrials(
    session: "api.Session",
    *,
    endBatches: "typing.Optional[int]" = None,
    maxDatapoints: "typing.Optional[int]" = None,
    metricNames: "typing.Optional[typing.Sequence[str]]" = None,
    metricType: "typing.Optional[v1MetricType]" = None,
    scale: "typing.Optional[v1Scale]" = None,
    startBatches: "typing.Optional[int]" = None,
    trialIds: "typing.Optional[typing.Sequence[int]]" = None,
) -> "v1CompareTrialsResponse":
    _params = {
        "endBatches": endBatches,
        "maxDatapoints": maxDatapoints,
        "metricNames": metricNames,
        "metricType": metricType.value if metricType is not None else None,
        "scale": scale.value if scale is not None else None,
        "startBatches": startBatches,
        "trialIds": trialIds,
    }
    _resp = session._do_request(
        method="GET",
        path="/api/v1/trials/compare",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return v1CompareTrialsResponse.from_json(_resp.json())
    raise APIHttpError("get_CompareTrials", _resp)

def post_CompleteTrialSearcherValidation(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_CompleteTrialSearcherValidation", _resp)

def post_ComputeHPImportance(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_ComputeHPImportance", _resp)

def post_CreateExperiment(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1CreateExperimentResponse.from_json(_resp.json())
    raise APIHttpError("post_CreateExperiment", _resp)

def post_CreateGroup(
    session: "api.Session",
    *,
    body: "v1CreateGroupRequest",
) -> "v1CreateGroupResponse":
    _params = None
    _resp = session._do_request(
        method="POST",
        path="/api/v1/groups",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return v1CreateGroupResponse.from_json(_resp.json())
    raise APIHttpError("post_CreateGroup", _resp)

def post_CreateTrialsCollection(
    session: "api.Session",
    *,
    body: "v1CreateTrialsCollectionRequest",
) -> "v1CreateTrialsCollectionResponse":
    _params = None
    _resp = session._do_request(
        method="POST",
        path="/api/v1/trial-comparison/collections",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return v1CreateTrialsCollectionResponse.from_json(_resp.json())
    raise APIHttpError("post_CreateTrialsCollection", _resp)

def get_CurrentUser(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1CurrentUserResponse.from_json(_resp.json())
    raise APIHttpError("get_CurrentUser", _resp)

def delete_DeleteCheckpoints(
    session: "api.Session",
    *,
    body: "v1DeleteCheckpointsRequest",
) -> None:
    _params = None
    _resp = session._do_request(
        method="DELETE",
        path="/api/v1/checkpoints",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("delete_DeleteCheckpoints", _resp)

def delete_DeleteExperiment(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("delete_DeleteExperiment", _resp)

def delete_DeleteGroup(
    session: "api.Session",
    *,
    groupId: int,
) -> None:
    _params = None
    _resp = session._do_request(
        method="DELETE",
        path=f"/api/v1/groups/{groupId}",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("delete_DeleteGroup", _resp)

def delete_DeleteModel(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("delete_DeleteModel", _resp)

def delete_DeleteModelVersion(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("delete_DeleteModelVersion", _resp)

def delete_DeleteProject(
    session: "api.Session",
    *,
    id: int,
) -> "v1DeleteProjectResponse":
    _params = None
    _resp = session._do_request(
        method="DELETE",
        path=f"/api/v1/projects/{id}",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return v1DeleteProjectResponse.from_json(_resp.json())
    raise APIHttpError("delete_DeleteProject", _resp)

def delete_DeleteTemplate(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("delete_DeleteTemplate", _resp)

def delete_DeleteTrialsCollection(
    session: "api.Session",
    *,
    id: "typing.Optional[int]" = None,
) -> None:
    _params = {
        "id": id,
    }
    _resp = session._do_request(
        method="DELETE",
        path="/api/v1/trial-comparison/collections",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("delete_DeleteTrialsCollection", _resp)

def delete_DeleteWebhook(
    session: "api.Session",
    *,
    id: int,
) -> None:
    _params = None
    _resp = session._do_request(
        method="DELETE",
        path=f"/api/v1/webhooks/{id}",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("delete_DeleteWebhook", _resp)

def delete_DeleteWorkspace(
    session: "api.Session",
    *,
    id: int,
) -> "v1DeleteWorkspaceResponse":
    _params = None
    _resp = session._do_request(
        method="DELETE",
        path=f"/api/v1/workspaces/{id}",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return v1DeleteWorkspaceResponse.from_json(_resp.json())
    raise APIHttpError("delete_DeleteWorkspace", _resp)

def post_DisableAgent(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1DisableAgentResponse.from_json(_resp.json())
    raise APIHttpError("post_DisableAgent", _resp)

def post_DisableSlot(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1DisableSlotResponse.from_json(_resp.json())
    raise APIHttpError("post_DisableSlot", _resp)

def post_EnableAgent(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1EnableAgentResponse.from_json(_resp.json())
    raise APIHttpError("post_EnableAgent", _resp)

def post_EnableSlot(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1EnableSlotResponse.from_json(_resp.json())
    raise APIHttpError("post_EnableSlot", _resp)

def get_ExpCompareMetricNames(
    session: "api.Session",
    *,
    trialId: "typing.Sequence[int]",
    periodSeconds: "typing.Optional[int]" = None,
) -> "typing.Iterable[v1ExpCompareMetricNamesResponse]":
    _params = {
        "periodSeconds": periodSeconds,
        "trialId": trialId,
    }
    _resp = session._do_request(
        method="GET",
        path="/api/v1/trials/metrics-stream/metric-names",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
        stream=True,
    )
    if _resp.status_code == 200:
        for _line in _resp.iter_lines():
            _j = json.loads(_line)
            if "error" in _j:
                raise APIHttpStreamError(
                    "get_ExpCompareMetricNames",
                    runtimeStreamError.from_json(_j["error"])
            )
            yield v1ExpCompareMetricNamesResponse.from_json(_j["result"])
        return
    raise APIHttpError("get_ExpCompareMetricNames", _resp)

def get_ExpCompareTrialsSample(
    session: "api.Session",
    *,
    experimentIds: "typing.Sequence[int]",
    metricName: str,
    metricType: "v1MetricType",
    endBatches: "typing.Optional[int]" = None,
    maxDatapoints: "typing.Optional[int]" = None,
    maxTrials: "typing.Optional[int]" = None,
    periodSeconds: "typing.Optional[int]" = None,
    startBatches: "typing.Optional[int]" = None,
) -> "typing.Iterable[v1ExpCompareTrialsSampleResponse]":
    _params = {
        "endBatches": endBatches,
        "experimentIds": experimentIds,
        "maxDatapoints": maxDatapoints,
        "maxTrials": maxTrials,
        "metricName": metricName,
        "metricType": metricType.value,
        "periodSeconds": periodSeconds,
        "startBatches": startBatches,
    }
    _resp = session._do_request(
        method="GET",
        path="/api/v1/experiments-compare",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
        stream=True,
    )
    if _resp.status_code == 200:
        for _line in _resp.iter_lines():
            _j = json.loads(_line)
            if "error" in _j:
                raise APIHttpStreamError(
                    "get_ExpCompareTrialsSample",
                    runtimeStreamError.from_json(_j["error"])
            )
            yield v1ExpCompareTrialsSampleResponse.from_json(_j["result"])
        return
    raise APIHttpError("get_ExpCompareTrialsSample", _resp)

def get_GetActiveTasksCount(
    session: "api.Session",
) -> "v1GetActiveTasksCountResponse":
    _params = None
    _resp = session._do_request(
        method="GET",
        path="/api/v1/tasks/count",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetActiveTasksCountResponse.from_json(_resp.json())
    raise APIHttpError("get_GetActiveTasksCount", _resp)

def get_GetAgent(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetAgentResponse.from_json(_resp.json())
    raise APIHttpError("get_GetAgent", _resp)

def get_GetAgents(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetAgentsResponse.from_json(_resp.json())
    raise APIHttpError("get_GetAgents", _resp)

def get_GetBestSearcherValidationMetric(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetBestSearcherValidationMetricResponse.from_json(_resp.json())
    raise APIHttpError("get_GetBestSearcherValidationMetric", _resp)

def get_GetCheckpoint(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetCheckpointResponse.from_json(_resp.json())
    raise APIHttpError("get_GetCheckpoint", _resp)

def get_GetCommand(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetCommandResponse.from_json(_resp.json())
    raise APIHttpError("get_GetCommand", _resp)

def get_GetCommands(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetCommandsResponse.from_json(_resp.json())
    raise APIHttpError("get_GetCommands", _resp)

def get_GetCurrentTrialSearcherOperation(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetCurrentTrialSearcherOperationResponse.from_json(_resp.json())
    raise APIHttpError("get_GetCurrentTrialSearcherOperation", _resp)

def get_GetExperiment(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetExperimentResponse.from_json(_resp.json())
    raise APIHttpError("get_GetExperiment", _resp)

def get_GetExperimentCheckpoints(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetExperimentCheckpointsResponse.from_json(_resp.json())
    raise APIHttpError("get_GetExperimentCheckpoints", _resp)

def get_GetExperimentLabels(
    session: "api.Session",
    *,
    projectId: "typing.Optional[int]" = None,
) -> "v1GetExperimentLabelsResponse":
    _params = {
        "projectId": projectId,
    }
    _resp = session._do_request(
        method="GET",
        path="/api/v1/experiment/labels",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetExperimentLabelsResponse.from_json(_resp.json())
    raise APIHttpError("get_GetExperimentLabels", _resp)

def get_GetExperimentTrials(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetExperimentTrialsResponse.from_json(_resp.json())
    raise APIHttpError("get_GetExperimentTrials", _resp)

def get_GetExperimentValidationHistory(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetExperimentValidationHistoryResponse.from_json(_resp.json())
    raise APIHttpError("get_GetExperimentValidationHistory", _resp)

def get_GetExperiments(
    session: "api.Session",
    *,
    archived: "typing.Optional[bool]" = None,
    description: "typing.Optional[str]" = None,
    experimentIdFilter_gt: "typing.Optional[int]" = None,
    experimentIdFilter_gte: "typing.Optional[int]" = None,
    experimentIdFilter_incl: "typing.Optional[typing.Sequence[int]]" = None,
    experimentIdFilter_lt: "typing.Optional[int]" = None,
    experimentIdFilter_lte: "typing.Optional[int]" = None,
    experimentIdFilter_notIn: "typing.Optional[typing.Sequence[int]]" = None,
    labels: "typing.Optional[typing.Sequence[str]]" = None,
    limit: "typing.Optional[int]" = None,
    name: "typing.Optional[str]" = None,
    offset: "typing.Optional[int]" = None,
    orderBy: "typing.Optional[v1OrderBy]" = None,
    projectId: "typing.Optional[int]" = None,
    sortBy: "typing.Optional[v1GetExperimentsRequestSortBy]" = None,
    states: "typing.Optional[typing.Sequence[determinedexperimentv1State]]" = None,
    userIds: "typing.Optional[typing.Sequence[int]]" = None,
    users: "typing.Optional[typing.Sequence[str]]" = None,
) -> "v1GetExperimentsResponse":
    _params = {
        "archived": str(archived).lower() if archived is not None else None,
        "description": description,
        "experimentIdFilter.gt": experimentIdFilter_gt,
        "experimentIdFilter.gte": experimentIdFilter_gte,
        "experimentIdFilter.incl": experimentIdFilter_incl,
        "experimentIdFilter.lt": experimentIdFilter_lt,
        "experimentIdFilter.lte": experimentIdFilter_lte,
        "experimentIdFilter.notIn": experimentIdFilter_notIn,
        "labels": labels,
        "limit": limit,
        "name": name,
        "offset": offset,
        "orderBy": orderBy.value if orderBy is not None else None,
        "projectId": projectId,
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetExperimentsResponse.from_json(_resp.json())
    raise APIHttpError("get_GetExperiments", _resp)

def get_GetGroup(
    session: "api.Session",
    *,
    groupId: int,
) -> "v1GetGroupResponse":
    _params = None
    _resp = session._do_request(
        method="GET",
        path=f"/api/v1/groups/{groupId}",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetGroupResponse.from_json(_resp.json())
    raise APIHttpError("get_GetGroup", _resp)

def post_GetGroups(
    session: "api.Session",
    *,
    body: "v1GetGroupsRequest",
) -> "v1GetGroupsResponse":
    _params = None
    _resp = session._do_request(
        method="POST",
        path="/api/v1/groups/search",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetGroupsResponse.from_json(_resp.json())
    raise APIHttpError("post_GetGroups", _resp)

def get_GetGroupsAndUsersAssignedToWorkspace(
    session: "api.Session",
    *,
    workspaceId: int,
    name: "typing.Optional[str]" = None,
) -> "v1GetGroupsAndUsersAssignedToWorkspaceResponse":
    _params = {
        "name": name,
    }
    _resp = session._do_request(
        method="GET",
        path=f"/api/v1/roles/workspace/{workspaceId}",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetGroupsAndUsersAssignedToWorkspaceResponse.from_json(_resp.json())
    raise APIHttpError("get_GetGroupsAndUsersAssignedToWorkspace", _resp)

def get_GetHPImportance(
    session: "api.Session",
    *,
    experimentId: int,
    periodSeconds: "typing.Optional[int]" = None,
) -> "typing.Iterable[v1GetHPImportanceResponse]":
    _params = {
        "periodSeconds": periodSeconds,
    }
    _resp = session._do_request(
        method="GET",
        path=f"/api/v1/experiments/{experimentId}/hyperparameter-importance",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
        stream=True,
    )
    if _resp.status_code == 200:
        for _line in _resp.iter_lines():
            _j = json.loads(_line)
            if "error" in _j:
                raise APIHttpStreamError(
                    "get_GetHPImportance",
                    runtimeStreamError.from_json(_j["error"])
            )
            yield v1GetHPImportanceResponse.from_json(_j["result"])
        return
    raise APIHttpError("get_GetHPImportance", _resp)

def get_GetJobQueueStats(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetJobQueueStatsResponse.from_json(_resp.json())
    raise APIHttpError("get_GetJobQueueStats", _resp)

def get_GetJobs(
    session: "api.Session",
    *,
    limit: "typing.Optional[int]" = None,
    offset: "typing.Optional[int]" = None,
    orderBy: "typing.Optional[v1OrderBy]" = None,
    resourcePool: "typing.Optional[str]" = None,
    states: "typing.Optional[typing.Sequence[determinedjobv1State]]" = None,
) -> "v1GetJobsResponse":
    _params = {
        "limit": limit,
        "offset": offset,
        "orderBy": orderBy.value if orderBy is not None else None,
        "resourcePool": resourcePool,
        "states": [x.value for x in states] if states is not None else None,
    }
    _resp = session._do_request(
        method="GET",
        path="/api/v1/job-queues",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetJobsResponse.from_json(_resp.json())
    raise APIHttpError("get_GetJobs", _resp)

def get_GetMaster(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetMasterResponse.from_json(_resp.json())
    raise APIHttpError("get_GetMaster", _resp)

def get_GetMasterConfig(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetMasterConfigResponse.from_json(_resp.json())
    raise APIHttpError("get_GetMasterConfig", _resp)

def get_GetModel(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetModelResponse.from_json(_resp.json())
    raise APIHttpError("get_GetModel", _resp)

def get_GetModelDef(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetModelDefResponse.from_json(_resp.json())
    raise APIHttpError("get_GetModelDef", _resp)

def post_GetModelDefFile(
    session: "api.Session",
    *,
    body: "v1GetModelDefFileRequest",
    experimentId: int,
) -> "v1GetModelDefFileResponse":
    _params = None
    _resp = session._do_request(
        method="POST",
        path=f"/api/v1/experiments/{experimentId}/file",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetModelDefFileResponse.from_json(_resp.json())
    raise APIHttpError("post_GetModelDefFile", _resp)

def get_GetModelDefTree(
    session: "api.Session",
    *,
    experimentId: int,
) -> "v1GetModelDefTreeResponse":
    _params = None
    _resp = session._do_request(
        method="GET",
        path=f"/api/v1/experiments/{experimentId}/file_tree",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetModelDefTreeResponse.from_json(_resp.json())
    raise APIHttpError("get_GetModelDefTree", _resp)

def get_GetModelLabels(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetModelLabelsResponse.from_json(_resp.json())
    raise APIHttpError("get_GetModelLabels", _resp)

def get_GetModelVersion(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetModelVersionResponse.from_json(_resp.json())
    raise APIHttpError("get_GetModelVersion", _resp)

def get_GetModelVersions(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetModelVersionsResponse.from_json(_resp.json())
    raise APIHttpError("get_GetModelVersions", _resp)

def get_GetModels(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetModelsResponse.from_json(_resp.json())
    raise APIHttpError("get_GetModels", _resp)

def get_GetNotebook(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetNotebookResponse.from_json(_resp.json())
    raise APIHttpError("get_GetNotebook", _resp)

def get_GetNotebooks(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetNotebooksResponse.from_json(_resp.json())
    raise APIHttpError("get_GetNotebooks", _resp)

def get_GetPermissionsSummary(
    session: "api.Session",
) -> "v1GetPermissionsSummaryResponse":
    _params = None
    _resp = session._do_request(
        method="GET",
        path="/api/v1/permissions/summary",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetPermissionsSummaryResponse.from_json(_resp.json())
    raise APIHttpError("get_GetPermissionsSummary", _resp)

def get_GetProject(
    session: "api.Session",
    *,
    id: int,
) -> "v1GetProjectResponse":
    _params = None
    _resp = session._do_request(
        method="GET",
        path=f"/api/v1/projects/{id}",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetProjectResponse.from_json(_resp.json())
    raise APIHttpError("get_GetProject", _resp)

def get_GetResourcePools(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetResourcePoolsResponse.from_json(_resp.json())
    raise APIHttpError("get_GetResourcePools", _resp)

def get_GetRolesAssignedToGroup(
    session: "api.Session",
    *,
    groupId: int,
) -> "v1GetRolesAssignedToGroupResponse":
    _params = None
    _resp = session._do_request(
        method="GET",
        path=f"/api/v1/roles/search/by-group/{groupId}",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetRolesAssignedToGroupResponse.from_json(_resp.json())
    raise APIHttpError("get_GetRolesAssignedToGroup", _resp)

def get_GetRolesAssignedToUser(
    session: "api.Session",
    *,
    userId: int,
) -> "v1GetRolesAssignedToUserResponse":
    _params = None
    _resp = session._do_request(
        method="GET",
        path=f"/api/v1/roles/search/by-user/{userId}",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetRolesAssignedToUserResponse.from_json(_resp.json())
    raise APIHttpError("get_GetRolesAssignedToUser", _resp)

def post_GetRolesByID(
    session: "api.Session",
    *,
    body: "v1GetRolesByIDRequest",
) -> "v1GetRolesByIDResponse":
    _params = None
    _resp = session._do_request(
        method="POST",
        path="/api/v1/roles/search/by-ids",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetRolesByIDResponse.from_json(_resp.json())
    raise APIHttpError("post_GetRolesByID", _resp)

def get_GetSearcherEvents(
    session: "api.Session",
    *,
    experimentId: int,
) -> "v1GetSearcherEventsResponse":
    _params = None
    _resp = session._do_request(
        method="GET",
        path=f"/api/v1/experiments/{experimentId}/searcher_events",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetSearcherEventsResponse.from_json(_resp.json())
    raise APIHttpError("get_GetSearcherEvents", _resp)

def get_GetShell(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetShellResponse.from_json(_resp.json())
    raise APIHttpError("get_GetShell", _resp)

def get_GetShells(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetShellsResponse.from_json(_resp.json())
    raise APIHttpError("get_GetShells", _resp)

def get_GetSlot(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetSlotResponse.from_json(_resp.json())
    raise APIHttpError("get_GetSlot", _resp)

def get_GetSlots(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetSlotsResponse.from_json(_resp.json())
    raise APIHttpError("get_GetSlots", _resp)

def get_GetTask(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetTaskResponse.from_json(_resp.json())
    raise APIHttpError("get_GetTask", _resp)

def get_GetTelemetry(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetTelemetryResponse.from_json(_resp.json())
    raise APIHttpError("get_GetTelemetry", _resp)

def get_GetTemplate(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetTemplateResponse.from_json(_resp.json())
    raise APIHttpError("get_GetTemplate", _resp)

def get_GetTemplates(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetTemplatesResponse.from_json(_resp.json())
    raise APIHttpError("get_GetTemplates", _resp)

def get_GetTensorboard(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetTensorboardResponse.from_json(_resp.json())
    raise APIHttpError("get_GetTensorboard", _resp)

def get_GetTensorboards(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetTensorboardsResponse.from_json(_resp.json())
    raise APIHttpError("get_GetTensorboards", _resp)

def get_GetTrial(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetTrialResponse.from_json(_resp.json())
    raise APIHttpError("get_GetTrial", _resp)

def get_GetTrialCheckpoints(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetTrialCheckpointsResponse.from_json(_resp.json())
    raise APIHttpError("get_GetTrialCheckpoints", _resp)

def get_GetTrialProfilerAvailableSeries(
    session: "api.Session",
    *,
    trialId: int,
    follow: "typing.Optional[bool]" = None,
) -> "typing.Iterable[v1GetTrialProfilerAvailableSeriesResponse]":
    _params = {
        "follow": str(follow).lower() if follow is not None else None,
    }
    _resp = session._do_request(
        method="GET",
        path=f"/api/v1/trials/{trialId}/profiler/available_series",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
        stream=True,
    )
    if _resp.status_code == 200:
        for _line in _resp.iter_lines():
            _j = json.loads(_line)
            if "error" in _j:
                raise APIHttpStreamError(
                    "get_GetTrialProfilerAvailableSeries",
                    runtimeStreamError.from_json(_j["error"])
            )
            yield v1GetTrialProfilerAvailableSeriesResponse.from_json(_j["result"])
        return
    raise APIHttpError("get_GetTrialProfilerAvailableSeries", _resp)

def get_GetTrialProfilerMetrics(
    session: "api.Session",
    *,
    labels_trialId: int,
    follow: "typing.Optional[bool]" = None,
    labels_agentId: "typing.Optional[str]" = None,
    labels_gpuUuid: "typing.Optional[str]" = None,
    labels_metricType: "typing.Optional[TrialProfilerMetricLabelsProfilerMetricType]" = None,
    labels_name: "typing.Optional[str]" = None,
) -> "typing.Iterable[v1GetTrialProfilerMetricsResponse]":
    _params = {
        "follow": str(follow).lower() if follow is not None else None,
        "labels.agentId": labels_agentId,
        "labels.gpuUuid": labels_gpuUuid,
        "labels.metricType": labels_metricType.value if labels_metricType is not None else None,
        "labels.name": labels_name,
    }
    _resp = session._do_request(
        method="GET",
        path=f"/api/v1/trials/{labels_trialId}/profiler/metrics",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
        stream=True,
    )
    if _resp.status_code == 200:
        for _line in _resp.iter_lines():
            _j = json.loads(_line)
            if "error" in _j:
                raise APIHttpStreamError(
                    "get_GetTrialProfilerMetrics",
                    runtimeStreamError.from_json(_j["error"])
            )
            yield v1GetTrialProfilerMetricsResponse.from_json(_j["result"])
        return
    raise APIHttpError("get_GetTrialProfilerMetrics", _resp)

def get_GetTrialWorkloads(
    session: "api.Session",
    *,
    trialId: int,
    filter: "typing.Optional[GetTrialWorkloadsRequestFilterOption]" = None,
    includeBatchMetrics: "typing.Optional[bool]" = None,
    limit: "typing.Optional[int]" = None,
    metricType: "typing.Optional[v1MetricType]" = None,
    offset: "typing.Optional[int]" = None,
    orderBy: "typing.Optional[v1OrderBy]" = None,
    sortKey: "typing.Optional[str]" = None,
) -> "v1GetTrialWorkloadsResponse":
    _params = {
        "filter": filter.value if filter is not None else None,
        "includeBatchMetrics": str(includeBatchMetrics).lower() if includeBatchMetrics is not None else None,
        "limit": limit,
        "metricType": metricType.value if metricType is not None else None,
        "offset": offset,
        "orderBy": orderBy.value if orderBy is not None else None,
        "sortKey": sortKey,
    }
    _resp = session._do_request(
        method="GET",
        path=f"/api/v1/trials/{trialId}/workloads",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetTrialWorkloadsResponse.from_json(_resp.json())
    raise APIHttpError("get_GetTrialWorkloads", _resp)

def get_GetTrialsCollections(
    session: "api.Session",
    *,
    projectId: "typing.Optional[int]" = None,
) -> "v1GetTrialsCollectionsResponse":
    _params = {
        "projectId": projectId,
    }
    _resp = session._do_request(
        method="GET",
        path="/api/v1/trial-comparison/collections",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetTrialsCollectionsResponse.from_json(_resp.json())
    raise APIHttpError("get_GetTrialsCollections", _resp)

def get_GetUser(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetUserResponse.from_json(_resp.json())
    raise APIHttpError("get_GetUser", _resp)

def get_GetUserSetting(
    session: "api.Session",
) -> "v1GetUserSettingResponse":
    _params = None
    _resp = session._do_request(
        method="GET",
        path="/api/v1/users/setting",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetUserSettingResponse.from_json(_resp.json())
    raise APIHttpError("get_GetUserSetting", _resp)

def get_GetUsers(
    session: "api.Session",
    *,
    limit: "typing.Optional[int]" = None,
    offset: "typing.Optional[int]" = None,
    orderBy: "typing.Optional[v1OrderBy]" = None,
    sortBy: "typing.Optional[v1GetUsersRequestSortBy]" = None,
) -> "v1GetUsersResponse":
    _params = {
        "limit": limit,
        "offset": offset,
        "orderBy": orderBy.value if orderBy is not None else None,
        "sortBy": sortBy.value if sortBy is not None else None,
    }
    _resp = session._do_request(
        method="GET",
        path="/api/v1/users",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetUsersResponse.from_json(_resp.json())
    raise APIHttpError("get_GetUsers", _resp)

def get_GetWebhooks(
    session: "api.Session",
) -> "v1GetWebhooksResponse":
    _params = None
    _resp = session._do_request(
        method="GET",
        path="/api/v1/webhooks",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetWebhooksResponse.from_json(_resp.json())
    raise APIHttpError("get_GetWebhooks", _resp)

def get_GetWorkspace(
    session: "api.Session",
    *,
    id: int,
) -> "v1GetWorkspaceResponse":
    _params = None
    _resp = session._do_request(
        method="GET",
        path=f"/api/v1/workspaces/{id}",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetWorkspaceResponse.from_json(_resp.json())
    raise APIHttpError("get_GetWorkspace", _resp)

def get_GetWorkspaceProjects(
    session: "api.Session",
    *,
    id: int,
    archived: "typing.Optional[bool]" = None,
    limit: "typing.Optional[int]" = None,
    name: "typing.Optional[str]" = None,
    offset: "typing.Optional[int]" = None,
    orderBy: "typing.Optional[v1OrderBy]" = None,
    sortBy: "typing.Optional[v1GetWorkspaceProjectsRequestSortBy]" = None,
    users: "typing.Optional[typing.Sequence[str]]" = None,
) -> "v1GetWorkspaceProjectsResponse":
    _params = {
        "archived": str(archived).lower() if archived is not None else None,
        "limit": limit,
        "name": name,
        "offset": offset,
        "orderBy": orderBy.value if orderBy is not None else None,
        "sortBy": sortBy.value if sortBy is not None else None,
        "users": users,
    }
    _resp = session._do_request(
        method="GET",
        path=f"/api/v1/workspaces/{id}/projects",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetWorkspaceProjectsResponse.from_json(_resp.json())
    raise APIHttpError("get_GetWorkspaceProjects", _resp)

def get_GetWorkspaces(
    session: "api.Session",
    *,
    archived: "typing.Optional[bool]" = None,
    limit: "typing.Optional[int]" = None,
    name: "typing.Optional[str]" = None,
    offset: "typing.Optional[int]" = None,
    orderBy: "typing.Optional[v1OrderBy]" = None,
    pinned: "typing.Optional[bool]" = None,
    sortBy: "typing.Optional[v1GetWorkspacesRequestSortBy]" = None,
    users: "typing.Optional[typing.Sequence[str]]" = None,
) -> "v1GetWorkspacesResponse":
    _params = {
        "archived": str(archived).lower() if archived is not None else None,
        "limit": limit,
        "name": name,
        "offset": offset,
        "orderBy": orderBy.value if orderBy is not None else None,
        "pinned": str(pinned).lower() if pinned is not None else None,
        "sortBy": sortBy.value if sortBy is not None else None,
        "users": users,
    }
    _resp = session._do_request(
        method="GET",
        path="/api/v1/workspaces",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return v1GetWorkspacesResponse.from_json(_resp.json())
    raise APIHttpError("get_GetWorkspaces", _resp)

def put_IdleNotebook(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("put_IdleNotebook", _resp)

def post_KillCommand(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1KillCommandResponse.from_json(_resp.json())
    raise APIHttpError("post_KillCommand", _resp)

def post_KillExperiment(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_KillExperiment", _resp)

def post_KillNotebook(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1KillNotebookResponse.from_json(_resp.json())
    raise APIHttpError("post_KillNotebook", _resp)

def post_KillShell(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1KillShellResponse.from_json(_resp.json())
    raise APIHttpError("post_KillShell", _resp)

def post_KillTensorboard(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1KillTensorboardResponse.from_json(_resp.json())
    raise APIHttpError("post_KillTensorboard", _resp)

def post_KillTrial(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_KillTrial", _resp)

def post_LaunchCommand(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1LaunchCommandResponse.from_json(_resp.json())
    raise APIHttpError("post_LaunchCommand", _resp)

def post_LaunchNotebook(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1LaunchNotebookResponse.from_json(_resp.json())
    raise APIHttpError("post_LaunchNotebook", _resp)

def post_LaunchShell(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1LaunchShellResponse.from_json(_resp.json())
    raise APIHttpError("post_LaunchShell", _resp)

def post_LaunchTensorboard(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1LaunchTensorboardResponse.from_json(_resp.json())
    raise APIHttpError("post_LaunchTensorboard", _resp)

def post_ListRoles(
    session: "api.Session",
    *,
    body: "v1ListRolesRequest",
) -> "v1ListRolesResponse":
    _params = None
    _resp = session._do_request(
        method="POST",
        path="/api/v1/roles/search",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return v1ListRolesResponse.from_json(_resp.json())
    raise APIHttpError("post_ListRoles", _resp)

def post_Login(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1LoginResponse.from_json(_resp.json())
    raise APIHttpError("post_Login", _resp)

def post_Logout(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_Logout", _resp)

def post_MarkAllocationResourcesDaemon(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_MarkAllocationResourcesDaemon", _resp)

def get_MasterLogs(
    session: "api.Session",
    *,
    follow: "typing.Optional[bool]" = None,
    limit: "typing.Optional[int]" = None,
    offset: "typing.Optional[int]" = None,
) -> "typing.Iterable[v1MasterLogsResponse]":
    _params = {
        "follow": str(follow).lower() if follow is not None else None,
        "limit": limit,
        "offset": offset,
    }
    _resp = session._do_request(
        method="GET",
        path="/api/v1/master/logs",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
        stream=True,
    )
    if _resp.status_code == 200:
        for _line in _resp.iter_lines():
            _j = json.loads(_line)
            if "error" in _j:
                raise APIHttpStreamError(
                    "get_MasterLogs",
                    runtimeStreamError.from_json(_j["error"])
            )
            yield v1MasterLogsResponse.from_json(_j["result"])
        return
    raise APIHttpError("get_MasterLogs", _resp)

def get_MetricBatches(
    session: "api.Session",
    *,
    experimentId: int,
    metricName: str,
    metricType: "v1MetricType",
    periodSeconds: "typing.Optional[int]" = None,
) -> "typing.Iterable[v1MetricBatchesResponse]":
    _params = {
        "metricName": metricName,
        "metricType": metricType.value,
        "periodSeconds": periodSeconds,
    }
    _resp = session._do_request(
        method="GET",
        path=f"/api/v1/experiments/{experimentId}/metrics-stream/batches",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
        stream=True,
    )
    if _resp.status_code == 200:
        for _line in _resp.iter_lines():
            _j = json.loads(_line)
            if "error" in _j:
                raise APIHttpStreamError(
                    "get_MetricBatches",
                    runtimeStreamError.from_json(_j["error"])
            )
            yield v1MetricBatchesResponse.from_json(_j["result"])
        return
    raise APIHttpError("get_MetricBatches", _resp)

def get_MetricNames(
    session: "api.Session",
    *,
    experimentId: int,
    periodSeconds: "typing.Optional[int]" = None,
) -> "typing.Iterable[v1MetricNamesResponse]":
    _params = {
        "periodSeconds": periodSeconds,
    }
    _resp = session._do_request(
        method="GET",
        path=f"/api/v1/experiments/{experimentId}/metrics-stream/metric-names",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
        stream=True,
    )
    if _resp.status_code == 200:
        for _line in _resp.iter_lines():
            _j = json.loads(_line)
            if "error" in _j:
                raise APIHttpStreamError(
                    "get_MetricNames",
                    runtimeStreamError.from_json(_j["error"])
            )
            yield v1MetricNamesResponse.from_json(_j["result"])
        return
    raise APIHttpError("get_MetricNames", _resp)

def post_MoveExperiment(
    session: "api.Session",
    *,
    body: "v1MoveExperimentRequest",
    experimentId: int,
) -> None:
    _params = None
    _resp = session._do_request(
        method="POST",
        path=f"/api/v1/experiments/{experimentId}/move",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_MoveExperiment", _resp)

def post_MoveProject(
    session: "api.Session",
    *,
    body: "v1MoveProjectRequest",
    projectId: int,
) -> None:
    _params = None
    _resp = session._do_request(
        method="POST",
        path=f"/api/v1/projects/{projectId}/move",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_MoveProject", _resp)

def patch_PatchExperiment(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1PatchExperimentResponse.from_json(_resp.json())
    raise APIHttpError("patch_PatchExperiment", _resp)

def patch_PatchModel(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1PatchModelResponse.from_json(_resp.json())
    raise APIHttpError("patch_PatchModel", _resp)

def patch_PatchModelVersion(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1PatchModelVersionResponse.from_json(_resp.json())
    raise APIHttpError("patch_PatchModelVersion", _resp)

def patch_PatchProject(
    session: "api.Session",
    *,
    body: "v1PatchProject",
    id: int,
) -> "v1PatchProjectResponse":
    _params = None
    _resp = session._do_request(
        method="PATCH",
        path=f"/api/v1/projects/{id}",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return v1PatchProjectResponse.from_json(_resp.json())
    raise APIHttpError("patch_PatchProject", _resp)

def patch_PatchTrialsCollection(
    session: "api.Session",
    *,
    body: "v1PatchTrialsCollectionRequest",
) -> "v1PatchTrialsCollectionResponse":
    _params = None
    _resp = session._do_request(
        method="PATCH",
        path="/api/v1/trial-comparison/collections",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return v1PatchTrialsCollectionResponse.from_json(_resp.json())
    raise APIHttpError("patch_PatchTrialsCollection", _resp)

def patch_PatchUser(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1PatchUserResponse.from_json(_resp.json())
    raise APIHttpError("patch_PatchUser", _resp)

def patch_PatchWorkspace(
    session: "api.Session",
    *,
    body: "v1PatchWorkspace",
    id: int,
) -> "v1PatchWorkspaceResponse":
    _params = None
    _resp = session._do_request(
        method="PATCH",
        path=f"/api/v1/workspaces/{id}",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return v1PatchWorkspaceResponse.from_json(_resp.json())
    raise APIHttpError("patch_PatchWorkspace", _resp)

def post_PauseExperiment(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_PauseExperiment", _resp)

def post_PinWorkspace(
    session: "api.Session",
    *,
    id: int,
) -> None:
    _params = None
    _resp = session._do_request(
        method="POST",
        path=f"/api/v1/workspaces/{id}/pin",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_PinWorkspace", _resp)

def post_PostAllocationProxyAddress(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_PostAllocationProxyAddress", _resp)

def post_PostCheckpointMetadata(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1PostCheckpointMetadataResponse.from_json(_resp.json())
    raise APIHttpError("post_PostCheckpointMetadata", _resp)

def post_PostModel(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1PostModelResponse.from_json(_resp.json())
    raise APIHttpError("post_PostModel", _resp)

def post_PostModelVersion(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1PostModelVersionResponse.from_json(_resp.json())
    raise APIHttpError("post_PostModelVersion", _resp)

def post_PostProject(
    session: "api.Session",
    *,
    body: "v1PostProjectRequest",
    workspaceId: int,
) -> "v1PostProjectResponse":
    _params = None
    _resp = session._do_request(
        method="POST",
        path=f"/api/v1/workspaces/{workspaceId}/projects",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return v1PostProjectResponse.from_json(_resp.json())
    raise APIHttpError("post_PostProject", _resp)

def post_PostSearcherOperations(
    session: "api.Session",
    *,
    body: "v1PostSearcherOperationsRequest",
    experimentId: int,
) -> None:
    _params = None
    _resp = session._do_request(
        method="POST",
        path=f"/api/v1/experiments/{experimentId}/searcher_operations",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_PostSearcherOperations", _resp)

def post_PostTrialProfilerMetricsBatch(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_PostTrialProfilerMetricsBatch", _resp)

def post_PostTrialRunnerMetadata(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_PostTrialRunnerMetadata", _resp)

def post_PostUser(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1PostUserResponse.from_json(_resp.json())
    raise APIHttpError("post_PostUser", _resp)

def post_PostUserSetting(
    session: "api.Session",
    *,
    body: "v1PostUserSettingRequest",
) -> None:
    _params = None
    _resp = session._do_request(
        method="POST",
        path="/api/v1/users/setting",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_PostUserSetting", _resp)

def post_PostWebhook(
    session: "api.Session",
    *,
    body: "v1Webhook",
) -> "v1PostWebhookResponse":
    _params = None
    _resp = session._do_request(
        method="POST",
        path="/api/v1/webhooks",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return v1PostWebhookResponse.from_json(_resp.json())
    raise APIHttpError("post_PostWebhook", _resp)

def post_PostWorkspace(
    session: "api.Session",
    *,
    body: "v1PostWorkspaceRequest",
) -> "v1PostWorkspaceResponse":
    _params = None
    _resp = session._do_request(
        method="POST",
        path="/api/v1/workspaces",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return v1PostWorkspaceResponse.from_json(_resp.json())
    raise APIHttpError("post_PostWorkspace", _resp)

def post_PreviewHPSearch(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1PreviewHPSearchResponse.from_json(_resp.json())
    raise APIHttpError("post_PreviewHPSearch", _resp)

def put_PutProjectNotes(
    session: "api.Session",
    *,
    body: "v1PutProjectNotesRequest",
    projectId: int,
) -> "v1PutProjectNotesResponse":
    _params = None
    _resp = session._do_request(
        method="PUT",
        path=f"/api/v1/projects/{projectId}/notes",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return v1PutProjectNotesResponse.from_json(_resp.json())
    raise APIHttpError("put_PutProjectNotes", _resp)

def put_PutTemplate(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1PutTemplateResponse.from_json(_resp.json())
    raise APIHttpError("put_PutTemplate", _resp)

def post_QueryTrials(
    session: "api.Session",
    *,
    body: "v1QueryTrialsRequest",
) -> "v1QueryTrialsResponse":
    _params = None
    _resp = session._do_request(
        method="POST",
        path="/api/v1/trial-comparison/query",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return v1QueryTrialsResponse.from_json(_resp.json())
    raise APIHttpError("post_QueryTrials", _resp)

def post_RemoveAssignments(
    session: "api.Session",
    *,
    body: "v1RemoveAssignmentsRequest",
) -> None:
    _params = None
    _resp = session._do_request(
        method="POST",
        path="/api/v1/roles/remove-assignments",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_RemoveAssignments", _resp)

def post_ReportCheckpoint(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_ReportCheckpoint", _resp)

def post_ReportTrialProgress(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_ReportTrialProgress", _resp)

def post_ReportTrialSearcherEarlyExit(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_ReportTrialSearcherEarlyExit", _resp)

def post_ReportTrialTrainingMetrics(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_ReportTrialTrainingMetrics", _resp)

def post_ReportTrialValidationMetrics(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_ReportTrialValidationMetrics", _resp)

def post_ResetUserSetting(
    session: "api.Session",
) -> None:
    _params = None
    _resp = session._do_request(
        method="POST",
        path="/api/v1/users/setting/reset",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_ResetUserSetting", _resp)

def get_ResourceAllocationAggregated(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1ResourceAllocationAggregatedResponse.from_json(_resp.json())
    raise APIHttpError("get_ResourceAllocationAggregated", _resp)

def get_ResourceAllocationRaw(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1ResourceAllocationRawResponse.from_json(_resp.json())
    raise APIHttpError("get_ResourceAllocationRaw", _resp)

def post_SearchRolesAssignableToScope(
    session: "api.Session",
    *,
    body: "v1SearchRolesAssignableToScopeRequest",
) -> "v1SearchRolesAssignableToScopeResponse":
    _params = None
    _resp = session._do_request(
        method="POST",
        path="/api/v1/roles/search/by-assignability",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return v1SearchRolesAssignableToScopeResponse.from_json(_resp.json())
    raise APIHttpError("post_SearchRolesAssignableToScope", _resp)

def post_SetCommandPriority(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1SetCommandPriorityResponse.from_json(_resp.json())
    raise APIHttpError("post_SetCommandPriority", _resp)

def post_SetNotebookPriority(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1SetNotebookPriorityResponse.from_json(_resp.json())
    raise APIHttpError("post_SetNotebookPriority", _resp)

def post_SetShellPriority(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1SetShellPriorityResponse.from_json(_resp.json())
    raise APIHttpError("post_SetShellPriority", _resp)

def post_SetTensorboardPriority(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1SetTensorboardPriorityResponse.from_json(_resp.json())
    raise APIHttpError("post_SetTensorboardPriority", _resp)

def post_SetUserPassword(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return v1SetUserPasswordResponse.from_json(_resp.json())
    raise APIHttpError("post_SetUserPassword", _resp)

def get_SummarizeTrial(
    session: "api.Session",
    *,
    trialId: int,
    endBatches: "typing.Optional[int]" = None,
    maxDatapoints: "typing.Optional[int]" = None,
    metricNames: "typing.Optional[typing.Sequence[str]]" = None,
    metricType: "typing.Optional[v1MetricType]" = None,
    scale: "typing.Optional[v1Scale]" = None,
    startBatches: "typing.Optional[int]" = None,
) -> "v1SummarizeTrialResponse":
    _params = {
        "endBatches": endBatches,
        "maxDatapoints": maxDatapoints,
        "metricNames": metricNames,
        "metricType": metricType.value if metricType is not None else None,
        "scale": scale.value if scale is not None else None,
        "startBatches": startBatches,
    }
    _resp = session._do_request(
        method="GET",
        path=f"/api/v1/trials/{trialId}/summarize",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return v1SummarizeTrialResponse.from_json(_resp.json())
    raise APIHttpError("get_SummarizeTrial", _resp)

def get_TaskLogs(
    session: "api.Session",
    *,
    taskId: str,
    agentIds: "typing.Optional[typing.Sequence[str]]" = None,
    allocationIds: "typing.Optional[typing.Sequence[str]]" = None,
    containerIds: "typing.Optional[typing.Sequence[str]]" = None,
    follow: "typing.Optional[bool]" = None,
    levels: "typing.Optional[typing.Sequence[v1LogLevel]]" = None,
    limit: "typing.Optional[int]" = None,
    orderBy: "typing.Optional[v1OrderBy]" = None,
    rankIds: "typing.Optional[typing.Sequence[int]]" = None,
    searchText: "typing.Optional[str]" = None,
    sources: "typing.Optional[typing.Sequence[str]]" = None,
    stdtypes: "typing.Optional[typing.Sequence[str]]" = None,
    timestampAfter: "typing.Optional[str]" = None,
    timestampBefore: "typing.Optional[str]" = None,
) -> "typing.Iterable[v1TaskLogsResponse]":
    _params = {
        "agentIds": agentIds,
        "allocationIds": allocationIds,
        "containerIds": containerIds,
        "follow": str(follow).lower() if follow is not None else None,
        "levels": [x.value for x in levels] if levels is not None else None,
        "limit": limit,
        "orderBy": orderBy.value if orderBy is not None else None,
        "rankIds": rankIds,
        "searchText": searchText,
        "sources": sources,
        "stdtypes": stdtypes,
        "timestampAfter": timestampAfter,
        "timestampBefore": timestampBefore,
    }
    _resp = session._do_request(
        method="GET",
        path=f"/api/v1/tasks/{taskId}/logs",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
        stream=True,
    )
    if _resp.status_code == 200:
        for _line in _resp.iter_lines():
            _j = json.loads(_line)
            if "error" in _j:
                raise APIHttpStreamError(
                    "get_TaskLogs",
                    runtimeStreamError.from_json(_j["error"])
            )
            yield v1TaskLogsResponse.from_json(_j["result"])
        return
    raise APIHttpError("get_TaskLogs", _resp)

def get_TaskLogsFields(
    session: "api.Session",
    *,
    taskId: str,
    follow: "typing.Optional[bool]" = None,
) -> "typing.Iterable[v1TaskLogsFieldsResponse]":
    _params = {
        "follow": str(follow).lower() if follow is not None else None,
    }
    _resp = session._do_request(
        method="GET",
        path=f"/api/v1/tasks/{taskId}/logs/fields",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
        stream=True,
    )
    if _resp.status_code == 200:
        for _line in _resp.iter_lines():
            _j = json.loads(_line)
            if "error" in _j:
                raise APIHttpStreamError(
                    "get_TaskLogsFields",
                    runtimeStreamError.from_json(_j["error"])
            )
            yield v1TaskLogsFieldsResponse.from_json(_j["result"])
        return
    raise APIHttpError("get_TaskLogsFields", _resp)

def post_TestWebhook(
    session: "api.Session",
    *,
    id: int,
) -> "v1TestWebhookResponse":
    _params = None
    _resp = session._do_request(
        method="POST",
        path=f"/api/v1/webhooks/{id}/test",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return v1TestWebhookResponse.from_json(_resp.json())
    raise APIHttpError("post_TestWebhook", _resp)

def get_TrialLogs(
    session: "api.Session",
    *,
    trialId: int,
    agentIds: "typing.Optional[typing.Sequence[str]]" = None,
    containerIds: "typing.Optional[typing.Sequence[str]]" = None,
    follow: "typing.Optional[bool]" = None,
    levels: "typing.Optional[typing.Sequence[v1LogLevel]]" = None,
    limit: "typing.Optional[int]" = None,
    orderBy: "typing.Optional[v1OrderBy]" = None,
    rankIds: "typing.Optional[typing.Sequence[int]]" = None,
    searchText: "typing.Optional[str]" = None,
    sources: "typing.Optional[typing.Sequence[str]]" = None,
    stdtypes: "typing.Optional[typing.Sequence[str]]" = None,
    timestampAfter: "typing.Optional[str]" = None,
    timestampBefore: "typing.Optional[str]" = None,
) -> "typing.Iterable[v1TrialLogsResponse]":
    _params = {
        "agentIds": agentIds,
        "containerIds": containerIds,
        "follow": str(follow).lower() if follow is not None else None,
        "levels": [x.value for x in levels] if levels is not None else None,
        "limit": limit,
        "orderBy": orderBy.value if orderBy is not None else None,
        "rankIds": rankIds,
        "searchText": searchText,
        "sources": sources,
        "stdtypes": stdtypes,
        "timestampAfter": timestampAfter,
        "timestampBefore": timestampBefore,
    }
    _resp = session._do_request(
        method="GET",
        path=f"/api/v1/trials/{trialId}/logs",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
        stream=True,
    )
    if _resp.status_code == 200:
        for _line in _resp.iter_lines():
            _j = json.loads(_line)
            if "error" in _j:
                raise APIHttpStreamError(
                    "get_TrialLogs",
                    runtimeStreamError.from_json(_j["error"])
            )
            yield v1TrialLogsResponse.from_json(_j["result"])
        return
    raise APIHttpError("get_TrialLogs", _resp)

def get_TrialLogsFields(
    session: "api.Session",
    *,
    trialId: int,
    follow: "typing.Optional[bool]" = None,
) -> "typing.Iterable[v1TrialLogsFieldsResponse]":
    _params = {
        "follow": str(follow).lower() if follow is not None else None,
    }
    _resp = session._do_request(
        method="GET",
        path=f"/api/v1/trials/{trialId}/logs/fields",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
        stream=True,
    )
    if _resp.status_code == 200:
        for _line in _resp.iter_lines():
            _j = json.loads(_line)
            if "error" in _j:
                raise APIHttpStreamError(
                    "get_TrialLogsFields",
                    runtimeStreamError.from_json(_j["error"])
            )
            yield v1TrialLogsFieldsResponse.from_json(_j["result"])
        return
    raise APIHttpError("get_TrialLogsFields", _resp)

def get_TrialsSample(
    session: "api.Session",
    *,
    experimentId: int,
    metricName: str,
    metricType: "v1MetricType",
    endBatches: "typing.Optional[int]" = None,
    maxDatapoints: "typing.Optional[int]" = None,
    maxTrials: "typing.Optional[int]" = None,
    periodSeconds: "typing.Optional[int]" = None,
    startBatches: "typing.Optional[int]" = None,
) -> "typing.Iterable[v1TrialsSampleResponse]":
    _params = {
        "endBatches": endBatches,
        "maxDatapoints": maxDatapoints,
        "maxTrials": maxTrials,
        "metricName": metricName,
        "metricType": metricType.value,
        "periodSeconds": periodSeconds,
        "startBatches": startBatches,
    }
    _resp = session._do_request(
        method="GET",
        path=f"/api/v1/experiments/{experimentId}/metrics-stream/trials-sample",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
        stream=True,
    )
    if _resp.status_code == 200:
        for _line in _resp.iter_lines():
            _j = json.loads(_line)
            if "error" in _j:
                raise APIHttpStreamError(
                    "get_TrialsSample",
                    runtimeStreamError.from_json(_j["error"])
            )
            yield v1TrialsSampleResponse.from_json(_j["result"])
        return
    raise APIHttpError("get_TrialsSample", _resp)

def get_TrialsSnapshot(
    session: "api.Session",
    *,
    batchesProcessed: int,
    experimentId: int,
    metricName: str,
    metricType: "v1MetricType",
    batchesMargin: "typing.Optional[int]" = None,
    periodSeconds: "typing.Optional[int]" = None,
) -> "typing.Iterable[v1TrialsSnapshotResponse]":
    _params = {
        "batchesMargin": batchesMargin,
        "batchesProcessed": batchesProcessed,
        "metricName": metricName,
        "metricType": metricType.value,
        "periodSeconds": periodSeconds,
    }
    _resp = session._do_request(
        method="GET",
        path=f"/api/v1/experiments/{experimentId}/metrics-stream/trials-snapshot",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
        stream=True,
    )
    if _resp.status_code == 200:
        for _line in _resp.iter_lines():
            _j = json.loads(_line)
            if "error" in _j:
                raise APIHttpStreamError(
                    "get_TrialsSnapshot",
                    runtimeStreamError.from_json(_j["error"])
            )
            yield v1TrialsSnapshotResponse.from_json(_j["result"])
        return
    raise APIHttpError("get_TrialsSnapshot", _resp)

def post_UnarchiveExperiment(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_UnarchiveExperiment", _resp)

def post_UnarchiveModel(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_UnarchiveModel", _resp)

def post_UnarchiveProject(
    session: "api.Session",
    *,
    id: int,
) -> None:
    _params = None
    _resp = session._do_request(
        method="POST",
        path=f"/api/v1/projects/{id}/unarchive",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_UnarchiveProject", _resp)

def post_UnarchiveWorkspace(
    session: "api.Session",
    *,
    id: int,
) -> None:
    _params = None
    _resp = session._do_request(
        method="POST",
        path=f"/api/v1/workspaces/{id}/unarchive",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_UnarchiveWorkspace", _resp)

def post_UnpinWorkspace(
    session: "api.Session",
    *,
    id: int,
) -> None:
    _params = None
    _resp = session._do_request(
        method="POST",
        path=f"/api/v1/workspaces/{id}/unpin",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_UnpinWorkspace", _resp)

def put_UpdateGroup(
    session: "api.Session",
    *,
    body: "v1UpdateGroupRequest",
    groupId: int,
) -> "v1UpdateGroupResponse":
    _params = None
    _resp = session._do_request(
        method="PUT",
        path=f"/api/v1/groups/{groupId}",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return v1UpdateGroupResponse.from_json(_resp.json())
    raise APIHttpError("put_UpdateGroup", _resp)

def post_UpdateJobQueue(
    session: "api.Session",
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
        stream=False,
    )
    if _resp.status_code == 200:
        return
    raise APIHttpError("post_UpdateJobQueue", _resp)

def post_UpdateTrialTags(
    session: "api.Session",
    *,
    body: "v1UpdateTrialTagsRequest",
) -> "v1UpdateTrialTagsResponse":
    _params = None
    _resp = session._do_request(
        method="POST",
        path="/api/v1/trial-comparison/update-trial-tags",
        params=_params,
        json=body.to_json(),
        data=None,
        headers=None,
        timeout=None,
        stream=False,
    )
    if _resp.status_code == 200:
        return v1UpdateTrialTagsResponse.from_json(_resp.json())
    raise APIHttpError("post_UpdateTrialTags", _resp)

# Paginated is a union type of objects whose .pagination
# attribute is a v1Pagination-type object.
Paginated = typing.Union[
    v1GetAgentsResponse,
    v1GetCommandsResponse,
    v1GetExperimentCheckpointsResponse,
    v1GetExperimentTrialsResponse,
    v1GetExperimentsResponse,
    v1GetGroupsResponse,
    v1GetJobsResponse,
    v1GetModelVersionsResponse,
    v1GetModelsResponse,
    v1GetNotebooksResponse,
    v1GetResourcePoolsResponse,
    v1GetShellsResponse,
    v1GetTemplatesResponse,
    v1GetTensorboardsResponse,
    v1GetTrialCheckpointsResponse,
    v1GetTrialWorkloadsResponse,
    v1GetUsersResponse,
    v1GetWorkspaceProjectsResponse,
    v1GetWorkspacesResponse,
    v1ListRolesResponse,
    v1SearchRolesAssignableToScopeResponse,
]
