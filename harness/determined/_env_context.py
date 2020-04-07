import logging
from typing import Any, Dict, List, Optional, Tuple

import determined as det
from determined import constants, workload
from determined_common import types


class EnvContext:
    def __init__(
        self,
        master_addr: str,
        master_port: int,
        container_id: str,
        experiment_config: Dict[str, Any],
        hparams: Dict[str, Any],
        initial_workload: workload.Workload,
        latest_checkpoint: Optional[Dict[str, Any]],
        use_gpu: bool,
        container_gpus: List[str],
        slot_ids: List[str],
        debug: bool,
        workload_manager_type: str,
        det_rendezvous_ports: str,
        det_trial_runner_network_interface: str,
        det_trial_id: str,
        det_experiment_id: str,
        det_cluster_id: str,
        trial_seed: int,
    ):
        self.master_addr = master_addr
        self.master_port = master_port
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
        self.det_trial_runner_network_interface = det_trial_runner_network_interface
        self.det_trial_id = det_trial_id
        self.det_experiment_id = det_experiment_id
        self.det_cluster_id = det_cluster_id
        self.trial_seed = trial_seed

    def first_step(self) -> types.StepID:
        return self.initial_workload.step_id

    def rendezvous_ports(self) -> Tuple[int, int]:
        ports = [int(x) for x in self.det_rendezvous_ports.split(",")]
        if len(ports) != 2:
            logging.warning("DET_RENDEZVOUS_PORTS not set, falling back on LOCAL_RENDEZVOUS_PORTS")
            ports = [constants.LOCAL_RENDEZVOUS_PORT, constants.LOCAL_RENDEZVOUS_PORT + 1]
        return ports[0], ports[1]
