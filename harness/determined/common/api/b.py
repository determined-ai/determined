import enum
import math
import typing

if typing.TYPE_CHECKING:
    from determined.experimental import client

# flake8: noqa
Json = typing.Any


def dump_float(val: typing.Any) -> typing.Any:
    if math.isnan(val):
        return "Nan"
    if math.isinf(val):
        return "Infinity" if val > 0 else "-Infinity"
    return val


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
        summary: "typing.Optional[v1JobSummary]" = None,
        weight: "typing.Optional[float]" = None,
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

    @classmethod
    def from_json(cls, obj: Json) -> "v1Job":
        return cls(
            summary=v1JobSummary.from_json(obj["summary"])
            if obj.get("summary", None) is not None
            else None,
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
        )

    def to_json(self) -> typing.Any:
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

    def to_json(self) -> typing.Any:
        return {
            "state": self.state,
            "jobsAhead": self.jobsAhead,
        }


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
            offset=obj["offset"] if obj.get("offset", None) is not None else None,
            limit=obj["limit"] if obj.get("limit", None) is not None else None,
            startIndex=obj["startIndex"] if obj.get("startIndex", None) is not None else None,
            endIndex=obj["endIndex"] if obj.get("endIndex", None) is not None else None,
            total=obj["total"] if obj.get("total", None) is not None else None,
        )

    def to_json(self) -> typing.Any:
        return {
            "offset": self.offset if self.offset is not None else None,
            "limit": self.limit if self.limit is not None else None,
            "startIndex": self.startIndex if self.startIndex is not None else None,
            "endIndex": self.endIndex if self.endIndex is not None else None,
            "total": self.total if self.total is not None else None,
        }


def get_GetJobs(
    session: "client.Session",
    *,
    orderBy: "typing.Optional[str]" = None,
    pagination_limit: "typing.Optional[int]" = None,
    pagination_offset: "typing.Optional[int]" = None,
    resourcePool: "typing.Optional[str]" = None,
) -> "v1GetJobsResponse":
    _params = {
        "orderBy": orderBy,
        "pagination.limit": pagination_limit,
        "pagination.offset": pagination_offset,
        "resourcePool": resourcePool,
    }
    _req = session._do_request(
        method="GET",
        path="/api/v1/job-queues",
        params=_params,
        json=None,
        data=None,
        headers=None,
        timeout=None,
    )
    if _req.status_code == 200:
        return v1GetJobsResponse.from_json(_req.json())
    raise ValueError(_req.status_code)
