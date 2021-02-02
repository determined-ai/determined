import contextlib
import os

import pytest
import requests

from determined_common.api import request

TRUSTED_DOMAIN = "https://example.com"
UNTRUSTED_DOMAIN = "https://untrusted-root.badssl.com"
UNTRUSTED_CERT_FILE = os.path.join(os.path.dirname(__file__), "untrusted-root.badssl.com.crt")


def test_custom_tls_certs() -> None:
    for bundle, raises in [
        (False, False),
        (True, True),
        (UNTRUSTED_CERT_FILE, False),
        (None, True),
    ]:
        request.set_master_cert_bundle(bundle)  # type: ignore

        request.get(TRUSTED_DOMAIN, "", authenticated=False)
        with contextlib.ExitStack() as ctx:
            if raises:
                ctx.enter_context(pytest.raises(requests.exceptions.SSLError))
            request.get(UNTRUSTED_DOMAIN, "", authenticated=False)
