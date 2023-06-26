import json
import subprocess
from typing import Any, List

from determined.common.api import authentication
from tests import config as conf


def det_cmd(cmd: List[str], **kwargs: Any) -> subprocess.CompletedProcess:
    return subprocess.run(
        ["det", "-m", conf.make_master_url()] + cmd,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        **kwargs,
    )


def det_cmd_json(cmd: List[str]) -> Any:
    res = det_cmd(cmd, check=True)
    return json.loads(res.stdout)


def det_cmd_expect_error(cmd: List[str], expected: str) -> None:
    res = det_cmd(cmd)
    assert res.returncode != 0
    assert expected in res.stderr.decode()


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
