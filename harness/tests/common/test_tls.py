import contextlib
import os

import pytest
import requests

from determined.common.api import certs, request

TRUSTED_DOMAIN = "https://example.com"
UNTRUSTED_DOMAIN = "https://untrusted-root.badssl.com"
UNTRUSTED_CERT_FILE = os.path.join(os.path.dirname(__file__), "untrusted-root.badssl.com.crt")


def test_custom_tls_certs() -> None:
    with open(UNTRUSTED_CERT_FILE) as f:
        untrusted_pem = f.read()

    for kwargs, raises in [
        ({"noverify": True}, False),
        ({"noverify": False}, True),
        ({"cert_pem": untrusted_pem}, False),
        ({}, True),
    ]:
        assert isinstance(kwargs, dict)
        cert = certs.Cert(**kwargs)

        # Trusted domains should always work.
        request.get(TRUSTED_DOMAIN, "", authenticated=False, cert=cert)

        with contextlib.ExitStack() as ctx:
            if raises:
                ctx.enter_context(pytest.raises(requests.exceptions.SSLError))
            request.get(UNTRUSTED_DOMAIN, "", authenticated=False, cert=cert)
