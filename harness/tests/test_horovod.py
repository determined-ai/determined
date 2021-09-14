import pytest

from determined import constants, horovod


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
        inter_node_network_interface=None,
        optimizations=optimizations,
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
