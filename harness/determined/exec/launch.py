import json
import logging
import os
import signal
import subprocess
import sys
import types

import determined as det
from determined.common import constants, storage
from determined.common.api import authentication, certs
from determined.exec import prep_container

logger = logging.getLogger("determined")


# Signal handler to intercept SLURM SIGTERM notification of pending preemption
def trigger_preemption(signum: int, frame: types.FrameType) -> None:
    info = det.get_cluster_info()
    if info and info.container_rank == 0:
        # Chief container, requests preemption, others ignore
        logger.info("SIGTERM: Preemption imminent.")
        # Notify the master that we need to be preempted
        cert = certs.default_load(info.master_url)
        sess = authentication.login_with_cache(info.master_url, cert=cert)
        sess.post(f"/api/v1/allocations/{info.allocation_id}/signals/pending_preemption")


def launch(experiment_config: det.ExperimentConfig) -> int:
    entrypoint = experiment_config.get_entrypoint()

    if isinstance(entrypoint, str) and det.util.match_legacy_trial_class(entrypoint):
        # Legacy entrypoint ("model_def:Trial") detected
        entrypoint = [
            "python3",
            "-m",
            "determined.launch.torch_distributed",
            "--",
            "--trial",
            entrypoint,
        ]

    if isinstance(entrypoint, str):
        entrypoint = ["sh", "-c", entrypoint]

    # Signals we want to forward from wrapper process to the child
    sig_names = ["SIGINT", "SIGTERM", "SIGHUP", "SIGUSR1", "SIGUSR2", "SIGWINCH", "SIGBREAK"]

    if os.environ.get("DET_RESOURCES_TYPE") == prep_container.RESOURCES_TYPE_SLURM_JOB:
        # SLURM sends SIGTERM to notify of pending preemption, so we register a custom
        # handler to intercept it in the chief rank, and ignore in others.   We invoke
        # trigger_preemption to cause a checkpoint and clean exit (given enough time).
        signal.signal(signal.SIGTERM, trigger_preemption)
        # Drop SIGTERM from forwarding so that we handle it in trigger_preemption
        sig_names.remove("SIGTERM")

    logger.info(f"Launching: {entrypoint}")

    p = subprocess.Popen(entrypoint)
    # Convert from signal names to Signal enums because SIGBREAK is windows-specific
    forwaded_signals = [getattr(signal, name) for name in sig_names if hasattr(signal, name)]
    with det.util.forward_signals(p, *forwaded_signals):
        return p.wait()


if __name__ == "__main__":
    info = det.get_cluster_info()
    assert info is not None, "must be run on-cluster"
    assert info.task_type == "TRIAL", f'must be run with task_type="TRIAL", not "{info.task_type}"'

    # Hack: get the resources id from the environment.
    resources_id = os.environ.get("DET_RESOURCES_ID")
    assert resources_id is not None, "Unable to run with DET_RESOURCES_ID unset"

    # Hack: read the full config.  The experiment config is not a stable API!
    experiment_config = det.ExperimentConfig(info.trial._config)

    det.common.set_logger(experiment_config.debug_enabled())

    logger.info(
        f"New trial runner in (container {resources_id}) on agent {info.agent_id}: "
        + json.dumps(det.util.mask_config_dict(info.trial._config))
    )

    # Perform validations
    try:
        logger.info("Validating checkpoint storage ...")
        storage.validate_config(
            experiment_config.get_checkpoint_storage(),
            container_path=constants.SHARED_FS_CONTAINER_PATH,
        )
    except Exception as e:
        logger.error("Checkpoint storage validation failed: {}".format(e))
        sys.exit(1)

    sys.exit(launch(experiment_config))
