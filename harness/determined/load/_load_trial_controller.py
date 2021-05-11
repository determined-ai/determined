import logging
import pathlib
from typing import Optional, Tuple, Type, cast

import determined as det
from determined import horovod, load, profiler, tensorboard, workload
from determined.common import check


def load_controller_from_trial(
    trial_class: Type[det.Trial],
    env: det.EnvContext,
    workloads: workload.Stream,
    load_path: Optional[pathlib.Path],
    rendezvous_info: det.RendezvousInfo,
    hvd_config: horovod.HorovodContext,
    prof: profiler.ProfilerAgent,
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
    trial_context = trial_class.trial_context_class(env, hvd_config, rendezvous_info)

    # Step 3: Instantiate the user's Trial.
    trial_inst = trial_class(trial_context)

    # Step 4: Return the TrialController.
    logging.info(f"Creating {controller_class.__name__} with {trial_class.__name__}.")
    return controller_class.from_trial(
        trial_inst=trial_inst,
        prof=prof,
        context=trial_context,
        env=env,
        workloads=workloads,
        load_path=load_path,
        rendezvous_info=rendezvous_info,
        hvd_config=hvd_config,
    )


def load_trial_implementation_controller(
    env: det.EnvContext,
    workloads: workload.Stream,
    load_path: Optional[pathlib.Path],
    rendezvous_info: det.RendezvousInfo,
    hvd_config: horovod.HorovodContext,
    prof: profiler.ProfilerAgent,
) -> det.TrialController:
    trial_class = load.trial_class_from_entrypoint(env.experiment_config["entrypoint"])
    return load_controller_from_trial(
        trial_class=trial_class,
        prof=prof,
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
    prof: profiler.ProfilerAgent,
) -> det.TrialController:
    check.true(
        env.experiment_config.native_enabled(),
        "Experiment configuration does not have an internal.native "
        f"configuration: {env.experiment_config}",
    )

    context, trial_class, controller_class = load.load_native_implementation(
        env, hvd_config, rendezvous_info
    )

    if trial_class is not None:
        return load_controller_from_trial(
            trial_class=trial_class,
            env=env,
            workloads=workloads,
            load_path=load_path,
            rendezvous_info=rendezvous_info,
            hvd_config=hvd_config,
            prof=prof,
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
            prof=prof,
        )


def prepare_controller(
    env: det.EnvContext,
    workloads: workload.Stream,
    load_path: Optional[pathlib.Path],
    rendezvous_info: det.RendezvousInfo,
    hvd_config: horovod.HorovodContext,
    prof: profiler.ProfilerAgent,
) -> det.TrialController:
    """
    Load a user's python code, locate the Trial and Trial Controller, then instantiate one.
    """

    if env.experiment_config.native_enabled():
        controller = load_native_implementation_controller(
            env=env,
            workloads=workloads,
            load_path=load_path,
            rendezvous_info=rendezvous_info,
            hvd_config=hvd_config,
            prof=prof,
        )
    else:
        controller = load_trial_implementation_controller(
            env=env,
            workloads=workloads,
            load_path=load_path,
            rendezvous_info=rendezvous_info,
            hvd_config=hvd_config,
            prof=prof,
        )

    return controller


def prepare_tensorboard(
    env: det.EnvContext,
    container_path: Optional[str] = None,
) -> Tuple[tensorboard.TensorboardManager, tensorboard.BatchMetricWriter]:
    tensorboard_mgr = tensorboard.build(
        env.det_cluster_id,
        env.det_experiment_id,
        env.det_trial_id,
        env.experiment_config["checkpoint_storage"],
        container_path,
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
