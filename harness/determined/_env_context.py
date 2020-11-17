import logging
from typing import Any, Dict, List, Optional, Tuple, cast

import determined as det
from determined import constants, workload
from determined_common import check, types


class EnvContext:
    def __init__(
        self,
        master_addr: str,
        master_port: int,
        use_tls: bool,
        master_cert_file: Optional[str],
        master_cert_name: Optional[str],
        container_id: str,
        experiment_config: Dict[str, Any],
        hparams: Dict[str, Any],
        initial_workload: workload.Workload,
        latest_checkpoint: Optional[Dict[str, Any]],
        use_gpu: bool,
        container_gpus: List[str],
        slot_ids: List[int],
        debug: bool,
        workload_manager_type: str,
        det_rendezvous_ports: str,
        det_trial_unique_port_offset: int,
        det_trial_runner_network_interface: str,
        det_trial_id: str,
        det_experiment_id: str,
        det_cluster_id: str,
        trial_seed: int,
        managed_training: bool = True,
    ):
        self.master_addr = master_addr
        self.master_port = master_port
        self.use_tls = use_tls
        self.master_cert_file = master_cert_file
        self.master_cert_name = master_cert_name
        self.container_id = container_id
        self.experiment_config = det.ExperimentConfig(experiment_config)
        self.hparams = hparams
        self.initial_workload = initial_workload
        self.latest_checkpoint = latest_checkpoint
        self.use_gpu = use_gpu
        self.container_gpus = container_gpus
        self.slot_ids = slot_ids
        self.debug = debug
        self.workload_manager_type = workload_manager_type
        self.det_rendezvous_ports = det_rendezvous_ports
        self.det_trial_unique_port_offset = det_trial_unique_port_offset
        self.det_trial_runner_network_interface = det_trial_runner_network_interface
        self.det_trial_id = det_trial_id
        self.det_experiment_id = det_experiment_id
        self.det_cluster_id = det_cluster_id
        self.trial_seed = trial_seed
        self.managed_training = managed_training

        self._per_slot_batch_size, self._global_batch_size = self._calculate_batch_sizes()

    def first_step(self) -> types.StepID:
        return self.initial_workload.step_id

    def rendezvous_ports(self) -> Tuple[int, int]:
        ports = [int(x) for x in self.det_rendezvous_ports.split(",")]
        if len(ports) != 2:
            logging.warning("DET_RENDEZVOUS_PORTS not set, falling back on LOCAL_RENDEZVOUS_PORTS")
            ports = [constants.LOCAL_RENDEZVOUS_PORT, constants.LOCAL_RENDEZVOUS_PORT + 1]
        return ports[0], ports[1]

    def _calculate_batch_sizes(self) -> Tuple[int, int]:
        if "global_batch_size" not in self.hparams.keys():
            raise AssertionError(
                "Please specify `global_batch_size` under `hyperparameters` "
                "in experiment config."
            )

        if "batch_size" in self.hparams.keys():
            logging.warning(
                "Use `global_batch_size` not `batch_size` under `hyperparameters` "
                "in experiment config."
            )

        global_batch_size = self.hparams["global_batch_size"]
        check.is_instance(global_batch_size, int, "`global_batch_size` hparam must be an int.")
        global_batch_size = cast(int, global_batch_size)

        if self.experiment_config.native_parallel_enabled():
            return global_batch_size, global_batch_size

        # Configure batch sizes.
        slots_per_trial = self.experiment_config.slots_per_trial()
        if global_batch_size < slots_per_trial:
            raise AssertionError(
                "Please set the `global_batch_size` hyperparameter to be greater or equal to the "
                f"number of slots. Current batch_size: {global_batch_size}, slots_per_trial: "
                f"{slots_per_trial}."
            )

        per_gpu_batch_size = global_batch_size // slots_per_trial
        effective_batch_size = per_gpu_batch_size * slots_per_trial
        if effective_batch_size != global_batch_size:
            logging.warning(
                f"`global_batch_size` changed from {global_batch_size} to {effective_batch_size} "
                f"to divide equally across {slots_per_trial} slots."
            )

        return per_gpu_batch_size, effective_batch_size

    @property
    def per_slot_batch_size(self) -> int:
        return self._per_slot_batch_size

    @property
    def global_batch_size(self) -> int:
        return self._global_batch_size
