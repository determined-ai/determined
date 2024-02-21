import contextlib
import os
import re
import subprocess
from typing import IO, Any, Iterator, List, Optional

import requests

from determined.common import api
from tests import config as conf
from tests import detproc


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
            m = None
            max_iterations = 2
            iterations = 0
            while not m and iterations < max_iterations:
                line = next(iterator)
                iterations += 1
                m = re.search(rb"Launched .* \(id: (.*)\)", line)
            assert m is not None
            self.task_id = m.group(1).decode() if m else None

    @property
    def stdout(self) -> Iterator[str]:
        assert self.process.stdout is not None
        for line in self.process.stdout:
            yield line.decode()

    @property
    def stderr(self) -> Iterator[str]:
        assert self.process.stderr is not None
        return (line.decode() for line in self.process.stderr)

    @property
    def stdin(self) -> IO:
        assert self.process.stdin is not None
        return self.process.stdin


@contextlib.contextmanager
def interactive_command(sess: api.Session, args: List[str]) -> Iterator[_InteractiveCommandProcess]:
    """
    Runs a Determined CLI command in a subprocess. On exit, it kills the
    corresponding Determined task if possible before closing the subprocess.

    Example usage:

    with util.interactive_command(sess, ["notebook", "start"]) as notebook:
        for line in notebook.stdout:
            if "Jupyter Notebook is running" in line:
                break
    """

    with detproc.Popen(
        sess,
        ["det"] + args,
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
    ) as p:
        cmd = _InteractiveCommandProcess(p, detach="--detach" in args)
        if cmd.task_id is None:
            raise AssertionError(
                f"Task ID for '{args}' could not be found. "
                "If it is still active, this command may persist "
                "in the Determined test deployment..."
            )
        try:
            yield cmd
        finally:
            detproc.check_call(sess, ["det", str(args[0]), "kill", cmd.task_id])
            p.kill()


def get_num_active_commands(sess: api.Session) -> int:
    r = sess.get("api/v1/commands")
    assert r.status_code == requests.codes.ok, r.text

    return len(
        [
            command
            for command in r.json()["commands"]
            if (
                command["state"] == "STATE_PULLING"
                or command["state"] == "STATE_STARTING"
                or command["state"] == "STATE_RUNNING"
            )
        ]
    )


def get_command(sess: api.Session, command_id: str) -> Any:
    r = sess.get("api/v1/commands/" + command_id)
    assert r.status_code == requests.codes.ok, r.text
    return r.json()["command"]


def get_command_config(sess: api.Session, command_type: str, task_id: str) -> str:
    assert command_type in ["command", "notebook", "shell"]
    command = ["det", "-m", conf.make_master_url(), command_type, "config", task_id]
    env = os.environ.copy()
    env["DET_DEBUG"] = "true"
    return detproc.check_output(sess, command, env)


def print_command_logs(sess: api.Session, task_id: str) -> bool:
    for tl in api.task_logs(sess, task_id):
        print(tl.message)
    return True
