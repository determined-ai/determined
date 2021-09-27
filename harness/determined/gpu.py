import csv
import logging
import subprocess
from typing import List, NamedTuple

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
        logging.info("detected 0 gpus (nvidia-smi not found)")
        # This case is expected if NVIDIA drivers are not available.
        return []
    except Exception as e:
        logging.warning(f"detected 0 gpus (error with nvidia-smi: {e})")
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

    logging.info(f"detected {len(gpus)} gpus")
    return gpus


def get_gpu_uuids() -> List[str]:
    return [gpu.uuid for gpu in sorted(get_gpus(), key=lambda gpu: gpu.id)]
