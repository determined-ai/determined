import pytest

from determined.common.api import bindings
from tests import config as conf
from tests import experiment as exp


@pytest.mark.e2e_cpu
def test_get_searcher_ops() -> None:
    exp_id = exp.run_basic_test(
        conf.fixtures_path("no_op/single.yaml"), conf.fixtures_path("no_op"), 1
    )
    sess = exp.determined_test_session()
    bindings.get_GetSearcherEvents(sess, experimentId=exp_id)


@pytest.mark.e2e_cpu
def test_post_searcher_ops() -> None:
    exp_id = exp.run_basic_test(
        conf.fixtures_path("no_op/single.yaml"), conf.fixtures_path("no_op"), 1
    )
    sess = exp.determined_test_session()
    op1 = bindings.v1SearcherOperation(
        createTrial="createTrial"
    )  # what do we need to include for an operation such as createTrial?
    body = bindings.v1PostSearcherOperationsRequest(
        experimentId=exp_id, searchOperations=[op1], triggeredByEvent="Initial_Operations_0"
    )
    bindings.post_PostSearcherOperations(sess, experimentId=exp_id, body=body)
