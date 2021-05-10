import csv
import logging
import subprocess
from typing import List, NamedTuple, Optional, Tuple

import determined as det
from determined.common import check

gpu_fields = [
    "index",
    "uuid",
    "utilization.gpu",
    "memory.used",
    "memory.total",
]


class GPU(NamedTuple):
    id: int
    uuid: str
    load: float
    memoryUtil: float


warned_fields = set()


def float_or_default(fields: dict, key: str, default: float) -> float:
    try:
        return float(fields[key])
    except ValueError:
        if key not in warned_fields:
            warned_fields.add(key)
            logging.warning(f"Unable to get {key} from nvidia-smi")
        return default


def get_gpus() -> List[GPU]:
    try:
        proc = subprocess.Popen(
            ["nvidia-smi", "--query-gpu=" + ",".join(gpu_fields), "--format=csv,noheader,nounits"],
            stdout=subprocess.PIPE,
            universal_newlines=True,
        )
    except FileNotFoundError:
        # This case is expected if NVIDIA drivers are not available.
        return []
    except Exception as e:
        logging.warning(f"Couldn't query GPUs with `nvidia-smi`; assuming there are none: {e}")
        return []

    gpus = []
    with proc:
        for field_list in csv.reader(proc.stdout):  # type: ignore
            if len(field_list) != len(gpu_fields):
                logging.warning(f"Ignoring unexpected nvidia-smi output: {field_list}")
                continue
            fields = dict(zip(gpu_fields, field_list))
            try:
                gpus.append(
                    GPU(
                        id=int(fields["index"]),
                        uuid=fields["uuid"].strip(),
                        load=float_or_default(fields, "utilization.gpu", 0.0) / 100,
                        memoryUtil=float_or_default(fields, "memory.used", 0.0)
                        / float_or_default(fields, "memory.total", 1.0),
                    )
                )
            except ValueError:
                logging.warning(f"Ignoring GPU with unexpected nvidia-smi output: {fields}")
    if proc.returncode:
        logging.warning(f"`nvidia-smi` exited with failure status code {proc.returncode}")
    return gpus


def get_gpu_ids_and_uuids() -> Tuple[List[int], List[str]]:
    gpus = get_gpus()
    return [gpu.id for gpu in gpus], [gpu.uuid for gpu in gpus]


def get_gpu_uuids_and_validate(use_gpu: bool, slot_ids: Optional[List[str]] = None) -> List[str]:
    if use_gpu:
        # Sanity check: if this trial is expected to run on the GPU but
        # no GPUs are available, this indicates a misconfiguration.
        _, gpu_uuids = get_gpu_ids_and_uuids()
        if not gpu_uuids:
            raise det.errors.InternalException("Failed to find GPUs for GPU-only trial")

        if slot_ids is not None:
            check.equal_lengths(slot_ids, gpu_uuids, "Mismatched slot_ids and container_gpus.")
        return gpu_uuids
    return []
