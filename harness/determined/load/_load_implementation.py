import contextlib
import importlib
import json
import logging
import runpy
import sys
from typing import Any, Iterator, List, Optional, Tuple, Type, cast

import determined as det
from determined import horovod
from determined_common import check


def load_trial_implementation(entrypoint_spec: str) -> Type[det.Trial]:
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


class RunpyGlobals:
    """
    RunpyGlobals is a singleton class that is used to share states between
    the harness and user code when loading user code. It should be used as a
    context manager. Below is how it is used through the procedure of loading
    user code.

    1. Before running user code, the environment context is used for
    instantiating RunpyGlobals.

    2. During loading user code, user code retrieves the environment context
    from RunpyGlobals and use it to run framework-specific initialization.
    User code also passes on the native context, which wraps the model and
    other objects used for training, to RunpyGlobals.

    3. After running user code, the native context is used for instantiating
    trial controller.
    """

    _instance = None  # type: Optional[RunpyGlobals]

    def __init__(self, env: det.EnvContext, hvd_config: horovod.HorovodContext):
        self.env = env  # type: det.EnvContext
        self.hvd_config = hvd_config  # type: horovod.HorovodContext
        self.context = None  # type: Optional[det.NativeContext]
        self.trial_cls = None  # type: Optional[Type[det.Trial]]
        self.controller_cls = None  # type: Optional[Type[det.TrialController]]

    def __enter__(self) -> "RunpyGlobals":
        check.true(
            RunpyGlobals._instance is None, "Please only use RunpyGlobals context once at a time."
        )
        RunpyGlobals._instance = self
        return self

    def __exit__(self, *_: Any) -> None:
        RunpyGlobals._instance = None

    @classmethod
    def is_initialized(cls) -> bool:
        return cls._instance is not None

    @classmethod
    def get_instance(cls) -> "RunpyGlobals":
        check.is_not_none(cls._instance, "Please initialize RunpyGlobals context first.")
        return cast(RunpyGlobals, cls._instance)

    @classmethod
    def set_runpy_native_result(
        cls, context: det.NativeContext, controller_cls: Type[det.TrialController]
    ) -> None:
        check.true(cls.get_instance().controller_cls is None, "Please don't load twice.")
        cls.get_instance().context = context
        cls.get_instance().controller_cls = controller_cls

    @classmethod
    def set_runpy_trial_result(
        cls, trial_cls: Type[det.Trial], controller_cls: Type[det.TrialController]
    ) -> None:
        check.true(cls.get_instance().controller_cls is None, "Please don't load twice.")
        cls.get_instance().trial_cls = trial_cls
        cls.get_instance().controller_cls = controller_cls
        raise det.errors.StopLoadingImplementation()

    @classmethod
    def get_runpy_result(
        cls,
    ) -> Tuple[Optional[det.NativeContext], Optional[Type[det.Trial]], Type[det.TrialController]]:
        check.true(
            cls.get_instance().controller_cls is not None, "Please load native implementation."
        )
        return (
            cls.get_instance().context,
            cls.get_instance().trial_cls,
            cast(Type[det.TrialController], cls.get_instance().controller_cls),
        )


def convert_notebook_to_python_script(notebook_path: str) -> str:
    check.check_true(
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


def load_native_implementation(
    env: det.EnvContext, hvd_config: horovod.HorovodContext
) -> Tuple[Optional[det.NativeContext], Optional[Type[det.Trial]], Type[det.TrialController]]:
    # For now, we assume the entrypoint_cmd is a python invocation like
    # "python <command>"
    command = env.experiment_config["internal"]["native"]["command"]  # type: List[str]
    logging.info(f"Loading Native implementation with command {command}.")
    if len(command) < 1:
        raise AssertionError("Expected non-empty command, but was empty.")

    if command[0].endswith(".ipynb"):
        command[0] = convert_notebook_to_python_script(command[0])

    with RunpyGlobals(env, hvd_config) as loader:
        with overwrite_sys_args(command):
            try:
                runpy.run_path(command[0], run_name="__main__")
            except det.errors.StopLoadingImplementation:
                # If caught this exception, will skip running the rest of the user code.
                pass
        context, trial_cls, controller_cls = loader.get_runpy_result()

    return context, trial_cls, controller_cls
