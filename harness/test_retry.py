# test requests with retry

import pytest
import requests.adapters
import requests.sessions
import requests
import requests_mock as mock
import urllib3

retry = urllib3.util.retry.Retry(
    total=5,
    backoff_factor=0.1,
    status_forcelist=[502, 503, 504],
)
adapter = requests.adapters.HTTPAdapter(max_retries=retry)


def test_direct_retry(requests_mock: mock.Mocker):
    url = "http://localhost:8080/test"
    requests_mock.get(url, status_code=503)
    # with mock.Mocker(adapter=mock.Adapter(max_retries=retry)) as m:
    # mock_adapter = mock.get_adapter(url)
    # mock_adapter.max_retries = retry

    with requests.Session() as session:
        session.mount("http://", adapter)
        session.mount("https://", adapter)

        with pytest.raises(urllib3.util.retry.MaxRetryError) as e:
            session.get(url)
        assert requests_mock.call_count > 1


def test_retry(requests_mock):
    requests_mock.get("http://localhost:8080/test", status_code=503)
    session = requests.Session()

    adapter = requests.adapters.HTTPAdapter(max_retries=retry)
    session.mount("http://", adapter)
    session.mount("https://", adapter)
    assert requests_mock.call_count == 0
    response = session.get("http://localhost:8080/test", verify=False)
    assert requests_mock.call_count == 5
    assert response.status_code == 200
