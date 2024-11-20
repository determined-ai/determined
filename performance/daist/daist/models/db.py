from pathlib import Path
from typing import Any, Iterable, NewType, Optional, Union

import psycopg2
from locust.stats import calculate_response_time_percentile, median_from_dict
from prettytable import PrettyTable
from psycopg2 import sql

from ..framework import repo
from ..models.environment import environment
from ..utils.timestamp import BASE_FMT, UTC_Z, TS_t, get_utc_iso_ts_str
from .base import BaseDict, BaseList, BaseObj, Format
from .locust import LocustStatsList


class PerfTestRun(BaseObj):
    DB_NAME = 'postgres'

    class Table:
        PERF_TEST_RUNS = 'perf_test_runs'
        PERF_TESTS = 'perf_tests'

    def __init__(self, locust_stats: LocustStatsList,
                 time: Optional[TS_t] = None):
        super().__init__()
        self._perf_test_runs_table = PerfTestRunsTable()
        self._perf_test_runs_table.add_row(time)
        self._perf_tests_table = PerfTestsTable(locust_stats)

    def get_filename(self, tags: Optional[Iterable[str]] = None,
                     fmt: Format.type_ = Format.TXT) -> Path:
        return super().get_filename(fmt, tags=tags)

    def __str__(self):
        return '\n'.join([f'{self._perf_test_runs_table.get_qualname()}:',
                          str(self._perf_test_runs_table),
                          '',
                          f'{self._perf_tests_table.get_qualname()}:',
                          str(self._perf_tests_table)])


class PerfTestRunsTable(BaseList):
    def add_row(self, time: Optional[TS_t] = None):
        self.append(PerfTestRunsRow(time=time))

    @classmethod
    def _serialize(cls, value: Union[dict, 'PerfTestRunsRow']) -> dict:
        return cls._serialize_to_dict(value)

    @staticmethod
    def _deserialize(value: dict) -> 'PerfTestRunsRow':
        return PerfTestRunsRow(value)

    def __str__(self):
        table = PrettyTable()
        table.field_names = (_fmt_key_for_table(key) for key in PerfTestRunsRow.Key.all_)
        table.align = "l"
        for row in self:
            table.add_row([row.commit, row.branch, row.time])
        return table.get_string()


class PerfTestRunsRow(BaseDict):
    TIME_FMT = BASE_FMT + UTC_Z

    class Key:
        type_ = NewType('LocustResults.Key', str)
        BRANCH: type_ = 'branch'
        COMMIT: type_ = 'commit'
        TIME: type_ = 'time'

        all_ = (COMMIT, BRANCH, TIME)

    def __init__(self, dict_=None, /, **kwargs):
        super().__init__(dict_, **kwargs)

        if self.get(self.Key.TIME) is None:
            # Setting this to None will cause a timestamp sample to be taken.
            self.time = None

        if self.Key.BRANCH not in self:
            self[self.Key.BRANCH] = repo.get_branch()

        if self.Key.COMMIT not in self:
            self[self.Key.COMMIT] = repo.get_commit(mark_dirty_if_needed=False)

    @property
    def branch(self) -> Optional[str]:
        return self.get(self.Key.BRANCH)

    @branch.setter
    def branch(self, value: str):
        self[self.Key.BRANCH] = value

    @property
    def commit(self) -> Optional[str]:
        return self.get(self.Key.COMMIT)

    @commit.setter
    def commit(self, value: str):
        self[self.Key.COMMIT] = value

    @property
    def time(self) -> str:
        return self[self.Key.TIME]

    @time.setter
    def time(self, value: TS_t):
        self[self.Key.TIME] = value

    @classmethod
    def _serialize(cls, key, value) -> Any:
        if key == cls.Key.TIME:
            value = get_utc_iso_ts_str(value, fmt=cls.TIME_FMT, timespec='seconds')
        return super()._serialize(key, value)


class PerfTestsTable(BaseList):
    def __init__(self, initlist: LocustStatsList):
        super().__init__()

        for row in initlist:
            num_response_times = len(row.response_times)
            self.append({
                PerfTestsRow.Key.TEST_NAME: initlist.get_test_name(row),
                PerfTestsRow.Key.AVG: row.total_response_time / row.num_requests,
                PerfTestsRow.Key.MIN: row.min_response_time,
                PerfTestsRow.Key.MED: median_from_dict(num_response_times, row.response_times),
                PerfTestsRow.Key.MAX: row.max_response_time,
                PerfTestsRow.Key.P90: calculate_response_time_percentile(row.response_times,
                                                                         num_response_times, 0.90),
                PerfTestsRow.Key.P95: calculate_response_time_percentile(row.response_times,
                                                                         num_response_times, 0.95),
                PerfTestsRow.Key.PASSES: row.num_requests - row.num_failures,
                PerfTestsRow.Key.FAILS: row.num_failures
            })

    @classmethod
    def _serialize(cls, value: Union[dict, 'PerfTestsRow']) -> dict:
        return cls._serialize_to_dict(value)

    @staticmethod
    def _deserialize(value: dict) -> 'PerfTestsRow':
        return PerfTestsRow(value)

    def __iter__(self) -> 'PerfTestsRow':
        for value in super().__iter__():
            yield PerfTestsRow(value)

    def __str__(self):
        table = PrettyTable()
        table.field_names = (_fmt_key_for_table(key) for key in PerfTestsRow.Key.all_)
        table.align = "l"
        table.align[_fmt_key_for_table(PerfTestsRow.Key.PASSES)] = "r"
        table.align[_fmt_key_for_table(PerfTestsRow.Key.FAILS)] = "r"
        for row in self:
            table.add_row([
                row.test_name,
                row.avg,
                row.min,
                row.med,
                row.max,
                row.p90,
                row.p95,
                row.passes,
                row.fails])
        return table.get_string()


class PerfTestsRow(BaseDict):
    class Key:
        type_ = NewType('PerfResults.Key', str)

        TEST_NAME: type_ = 'test_name'
        AVG: type_ = 'avg'
        MIN: type_ = 'min'
        MED: type_ = 'med'
        MAX: type_ = 'max'
        P90: type_ = 'p90'
        P95: type_ = 'p95'
        PASSES: type_ = 'passes'
        FAILS: type_ = 'fails'

        all_ = (TEST_NAME, AVG, MIN, MED, MAX, P90, P95, PASSES, FAILS)

    @property
    def test_name(self) -> str:
        return self[self.Key.TEST_NAME]

    @test_name.setter
    def test_name(self, value: str):
        self[self.Key.TEST_NAME] = value

    @property
    def avg(self) -> float:
        return self[self.Key.AVG]

    @avg.setter
    def avg(self, value: float):
        self[self.Key.AVG] = value

    @property
    def min(self) -> float:
        return self[self.Key.MIN]

    @min.setter
    def min(self, value: float):
        self[self.Key.MIN] = value

    @property
    def med(self) -> float:
        return self[self.Key.MED]

    @med.setter
    def med(self, value: float):
        self[self.Key.MED] = value

    @property
    def max(self) -> float:
        return self[self.Key.MAX]

    @max.setter
    def max(self, value: float):
        self[self.Key.MAX] = value

    @property
    def p90(self) -> float:
        return self[self.Key.P90]

    @p90.setter
    def p90(self, value: float):
        self[self.Key.P90] = value

    @property
    def p95(self) -> float:
        return self[self.Key.P95]

    @p95.setter
    def p95(self, value: float):
        self[self.Key.P95] = value

    @property
    def passes(self) -> int:
        return self[self.Key.PASSES]

    @passes.setter
    def passes(self, value: int):
        self[self.Key.PASSES] = value

    @property
    def fails(self) -> int:
        return self[self.Key.FAILS]

    @fails.setter
    def fails(self, value: int):
        self[self.Key.FAILS] = value


def _fmt_key_for_table(key: str) -> str:
    return key.replace('_', ' ').title()
