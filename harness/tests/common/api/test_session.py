import pytest

from determined.common import api
from determined.common.api import certs
from tests.common import api_server


class TestSession:
    @staticmethod
    @pytest.fixture(scope="class")
    def test_master() -> str:
        with api_server.run_api_server(address=("localhost", 8080)) as master_url:
            yield master_url

    @staticmethod
    @pytest.fixture(scope="function")
    def session(test_master: str) -> api.Session:
        yield api.Session(
            master=test_master, username="me", token="t1o2k3e4n5", cert=certs.Cert(noverify=True)
        )

    def test_direct_instantiation_doesnt_reuse_requests_sessions(
        self, session: api.Session
    ) -> None:
        # Make a few requests, and expect the request's session to be closed immediately after
        # each request.
        # To requests and underlying urllib3, this means the PoolManager should not contain any
        # ConnectionPools.
        for _ in range(3):
            resp = session.get(path="/info")
            assert resp.connection  # for mypy
            assert len(resp.connection.poolmanager.pools) == 0

    def test_context_manager_reuses_requests_sessions(self, session: api.Session) -> None:
        connection_pools = []
        # Make a few requests across the same path and a different path and verify that the
        # requests are made with the same requests session.
        with session as sess:
            for path in ["/info", "/info", "/users/me"]:
                resp = sess.get(path=path)

                # urllib3 creates HTTP connections from the PoolManager's connection pool, so we
                # assume that if there was one connection pool, it was the one the request used.
                assert len(resp.connection.poolmanager.pools) == 1

                pool_key = resp.connection.poolmanager.pools.keys()[0]
                connection_pool = resp.connection.poolmanager.pools.get(pool_key)
                connection_pools.append(connection_pool)
        assert len(set(connection_pools)) == 1
