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
    const_hp = bindings.v1ConstantHyperparameter(val = 0.2)
    lr = bindings.v1Hyperparameter(constantHyperparam = const_hp)
    # need to add more variations above according to what's in the HyperParameterVO struct,
    # will do that after i figure out some of the types.
    hyperparams = {"optimizer": lr}
    create_trial_op = bindings.v1CreateTrialOperation(hyperparams=hyperparams)
    op1 = bindings.v1SearcherOperation(createTrial=create_trial_op)
    body = bindings.v1PostSearcherOperationsRequest(
        experimentId=exp_id, searcherOperations=[op1], triggeredByEvent="Initial_Operations_0"
    )
    bindings.post_PostSearcherOperations(sess, experimentId=exp_id, body=body)
