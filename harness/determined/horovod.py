import importlib
import os
from typing import Any, Dict, List, Optional

from determined import constants
from determined.common import check


class _PolyHorovod:
    """
    Importing two different types of horovod in the same python process (horovod.tensorflow and
    horovod.pytorch, for instance) results in a segfault.

    _PolyHorovod is a wrapper around the horovod module to delay the actual importing of horovod
    until it is known which version is actually needed for the task. The result is that horovod
    importing across Determined becomes simple, easy, and robust.

    After require_horovod_type() is called once, horovod is imported, and _PolyHorovod passes all
    other calls to the real horovod module.
    """

    def __init__(self) -> None:
        self._poly_hvd_type = None  # type: Optional[str]
        self._poly_hvd_first_reason = "(horovod type has not been set)"
        self._poly_hvd_module = None  # type: Any
        self._poly_hvd_imported = False

    def require_horovod_type(self, horovod_type: str, reason: str) -> None:
        """
        Declare the required type of horovod and give a unique reason as to why it is required.

        The reason makes for clear error reporting if require_horovod_type() is called a second
        time but with a different type.
        """

        known_types = {"tensorflow", "tensorflow.keras", "torch"}
        check.is_in(horovod_type, known_types, "Unknown horovod type requested.")

        if self._poly_hvd_type is not None:
            check.eq(
                horovod_type,
                self._poly_hvd_type,
                f"require_horovod_type() called with with type {horovod_type} after a previous "
                f"call with type {self._poly_hvd_type} in the same process. The reason for the "
                f"first call was '{self._poly_hvd_first_reason}'; the reason for this call is "
                f"'{reason}'.",
            )
        else:
            self._poly_hvd_type = horovod_type
            self._poly_hvd_first_reason = reason
            # If horovod has not been imported yet, do it now.
            try:
                self._poly_hvd_module = importlib.import_module(f"horovod.{horovod_type}")
            except ImportError:
                pass

    def __getattr__(self, attr: str) -> Any:
        check.is_not_none(
            self._poly_hvd_type,
            "You must call det.horovod.hvd.require_horovod_type() before any other calls.",
        )
        check.is_not_none(self._poly_hvd_module, "Horovod could not be imported in this process.")
        return getattr(self._poly_hvd_module, attr)

    def cross_rank(self) -> Any:
        """
        When hvd.cross_rank() is not reliably present (version =< v0.22.1) we fall back to reading
        HOROVOD_CROSS_RANK, the environment variable set by the gloo controller as far back as
        v0.17.0.
        """
        if hasattr(self._poly_hvd_module, "cross_rank"):
            return self._poly_hvd_module.cross_rank()
        if "HOROVOD_CROSS_RANK" in os.environ:
            return int(os.environ["HOROVOD_CROSS_RANK"])
        raise RuntimeError("hvd has no cross_rank() and HOROVOD_CROSS_RANK is not set")

    def cross_size(self) -> Any:
        if hasattr(self._poly_hvd_module, "cross_size"):
            return self._poly_hvd_module.cross_size()
        if "HOROVOD_CROSS_SIZE" in os.environ:
            return int(os.environ["HOROVOD_CROSS_SIZE"])
        raise RuntimeError("hvd has no cross_size() and HOROVOD_CROSS_SIZE is not set")


hvd = _PolyHorovod()


def create_hostlist_arg(num_proc_per_machine: int, ip_addresses: List[str]) -> str:
    trial_runner_hosts = ip_addresses.copy()
    # Horovodrun does not interpret "0.0.0.0" correctly.
    trial_runner_hosts[0] = "localhost"
    return ",".join([f"{host}:{num_proc_per_machine}" for host in trial_runner_hosts])


def create_performance_args(optimizations: Dict[str, Any]) -> List[str]:
    check.check_in("auto_tune_tensor_fusion", optimizations)
    check.check_in("tensor_fusion_threshold", optimizations)
    check.check_in("tensor_fusion_cycle_time", optimizations)

    if optimizations.get("auto_tune_tensor_fusion"):
        performance_args = [
            "--autotune",
            "--autotune-log-file",
            str(constants.HOROVOD_AUTOTUNE_LOG_FILEPATH),
        ]
    else:
        performance_args = [
            "--fusion-threshold-mb",
            str(optimizations.get("tensor_fusion_threshold")),
            "--cycle-time-ms",
            str(optimizations.get("tensor_fusion_cycle_time")),
        ]

    # Prevent horovod from auto-tuning these parameters.
    performance_args.extend(
        [
            "--cache-capacity",
            str(1024),
            "--no-hierarchical-allreduce",
            "--no-hierarchical-allgather",
        ]
    )
    return performance_args


def create_run_command(
    num_proc_per_machine: int,
    ip_addresses: List[str],
    inter_node_network_interface: Optional[str],
    optimizations: Dict[str, Any],
    debug: bool,
    optional_args: List[str],
) -> List[str]:
    num_machines = len(ip_addresses)
    num_proc_total = num_proc_per_machine * num_machines

    # Construct the horovodrun command.
    horovod_process_cmd = [
        "horovodrun",
        "-np",
        str(num_proc_total),
        "-p",
        str(constants.DTRAIN_SSH_PORT),
        "-H",
        create_hostlist_arg(num_proc_per_machine, ip_addresses),
        "--start-timeout",
        str(constants.HOROVOD_STARTUP_TIMEOUT_SECONDS),
        "--gloo-timeout-seconds",
        str(constants.HOROVOD_GLOO_TIMEOUT_SECONDS),
    ]
    if inter_node_network_interface is not None and num_machines > 1:
        horovod_process_cmd.extend(["--network-interface", inter_node_network_interface])
    horovod_process_cmd.extend(create_performance_args(optimizations))
    if debug:
        horovod_process_cmd.append("--verbose")
    horovod_process_cmd.extend(optional_args)
    horovod_process_cmd.append("--")
    return horovod_process_cmd
