import contextlib
import logging
import os
import pathlib
import sys
from typing import Any, Dict, Iterator, List, Optional, Tuple

import determined as det
from determined import constants, gpu, horovod, workload
from determined_common import api


def _get_gpus() -> Tuple[bool, List[str], List[int]]:
    gpu_ids, gpu_uuids = gpu.get_gpu_ids_and_uuids()
    use_gpu = len(gpu_uuids) > 0
    return use_gpu, gpu_uuids, gpu_ids


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


def _make_local_execution_exp_config(input_config: Optional[Dict[str, Any]]) -> Dict[str, Any]:
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
        if key in input_config:
            logging.info(
                f"'{key}' configuration key is not supported by local test mode and will be ignored"
            )
            del input_config[key]

    return {**constants.DEFAULT_EXP_CFG, **input_config}


def _make_local_execution_env(
    managed_training: bool,
    config: Optional[Dict[str, Any]],
    hparams: Optional[Dict[str, Any]] = None,
) -> Tuple[det.EnvContext, det.RendezvousInfo, horovod.HorovodContext]:
    config = det.ExperimentConfig(_make_local_execution_exp_config(config))
    hparams = hparams or api.generate_random_hparam_values(config.get("hyperparameters", {}))
    use_gpu, container_gpus, slot_ids = _get_gpus()
    local_rendezvous_ports = (
        f"{constants.LOCAL_RENDEZVOUS_PORT},{constants.LOCAL_RENDEZVOUS_PORT+1}"
    )

    env = det.EnvContext(
        master_addr="",
        master_port=0,
        use_tls=False,
        master_cert_file=None,
        master_cert_name=None,
        container_id="",
        experiment_config=config,
        hparams=hparams,
        initial_workload=workload.train_workload(1, 1, 1, config.scheduling_unit()),
        latest_checkpoint=None,
        use_gpu=use_gpu,
        container_gpus=container_gpus,
        slot_ids=slot_ids,
        debug=config.debug_enabled(),
        workload_manager_type="",
        det_rendezvous_ports=local_rendezvous_ports,
        det_trial_unique_port_offset=0,
        det_trial_runner_network_interface=constants.AUTO_DETECT_TRIAL_RUNNER_NETWORK_INTERFACE,
        det_trial_id="",
        det_experiment_id="",
        det_cluster_id="",
        trial_seed=config.experiment_seed(),
        managed_training=managed_training,
    )
    rendezvous_ports = env.rendezvous_ports()
    rendezvous_info = det.RendezvousInfo(
        addrs=[f"0.0.0.0:{rendezvous_ports[0]}"], addrs2=[f"0.0.0.0:{rendezvous_ports[1]}"], rank=0
    )
    hvd_config = horovod.HorovodContext.from_configs(
        env.experiment_config, rendezvous_info, env.hparams
    )

    return env, rendezvous_info, hvd_config


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
