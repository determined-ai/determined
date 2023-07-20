import pytest
import responses

from determined.common import api
from determined.common.api import bindings
from determined.common.experimental import resource_pool
from tests.fixtures import api_responses

_MASTER = "http://localhost:8080"


@pytest.fixture
def standard_session() -> api.Session:
    return api.Session(master=_MASTER, user=None, auth=None, cert=None)


@pytest.fixture
def single_item_resource_pools() -> bindings.v1GetResourcePoolsResponse:
    sample_resource_pools = api_responses.sample_get_resource_pool().resourcePools
    single_item_pagination = bindings.v1Pagination(endIndex=1, startIndex=0, total=1)
    return bindings.v1GetResourcePoolsResponse(
        resourcePools=sample_resource_pools, pagination=single_item_pagination
    )


@responses.activate
def test_bind_rp_to_workspaces_errors_on_invalid_rps(
    standard_session: api.Session,
) -> None:
    resource_pool_name = "invalid_rp"
    responses.post(
        f"{_MASTER}/api/v1/resource-pools/{resource_pool_name}/workspace-bindings",
        status=500,
    )
    rp = resource_pool.ResourcePool(session=standard_session, name="invalid_rp")

    try:
        rp.bind("foo")
        raise AssertionError("Server's 500 should raise an exception")
    except api.errors.APIException:
        pass
