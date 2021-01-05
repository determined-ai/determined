from datetime import datetime, timezone

from determined import workload


def _current_timestamp() -> datetime:
    """Returns the current time as a datetime object in the UTC timezone."""
    return datetime.now(timezone.utc)


class TimerLayer(workload.Source):
    """
    TimerLayer just measures start_time and end_time of the layers below it.
    """

    def __init__(
        self,
        workloads: workload.Stream,
    ) -> None:
        self.workloads = workloads

    def __iter__(self) -> workload.Stream:
        for wkld, args, response_func in self.workloads:
            start_time = _current_timestamp()

            def _respond(in_response: workload.Response) -> None:
                if isinstance(in_response, dict):
                    in_response["start_time"] = start_time
                    in_response["end_time"] = _current_timestamp()
                response_func(in_response)

            yield wkld, args, _respond
