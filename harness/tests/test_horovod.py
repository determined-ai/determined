from typing import Any, Dict

import pytest

import determined as det
from determined import constants, horovod


def create_default_env_context(experiment_config: Dict[str, Any]) -> det.EnvContext:
    det_trial_runner_network_interface = constants.AUTO_DETECT_TRIAL_RUNNER_NETWORK_INTERFACE
    return det.EnvContext(
        experiment_config=experiment_config,
        master_addr="",
        master_port=0,
        use_tls=False,
        master_cert_file=None,
        master_cert_name=None,
        container_id="",
        hparams={"global_batch_size": 32},
        latest_checkpoint=None,
        latest_batch=0,
        use_gpu=False,
        container_gpus=[],
        slot_ids=[],
        debug=False,
        det_rendezvous_port="",
        det_trial_unique_port_offset=0,
        det_trial_runner_network_interface=det_trial_runner_network_interface,
        det_trial_id="1",
        det_agent_id="1",
        det_experiment_id="1",
        det_allocation_token="",
        det_cluster_id="uuid-123",
        trial_seed=0,
        trial_run_id=1,
        allocation_id="",
        managed_training=True,
        test_mode=False,
        on_cluster=False,
    )


@pytest.mark.parametrize("debug", [True, False])  # type: ignore
@pytest.mark.parametrize("auto_tune", [True, False])  # type: ignore
@pytest.mark.parametrize("tensor_fusion_threshold", [64, 128, 512])  # type: ignore
@pytest.mark.parametrize("tensor_fusion_cycle_time", [5, 20])  # type: ignore
def test_create_run_command(
    debug: bool, auto_tune: bool, tensor_fusion_threshold: int, tensor_fusion_cycle_time: int
) -> None:
    ip_addresses = ["localhost", "128.140.2.4"]
    num_proc_per_machine = 8
    optimizations = {
        "auto_tune_tensor_fusion": auto_tune,
        "tensor_fusion_threshold": tensor_fusion_threshold,
        "tensor_fusion_cycle_time": tensor_fusion_cycle_time,
    }
    experiment_config = {
        "optimizations": optimizations,
        "resources": {"slots_per_trial": 1, "native_parallel": False},
    }
    env = create_default_env_context(experiment_config)

    expected_horovod_run_cmd = [
        "horovodrun",
        "-np",
        "16",
        "-p",
        str(constants.HOROVOD_SSH_PORT),
        "-H",
        "localhost:8,128.140.2.4:8",
        "--start-timeout",
        str(constants.HOROVOD_STARTUP_TIMEOUT_SECONDS),
        "--gloo-timeout-seconds",
        str(constants.HOROVOD_GLOO_TIMEOUT_SECONDS),
    ]
    if auto_tune:
        expected_horovod_run_cmd.extend(
            ["--autotune", "--autotune-log-file", str(constants.HOROVOD_AUTOTUNE_LOG_FILEPATH)]
        )
    else:
        expected_horovod_run_cmd.extend(
            [
                "--fusion-threshold-mb",
                str(tensor_fusion_threshold),
                "--cycle-time-ms",
                str(tensor_fusion_cycle_time),
            ]
        )
    expected_horovod_run_cmd.extend(
        [
            "--cache-capacity",
            str(1024),
            "--no-hierarchical-allreduce",
            "--no-hierarchical-allgather",
        ]
    )
    if debug:
        expected_horovod_run_cmd.append("--verbose")
    expected_horovod_run_cmd.append("--")

    created_horovod_run_cmd = horovod.create_run_command(
        num_proc_per_machine=num_proc_per_machine,
        ip_addresses=ip_addresses,
        env=env,
        debug=debug,
        optional_args=[],
    )
    assert expected_horovod_run_cmd == created_horovod_run_cmd


def test_create_hostlist_arg() -> None:
    ip_addresses = ["localhost", "128.140.2.4"]
    num_proc_per_machine = 8
    expected_horovod_hostlist_arg = (
        f"{ip_addresses[0]}:{num_proc_per_machine},{ip_addresses[1]}:{num_proc_per_machine}"
    )
    created_horovod_hostlist_arg = horovod.create_hostlist_arg(
        num_proc_per_machine=num_proc_per_machine, ip_addresses=ip_addresses
    )
    assert expected_horovod_hostlist_arg == created_horovod_hostlist_arg
