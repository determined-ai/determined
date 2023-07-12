from typing import List, NamedTuple

import pytest

from determined.common.api import request

Case = NamedTuple("Case", [("base", str), ("path", str), ("expected", str)])


def gen_make_url_new_cases() -> List[Case]:
    host = "http://localhost:8080"
    path = "api/v1/experiments"
    cases: List[Case] = [
        Case("http://localhost:8080/", "", f"{host}/"),
        Case("http://localhost:8080", "", f"{host}"),
        Case("http://localhost:8080", "/api/v1/experiments", f"{host}/{path}"),
        Case("http://localhost:8080", "api/v1/experiments/", f"{host}/{path}/"),
        Case("http://localhost:8080/", "/api/v1/experiments", f"{host}/{path}"),
        Case("http://localhost:8080/", "api/v1/experiments/", f"{host}/{path}/"),
        Case("http://localhost:8080/proxied/", "", f"{host}/proxied/"),
        Case("http://localhost:8080/proxied", "", f"{host}/proxied"),
        Case("http://localhost:8080/proxied", f"/{path}", f"{host}/proxied/{path}"),
        Case("http://localhost:8080/proxied", f"{path}/", f"{host}/proxied/{path}/"),
        Case("http://localhost:8080/proxied/", f"/{path}", f"{host}/proxied/{path}"),
        Case("http://localhost:8080/proxied/", f"{path}/", f"{host}/proxied/{path}/"),
        Case(
            "http://localhost:8080", f"{host}/{path}/", f"{host}/{path}/"
        ),  # invalid path. unexpected case supported through (deprecated) make_url.
    ]
    return cases


@pytest.mark.parametrize("base, path, expected", gen_make_url_new_cases())
def test_make_url_new(base: str, path: str, expected: str) -> None:
    actual = request.make_url_new(base, path)
    assert actual == expected, f"base: {base}, path: {path}"


def gen_make_url_cases() -> List[Case]:
    host = "http://localhost:8080"
    path = "api/v1/experiments"
    cases: List[Case] = [
        Case("http://localhost:8080/", "", f"{host}/"),
        Case("http://localhost:8080", "", f"{host}"),
        Case("http://localhost:8080", "/api/v1/experiments", f"{host}/{path}"),
        Case("http://localhost:8080", "api/v1/experiments/", f"{host}/{path}/"),
        Case("http://localhost:8080/", "/api/v1/experiments", f"{host}/{path}"),
        Case("http://localhost:8080/", "api/v1/experiments/", f"{host}/{path}/"),
        Case("http://localhost:8080/proxied/", "", f"{host}/proxied/"),
        Case("http://localhost:8080/proxied", "", f"{host}/proxied"),
        Case("http://localhost:8080/proxied/", f"{path}/", f"{host}/proxied/{path}/"),
        Case(
            "http://localhost:8080", f"{host}/{path}/", f"{host}/{path}/"
        ),  # invalid path. unexpected case.
        # unsupported cases
        # Case("http://localhost:8080/proxied", f"/{path}", f"{host}/proxied/{path}"),
        # Case("http://localhost:8080/proxied", f"{path}/", f"{host}/proxied/{path}/"),
        # Case("http://localhost:8080/proxied/", f"/{path}", f"{host}/proxied/{path}"),
    ]
    return cases


@pytest.mark.parametrize("base, path, expected", gen_make_url_cases())
def test_make_url(base: str, path: str, expected: str) -> None:
    actual = request.make_url(base, path)
    assert actual == expected, f"base: {base}, path: {path}"
