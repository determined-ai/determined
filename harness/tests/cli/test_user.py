import contextlib
from unittest import mock

import responses
from responses import matchers

from determined.cli import cli
from determined.common import api
from determined.common.api import bindings
from tests.cli import util


@mock.patch("getpass.getpass")
def test_user_change_password(mock_getpass: mock.MagicMock) -> None:
    mock_getpass.side_effect = lambda *_: "newpass"
    with util.standard_cli_rsps() as rsps:
        userobj = bindings.v1User(active=True, admin=False, username="det-user", id=101)
        rsps.get(
            "http://localhost:8080/api/v1/users/tgt-user/by-username",
            status=200,
            json={"user": userobj.to_json()},
        )

        patchobj = bindings.v1PatchUser(isHashed=True, password=api.salt_and_hash("newpass"))
        rsps.patch(
            "http://localhost:8080/api/v1/users/101",
            status=200,
            match=[
                matchers.json_params_matcher(patchobj.to_json(True)),
            ],
            json={"user": userobj.to_json()},
        )

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


@mock.patch("getpass.getpass")
@mock.patch("determined.cli.cli.die")
def test_login_with_invalid_credentials_error_message(
    mock_die: mock.MagicMock, mock_getpass: mock.MagicMock
) -> None:
    mock_getpass.side_effect = lambda *_: "newpass"
    with responses.RequestsMock(
        registry=responses.registries.OrderedRegistry, assert_all_requests_are_fired=True
    ) as rsps:
        util.expect_get_info(rsps)
        rsps.post(
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
