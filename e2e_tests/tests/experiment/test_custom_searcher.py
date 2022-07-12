import logging
import uuid
from random import random
from typing import Tuple, List, Optional

import pytest

from determined.common.api import bindings
from determined.common.experimental import ExperimentReference
from determined.experimental import client
from determined.searcher.search_method import SearchMethod, SearchState, Operation, Create
from determined.searcher.search_runner import SearchRunner
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
    const_hp = bindings.v1ConstantHyperparameter(val=0.2)
    lr = bindings.v1Hyperparameter(constantHyperparam=const_hp)
    # need to add more variations above according to what's in the HyperParameterVO struct,
    # will do that after i figure out some of the types.
    hyperparams = {"optimizer": lr}
    create_trial_op = bindings.v1CreateTrialOperation(hyperparams=hyperparams)
    op1 = bindings.v1SearcherOperation(createTrial=create_trial_op)
    init_op = bindings.v1InitialOperations(id=0)
    init_event = bindings.v1SearcherEvent(initialOperations=init_op)
    body = bindings.v1PostSearcherOperationsRequest(
        experimentId=exp_id, searcherOperations=[op1], triggeredByEvent=init_event
    )
    bindings.post_PostSearcherOperations(sess, experimentId=exp_id, body=body)


@pytest.mark.e2e_cpu
def test_run_custom_searcher_experiment() -> None:
    # example searcher script
    config = conf.load_config(conf.fixtures_path("no_op/single.yaml"))
    config["searcher"] = {
        "name": "custom",
        "metric": "validation_error",
        "smaller_is_better": True,
    }
    experiment: ExperimentReference = client.create_experiment(config, conf.fixtures_path("no_op"))

    search_method = SingleSearchMethod(config)
    search_runner = SearchRunner(search_method)
    search_runner.run()


class SingleSearchMethod(SearchMethod):
    def __init__(self, experiment_config: dict) -> None:
        super(self).__init__(SearchState(None))
        self.hparams = experiment_config["hyperparameters"]

    def initial_operations(self) -> Tuple[List[Operation], Optional[str]]:
        logging.info("initial_operations")
        create = Create(
            request_id=uuid.uuid4(),
            trial_seed=random.randint(0, 2**31),
            hparams=self.hparams,
        )
        return [create]
