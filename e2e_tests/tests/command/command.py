import os
import re
import subprocess
from contextlib import contextmanager
from typing import IO, Any, Generator, Optional

import requests

import determined.common.api.authentication as auth
from determined.common import api
from tests import config as conf


@contextmanager
def interactive_command(*args: str) -> Generator:
    """
    Runs a Determined CLI command in a subprocess. On exit, it kills the
    corresponding Determined task if possible before closing the subprocess.

    Example usage:

    with util.interactive_command("notebook", "start") as notebook:
        for line in notebook.stdout:
            if "Jupyter Notebook is running" in line:
                break
    """

    class _InteractiveCommandProcess:
        def __init__(self, process: subprocess.Popen, detach: bool = False):
            self.process = process
            self.detach = detach
            self.task_id = None  # type: Optional[str]

            if self.detach:
                iterator = iter(self.process.stdout)  # type: ignore
                line = next(iterator)
                self.task_id = line.decode().strip()
            else:
                iterator = iter(self.process.stdout)  # type: ignore
                line = next(iterator)
                m = re.search(rb"Scheduling .* \(id: (.*)\)\.\.\.", line)
                assert m is not None
                self.task_id = m.group(1).decode() if m else None

        @property
        def stdout(self) -> Generator[str, None, None]:
            assert self.process.stdout is not None
            for line in self.process.stdout:
                yield line.decode()

        @property
        def stderr(self) -> Generator[str, None, None]:
            assert self.process.stderr is not None
            return (line.decode() for line in self.process.stderr)

        @property
        def stdin(self) -> IO:
            assert self.process.stdin is not None
            return self.process.stdin

    with subprocess.Popen(
        ("det", "-m", conf.make_master_url()) + args,
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        env={"PYTHONUNBUFFERED": "1", **os.environ},
    ) as p:
        cmd = _InteractiveCommandProcess(p, detach="--detach" in args)
        if cmd.task_id is None:
            raise AssertionError(
                "Task ID for '{}' could not be found. "
                "If it is still active, this command may persist "
                "in the Determined test deployment...".format(args)
            )
        try:
            yield cmd
        finally:
            subprocess.check_call(
                ["det", "-m", conf.make_master_url(), str(args[0]), "kill", cmd.task_id]
            )
            p.kill()


def get_num_running_commands() -> int:
    auth.initialize_session(conf.make_master_url(), try_reauth=True)
    r = api.get(conf.make_master_url(), "api/v1/commands")
    assert r.status_code == requests.codes.ok, r.text

    return len([command for command in r.json()["commands"] if command["state"] == "STATE_RUNNING"])


def get_command(id: str) -> Any:
    auth.initialize_session(conf.make_master_url(), try_reauth=True)
    r = api.get(conf.make_master_url(), "api/v1/commands/" + id)
    assert r.status_code == requests.codes.ok, r.text
    return r.json()["command"]


def get_command_config(command_type: str, id: str) -> str:
    assert command_type in ["command", "notebook", "shell"]
    command = ["det", "-m", conf.make_master_url(), command_type, "config", id]
    env = os.environ.copy()
    env["DET_DEBUG"] = "true"
    completed_process = subprocess.run(
        command,
        universal_newlines=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        env=env,
    )
    assert completed_process.returncode == 0, "\nstdout:\n{} \nstderr:\n{}".format(
        completed_process.stdout, completed_process.stderr
    )
    return str(completed_process.stdout)
