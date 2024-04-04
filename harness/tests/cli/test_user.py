from unittest import mock

import pytest
from responses import matchers

from determined.cli import cli
from determined.common import api
from determined.common.api import bindings
from tests.cli import util


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

    # cannot set password to blank
    mock_getpass.side_effect = lambda *_: ""
    with util.standard_cli_rsps() as rsps:
        rsps.get(
            "http://localhost:8080/api/v1/users/tgt-user/by-username",
            status=200,
            json={"user": userobj.to_json()},
        )
        with pytest.raises(SystemExit):
            cli.main(["user", "change-password", "tgt-user"])

    # cannot set password to something weak
    mock_getpass.side_effect = lambda *_: "password"
    with util.standard_cli_rsps() as rsps:
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
