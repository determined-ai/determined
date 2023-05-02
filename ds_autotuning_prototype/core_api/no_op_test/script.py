import json
import logging
import random
from typing import Any, Dict

import determined as det
from attrdict import AttrDict
from determined.pytorch import dsat
from determined.pytorch.dsat import _defaults

possible_paths = [_defaults.MODEL_INFO_PROFILING_PATH, _defaults.AUTOTUNING_RESULTS_PATH]


def main(
    core_context: det.core.Context,
    hparams: Dict[str, Any],
) -> None:
    hparams = AttrDict(hparams)
    # TODO: Remove hack for seeing actual used HPs after Web UI is fixed.
    logging.info(f"HPs seen by trial: {hparams}")
    # Hack for clashing 'type' key. Need to change config parsing behavior so that
    # user scripts don't need to inject helper functions like this.
    ds_config = dsat.get_ds_config_from_hparams(hparams)
    is_model_profile_info_run = ds_config.get("autotuning", {}).get("model_info_path") is not None

    # We will simulate periodic OOMs.
    should_oom = False
    if is_model_profile_info_run:
        path = _defaults.MODEL_INFO_PROFILING_PATH
        metrics = {
            "num_params": 1,
            "trainable_num_params": 1,
            "activation_mem_per_gpu": 1,
            "rank": 0,
        }
    else:
        should_oom = random.randint(0, 3) == 0
        path = _defaults.AUTOTUNING_RESULTS_PATH
        metrics = {"throughput": random.randint(1, 100)}
    if not should_oom:
        with open(path, "w") as f:
            json.dump(metrics, f)

    is_chief = core_context.distributed.rank == 0
    for op in core_context.searcher.operations():
        for steps_completed in range(1, op.length + 1):
            with dsat.dsat_reporting_context(core_context, op):
                if not is_model_profile_info_run and should_oom:
                    raise RuntimeError("CUDA out of memory.")
                else:
                    if steps_completed >= op.length:
                        exit()
                    elif is_chief:
                        core_context.train.report_validation_metrics(
                            steps_completed=steps_completed,
                            metrics={"steps_completed": steps_completed},
                        )
            if core_context.preempt.should_preempt():
                return


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)
    info = det.get_cluster_info()
    hparams = info.trial.hparams
    distributed = det.core.DistributedContext.from_torch_distributed()
    with det.core.init(distributed=distributed) as core_context:
        main(core_context, hparams)
