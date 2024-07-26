from pathlib import Path
from typing import Iterable, NewType, Optional, Union

from .base import BaseDict, BaseList, Format
from ..rest_api.locust_utils import LocustTasksWithMeta


class LocustStatsList(BaseList):
    def __init__(self, initlist: Optional[Iterable] = None,
                 locust_tasks_with_meta: Optional[LocustTasksWithMeta] = None):
        super().__init__(initlist)

        self._locust_tasks_with_meta = locust_tasks_with_meta

    @classmethod
    def _serialize(cls, value: Union['LocustStats', dict]) -> dict:
        return cls._serialize_to_dict(value)

    @staticmethod
    def _deserialize(value: dict) -> 'LocustStats':
        return LocustStats(value)

    def __iter__(self) -> 'LocustStats':
        for value in super().__iter__():
            yield LocustStats(value)

    def get_filename(self, tags: Optional[Iterable[str]] = None,
                     fmt: Format.type_ = Format.PICKLE) -> Path:
        return super().get_filename(fmt, tags=tags)

    def get_test_name(self, locust_stats: 'LocustBaseStats') -> str:
        if self._locust_tasks_with_meta is not None:
            for task in self._locust_tasks_with_meta:
                if task.url == locust_stats.name:
                    return task.test_name

        # Return the endpoint as the default
        return locust_stats.name


class LocustBaseStats(BaseDict):
    class Key:
        type_ = NewType('LocustBaseStats.Key', str)
        NAME: type_ = 'name'
        METHOD: type_ = 'method'

    @property
    def name(self) -> str:
        return self[self.Key.NAME]

    @property
    def method(self) -> str:
        return self[self.Key.METHOD]


class LocustStats(LocustBaseStats):
    class Key(LocustBaseStats.Key):
        type_ = NewType('LocustInternalStatsEntryDict', str)
        LAST_REQUEST_TIMESTAMP: type_ = 'last_request_timestamp'
        START_TIME: type_ = 'start_time'
        NUM_REQUESTS: type_ = 'num_requests'
        NUM_NONE_REQUESTS: type_ = 'num_none_requests'
        NUM_FAILURES: type_ = 'num_failures'
        TOTAL_RESPONSE_TIME: type_ = 'total_response_time'
        MAX_RESPONSE_TIME: type_ = 'max_response_time'
        MIN_RESPONSE_TIME: type_ = 'min_response_time'
        TOTAL_CONTENT_LENGTH: type_ = 'total_content_length'
        RESPONSE_TIMES: type_ = 'response_times'
        NUM_REQS_PER_SEC: type_ = 'num_reqs_per_sec'
        NUM_FAIL_PER_SEC: type_ = 'num_fail_per_sec'

    @property
    def last_request_timestamp(self) -> Optional[float]:
        return self[self.Key.LAST_REQUEST_TIMESTAMP]

    @last_request_timestamp.setter
    def last_request_timestamp(self, value):
        self[self.Key.LAST_REQUEST_TIMESTAMP] = value

    @property
    def start_time(self) -> float:
        return self[self.Key.START_TIME]

    @start_time.setter
    def start_time(self, value):
        self[self.Key.START_TIME] = value

    @property
    def num_requests(self) -> int:
        return self[self.Key.NUM_REQUESTS]

    @num_requests.setter
    def num_requests(self, value):
        self[self.Key.NUM_REQUESTS] = value

    @property
    def num_none_requests(self) -> int:
        return self[self.Key.NUM_NONE_REQUESTS]

    @num_none_requests.setter
    def num_none_requests(self, value):
        self[self.Key.NUM_NONE_REQUESTS] = value

    @property
    def num_failures(self) -> int:
        return self[self.Key.NUM_FAILURES]

    @num_failures.setter
    def num_failures(self, value):
        self[self.Key.NUM_FAILURES] = value

    @property
    def total_response_time(self) -> int:
        return self[self.Key.TOTAL_RESPONSE_TIME]

    @total_response_time.setter
    def total_response_time(self, value):
        self[self.Key.TOTAL_RESPONSE_TIME] = value

    @property
    def max_response_time(self) -> int:
        return self[self.Key.MAX_RESPONSE_TIME]

    @max_response_time.setter
    def max_response_time(self, value):
        self[self.Key.MAX_RESPONSE_TIME] = value

    @property
    def min_response_time(self) -> Optional[int]:
        return self[self.Key.MIN_RESPONSE_TIME]

    @min_response_time.setter
    def min_response_time(self, value):
        self[self.Key.MIN_RESPONSE_TIME] = value

    @property
    def total_content_length(self) -> int:
        return self[self.Key.TOTAL_CONTENT_LENGTH]

    @total_content_length.setter
    def total_content_length(self, value):
        self[self.Key.TOTAL_CONTENT_LENGTH] = value

    @property
    def response_times(self) -> dict[int, int]:
        return self[self.Key.RESPONSE_TIMES]

    @response_times.setter
    def response_times(self, value):
        self[self.Key.RESPONSE_TIMES] = value

    @property
    def num_reqs_per_sec(self) -> dict[int, int]:
        return self[self.Key.NUM_REQS_PER_SEC]

    @num_reqs_per_sec.setter
    def num_reqs_per_sec(self, value):
        self[self.Key.NUM_REQS_PER_SEC] = value

    @property
    def num_fail_per_sec(self) -> dict[int, int]:
        return self[self.Key.NUM_FAIL_PER_SEC]

    @num_fail_per_sec.setter
    def num_fail_per_sec(self, value):
        self[self.Key.NUM_FAIL_PER_SEC] = value
