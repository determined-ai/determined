from determined.common import api
from determined.common.api import certs
from tests.common import api_server


def test_session_non_persistent_http_sessions():
    with api_server.run_api_server(address=("localhost", 8080)) as master_url:
        session = api.Session(
            master=master_url, username="me", token="t1o2k3e4n5", cert=certs.Cert(noverify=True)
        )

        # No persistent session was created, and we shouldn't have created any HTTP sessions.
        assert session._http_session is None

        resp_connections = []
        num_requests = 5

        # Make a few requests using the same session object.
        for i in range(num_requests):
            resp = session.get(
                path="/info",
                params=None,
                headers=None,
                timeout=None,
                stream=False,
            )

            # The underlying HTTP connection should be closed immediately after request.
            # To requests and underlying urllib3, this means there are no pools in the poolmanager
            assert len(resp.connection.poolmanager.pools) == 0
            resp_connections.append(resp.connection)

        # Expect each request to have been made with a different underlying HTTP connection.
        assert len(set(resp_connections)) == len(resp_connections)


def test_session_persistent_http_sessions():
    with api_server.run_api_server(address=("localhost", 8080)) as master_url:
        session = api.Session(
            master=master_url, username="me", token="t1o2k3e4n5", cert=certs.Cert(noverify=True)
        )

        resp_connections = []

        # Within a session context, make a few requests.
        with session as persistent_session:
            # Expect a persistent HTTP session to be set.
            assert persistent_session._http_session

            num_requests = 5
            for i in range(num_requests):
                resp = persistent_session.get(
                    path="/info",
                    params=None,
                    headers=None,
                    timeout=None,
                    stream=False,
                )

                # There should always be one HTTP connection pool open.
                assert len(resp.connection.poolmanager.pools) == 1

                resp_connections.append(resp.connection)

        # Expect each request to have been made with the same underlying HTTP connection.
        assert len(set(resp_connections)) == 1
