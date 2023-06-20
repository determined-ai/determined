import contextlib
import json
import ssl
import threading
from http.server import HTTPServer, SimpleHTTPRequestHandler
from pathlib import Path
from typing import Any, Callable, Dict, Iterator, Optional, Tuple

import determined
from determined.common.api import bindings
from determined.common.api.authentication import salt_and_hash

CERTS_DIR = Path(__file__).parent / "multimaster-certs"
CERTS1 = {
    "keyfile": CERTS_DIR / "key1.pem",
    "certfile": CERTS_DIR / "cert1.pem",
}
CERTS2 = {
    "keyfile": CERTS_DIR / "key2.pem",
    "certfile": CERTS_DIR / "cert2.pem",
}

DEFAULT_HOST = "localhost"
DEFAULT_PORT = 12345
DEFAULT_USER = "user1"
DEFAULT_PASSWORD = "password1"
DEFAULT_TOKEN = "token1"

FIXTURES_DIR = Path(__file__).parent.parent / "fixtures"


def sample_get_experiment(**kwargs: Any) -> bindings.v1GetExperimentResponse:
    """Get an experiment from a fixture and optionally override some fields.

    Load a sample experiment from a fixture.  It's assumed that generally a caller cares only that
    the response is well-formed. If instead the caller cares about any particular fields, they can
    override them by passing them as keyword arguments.

    Args:
        **kwargs: Fields to override in the experiment.

    Returns:
        A bindings.v1GetExperimentResponse object with the experiment. NOTE: The returned object
        is a bindings type, *not* a ExperimentReference.
    """
    with open(FIXTURES_DIR / "experiment.json") as f:
        resp = bindings.v1GetExperimentResponse.from_json(json.load(f))
        for k, v in kwargs.items():
            setattr(resp.experiment, k, v)
        return resp


@contextlib.contextmanager
def run_api_server(
    address: Tuple[str, int] = (DEFAULT_HOST, DEFAULT_PORT),
    credentials: Tuple[str, str, str] = (DEFAULT_USER, DEFAULT_PASSWORD, DEFAULT_TOKEN),
    ssl_keys: Optional[Dict[str, Path]] = CERTS1,
) -> Iterator[str]:
    user, password, token = credentials
    lock = threading.RLock()
    state: Dict[str, Any] = {}

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
            fake_user = {"username": user, "admin": True, "active": True}
            return {"token": token, "user": fake_user}

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

        def get_experiment_longrunning(self) -> Dict[str, Any]:
            """A master response to get_GetExperiment for a long-running experiment.

            This function models an experiment that may take a long time to complete. The first
            two times get_experiment is called, the experiment state is still in a
            bindings.experimentv1State.RUNNING. On the third call, its state is
            bindings.experimentv1State.COMPLETED.

            Returns:
                If successful, a JSON-encoded sample experiment. Else None.
            """
            key = "get_experiment_longrunning_n_calls"
            n_calls = 2
            with open(FIXTURES_DIR / "experiment.json") as f:
                sample_experiment = bindings.v1GetExperimentResponse.from_json(json.load(f))

            with lock:
                state[key] = state.get(key, 0) + 1
                if state[key] <= n_calls:
                    sample_experiment.experiment.state = bindings.experimentv1State.RUNNING
                else:
                    sample_experiment.experiment.state = bindings.experimentv1State.COMPLETED
            return sample_experiment.to_json()

        def get_experiment_flaky(self) -> Dict[str, Any]:
            """A master response to get_GetExperiment for a long-running experiment.

            This function models an experiment where master sometimes cannot be reached. The first
            two times get_experiment is called, the call to the master returns a 504 HTTP code.
            The third call is successful.

            Returns:
                If successful, a JSON-encoded sample experiment. Else None.
            """
            key = "get_experiment_flaky_n_calls"
            fail_for = 2
            with lock:
                state[key] = state.get(key, 0) + 1
                if state[key] <= fail_for:
                    self.send_error(504)
                    return {}
            with open(FIXTURES_DIR / "experiment.json") as f:
                sample_experiment = bindings.v1GetExperimentResponse.from_json(json.load(f))
            return sample_experiment.to_json()

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
                "/api/v1/experiments/1": self.get_experiment_flaky,
                "/api/v1/experiments/2": self.get_experiment_longrunning,
            }.get(self.path.split("?")[0])
            self.do_core(fn)

        def do_POST(self) -> None:
            fn = {
                "/api/v1/auth/login": self._login,
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

    if ssl_keys is not None:
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
        protocol = "https" if ssl_keys is not None else "http"
        yield f"{protocol}://{host}:{port}"
    finally:
        server.shutdown()
        thread.join()
