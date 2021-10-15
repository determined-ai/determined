import importlib
import logging
import sys
from typing import Optional, Type, cast

import determined as det
from determined import horovod, load, workload
from determined.common import check


def trial_class_from_entrypoint(entrypoint_spec: str) -> Type[det.Trial]:
    """
    Load and initialize a Trial class from an entrypoint specification.

    An entrypoint specification is expected to take the form:

        "<module>:<object reference>"

    <module> specifies the module containing the trial class within the model
    definition, relative to the root.

    <object reference> specifies the naming of the trial class within the
    module. It may be a nested object delimited by dots.

    Examples:

        "model_def:CIFAR10Trial": expects a "CIFAR10Trial" class that is
        defined in a file model_def.py

        "my_lib.trial:trial_classes.NestedTrial": expects a "NestedTrial"
        class that is an attribute of `trial_classes`, where `trial_classes` is
        defined in a file my_lib/trial.py

    Note that this follows the entrypoints specification loading logic defined
    in [1] with a single difference: the directory name of the model definition
    is prefixed to <module>, or used as the module if <module> is empty.

    [1] https://packaging.python.org/specifications/entry-points/
    """

    logging.info(f"Loading Trial implementation with entrypoint {entrypoint_spec}.")
    module, qualname_separator, qualname = entrypoint_spec.partition(":")

    # Exporting checkpoints reliably requires instantiating models from user
    # trials and loading their weights. The user may load multiple trials into
    # the same process. If the trials have the same module name, ie. model_def,
    # python will only load the module once. Thus, it would be impossible to
    # load trials from different experiments into the same process. To avoid
    # this, we remove the module name from sys.modules if it already exists to
    # force python to load the module regardless of its name.
    if module in sys.modules:
        sys.modules.pop(module)

    obj = importlib.import_module(module)
    if qualname_separator:
        for attr in qualname.split("."):
            obj = getattr(obj, attr)

    check.check_issubclass(
        obj, det.Trial, "Invalid type for specified 'entrypoint' ({})".format(entrypoint_spec)
    )

    return cast(Type[det.Trial], obj)


def load_trial(
    trial_class: Type[det.Trial],
    env: det.EnvContext,
    rendezvous_info: det.RendezvousInfo,
    hvd_config: horovod.HorovodContext,
    workloads: Optional[workload.Stream] = None,
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

    try:
        # Step 3: Instantiate the user's Trial.
        trial_inst = trial_class(trial_context)

        # Step 4: Return the TrialController.
        logging.info(f"Creating {controller_class.__name__} with {trial_class.__name__}.")
        return controller_class.from_trial(
            trial_inst=trial_inst,
            context=trial_context,
            env=env,
            rendezvous_info=rendezvous_info,
            hvd_config=hvd_config,
            workloads=workloads,
        )
    except Exception:
        # TODO: Refactor load_trial so that it takes a generic context as input.
        trial_context.distributed.close()
        raise


def prepare_controller(
    env: det.EnvContext,
    rendezvous_info: det.RendezvousInfo,
    hvd_config: horovod.HorovodContext,
) -> det.TrialController:
    """
    Load a user's python code, locate the Trial and Trial Controller, then instantiate one.
    """

    if env.experiment_config.native_enabled():
        controller = load.load_native(env, rendezvous_info, hvd_config)
    else:
        trial_class = trial_class_from_entrypoint(env.experiment_config["entrypoint"])
        controller = load_trial(trial_class, env, rendezvous_info, hvd_config)

    return controller
