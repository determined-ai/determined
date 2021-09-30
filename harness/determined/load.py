import contextlib
import importlib
import json
import logging
import runpy
import sys
from typing import Iterator, List, Optional, Tuple, Type, cast

import determined as det
from determined.common import check

in_runpy = None
runpy_trial_class = None  # type: Optional[Type[det.Trial]]


@contextlib.contextmanager
def running_in_runpy() -> Iterator:
    global in_runpy
    in_runpy = True
    try:
        yield
    finally:
        in_runpy = True


def convert_notebook_to_python_script(notebook_path: str) -> str:
    check.true(
        notebook_path.endswith(".ipynb"), f"Notebook file {notebook_path} must has a suffix .ipynb"
    )
    processed_cells_path = f"{notebook_path[:-6]}__det__.py"

    with open(notebook_path, "r") as f1, open(processed_cells_path, "w") as f2:
        obj = json.load(f1)
        check.true("cells" in obj, f"Invalid notebook file {notebook_path}")
        for cell in obj["cells"]:
            if cell["cell_type"] == "code":
                lines = [line for line in cell["source"] if not line.lstrip().startswith("!")]
                f2.writelines(lines)
                f2.write("\n")
    return processed_cells_path


@contextlib.contextmanager
def overwrite_sys_args(new_args: List[str]) -> Iterator:
    old_sys_args = sys.argv
    sys.argv = new_args
    try:
        yield
    finally:
        sys.argv = old_sys_args


def get_trial_class_from_native(command: List[str]) -> Type[det.Trial]:
    global runpy_trial_class

    # For now, we assume the entrypoint_cmd is a python invocation like
    # "python <command>"
    logging.info(f"Loading Native implementation with command {command}.")
    if len(command) < 1:
        raise AssertionError("Expected non-empty command, but was empty.")

    if command[0].endswith(".ipynb"):
        command[0] = convert_notebook_to_python_script(command[0])

    with overwrite_sys_args(command), running_in_runpy():
        try:
            runpy.run_path(command[0], run_name="__main__")
        except det.errors.StopLoadingImplementation:
            # If caught this exception, will skip running the rest of the user code.
            pass
        finally:
            trial_cls = runpy_trial_class
            runpy_trial_class = None

    if trial_cls is None:
        raise ValueError("Please load native implementation.")
    return trial_cls


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

    assert isinstance(obj, type), "entrypoint ({entrypoint_spec}) is not a class"
    assert issubclass(obj, det.Trial), "entrypoint ({entrypoint_spec}) is not a det.Trial subclass"
    return cast(Type[det.Trial], obj)


def get_trial_controller_class(trial_class: Type[det.Trial]) -> Type[det.TrialController]:
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


def get_trial_and_controller_class(
    experiment_config: det.ExperimentConfig,
) -> Tuple[Type[det.Trial], Type[det.TrialController]]:
    if experiment_config.native_enabled():
        command = experiment_config["internal"]["native"]["command"]  # type: List[str]
        trial_class = get_trial_class_from_native(command)
    else:
        trial_class = trial_class_from_entrypoint(experiment_config["entrypoint"])

    return trial_class, get_trial_controller_class(trial_class)
