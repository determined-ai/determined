import requests

from tests.common.api_server import run_api_server


def test_flaky_endpoint() -> None:
    def test(*args, **kwargs):
        with run_api_server(*args, **kwargs) as master_url:
            with requests.Session() as session:
                for _ in range(2):  # end point FAILS_FOR 2 times
                    response = session.get(master_url + "/api/v1/experiments/1", verify=False)
                    assert response.status_code == 504
                response = session.get(master_url + "/api/v1/experiments/1", verify=False)
                assert response.status_code == 200

    for _ in range(2):  # no state is shared between runs
        test()

    test(ssl_keys=None)
