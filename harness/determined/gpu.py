import sys
from typing import List, Optional, Tuple

from determined_common import check


def get_gpu_ids_and_uuids() -> Tuple[List[str], List[str]]:
    try:
        import GPUtil
    except ModuleNotFoundError:
        return [], []

    gpus = GPUtil.getGPUs()
    return [gpu.id for gpu in gpus], [gpu.uuid for gpu in gpus]


def get_gpu_uuids_and_validate(use_gpu: bool, slot_ids: Optional[List[str]] = None) -> List[str]:
    if use_gpu:
        # Sanity check: if this trial is expected to run on the GPU but
        # no GPUs are available, this indicates a misconfiguration.
        _, gpu_uuids = get_gpu_ids_and_uuids()
        if len(gpu_uuids) == 0:
            sys.exit("Failed to find GPUs for GPU-only trial")

        if slot_ids is not None:
            check.equal_lengths(slot_ids, gpu_uuids, "Mismatched slot_ids and container_gpus.")
        return gpu_uuids
    return []
