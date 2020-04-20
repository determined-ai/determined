import tempfile
from typing import Type

import pytest

import determined as det
from determined import experimental
from determined_common import check
from tests.unit.frameworks.fixtures import (
    estimator_xor_model,
    pytorch_xor_model,
    tf_keras_xor_model,
)


@pytest.mark.parametrize(  # type: ignore
    "trial_class",
    [estimator_xor_model.XORTrial, pytorch_xor_model.XORTrial, tf_keras_xor_model.XORTrial],
)
def test_create_trial_instance(trial_class: Type[det.Trial]) -> None:
    with tempfile.TemporaryDirectory() as td:
        trial_instance = experimental.create_trial_instance(
            trial_def=trial_class,
            config={"hyperparameters": {"global_batch_size": det.Constant(16)}},
            checkpoint_dir=td,
        )

        check.check_isinstance(trial_instance, det.Trial)
