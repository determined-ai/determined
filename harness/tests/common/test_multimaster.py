import contextlib
import json
import ssl
import threading
from http.server import HTTPServer, SimpleHTTPRequestHandler
from pathlib import Path
from typing import Any, Callable, Dict, Iterator, Optional, Tuple

import determined
from determined.common.api.authentication import salt_and_hash
from determined.common.experimental.determined import Determined
from tests.confdir import use_test_config_dir

CERTS_DIR = Path(__file__).parent / "multimaster-certs"
CERTS1 = {
    "keyfile": CERTS_DIR / "key1.pem",
    "certfile": CERTS_DIR / "cert1.pem",
}

CERTS2 = {
    "keyfile": CERTS_DIR / "key2.pem",
    "certfile": CERTS_DIR / "cert2.pem",
}


@contextlib.contextmanager
def run_api_server(
    address: Tuple[str, int] = ("localhost", 12345),
    credentials: Tuple[str, str, str] = ("user1", "password1", "token1"),
    ssl_keys: Dict[str, Path] = CERTS1,
) -> Iterator[str]:
    user, password, token = credentials

    class RequestHandler(SimpleHTTPRequestHandler):
        def _info(self) -> Dict[str, Any]:
            return {"cluster_id": "fake-cluster", "version": determined.__version__}

        def _users_me(self) -> Dict[str, Any]:
            return {"username": user}

        def _login(self) -> Dict[str, Any]:
            content_length = int(self.headers["Content-Length"])
            post_data = self.rfile.read(content_length)
            posted_credentials = json.loads(post_data)
            expected_password = salt_and_hash(password)
            assert posted_credentials.get("username") == user
            assert posted_credentials.get("password") == expected_password
            return {"token": token}

        def _api_v1_models(self) -> Dict[str, Any]:
            assert self.headers["Authorization"] == f"Bearer {token}"
            return {
                "models": [],
                "pagination": {
                    "offset": 0,
                    "limit": 100,
                    "startIndex": 0,
                    "endIndex": 0,
                    "total": 0,
                },
            }

        def do_core(self, fn: Optional[Callable[..., Dict[str, Any]]]) -> None:
            if fn is None:
                self.send_error(404, f"path not handled: {self.path}")
                return None

            result = fn()
            self._send_result(result)

        def do_GET(self) -> None:
            fn = {
                "/info": self._info,
                "/users/me": self._users_me,
                "/api/v1/models": self._api_v1_models,
            }.get(self.path.split("?")[0])
            self.do_core(fn)

        def do_POST(self) -> None:
            fn = {
                "/login": self._login,
            }.get(self.path)
            self.do_core(fn)

        def _send_result(self, result: Dict[str, Any]) -> None:
            response = json.dumps(result).encode("utf8")
            self.send_response(200)
            self.send_header("Content-type", "application/json")
            self.send_header("Content-Length", str(len(response)))
            self.end_headers()
            self.wfile.write(response)

    server = HTTPServer(address, RequestHandler)

    server.socket = ssl.wrap_socket(
        server.socket,
        keyfile=str(ssl_keys["keyfile"]),
        certfile=str(ssl_keys["certfile"]),
        server_side=True,
    )

    thread = threading.Thread(target=server.serve_forever, args=[0.1])
    thread.start()
    try:
        host = address[0]
        port = address[1]
        yield f"https://{host}:{port}/"
    finally:
        server.shutdown()
        thread.join()


def test_multimaster() -> None:
    with use_test_config_dir():
        conf1 = {
            "address": ("localhost", 12345),
            "credentials": ("user1", "password1", "token1"),
            "ssl_keys": CERTS1,
        }

        conf2 = {
            "address": ("localhost", 12346),
            "credentials": ("user2", "password2", "token2"),
            "ssl_keys": CERTS2,
        }

        with run_api_server(**conf1) as master_url1:  # type: ignore
            with run_api_server(**conf2) as master_url2:  # type: ignore
                d1 = Determined(
                    master_url1,
                    user="user1",
                    password="password1",
                    cert_path=str(CERTS1["certfile"]),
                )
                d2 = Determined(
                    master_url2,
                    user="user2",
                    password="password2",
                    cert_path=str(CERTS2["certfile"]),
                )
                d1.get_models()
                d2.get_models()
