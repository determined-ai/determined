import collections
import json
import logging
import os
import re
import subprocess
import typing

import pytest

from tests import config as conf
from tests.experiment import create_native_experiment, experiment, native_experiment_submit

NativeImplementation = collections.namedtuple(
    "NativeImplementation",
    ["cwd", "command", "configuration", "num_expected_steps_per_trial", "num_expected_trials"],
)


class NativeImplementations:
    PytorchMNISTCNNSingleGeneric = NativeImplementation(
        cwd=conf.official_examples_path("native/native_mnist_pytorch"),
        command=[
            "python",
            conf.official_examples_path("native/native_mnist_pytorch/trial_impl.py"),
        ],
        configuration={
            "checkpoint_storage": experiment.shared_fs_checkpoint_config(),
            "searcher": {
                "name": "single",
                "max_length": {"batches": 100},
                "metric": "validation_loss",
            },
            "max_restarts": 0,
        },
        num_expected_steps_per_trial=1,
        num_expected_trials=1,
    )
    TFEstimatorMNISTCNNSingle = NativeImplementation(
        cwd=conf.official_examples_path("native/native_mnist_estimator"),
        command=[
            "python",
            conf.official_examples_path("native/native_mnist_estimator/native_impl.py"),
        ],
        configuration={
            "scheduling_unit": 4,
            "checkpoint_storage": experiment.shared_fs_checkpoint_config(),
            "searcher": {"name": "single", "max_length": {"batches": 4}, "metric": "accuracy"},
            "max_restarts": 0,
        },
        num_expected_steps_per_trial=1,
        num_expected_trials=1,
    )

    TFEstimatorMNISTCNNSingleGeneric = NativeImplementation(
        cwd=conf.official_examples_path("native/native_mnist_estimator"),
        command=[
            "python",
            conf.official_examples_path("native/native_mnist_estimator/trial_impl.py"),
        ],
        configuration={
            "scheduling_unit": 4,
            "checkpoint_storage": experiment.shared_fs_checkpoint_config(),
            "searcher": {"name": "single", "max_length": {"batches": 4}, "metric": "accuracy"},
            "max_restarts": 0,
        },
        num_expected_steps_per_trial=1,
        num_expected_trials=1,
    )

    # Train a single tf.keras model using fit().
    TFKerasMNISTCNNSingleFit = NativeImplementation(
        cwd=conf.official_examples_path("native/native_fashion_mnist_tf_keras"),
        command=[
            "python",
            conf.official_examples_path("native/native_fashion_mnist_tf_keras/native_impl.py"),
            "--use-fit",
        ],
        configuration={
            "scheduling_unit": 4,
            "checkpoint_storage": experiment.shared_fs_checkpoint_config(),
            "searcher": {"name": "single", "max_length": {"batches": 4}, "metric": "val_accuracy"},
            "max_restarts": 2,
        },
        num_expected_steps_per_trial=1,
        num_expected_trials=1,
    )

    # Train a single tf.keras model using fit_generator().
    TFKerasMNISTCNNSingleFitGenerator = NativeImplementation(
        cwd=conf.official_examples_path("native/native_fashion_mnist_tf_keras"),
        command=[
            "python",
            conf.official_examples_path("native/native_fashion_mnist_tf_keras/native_impl.py"),
        ],
        configuration={
            "scheduling_unit": 4,
            "checkpoint_storage": experiment.shared_fs_checkpoint_config(),
            "searcher": {"name": "single", "max_length": {"batches": 4}, "metric": "val_accuracy"},
            "max_restarts": 2,
        },
        num_expected_steps_per_trial=1,
        num_expected_trials=1,
    )

    TFKerasMNISTCNNSingleGeneric = NativeImplementation(
        cwd=conf.official_examples_path("native/native_fashion_mnist_tf_keras"),
        command=[
            "python",
            conf.official_examples_path("native/native_fashion_mnist_tf_keras/trial_impl.py"),
        ],
        configuration={
            "scheduling_unit": 4,
            "checkpoint_storage": experiment.shared_fs_checkpoint_config(),
            "searcher": {"name": "single", "max_length": {"batches": 4}, "metric": "val_accuracy"},
            "max_restarts": 2,
        },
        num_expected_steps_per_trial=1,
        num_expected_trials=1,
    )


def maybe_create_experiment(implementation: NativeImplementation) -> typing.Optional[int]:
    logging.debug(implementation)

    target_env = os.environ.copy()
    target_env["DET_MASTER"] = conf.make_master_url()

    with subprocess.Popen(
        implementation.command + ["--config", json.dumps(implementation.configuration)],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        cwd=implementation.cwd,
        env=target_env,
    ) as p:
        assert p.stdout is not None
        for line in p.stdout:
            m = re.search(r"Created experiment (\d+)\n", line.decode())
            if m is not None:
                return int(m.group(1))

    return None


def create_experiment(implementation: NativeImplementation) -> int:
    return create_native_experiment(
        implementation.cwd,
        implementation.command + ["--config", json.dumps(implementation.configuration)],
    )


def run_warm_start_test(implementation: NativeImplementation) -> None:
    experiment_id1 = create_experiment(implementation)
    experiment.wait_for_experiment_state(
        experiment_id1, "COMPLETED", max_wait_secs=conf.DEFAULT_MAX_WAIT_SECS
    )
    assert experiment.num_active_trials(experiment_id1) == 0

    trials = experiment.experiment_trials(experiment_id1)
    assert len(trials) == 1

    first_trial = trials[0]
    first_trial_id = first_trial["id"]
    assert len(first_trial["steps"]) == implementation.num_expected_steps_per_trial
    first_checkpoint_id = first_trial["steps"][0]["checkpoint"]["id"]

    # Add a source trial ID to warm start from.
    second_exp = NativeImplementation(**implementation._asdict())
    second_exp.configuration["searcher"]["source_trial_id"] = first_trial_id

    experiment_id2 = create_experiment(second_exp)
    experiment.wait_for_experiment_state(
        experiment_id2, "COMPLETED", max_wait_secs=conf.DEFAULT_MAX_WAIT_SECS
    )
    assert experiment.num_active_trials(experiment_id2) == 0

    # The new trials should have a warm start checkpoint ID.
    trials = experiment.experiment_trials(experiment_id2)
    assert len(trials) == 1
    for trial in trials:
        assert trial["warm_start_checkpoint_id"] == first_checkpoint_id


@pytest.mark.parametrize(  # type: ignore
    "implementation",
    [
        NativeImplementations.TFKerasMNISTCNNSingleFitGenerator,
        NativeImplementations.TFKerasMNISTCNNSingleFit,
        NativeImplementations.TFKerasMNISTCNNSingleGeneric,
    ],
)
@pytest.mark.parametrize("tf2", [True, False])  # type: ignore
@pytest.mark.e2e_cpu  # type: ignore
def test_tf_keras_warm_start(implementation: NativeImplementation, tf2: bool) -> None:
    implementation = implementation._replace(
        configuration=(
            conf.set_tf2_image(implementation.configuration)
            if tf2
            else conf.set_tf1_image(implementation.configuration)
        )
    )
    run_warm_start_test(implementation)


@pytest.mark.parametrize(  # type: ignore
    "implementation",
    [
        NativeImplementations.TFEstimatorMNISTCNNSingle,
        NativeImplementations.TFEstimatorMNISTCNNSingleGeneric,
    ],
)
@pytest.mark.parametrize("tf2", [True, False])  # type: ignore
@pytest.mark.e2e_cpu  # type: ignore
def test_tf_estimator_warm_start(implementation: NativeImplementation, tf2: bool) -> None:
    implementation = implementation._replace(
        configuration=(
            conf.set_tf2_image(implementation.configuration)
            if tf2
            else conf.set_tf1_image(implementation.configuration)
        )
    )
    run_warm_start_test(implementation)


@pytest.mark.parametrize(  # type: ignore
    "implementation", [NativeImplementations.PytorchMNISTCNNSingleGeneric]
)
@pytest.mark.e2e_cpu  # type: ignore
def test_pytorch_warm_start(implementation: NativeImplementation) -> None:
    run_warm_start_test(implementation)


@pytest.mark.e2e_cpu  # type: ignore
def test_native_experiment_submit() -> None:
    context_dir = conf.official_examples_path("trial/mnist_pytorch")
    config = conf.load_config(conf.fixtures_path("mnist_pytorch/const-pytorch11.yaml"))
    native_experiment_submit(context_dir, config, follow_logs=True)
