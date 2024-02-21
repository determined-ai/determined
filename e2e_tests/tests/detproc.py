"""
detproc is a subprocess-like tool for calling our CLI with explicit session management.

e2e tests shouldn't really be relying on the persistence of cached api credentials in order to work;
they should be explicit about which login session should be used to make the test pass.

However, lots of e2e functionality is exercised today through the CLI.  Also, it's unfortunately
true that almost all of the CLI functionality is tested only with e2e tests.  So migrating the whole
e2e test suite to api.bindings or the SDK might be nice for e2e_tests but it would probably result
in huge parts of the CLI having no test coverage at all.

So the detprocess module avoids the dilemma by continuing to use the CLI in e2e tests but offering
a mechansism for explicit session management through the CLI subprocess boundary.
"""

import json
import os
import subprocess
from typing import Any, Dict, List, Optional

from determined.common import api


class CalledProcessError(subprocess.CalledProcessError):
    """
    Subclass subprocess.CalledProcessError in order to have a __str__ method that includes the
    stderr of the det cli call that failed.

    That way, the actual failure surfaces in test logs in the pytest summary info section at the
    bottom of the logs.
    """

    def __str__(self) -> str:
        return (
            f"Command '{self.cmd}' returned non-zero exit status {self.returncode}, "
            f"stderr={self.stderr}"
        )


def mkenv(sess: api.Session, env: Optional[Dict[str, str]]) -> Dict[str, str]:
    env = env or {**os.environ}
    assert "DET_USER" not in env, "if you set DET_USER you probably want to use normal subprocess"
    assert (
        "DET_USER_TOKEN" not in env
    ), "if you set DET_USER_TOKEN you probably want to use normal subprocess"
    # Point at the same master as the session.
    env["DET_MASTER"] = sess.master
    # Configure the username and token directly through the environment, via the codepath normally
    # designed for on-cluster auto-config situations.
    env["DET_USER"] = sess.username
    env["DET_USER_TOKEN"] = sess.token
    # Disable the authentication cache, which, by design, is allowed to override that on-cluster
    # auto-config situation.
    env["DET_DEBUG_CONFIG_PATH"] = "/tmp/disable-e2e-auth-cache"
    # Disable python's stdio buffering.
    env["PYTHONUNBUFFERED"] = "1"
    return env


def forbid_user_setting(cmd: List[str]) -> None:
    if "-u" in cmd or "--user" in cmd:
        raise ValueError(
            "you should never be passing -u or --user to detproc; that is for setting the user "
            "and that functionality belongs to the cli unit tests.  If you want to run as a "
            "different user, either use the sdk or pass in a different Session that is "
            f"authenticated as that user.  Command was: {cmd}"
        )


class Popen(subprocess.Popen):
    def __init__(
        self,
        sess: api.Session,
        cmd: List[str],
        *args: Any,
        env: Optional[Dict[str, str]] = None,
        **kwargs: Any,
    ) -> None:
        forbid_user_setting(cmd)
        super().__init__(cmd, *args, env=mkenv(sess, env), **kwargs)  # type: ignore


def run(
    sess: api.Session,
    cmd: List[str],
    *args: Any,
    env: Optional[Dict[str, str]] = None,
    **kwargs: Any,
) -> subprocess.CompletedProcess:
    forbid_user_setting(cmd)
    p = subprocess.run(cmd, *args, env=mkenv(sess, env), **kwargs)  # type: ignore
    assert isinstance(p, subprocess.CompletedProcess)
    return p


def check_call(
    sess: api.Session,
    cmd: List[str],
    env: Optional[Dict[str, str]] = None,
) -> subprocess.CompletedProcess:
    p = run(sess, cmd, env=env, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    if p.returncode != 0:
        assert p.stdout is not None and p.stderr is not None
        stdout = p.stdout.decode("utf8")
        stderr = p.stderr.decode("utf8")
        raise CalledProcessError(p.returncode, cmd, output=stdout, stderr=stderr)
    return p


def check_output(
    sess: api.Session,
    cmd: List[str],
    env: Optional[Dict[str, str]] = None,
) -> str:
    p = run(sess, cmd, env=env, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    if p.returncode != 0:
        assert p.stderr is not None
        stderr = p.stderr.decode("utf8")
        raise CalledProcessError(p.returncode, cmd, stderr=stderr)
    out = p.stdout.decode()
    assert isinstance(out, str)
    return out


def check_error(
    sess: api.Session,
    cmd: List[str],
    errmsg: str,
) -> subprocess.CompletedProcess:
    p = run(sess, cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    assert p.returncode != 0
    assert p.stderr is not None
    stderr = p.stderr.decode("utf8")
    assert errmsg.lower() in stderr.lower(), f"did not find '{errmsg}' in '{stderr}'"
    return p


def check_json(
    sess: api.Session,
    cmd: List[str],
) -> Any:
    return json.loads(check_output(sess, cmd))
