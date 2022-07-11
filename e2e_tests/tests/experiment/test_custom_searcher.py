import logging
import uuid
import random
from typing import Tuple, List, Optional

import pytest

from determined.common.api import bindings
from determined.common.experimental import ExperimentReference
from determined.experimental import client
from determined.searcher.hyperparameters import HparamSample, IntHparamValue, DoubleHparamValue, CategoricalHparamValue
from determined.searcher.search_method import SearchMethod, SearchState, Operation, Create, Shutdown, ValidateAfter, \
    Close, ExitedReason
from determined.searcher.search_runner import SearchRunner
from tests import config as conf
from tests import experiment as exp


@pytest.mark.e2e_cpu
def test_get_searcher_ops() -> None:
    exp_id = exp.run_basic_test(
        conf.fixtures_path("no_op/single.yaml"), conf.fixtures_path("no_op"), 1
    )
    sess = exp.determined_test_session()
    resp = bindings.get_GetSearcherEvents(sess, experimentId=exp_id)
    assert resp.searcherEvents is not None  # want to ensure it's not None for now.


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
    init_op = bindings.v1InitialOperations(holder="1")
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
        "max_length": {"batches": 3000},
    }
    config["description"] = "custom searcher"
    search_method = SingleSearchMethod(config)
    search_runner = SearchRunner(search_method)
    search_runner.run(config, context_dir=conf.fixtures_path("no_op"))


class SingleSearchMethod(SearchMethod):
    def __init__(self, experiment_config: dict) -> None:
        super().__init__(SearchState(None))
        self.hyperparameters = experiment_config["hyperparameters"]

    def on_trial_created(self, trial_id: int) -> List[Operation]:
        return []

    def on_validation_completed(self, metric: float) -> List[Operation]:
        return []

    def on_trial_closed(self, trial_id: int) -> List[Operation]:
        return []

    def progress(self) -> float:
        return 0.99  # TODO change signature

    def on_trial_exited_early(self, trial_id: int, exit_reason: ExitedReason) -> List[Operation]:
        logging.warning(f"Trial {trial_id} exited early: {exit_reason}")
        return [Shutdown()]

    def initial_operations(self) -> List[Operation]:
        logging.info("initial_operations")
        values = []
        for name, value in self.hyperparameters.items():
            if isinstance(value, int):
                values.append(IntHparamValue(name, value))
            elif isinstance(value, float):
                values.append(DoubleHparamValue(name, value))
            elif isinstance(value, str):
                values.append(CategoricalHparamValue(name, value))
            else:
                raise RuntimeError(f"unsupported hparam value type: {value}")

        hparams = HparamSample(values=values)
        create = Create(
            trial_id=uuid.uuid4(),
            hparams=hparams._to_hyperparameters(),
            checkpoint=None,
        )
        validate_after = ValidateAfter(trial_id=create.trial_id, length=3000)
        close = Close(trial_id=create.trial_id)
        logging.debug("".join(f"{k}:{v.to_json()}" for k, v in create.hparams.items()))
        return [create, validate_after, close]
