import json
import pathlib
import shutil

from determined.common.api import authentication
from tests import confdir

MOCK_MASTER_URL = "http://localhost:8080"
AUTH_V0_PATH = pathlib.Path(__file__).parent / "auth_v0.json"


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
