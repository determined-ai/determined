import contextlib
import logging
import os
import pathlib
import sys
from typing import Any, Dict, Iterator, List, Optional, Tuple, Type

import determined as det
from determined import constants, core, gpu, load
from determined.common import api


class InvalidHP(Exception):
    def __init__(self, msg: str = "...") -> None:
        if not isinstance(msg, str):
            raise TypeError(
                "InvalidHP exceptions can be initialized with a custom message, "
                f"but it must be a string, not {type(msg).__name__}"
            )
        self.msg = msg


def _get_gpus(limit_gpus: Optional[int]) -> Tuple[bool, List[str], List[int]]:
    gpus = gpu.get_gpus()

    if limit_gpus is not None:
        gpus = gpus[:limit_gpus]

    use_gpus = len(gpus) > 0

    return use_gpus, [gpu.uuid for gpu in gpus], [gpu.id for gpu in gpus]


@contextlib.contextmanager
def _catch_sys_exit() -> Any:
    try:
        yield
    except SystemExit as e:
        raise det.errors.InvalidExperimentException(
            "Caught a SystemExit exception. "
            "This might be raised by directly calling or using a library calling sys.exit(). "
            "Please remove any calls to sys.exit() from your model code."
        ) from e


def _make_local_execution_exp_config(
    input_config: Optional[Dict[str, Any]],
    checkpoint_dir: str,
    managed_training: bool,
    test_mode: bool,
) -> Dict[str, Any]:
    """
    Create a local experiment configuration based on an input configuration and
    defaults. Use a shallow merging policy to overwrite our default
    configuration with each entire subconfig specified by a user.

    The defaults and merging logic is not guaranteed to match the logic used by
    the Determined master. This function also does not do experiment
    configuration validation, which the Determined master does.
    """

    input_config = input_config.copy() if input_config else {}
    config_keys_to_ignore = {
        "bind_mounts",
        "checkpoint_storage",
        "environment",
        "resources",
        "optimizations",
    }

    for key in config_keys_to_ignore:
        if key not in input_config:
            continue
        # This codepath is used by checkpoint loading, where we do not want to emit any warnings,
        # so only warn if we are explicitly in --local --test mode.
        if test_mode and not managed_training:
            logging.info(
                f"'{key}' configuration key is not supported by local test mode and will be ignored"
            )
        del input_config[key]

    checkpoint_storage = {
        "type": "shared_fs",
        "host_path": os.path.abspath(checkpoint_dir),
    }

    return {"checkpoint_storage": checkpoint_storage, **constants.DEFAULT_EXP_CFG, **input_config}


def _make_local_execution_env(
    managed_training: bool,
    test_mode: bool,
    config: Optional[Dict[str, Any]],
    checkpoint_dir: str,
    hparams: Optional[Dict[str, Any]] = None,
    limit_gpus: Optional[int] = None,
) -> Tuple[core.Context, det.EnvContext]:
    config = det.ExperimentConfig(
        _make_local_execution_exp_config(
            config, checkpoint_dir, managed_training=managed_training, test_mode=test_mode
        )
    )
    hparams = hparams or api.generate_random_hparam_values(config.get("hyperparameters", {}))
    use_gpu, container_gpus, slot_ids = _get_gpus(limit_gpus)

    env = det.EnvContext(
        master_url="",
        master_cert_file=None,
        master_cert_name=None,
        experiment_config=config,
        hparams=hparams,
        latest_checkpoint=None,
        steps_completed=0,
        use_gpu=use_gpu,
        container_gpus=container_gpus,
        slot_ids=slot_ids,
        debug=config.debug_enabled(),
        det_trial_unique_port_offset=0,
        det_trial_id="",
        det_agent_id="",
        det_experiment_id="",
        det_cluster_id="",
        trial_seed=config.experiment_seed(),
        trial_run_id=1,
        allocation_id="",
        managed_training=managed_training,
        test_mode=test_mode,
        on_cluster=False,
    )

    core_context = core._dummy_init()

    return core_context, env


@contextlib.contextmanager
def _local_execution_manager(context_dir: pathlib.Path) -> Iterator:
    """
    A context manager used for local execution to mimic the environment of trial
    container execution.

    It does the following things:
    1. Set the current working directory to be the context directory.
    2. Add the current working directory to importing paths.
    3. Catch SystemExit.
    """
    current_directory = os.getcwd()
    current_path = sys.path[0]

    try:
        os.chdir(str(context_dir))

        # Python typically initializes sys.path[0] as the empty string which directs
        # Python to search modules in the current directory first when invoked
        # interactively. We set sys.path[0] manually to let Python importer search the
        # the current directory first.
        # Reference: https://docs.python.org/3/library/sys.html#sys.path
        sys.path[0] = ""
        with det._catch_sys_exit():
            yield
    finally:
        os.chdir(current_directory)
        sys.path[0] = current_path


def _load_trial_for_checkpoint_export(
    context_dir: pathlib.Path,
    managed_training: bool,
    trial_cls_spec: str,
    config: Dict[str, Any],
    hparams: Dict[str, Any],
) -> Tuple[Type[det.Trial], det.TrialContext]:
    with _local_execution_manager(context_dir):
        trial_class = load.trial_class_from_entrypoint(trial_cls_spec)
        core_context, env = _make_local_execution_env(
            managed_training=managed_training,
            test_mode=False,
            config=config,
            checkpoint_dir="/tmp",
            hparams=hparams,
        )
        trial_context = trial_class.trial_context_class(core_context, env)
    return trial_class, trial_context
