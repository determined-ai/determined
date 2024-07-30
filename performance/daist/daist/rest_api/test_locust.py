import logging
import time
from typing import List
from unittest import TestCase, skipIf

from locust import events
from locust.env import Environment as LocustEnvironment
from locust.env import LocalRunner
from locust.stats import StatsEntry
from requests.exceptions import HTTPError
from requests.models import Response

from ..models.db import PerfTestRun
from ..models.environment import environment
from ..models.locust import LocustStatsList
from ..models.result import FileMeta
from ..models.session import session
from ..utils import flags
from ..utils.misc import parse_class_and_method_name_from_test_id
from . import locust_utils, tasks
from .resources import Resources, get_resource_profile

# NOTE: uncomment below to debug logging of HTTP
# locust_utils.debug_http()

logger = logging.getLogger(__name__)


# noinspection PyUnusedLocal
@events.request.add_listener
def my_request_handler(request_type, name, response_time, response_length, response: Response,
                       context, exception, start_time, url, **kwargs):
    if exception:
        logger.error(f'Locust request error:\n'
                     f'Exception: {exception}\n'
                     f'Response text: {response.text}\n')


class TestRO(TestCase):
    _env: LocustEnvironment = None
    _resources: Resources = None
    _tasks: locust_utils.LocustTasksWithMeta = None
    _runner: LocalRunner = None
    _stop_timeout = 60.0
    _TEST_LENGTH_SEC = 300
    _USERS = 10

    @classmethod
    def setUpClass(cls):
        cls._resources = get_resource_profile(session.cfg)
        user_class = locust_utils.create_locust_user_class(session.determined.user,
                                                           session.determined.password,
                                                           cls._get_task_list())
        cls._env = LocustEnvironment(
            user_classes=[user_class],
            events=events,
            host=session.determined.host,
            reset_stats=True,
            stop_timeout=cls._stop_timeout,
        )
        cls._runner = cls._env.create_local_runner()

    @classmethod
    def _get_task_list(cls) -> locust_utils.LocustTasksWithMeta:
        if cls._tasks is None:
            cls._tasks = tasks.read_only_tasks(cls._resources)
        return cls._tasks

    def tearDown(self):
        self._save_to_results(self._env.stats.serialize_stats(), self._get_task_list())

    def _save_to_results(self, locust_stats: List[StatsEntry],
                         locust_tasks_with_meta: locust_utils.LocustTasksWithMeta):
        # Save the source data
        locust_stats = LocustStatsList(locust_stats, locust_tasks_with_meta)
        file_meta = FileMeta()
        file_meta.test_id = self.id()
        tag = parse_class_and_method_name_from_test_id(self.id())
        session.result.add_obj(locust_stats, locust_stats.get_filename(tags=(tag,)), file_meta)

        # Create and save the text representation of the performance test database tables.
        perf_tests_run = PerfTestRun(locust_stats)
        file_meta = FileMeta()
        file_meta.test_id = self.id()
        session.result.add_obj(perf_tests_run, perf_tests_run.get_filename(tags=(tag,)), file_meta)
        if environment.secrets.perf_result_db_pass is not None:
            logger.info(f'Uploading {self.id()} results.')
            try:
                perf_tests_run.upload()
            except Exception as err:
                logger.exception(f'Failed to upload {self.id()} results.')
                raise err

    def test(self):
        self._runner.start(self._USERS, spawn_rate=self._USERS)
        time.sleep(self._TEST_LENGTH_SEC)
        self._runner.quit()
        self._runner.greenlet.join()


@skipIf(not flags.SCALE_32, 'As of determined==0.34.0, this test causes the server to hang after a '
                            'few iterations.')
class TestExperimentCheckpoints(TestRO):
    @classmethod
    def _get_task_list(cls) -> locust_utils.LocustTasksWithMeta:
        if cls._tasks is None:
            cls._tasks = locust_utils.LocustTasksWithMeta()
            cls._tasks.append(locust_utils.LocustGetTaskWithMeta(
                              f"/api/v1/experiments/{cls._resources.experiment_id}/checkpoints",
                              test_name="get experiment checkpoints"))
        return cls._tasks
