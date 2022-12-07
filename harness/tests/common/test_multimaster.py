from determined.common.experimental.determined import Determined
from tests.common import api_server
from tests.confdir import use_test_config_dir


def test_multimaster() -> None:
    with use_test_config_dir():
        conf1 = {
            "address": ("localhost", 12345),
            "credentials": ("user1", "password1", "token1"),
            "ssl_keys": api_server.CERTS1,
        }

        conf2 = {
            "address": ("localhost", 12346),
            "credentials": ("user2", "password2", "token2"),
            "ssl_keys": api_server.CERTS2,
        }

        with api_server.run_api_server(**conf1) as master_url1:  # type: ignore
            with api_server.run_api_server(**conf2) as master_url2:  # type: ignore
                d1 = Determined(
                    master_url1,
                    user="user1",
                    password="password1",
                    cert_path=str(api_server.CERTS1["certfile"]),
                )
                d2 = Determined(
                    master_url2,
                    user="user2",
                    password="password2",
                    cert_path=str(api_server.CERTS2["certfile"]),
                )
                d1.get_models()
                d2.get_models()
