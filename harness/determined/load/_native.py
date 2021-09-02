import contextlib
import json
import logging
import runpy
import sys
from typing import Any, Iterator, List, Optional, Type, cast

import determined as det
from determined import horovod, load, workload
from determined.common import check


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

    def __init__(
        self,
        env: det.EnvContext,
        hvd_config: horovod.HorovodContext,
        rendezvous_info: det.RendezvousInfo,
    ) -> None:
        self.env = env  # type: det.EnvContext
        self.hvd_config = hvd_config  # type: horovod.HorovodContext
        self.rendezvous_info = rendezvous_info  # type: det.RendezvousInfo
        self.trial_cls = None  # type: Optional[Type[det.Trial]]

    def __enter__(self) -> "RunpyGlobals":
        check.is_none(
            RunpyGlobals._instance, "Please only use RunpyGlobals context once at a time."
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
    def set_runpy_trial_result(cls, trial_cls: Type[det.Trial]) -> None:
        check.is_none(cls.get_instance().trial_cls, "Please don't load twice.")
        cls.get_instance().trial_cls = trial_cls

    @classmethod
    def get_runpy_result(cls) -> Type[det.Trial]:
        trial_cls = cls.get_instance().trial_cls
        check.is_not_none(trial_cls, "Please load native implementation.")
        assert trial_cls is not None
        return trial_cls


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


def get_trial_from_native(
    env: det.EnvContext,
    hvd_config: horovod.HorovodContext,
    rendezvous_info: det.RendezvousInfo,
) -> Type[det.Trial]:
    # For now, we assume the entrypoint_cmd is a python invocation like
    # "python <command>"
    command = env.experiment_config["internal"]["native"]["command"]  # type: List[str]
    logging.info(f"Loading Native implementation with command {command}.")
    if len(command) < 1:
        raise AssertionError("Expected non-empty command, but was empty.")

    if command[0].endswith(".ipynb"):
        command[0] = convert_notebook_to_python_script(command[0])

    with RunpyGlobals(env, hvd_config, rendezvous_info) as loader:
        with overwrite_sys_args(command):
            try:
                runpy.run_path(command[0], run_name="__main__")
            except det.errors.StopLoadingImplementation:
                # If caught this exception, will skip running the rest of the user code.
                pass
        return loader.get_runpy_result()


def load_native(
    env: det.EnvContext,
    rendezvous_info: det.RendezvousInfo,
    hvd_config: horovod.HorovodContext,
    workloads: Optional[workload.Stream] = None,
) -> det.TrialController:
    check.true(
        env.experiment_config.native_enabled(),
        "Experiment configuration does not have an internal.native "
        f"configuration: {env.experiment_config}",
    )

    trial_class = get_trial_from_native(env, hvd_config, rendezvous_info)
    return load.load_trial(
        trial_class=trial_class,
        env=env,
        rendezvous_info=rendezvous_info,
        hvd_config=hvd_config,
        workloads=workloads,
    )
