from typing import Any, Dict, List, Optional

import determined as det


class EnvContext:
    def __init__(
        self,
        master_url: str,
        master_cert_file: Optional[str],
        master_cert_name: Optional[str],
        experiment_config: Dict[str, Any],
        hparams: Dict[str, Any],
        latest_checkpoint: Optional[str],
        steps_completed: int,
        use_gpu: bool,
        container_gpus: List[str],
        slot_ids: List[int],
        debug: bool,
        det_trial_unique_port_offset: int,
        det_trial_id: str,
        det_experiment_id: str,
        det_agent_id: str,
        det_cluster_id: str,
        trial_seed: int,
        trial_run_id: int,
        allocation_id: str,
        managed_training: bool,
        test_mode: bool,
        on_cluster: bool,
    ):
        self.master_url = master_url
        self.master_cert_file = master_cert_file
        self.master_cert_name = master_cert_name
        self.experiment_config = det.ExperimentConfig(experiment_config)
        self.hparams = hparams
        self.latest_checkpoint = latest_checkpoint
        self.steps_completed = steps_completed
        self.use_gpu = use_gpu
        self.container_gpus = container_gpus
        self.slot_ids = slot_ids
        self.debug = debug
        self.det_trial_unique_port_offset = det_trial_unique_port_offset
        self.det_trial_id = det_trial_id
        self.det_experiment_id = det_experiment_id
        self.det_agent_id = det_agent_id
        self.det_cluster_id = det_cluster_id
        self.trial_seed = trial_seed
        self.trial_run_id = trial_run_id
        self.allocation_id = allocation_id
        self.managed_training = managed_training
        self.test_mode = test_mode
        self.on_cluster = on_cluster
