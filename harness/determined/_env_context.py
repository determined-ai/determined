import logging
from typing import Any, Dict, List, Optional, Tuple, cast

import determined as det
from determined.common import check


class EnvContext:
    def __init__(
        self,
        master_url: str,
        master_cert_file: Optional[str],
        master_cert_name: Optional[str],
        container_id: str,
        experiment_config: Dict[str, Any],
        hparams: Dict[str, Any],
        latest_checkpoint: Optional[str],
        latest_batch: int,
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
        self.container_id = container_id
        self.experiment_config = det.ExperimentConfig(experiment_config)
        self.hparams = hparams
        self.latest_checkpoint = latest_checkpoint
        self.latest_batch = latest_batch
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

        self._per_slot_batch_size, self._global_batch_size = self._calculate_batch_sizes()

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
