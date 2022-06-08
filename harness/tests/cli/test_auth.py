import contextlib
import copy
import json
import os
import shutil
from pathlib import Path
from typing import Any, Dict, Iterator, Optional

import pytest
import requests_mock

from determined.common.api.authentication import Authentication, TokenStore
from determined.common.api.certs import default_load as certs_default_load
from determined.common.api.errors import CorruptTokenCacheException, UnauthenticatedException
from tests.confdir import use_test_config_dir

MOCK_MASTER_URL = "http://localhost:8080/"
AUTH_V0_PATH = Path(__file__).parent / "auth_v0.json"
UNTRUSTED_CERT_PATH = Path(__file__).parents[1] / "common" / "untrusted-root" / "127.0.0.1-ca.crt"
AUTH_JSON = {
    "version": 1,
    "masters": {
        "http://localhost:8080/": {
            "active_user": "bob",
            "tokens": {
                "determined": "det.token",
                "bob": "bob.token",
            },
        }
    },
}


@pytest.mark.parametrize("user", [None, "Bob"])
def test_auth_no_store_no_reauth(user: Optional[str]) -> None:
    with use_test_config_dir():
        with pytest.raises(UnauthenticatedException):
            Authentication(MOCK_MASTER_URL, user)


@pytest.mark.parametrize("user", [None, "bob", "determined"])
def test_auth_with_store(requests_mock: requests_mock.Mocker, user: Optional[str]) -> None:
    with use_test_config_dir() as config_dir:
        auth_json_path = config_dir / "auth.json"
        with open(auth_json_path, "w") as f:
            json.dump(AUTH_JSON, f)

        expected_user = "determined" if user == "determined" else "bob"
        expected_token = "det.token" if user == "determined" else "bob.token"
        requests_mock.get(
            "/users/me",
            status_code=200,
            json={"username": expected_user},
        )
        authentication = Authentication(MOCK_MASTER_URL, user)
        assert authentication.session.username == expected_user
        assert authentication.session.token == expected_token


@contextlib.contextmanager
def set_container_env_vars() -> Iterator[None]:
    try:
        os.environ["DET_USER"] = "alice"
        os.environ["DET_USER_TOKEN"] = "alice.token"
        yield
    finally:
        del os.environ["DET_USER"]
        del os.environ["DET_USER_TOKEN"]


@pytest.mark.parametrize("user", [None, "bob", "determined"])
@pytest.mark.parametrize("has_token_store", [True, False])
def test_auth_user_from_env(
    requests_mock: requests_mock.Mocker, user: Optional[str], has_token_store: bool
) -> None:
    with use_test_config_dir() as config_dir, set_container_env_vars():
        if has_token_store:
            auth_json_path = config_dir / "auth.json"
            with open(auth_json_path, "w") as f:
                json.dump(AUTH_JSON, f)

        requests_mock.get("/users/me", status_code=200, json={"username": "alice"})

        authentication = Authentication(MOCK_MASTER_URL, user)
        if has_token_store:
            assert authentication.session.username == user or "determined"
            assert authentication.session.token == (
                "det.token" if user == "determined" else "bob.token"
            )
        else:
            assert authentication.session.username == "alice"
            assert authentication.session.token == "alice.token"


def test_auth_json_v0_upgrade() -> None:
    with use_test_config_dir() as config_dir:
        auth_json_path = config_dir / "auth.json"
        shutil.copy2(AUTH_V0_PATH, auth_json_path)
        ts = TokenStore(MOCK_MASTER_URL, auth_json_path)

        assert ts.get_active_user() == "determined"
        assert ts.get_token("determined") == "v2.public.this.is.a.test"

        ts.set_token("determined", "ai")

        ts2 = TokenStore(MOCK_MASTER_URL, auth_json_path)
        assert ts2.get_token("determined") == "ai"

        with auth_json_path.open() as fin:
            data = json.load(fin)
            assert data.get("version") == 1
            assert "masters" in data and list(data["masters"].keys()) == [MOCK_MASTER_URL]


@pytest.mark.parametrize(
    "master_url,should_match",
    [
        ("localhost", True),
        ("127.0.0.1", False),
        ("localhost:8080", True),
        ("localhost/", True),
        ("localhost//", True),
        ("localhost/det/test", True), # Somewhat surprising behaviour.
        ("localhost/det/test/", True), # Somewhat surprising behaviour.
        ("http://localhost", False),
        ("https://localhost:8080", False),
        ("http://localhost:8080", True),
    ],
)
def test_auth_url_normalization(master_url: str, should_match: bool) -> None:
    with use_test_config_dir() as config_dir:
        auth_json_path = config_dir / "auth.json"
        with open(auth_json_path, "w") as f:
            json.dump(AUTH_JSON, f)
        ts = TokenStore(master_url, auth_json_path)

        if should_match:
            assert ts.get_active_user() == "bob"
            assert ts.get_token("determined") == "det.token"
        else:
            assert ts.get_active_user() is None
            assert ts.get_token("determined") is None


@pytest.mark.parametrize(
    "merge_url,should_corrupt",
    [
        ("localhost", True),
        ("localhost:8080", True),
        ("example.com", False),
        ("https://localhost:8080", False),
    ],
)
def test_auth_url_conflict(merge_url: str, should_corrupt: bool) -> None:
    with use_test_config_dir() as config_dir:
        auth_json_path = config_dir / "auth.json"
        with open(auth_json_path, "w") as f:
            auth_json: Dict[str, Any] = copy.deepcopy(AUTH_JSON)
            auth_json["masters"][merge_url] = {
                "active_user": "joe",
                "tokens": {
                    "joe": "joe.token",
                },
            }
            json.dump(auth_json, f)
        if should_corrupt:
            with pytest.raises(CorruptTokenCacheException):
                ts = TokenStore(merge_url, auth_json_path)
        else:
            ts = TokenStore(merge_url, auth_json_path)
            assert ts.get_active_user() == "joe"
            assert ts.get_token("joe") == "joe.token"


def test_cert_v0_upgrade() -> None:
    with use_test_config_dir() as config_dir:
        cert_path = config_dir / "master.crt"
        shutil.copy2(UNTRUSTED_CERT_PATH, cert_path)
        with cert_path.open() as fin:
            cert_data = fin.read()

        cert = certs_default_load(MOCK_MASTER_URL)
        assert isinstance(cert.bundle, str)
        with open(cert.bundle) as fin:
            loaded_cert_data = fin.read()
        assert loaded_cert_data.endswith(cert_data)
        assert not cert_path.exists()

        v1_certs_path = config_dir / "certs.json"
        assert v1_certs_path.exists()

        # Load once again from v1.
        cert2 = certs_default_load(MOCK_MASTER_URL)
        assert isinstance(cert2.bundle, str)
        with open(cert2.bundle) as fin:
            loaded_cert_data = fin.read()
        assert loaded_cert_data.endswith(cert_data)
