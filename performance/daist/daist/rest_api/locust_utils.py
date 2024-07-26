from abc import abstractmethod
from collections import UserList
import logging
from typing import Callable, Dict, Iterator, List, Optional, Union, Type
from urllib.parse import urlencode

from locust import HttpUser, TaskSet
from requests import Response

from ..framework.typelib import Url_t

Weight_t = int
HttpTask_t = Union[Callable[[HttpUser], Response], TaskSet]
HttpTasks_t = Union[List[HttpTask_t], Dict[HttpTask_t, Weight_t]]


def debug_http():
    # HTTP debugging
    from http.client import HTTPConnection

    HTTPConnection.debuglevel = 1
    logging.basicConfig()
    logging.getLogger().setLevel(logging.DEBUG)
    requests_log = logging.getLogger("requests.packages.urllib3")
    requests_log.setLevel(logging.DEBUG)
    requests_log.propagate = True


class LocustTasksWithMeta(UserList):
    def __init__(self, initlist: Optional[List['BaseLocustTaskWithMeta']] = None):
        super().__init__(initlist)

    def __iter__(self) -> Iterator['BaseLocustTaskWithMeta']:
        yield from super().__iter__()

    def __getitem__(self, item) -> 'BaseLocustTaskWithMeta':
        return super().__getitem__(item)

    @property
    def tasks(self) -> HttpTasks_t:
        return [entry.task for entry in self]


class BaseLocustTaskWithMeta:
    def __init__(self, endpoint: Url_t, test_name: Optional[str] = None):
        self.endpoint = endpoint
        self.test_name = test_name if test_name is not None else endpoint

    @abstractmethod
    def task(self, user: HttpUser) -> Response:
        ...

    @property
    @abstractmethod
    def url(self) -> Url_t:
        ...

    @url.setter
    @abstractmethod
    def url(self, value):
        ...


class LocustGetTaskWithMeta(BaseLocustTaskWithMeta):
    def __init__(self, endpoint: Url_t, *, params: Optional[Dict] = None,
                 test_name: Optional[str] = None):
        super().__init__(endpoint, test_name)

        self.params = params
        self._url = endpoint
        if params is not None:
            self._url += f'?{urlencode(params)}'

    @property
    def url(self) -> Url_t:
        return self._url

    def task(self, user: HttpUser) -> Response:
        return user.client.get(self.url)


class LocustPostTaskWithMeta(BaseLocustTaskWithMeta):
    def __init__(self, endpoint: Url_t, *, body: Optional[Dict] = None,
                 test_name: Optional[str] = None):
        super().__init__(endpoint, test_name)
        self.body = body

    @property
    def url(self) -> Url_t:
        return self.endpoint

    def task(self, user: HttpUser) -> Response:
        return user.client.post(self.endpoint, json=self.body)


def get_task(endpoint, params=None) -> HttpTask_t:
    query_string = ""
    if params is not None:
        query_string = f"?{urlencode(params)}"

    def task(user: HttpUser):
        return user.client.get(f"{endpoint}{query_string}")
    return task


def post_task(endpoint, body=None) -> HttpTask_t:
    def task(user: HttpUser):
        return user.client.post(endpoint, json=body)
    return task


def create_locust_user_class(admin_username: str, admin_password: str,
                             tasks_with_meta: LocustTasksWithMeta) -> Type[HttpUser]:
    class LocustUser(HttpUser):
        tasks = tasks_with_meta.tasks

        login_task_with_meta = LocustPostTaskWithMeta("/api/v1/auth/login", test_name="login")
        logout_task_with_meta = LocustPostTaskWithMeta("/api/v1/auth/logout", test_name="logout")
        tasks_with_meta.append(login_task_with_meta)
        tasks_with_meta.append(logout_task_with_meta)

        # TODO configure wait_time?

        def on_start(self):
            self.client.post(self.login_task_with_meta.url, json={
                "username": admin_username,
                "password": admin_password
            })
    
        def on_stop(self):
            self.client.post(self.logout_task_with_meta.url)
    return LocustUser
