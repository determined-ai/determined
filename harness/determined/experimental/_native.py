import logging
import tempfile
from typing import Any, Dict, Optional, Type

import determined as det
from determined import workload
from determined.common import check


def _make_test_workloads(config: det.ExperimentConfig) -> workload.Stream:
    interceptor = workload.WorkloadResponseInterceptor()

    logging.info("Training one batch")
    yield from interceptor.send(workload.train_workload(1))
    metrics = interceptor.metrics_result()
    batch_metrics = metrics["metrics"]["batch_metrics"]
    check.eq(len(batch_metrics), config.scheduling_unit())
    logging.info(f"Finished training, metrics: {batch_metrics}")

    logging.info("Validating one batch")
    yield from interceptor.send(workload.validation_workload(1))
    validation = interceptor.metrics_result()
    v_metrics = validation["metrics"]["validation_metrics"]
    logging.info(f"Finished validating, validation metrics: {v_metrics}")

    logging.info("Saving a checkpoint.")
    yield workload.checkpoint_workload(), workload.ignore_workload_response
    logging.info("Finished saving a checkpoint.")


def test_one_batch(
    trial_class: Type[det.Trial],
    config: Optional[Dict[str, Any]] = None,
) -> Any:
    # Override the scheduling_unit value to 1.
    config = {**(config or {}), "scheduling_unit": 1}

    logging.info("Running a minimal test experiment locally")
    with tempfile.TemporaryDirectory() as checkpoint_dir:
        core_context, env = det._make_local_execution_env(
            managed_training=True,
            test_mode=True,
            config=config,
            checkpoint_dir=checkpoint_dir,
            limit_gpus=1,
        )

        workloads = _make_test_workloads(env.experiment_config)
        logging.info(f"Using hyperparameters: {env.hparams}.")
        logging.debug(f"Using a test experiment config: {env.experiment_config}.")

        distributed_backend = det._DistributedBackend()
        controller_class = trial_class.trial_controller_class
        assert controller_class is not None
        controller_class.pre_execute_hook(env, distributed_backend)

        trial_context = trial_class.trial_context_class(core_context, env)
        logging.info(f"Creating {trial_class.__name__}.")
        trial_inst = trial_class(trial_context)

        controller = controller_class.from_trial(
            trial_inst=trial_inst,
            context=trial_context,
            env=env,
            workloads=workloads,
        )

        controller.run()

        logging.info("The test experiment passed.")
        logging.info(
            "Note: to submit an experiment to the cluster, change local parameter to False"
        )
