import logging
import uuid
from typing import List

import pytest

from determined.searcher.hyperparameters import (
    CategoricalHparamValue,
    DoubleHparamValue,
    HparamSample,
    HparamValue,
    IntHparamValue,
)
from determined.searcher.search_method import (
    Close,
    Create,
    ExitedReason,
    Operation,
    SearchMethod,
    SearchState,
    Shutdown,
    ValidateAfter,
)
from determined.searcher.search_runner import SearchRunner
from tests import config as conf


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

    def on_trial_created(self, trial_id: uuid.UUID) -> List[Operation]:
        return []

    def on_validation_completed(self, metric: float) -> List[Operation]:
        return []

    def on_trial_closed(self, trial_id: uuid.UUID) -> List[Operation]:
        return []

    def progress(self) -> float:
        return 0.99  # TODO change signature

    def on_trial_exited_early(
        self, trial_id: uuid.UUID, exit_reason: ExitedReason
    ) -> List[Operation]:
        logging.warning(f"Trial {trial_id} exited early: {exit_reason}")
        return [Shutdown()]

    def initial_operations(self) -> List[Operation]:
        logging.info("initial_operations")
        values: List[HparamValue] = []
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
