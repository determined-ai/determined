import json

from determined.common.api import authentication
from tests import confdir

MOCK_MASTER_URL = "http://localhost:8080"


def test_auth_json_v0_upgrade() -> None:
    with confdir.use_test_config_dir() as config_dir:
        auth_json_path = config_dir / "auth.json"
        v0 = {
            "active_user": "determined",
            "tokens": {
                "determined": "v2.public.this.is.a.test",
            },
        }
        auth_json_path.write_text(json.dumps(v0))

        ts = authentication.TokenStore(MOCK_MASTER_URL, auth_json_path)

        assert ts.get_active_user() == "determined"
        assert ts.get_token("determined") == "v2.public.this.is.a.test"

        ts.set_token("determined", "ai")

        ts2 = authentication.TokenStore(MOCK_MASTER_URL, auth_json_path)
        assert ts2.get_token("determined") == "ai"

        with auth_json_path.open() as fin:
            data = json.load(fin)
            assert data.get("version") == 2
            assert "masters" in data and list(data["masters"].keys()) == [MOCK_MASTER_URL]


def test_auth_json_v1_upgrade() -> None:
    with confdir.use_test_config_dir() as config_dir:
        auth_json_path = config_dir / "auth.json"
        v1 = {
            "version": 1,
            "masters": {
                # First merge: zero active users, no token overlap.
                "http://firstmaster": {
                    "tokens": {
                        "a": "a.token",
                    },
                },
                "http://firstmaster:80#fragment": {
                    "tokens": {
                        "b": "b.token",
                    },
                },
                # Second conflict: one active user, partial token overlap.
                "https://secondmaster": {
                    "active_user": "a",
                    "tokens": {
                        "a": "a.token",
                        "b": "b.token1",
                    },
                },
                "https://secondmaster/?key=value": {
                    "tokens": {
                        "b": "b.token2",
                        "c": "c.token",
                    },
                },
                # Third conflict: two active users, full token overlap.
                "thirdmaster": {
                    "active_user": "a",
                    "tokens": {
                        "a": "a.token1",
                        "b": "b.token1",
                    },
                },
                "http://user@thirdmaster:8080": {
                    "active_user": "b",
                    "tokens": {
                        "a": "a.token2",
                        "b": "b.token2",
                    },
                },
                # Special case: force a ValueError to make sure the shim_store_v1 discards totally
                # broken URLs (without crashing the CLI).
                #
                # This works because urlparse will reject the \u2100 character with a ValueError,
                # which exercises the right codepath.  Something about "NFKC normalization".
                #
                # This is a little phony because the ValueErrors we are worried about would be cases
                # where precanonicalize_v1_url isn't guarding against everything that
                # canonicalize_master_url would reject.  But since I don't know of any such things,
                # it's hard to exercise that codepath any other way.
                "http://\u2100": {
                    "active_user": "a",
                    "tokens": {
                        "a": "a.token",
                    },
                },
            },
        }
        auth_json_path.write_text(json.dumps(v1))

        ts1 = authentication.TokenStore("http://firstmaster:80", auth_json_path)
        assert ts1.get_active_user() is None
        assert set(ts1.get_all_users()) == {"a", "b"}
        assert ts1.get_token("a") == "a.token"
        assert ts1.get_token("b") == "b.token"

        ts2 = authentication.TokenStore("https://secondmaster:443", auth_json_path)
        assert ts2.get_active_user() == "a"
        assert set(ts2.get_all_users()) == {"a", "b", "c"}
        assert ts2.get_token("a") == "a.token"
        assert ts2.get_token("b") in ("b.token1", "b.token2")
        assert ts2.get_token("c") == "c.token"

        ts3 = authentication.TokenStore("http://thirdmaster:8080", auth_json_path)
        assert ts3.get_active_user() in ("a", "b")
        assert set(ts3.get_all_users()) == {"a", "b"}
        assert ts3.get_token("a") in ("a.token1", "a.token2")
        assert ts3.get_token("b") in ("b.token1", "b.token2")

        # Make sure the file is updated when we write to the TokenStore.
        ts3.set_active("a")
        obj = json.loads(auth_json_path.read_text())
        assert obj["version"] == 2
        # Make sure we got exactly the master urls we expected.
        exp = {"http://firstmaster:80", "https://secondmaster:443", "http://thirdmaster:8080"}
        assert set(obj["masters"]) == exp, list(obj["masters"])
