import unittest
from pathlib import Path
from unittest import TestCase

from daist.models.db import PerfTestRun, PerfTestRunsTable
from daist.models.environment import environment
from daist.models.locust import LocustStatsList
import daist


PATH_TO_SAMPLES = Path(__file__).parent / 'samples'
PATH_TO_SAMPLE_INPUT = (PATH_TO_SAMPLES /
                        'LocustStatsList-LocustTest.test_concurrent_read_only_users.pkl')
PATH_TO_REFERENCE_OUTPUT = (PATH_TO_SAMPLES /
                            'PerfTestRun-LocustTest.test_concurrent_read_only_users.txt')


class TestPerfTestDb(TestCase):
    _saved_branch = None
    _saved_commit = None
    _saved_version = None

    @classmethod
    def setUpClass(cls):
        super().setUpClass()
        cls._saved_branch = environment.git_branch
        cls._saved_commit = environment.git_commit
        cls._saved_version = daist.__version__

        environment[environment.Key.GIT_BRANCH] = 'None'
        environment[environment.Key.GIT_COMMIT] = 'None'
        daist.__version__ = None

    @classmethod
    def tearDownClass(cls):
        super().tearDownClass()
        environment[environment.Key.GIT_BRANCH] = cls._saved_branch
        environment[environment.Key.GIT_COMMIT] = cls._saved_commit
        daist.__version__ = cls._saved_version

    def test_open(self):
        locust_stats = LocustStatsList.open(PATH_TO_SAMPLE_INPUT)
        perf_tests_run = PerfTestRun(locust_stats, time=0)
        with open(PATH_TO_REFERENCE_OUTPUT, 'r') as inf:
            self.assertEqual(inf.read(), str(perf_tests_run))

    def test_table(self):
        exp = """\
+--------+--------+----------------------+
| Commit | Branch | Time                 |
+--------+--------+----------------------+
| None   | None   | 1970-01-01T00:00:00Z |
+--------+--------+----------------------+"""
        table = PerfTestRunsTable()
        table.add_row(time=0)
        self.assertEqual(exp, str(table))


@unittest.skipIf(True, 'For development/debugging purposes.')
class TestUpload(TestCase):
    _saved_daist_db_pass = None

    def test(self):
        locust_stats = LocustStatsList.open(PATH_TO_SAMPLE_INPUT)
        run = PerfTestRun(locust_stats, time=0)
        run.upload()
