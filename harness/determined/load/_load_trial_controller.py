import logging
import pathlib
from typing import List, Optional, Tuple, Type, cast

import determined as det
from determined import horovod, load, tensorboard, workload
from determined._trial import TrialCapabilities
from determined.estimator import EstimatorTrial
from determined.keras import TFKerasTrial
from determined.pytorch import PyTorchTrial
from determined_common import check
from determined_common.api import patch


def trial_framework(trial_class: Type[det.Trial]) -> Optional[Tuple[str, TrialCapabilities]]:
    classes: List[Type[det.Trial]] = [PyTorchTrial, TFKerasTrial, EstimatorTrial]
    for cls in classes:
        if issubclass(trial_class, cls):
            return cls.name(), cls.capabilities()

def report_framework(name: str, capabilities: TrialCapabilities, env: det.EnvContext) -> None:
    host = env.master_addr + ":" + str(env.master_port)
    path = f"/api/v1/experiments/{env.det_experiment_id}"

    name_val = "UNSPECIFIED"
    if name == "PyTorchTrial":
        name_val = "PYTORCH_TRIAL"
    name_val = "FRAMEWORK_" + name_val

    patch(host, path, body={"framework": name_val})

def load_controller_from_trial(
    trial_class: Type[det.Trial],
    env: det.EnvContext,
    workloads: workload.Stream,
    load_path: Optional[pathlib.Path],
    rendezvous_info: det.RendezvousInfo,
    hvd_config: horovod.HorovodContext,
) -> det.TrialController:
    # Step 1: Validate model definition.
    controller_class = trial_class.trial_controller_class
    check.is_not_none(
        controller_class,
        f"The class attribute `trial_controller_class` of {trial_class.__name__} is "
        "None; please set it the correct subclass of `det.TrialController`",
    )
    check.is_subclass(
        controller_class,
        det.TrialController,
        f"The class attribute `trial_controller_class` of {trial_class.__name__} is "
        "not a valid subclass of `det.TrialController`",
    )
    controller_class = cast(Type[det.TrialController], controller_class)

    # Step 2: Initialize framework-specific details (horovod, random seeds, etc).
    controller_class.pre_execute_hook(env, hvd_config)
    trial_context = trial_class.trial_context_class(env, hvd_config)

    # Step 3: Instantiate the user's Trial.
    trial_inst = trial_class(trial_context)


    # Step 4: Return the TrialController.
    logging.info(f"Creating {controller_class.__name__} with {trial_class.__name__}.")
    controller = controller_class.from_trial(
        trial_inst=trial_inst,
        context=trial_context,
        env=env,
        workloads=workloads,
        load_path=load_path,
        rendezvous_info=rendezvous_info,
        hvd_config=hvd_config,
    )

    fw = trial_framework(trial_class)
    if fw is not None:
        is_lead_trial = trial_context.distributed.get_rank() == 0 and \
            trial_context.distributed.get_local_rank() == 0
        if not env.test_mode and is_lead_trial:
            report_framework(fw[0], fw[1], env)

    return controller


def load_trial_implementation_controller(
    env: det.EnvContext,
    workloads: workload.Stream,
    load_path: Optional[pathlib.Path],
    rendezvous_info: det.RendezvousInfo,
    hvd_config: horovod.HorovodContext,
) -> det.TrialController:
    trial_class = load.load_trial_implementation(env.experiment_config["entrypoint"])
    return load_controller_from_trial(
        trial_class=trial_class,
        env=env,
        workloads=workloads,
        load_path=load_path,
        rendezvous_info=rendezvous_info,
        hvd_config=hvd_config,
    )


def load_native_implementation_controller(
    env: det.EnvContext,
    workloads: workload.Stream,
    load_path: Optional[pathlib.Path],
    rendezvous_info: det.RendezvousInfo,
    hvd_config: horovod.HorovodContext,
) -> det.TrialController:
    check.true(
        env.experiment_config.native_enabled(),
        "Experiment configuration does not have an internal.native "
        f"configuration: {env.experiment_config}",
    )

    context, trial_class, controller_class = load.load_native_implementation(env, hvd_config)

    if trial_class is not None:
        return load_controller_from_trial(
            trial_class=trial_class,
            env=env,
            workloads=workloads,
            load_path=load_path,
            rendezvous_info=rendezvous_info,
            hvd_config=hvd_config,
        )

    else:
        # Framework-specific native implementation.
        check.is_not_none(
            controller_class,
            "The class attribute `trial_controller_class` is "
            "None; please set it the correct subclass of `det.TrialController`",
        )
        check.is_subclass(
            controller_class,
            det.TrialController,
            "The class attribute `trial_controller_class` is "
            "not a valid subclass of `det.TrialController`",
        )
        logging.info(f"Creating {controller_class.__name__} with {type(context).__name__}.")
        return cast(det.TrialController, controller_class).from_native(
            context=cast(det.NativeContext, context),
            env=env,
            workloads=workloads,
            load_path=load_path,
            rendezvous_info=rendezvous_info,
            hvd_config=hvd_config,
        )


def prepare_controller(
    env: det.EnvContext,
    workloads: workload.Stream,
    load_path: Optional[pathlib.Path],
    rendezvous_info: det.RendezvousInfo,
    hvd_config: horovod.HorovodContext,
) -> det.TrialController:
    """
    Load a user's python code, locate the Trial and Trial Controller, then instantiate one.
    """

    if env.experiment_config.native_enabled():
        controller = load_native_implementation_controller(
            env, workloads, load_path, rendezvous_info, hvd_config
        )
    else:
        controller = load_trial_implementation_controller(
            env, workloads, load_path, rendezvous_info, hvd_config
        )
    return controller


def prepare_tensorboard(
    env: det.EnvContext,
    container_path: Optional[str] = None,
) -> Tuple[tensorboard.TensorboardManager, tensorboard.BatchMetricWriter]:
    tensorboard_mgr = tensorboard.build(
        env, env.experiment_config["checkpoint_storage"], container_path
    )
    try:
        from determined.tensorboard.metric_writers import tensorflow

        writer: tensorboard.MetricWriter = tensorflow.TFWriter()

    except ModuleNotFoundError:
        logging.warning("Tensorflow writer not found")
        from determined.tensorboard.metric_writers import pytorch

        writer = pytorch.TorchWriter()

    return (
        tensorboard_mgr,
        tensorboard.BatchMetricWriter(writer),
    )
