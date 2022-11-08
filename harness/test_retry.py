# test requests with retry

import pytest
import requests.adapters
import requests.sessions
import urllib3


def test_retry(requests_mock):
    requests_mock.get("http://localhost:8080/test", status_code=503)

    session = requests.Session()
    retry = urllib3.util.retry.Retry(
        total=5,
        # backoff_factor=0.1,
        status_forcelist=[502, 503, 504],
    )

    adapter = requests.adapters.HTTPAdapter(max_retries=retry)
    session.mount("http://", adapter)
    session.mount("https://", adapter)
    assert requests_mock.call_count == 0
    response = session.get("http://localhost:8080/test", verify=False)
    assert requests_mock.call_count == 5
    assert response.status_code == 200
