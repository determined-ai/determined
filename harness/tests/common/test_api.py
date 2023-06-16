from typing import NamedTuple

from determined.common.api import request


def test_make_url():
    Case = NamedTuple("Case", [("base", str), ("path", str), ("expected", str)])
    host = "http://localhost:8080"
    path = "api/v1/experiments"
    cases = [
        Case("http://localhost:8080/", "", f"{host}/"),
        Case("http://localhost:8080", "", f"{host}"),
        Case("http://localhost:8080", "/api/v1/experiments", f"{host}/{path}"),
        Case("http://localhost:8080", "api/v1/experiments/", f"{host}/{path}/"),
        Case("http://localhost:8080/", "/api/v1/experiments", f"{host}/{path}"),
        Case("http://localhost:8080/", "api/v1/experiments/", f"{host}/{path}/"),
        Case("http://localhost:8080/proxied/", "", f"{host}/proxied/"),
        Case("http://localhost:8080/proxied", "", f"{host}/proxied"),
        Case("http://localhost:8080/proxied", "/api/v1/experiments", f"{host}/proxied/{path}"),
        Case("http://localhost:8080/proxied", "api/v1/experiments/", f"{host}/proxied/{path}/"),
        Case("http://localhost:8080/proxied/", "/api/v1/experiments", f"{host}/proxied/{path}"),
        Case("http://localhost:8080/proxied/", "api/v1/experiments/", f"{host}/proxied/{path}/"),
    ]

    for idx, case in enumerate(cases):
        base, path, expected = case
        assert request.make_url(base, path) == expected, f"{idx}  {case}"
