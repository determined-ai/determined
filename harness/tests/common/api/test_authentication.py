import contextlib
import json
import pathlib
import shutil
from typing import Optional

import pytest
import responses
from responses import registries

from determined.common.api import authentication
from tests import confdir
from tests.cli import util

MOCK_MASTER_URL = "http://localhost:8080"
AUTH_V0_PATH = pathlib.Path(__file__).parent / "auth_v0.json"


@pytest.mark.parametrize("active_user", ["alice", "bob", None])
def test_logout_clears_active_user(active_user: Optional[str]) -> None:
    with contextlib.ExitStack() as es:
        es.enter_context(util.setenv_optional("DET_MASTER", MOCK_MASTER_URL))
        rsps = es.enter_context(
            responses.RequestsMock(
                registry=registries.OrderedRegistry,
                assert_all_requests_are_fired=True,
            )
        )
        mts = es.enter_context(util.MockTokenStore(strict=True))

        mts.get_active_user(retval=active_user)
        if active_user == "alice":
            mts.clear_active()
        mts.get_token("alice", retval="token")
        mts.drop_user("alice")
        rsps.post(f"{MOCK_MASTER_URL}/api/v1/auth/logout", status=200)

        authentication.logout(MOCK_MASTER_URL, "alice", None)


def test_auth_json_v0_upgrade() -> None:
    with confdir.use_test_config_dir() as config_dir:
        auth_json_path = config_dir / "auth.json"
        shutil.copy2(AUTH_V0_PATH, auth_json_path)
        ts = authentication.TokenStore(MOCK_MASTER_URL, auth_json_path)

        assert ts.get_active_user() == "determined"
        assert ts.get_token("determined") == "v2.public.this.is.a.test"

        ts.set_token("determined", "ai")

        ts2 = authentication.TokenStore(MOCK_MASTER_URL, auth_json_path)
        assert ts2.get_token("determined") == "ai"

        with auth_json_path.open() as fin:
            data = json.load(fin)
            assert data.get("version") == 1
            assert "masters" in data and list(data["masters"].keys()) == [MOCK_MASTER_URL]
