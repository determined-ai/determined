import importlib
import logging
import sys
from typing import Type, cast

import determined as det

logger = logging.getLogger("determined")


def trial_class_from_entrypoint(entrypoint_spec: str) -> Type[det.LegacyTrial]:
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

    logger.info(f"Loading Trial implementation with entrypoint {entrypoint_spec}.")
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

    assert isinstance(obj, type), f"entrypoint ({entrypoint_spec}) is not a class"
    assert issubclass(
        obj, det.LegacyTrial
    ), f"entrypoint ({entrypoint_spec}) is not a det.Trial subclass"
    return cast(Type[det.LegacyTrial], obj)


def get_trial_controller_class(trial_class: Type[det.LegacyTrial]) -> Type[det.TrialController]:
    # Validate the Trial class
    controller_class = trial_class.trial_controller_class
    if controller_class is None:
        raise ValueError(
            f"The class attribute `trial_controller_class` of {trial_class.__name__} is "
            "None; please set it the correct subclass of `det.TrialController`",
        )

    if not issubclass(controller_class, det.TrialController):
        raise ValueError(
            f"The class attribute `trial_controller_class` of {trial_class.__name__} is "
            "not a valid subclass of `det.TrialController`",
        )

    return controller_class
