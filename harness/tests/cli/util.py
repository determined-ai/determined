import contextlib
import difflib
import io
import os
from typing import Any, Iterator, List, Optional, cast

import responses
from responses import registries

import determined as det
from determined.cli import cli
from determined.common.api import authentication


@contextlib.contextmanager
def setenv_optional(key: str, val: Optional[str]) -> Iterator:
    old = os.environ.get(key)
    if val is None:
        os.environ.pop(key, None)
    else:
        os.environ[key] = val
    try:
        yield
    finally:
        if old is None:
            os.environ.pop(key, None)
        else:
            os.environ[key] = old


class MockTokenStore:
    def __init__(self, strict: bool) -> None:
        self._mock = MockTokenStoreInstance(self)
        self._strict = strict

    def __call__(self, *_: Any) -> "MockTokenStoreInstance":
        # For when somebody calls authentication.TokenStore().
        return self._mock

    def __enter__(self) -> "MockTokenStore":
        self._exp_calls: List[Any] = []
        self._retvals: List[Any] = []
        self._ncalls = 0
        self._real = authentication.TokenStore
        authentication.TokenStore = self  # type: ignore
        return self

    def __exit__(self, exc_type: Any, *_: Any) -> None:
        authentication.TokenStore = self._real  # type: ignore
        if exc_type is not None:
            return
        if not self._strict:
            return
        if self._ncalls == len(self._exp_calls):
            return
        raise ValueError(f"missing expected calls: {self._exp_calls[self._ncalls:]}")

    def _match_call(self, call: Any) -> Any:
        if self._strict:
            if self._ncalls == len(self._exp_calls):
                raise ValueError(f"unexpected call to TokenStore: {call}")
            if self._exp_calls[self._ncalls] != call:
                raise ValueError(
                    f"mismstached call to TokenStore: expected {self._exp_calls[self._ncalls]} "
                    f"but got {call}"
                )
            retval = self._retvals[self._ncalls]
            self._ncalls += 1
            return retval
        else:
            try:
                idx = self._exp_calls.index(call)
            except ValueError:
                raise ValueError(f"call to TokenStore has no match: {call}")
            return self._retvals[idx]

    def get_active_user(self, *, retval: Optional[str]) -> None:
        self._exp_calls.append("get_active_user")
        self._retvals.append(retval)

    def get_all_users(self, *, retval: List[str]) -> None:
        self._exp_calls.append("get_all_users")
        self._retvals.append(retval)

    def get_token(self, user: str, *, retval: Optional[str]) -> None:
        self._exp_calls.append(("get_token", user))
        self._retvals.append(retval)

    def drop_user(self, username: str) -> None:
        self._exp_calls.append(("drop_user", username))
        self._retvals.append(None)

    def set_token(self, username: str, token: str) -> None:
        self._exp_calls.append(("set_token", username, token))
        self._retvals.append(None)

    def set_active(self, username: str) -> None:
        self._exp_calls.append(("set_active", username))
        self._retvals.append(None)

    def clear_active(self) -> None:
        self._exp_calls.append("clear_active")
        self._retvals.append(None)


class MockTokenStoreInstance:
    def __init__(self, mts: MockTokenStore) -> None:
        self._mts = mts

    def get_active_user(self) -> Optional[str]:
        return cast(Optional[str], self._mts._match_call("get_active_user"))

    def get_all_users(self) -> Optional[str]:
        return cast(Optional[str], self._mts._match_call("get_all_users"))

    def get_token(self, user: str) -> Optional[str]:
        return cast(Optional[str], self._mts._match_call(("get_token", user)))

    def drop_user(self, username: str) -> None:
        self._mts._match_call(("drop_user", username))

    def set_token(self, username: str, token: str) -> None:
        self._mts._match_call(("set_token", username, token))

    def set_active(self, username: str) -> None:
        self._mts._match_call(("set_active", username))

    def clear_active(self) -> None:
        self._mts._match_call("clear_active")


@contextlib.contextmanager
def standard_cli_rsps() -> Iterator[responses.RequestsMock]:
    with contextlib.ExitStack() as es:
        es.enter_context(setenv_optional("DET_USER", "det-user"))
        es.enter_context(setenv_optional("DET_USER_TOKEN", "det-token"))
        mts = es.enter_context(MockTokenStore(strict=False))
        mts.get_active_user(retval="det-user")
        mts.get_token("det-user", retval="det-token")
        rsps = es.enter_context(
            responses.RequestsMock(
                registry=registries.OrderedRegistry, assert_all_requests_are_fired=True
            )
        )
        expect_get_info(rsps)
        rsps.get("http://localhost:8080/api/v1/me", status=200)
        yield rsps


def expect_get_info(
    rsps: Optional[responses.RequestsMock] = None, master_url: str = "http://localhost:8080"
) -> None:
    if rsps:
        rsps.get(f"{master_url}/info", status=200, json={"version": det.__version__})
    else:
        responses.get(f"{master_url}/info", status=200, json={"version": det.__version__})


def check_cli_output(args: List[str], expected: str) -> None:
    """
    Helper method to test CLI methods that checks redirected STDOUT from the executed command
    matches expected output.
    """
    with contextlib.redirect_stdout(io.StringIO()) as f:
        cli.main(args=args)
    actual = f.getvalue()
    exp_lines = expected.splitlines(keepends=True)
    act_lines = actual.splitlines(keepends=True)
    diff_lines = difflib.ndiff(act_lines, exp_lines)
    diff = "".join(diff_lines)
    assert actual == expected, f"CLI output for {args} actual(-) != expected(+):\n {diff}"
