from typing import Any

from determined.common.api import authentication
from tests import config as conf


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
