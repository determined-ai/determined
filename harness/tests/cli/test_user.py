from unittest import mock

import pytest
import responses
from responses import matchers

from determined.cli import cli
from determined.common import api
from determined.common.api import bindings
from tests.cli import util


def test_user_create_password_flag_does_not_prompt() -> None:
    with util.standard_cli_rsps() as rsps:
        userobj = bindings.v1User(active=True, admin=True, username="det-user", id=10017)
        rsps.post(
            "http://localhost:8080/api/v1/users",
            status=200,
            match=[
                matchers.json_params_matcher(
                    params={
                        "isHashed": True,
                        "user": {
                            "username": "test-user-1",
                        },
                        "password": api.salt_and_hash("5DCAB140-f49b-4260-a451-fad6a10017ca"),
                    },
                    strict_match=False,
                ),
            ],
            json={"user": userobj.to_json()},  # doesn't really match, but doesn't matter yet
        )
        cli.main(
            ["user", "create", "test-user-1", "--password", "5DCAB140-f49b-4260-a451-fad6a10017ca"]
        )


def test_user_create_remote_account_does_not_require_password() -> None:
    with util.standard_cli_rsps() as rsps:
        userobj = bindings.v1User(active=True, admin=True, username="det-user", id=10018)
        rsps.post(
            "http://localhost:8080/api/v1/users",
            status=200,
            match=[
                matchers.json_params_matcher(
                    params={
                        "isHashed": True,
                        "user": {
                            "username": "test-user-2",
                            "remote": True,
                        },
                    },
                    strict_match=False,
                ),
            ],
            json={"user": userobj.to_json()},
        )
        cli.main(["user", "create", "test-user-2", "--remote"])


@mock.patch("getpass.getpass")
def test_user_create_interactive_password(mock_getpass: mock.MagicMock) -> None:
    mock_getpass.side_effect = lambda *_: "8CBAAB59-21c5-45cb-b058-6e2f3ceaf03e"
    with util.standard_cli_rsps() as rsps:
        userobj = bindings.v1User(active=True, admin=True, username="det-user", id=10019)
        rsps.post(
            "http://localhost:8080/api/v1/users",
            status=200,
            match=[
                matchers.json_params_matcher(
                    params={
                        "isHashed": True,
                        "user": {
                            "username": "test-user-3",
                        },
                        "password": api.salt_and_hash("8CBAAB59-21c5-45cb-b058-6e2f3ceaf03e"),
                    },
                    strict_match=False,
                ),
            ],
            json={"user": userobj.to_json()},  # doesn't really match, but doesn't matter yet
        )
        cli.main(["user", "create", "test-user-3"])


@mock.patch("getpass.getpass")
def test_user_create_fails_with_empty_password(mock_getpass: mock.MagicMock) -> None:
    mock_getpass.side_effect = lambda *_: ""
    with pytest.raises(SystemExit):
        cli.main(["user", "create", "test-user-4"])


@mock.patch("getpass.getpass")
def test_user_change_password(mock_getpass: mock.MagicMock) -> None:
    mock_getpass.side_effect = lambda *_: "ce93AA76-2f62-4f29-ab5d-c56a3375e702"
    with util.standard_cli_rsps() as rsps:
        userobj = bindings.v1User(active=True, admin=False, username="det-user", id=101)
        rsps.get(
            "http://localhost:8080/api/v1/users/tgt-user/by-username",
            status=200,
            json={"user": userobj.to_json()},
        )

        patchobj = bindings.v1PatchUser(
            isHashed=True, password=api.salt_and_hash("ce93AA76-2f62-4f29-ab5d-c56a3375e702")
        )
        rsps.patch(
            "http://localhost:8080/api/v1/users/101",
            status=200,
            match=[
                matchers.json_params_matcher(patchobj.to_json(True)),
            ],
            json={"user": userobj.to_json()},
        )

        cli.main(["user", "change-password", "tgt-user"])


@mock.patch("getpass.getpass")
def test_user_change_password_blank_fails(mock_getpass: mock.MagicMock) -> None:
    mock_getpass.side_effect = lambda *_: ""
    with util.standard_cli_rsps() as rsps:
        userobj = bindings.v1User(active=True, admin=False, username="det-user", id=102)
        rsps.get(
            "http://localhost:8080/api/v1/users/tgt-user/by-username",
            status=200,
            json={"user": userobj.to_json()},
        )
        with pytest.raises(SystemExit):
            cli.main(["user", "change-password", "tgt-user"])


@mock.patch("getpass.getpass")
def test_user_change_password_weak_fails(mock_getpass: mock.MagicMock) -> None:
    mock_getpass.side_effect = lambda *_: "password"
    with util.standard_cli_rsps() as rsps:
        userobj = bindings.v1User(active=True, admin=False, username="det-user", id=103)
        rsps.get(
            "http://localhost:8080/api/v1/users/tgt-user/by-username",
            status=200,
            json={"user": userobj.to_json()},
        )
        with pytest.raises(SystemExit):
            cli.main(["user", "change-password", "tgt-user"])


@mock.patch("determined.cli.cli.die")
def test_user_edit_no_fields(mock_die: mock.MagicMock) -> None:
    with util.standard_cli_rsps() as rsps:
        userobj = bindings.v1User(active=True, admin=False, username="det-user", id=101)
        rsps.get(
            "http://localhost:8080/api/v1/users/det-user/by-username",
            status=200,
            json={"user": userobj.to_json()},
        )

        # No edited field should result in error
        cli.main(["user", "edit", "det-user"])
        mock_die.assert_has_calls(
            [mock.call("No field provided. Use 'det user edit -h' for usage.", exit_code=1)]
        )


@responses.activate()
@mock.patch("determined.common.api.authentication.TokenStore", util.MockTokenStore(strict=False))
@mock.patch("getpass.getpass", lambda *_: "newpass")
@mock.patch("determined.cli.cli.die")
def test_login_dies_with_invalid_credentials_error_message(mock_die: mock.MagicMock) -> None:
    util.expect_get_info()
    responses.post(
        "http://localhost:8080/api/v1/auth/login",
        status=401,
    )
    cli.main(["user", "login", "test-user"])
    mock_die.assert_has_calls(
        [
            mock.call(
                "Failed to log in user: Invalid username/password combination. Please try again."
            )
        ]
    )
