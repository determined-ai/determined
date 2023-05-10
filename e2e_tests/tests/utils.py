import time
from typing import Any, Callable, Tuple, TypeVar

from determined.common.api import authentication
from tests import config as conf

T = TypeVar("T")


def wait_for(predicate: Callable[[], Tuple[bool, T]], timeout: int) -> T:
    """
    Wait for the predicate to return (Done, ReturnValue) while
    checking for a timeout. without preempting the predicate.
    """

    start = time.time()
    done, rv = predicate()
    while not done:
        if time.time() - start > timeout:
            raise TimeoutError("timed out waiting for predicate")
        time.sleep(0.1)
    return rv


class CliArgsMock:
    """Mock the CLI args to mimic invoking the CLI with the given args."""

    def __init__(self, **kwargs: Any) -> None:
        if "master" not in kwargs:
            kwargs["master"] = conf.make_master_url()
        if "user" not in kwargs:
            token_store = authentication.TokenStore(kwargs["master"])
            kwargs["user"] = token_store.get_active_user()
        self.__dict__.update(kwargs)

    def __getattr__(self, name: Any) -> Any:
        return self.__dict__.get(name, None)
