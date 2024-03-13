import pytest

from determined.common import api


@pytest.mark.parametrize(
    "url,exp",
    [
        # Bare hostname[:port] is converted to http [and port 8080].
        ("host", "http://host:8080"),
        ("host:99", "http://host:99"),
        # Scheme-specific defaults are applied.
        ("http://host", "http://host:80"),
        ("https://host", "https://host:443"),
        # Path is allowed, but trailing "/" is removed.
        ("http://host/", "http://host:80"),
        ("http://host:99/", "http://host:99"),
        ("http://host:99/path", "http://host:99/path"),
        ("http://host:99/path/", "http://host:99/path"),
        # I guess if user and password are empty, we should remove the "@" and accept it.
        ("http://@host:8080", "http://host:8080"),
    ],
)
def test_canonicalize_master_url_valid(url: str, exp: str) -> None:
    got = api.canonicalize_master_url(url)
    assert got == exp


@pytest.mark.parametrize(
    "url,exp",
    [
        # Reject user, password, query, or fragment.
        ("http://user@host:8080", "must not contain username, password, query, or fragment"),
        ("http://user:pass@host:8080", "must not contain username, password, query, or fragment"),
        ("http://host?query=bad", "must not contain username, password, query, or fragment"),
        ("http://host#fragment", "must not contain username, password, query, or fragment"),
        # Reject empty hostname.
        ("http://:8080", "must contain a nonempty hostname"),
        ("http://", "must contain a nonempty hostname"),
    ],
)
def test_canonicalize_master_url_invalid(url: str, exp: str) -> None:
    with pytest.raises(ValueError, match=exp):
        api.canonicalize_master_url(url)
