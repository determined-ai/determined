import csv
import json
import logging
import subprocess
from typing import List, NamedTuple, Tuple

logger = logging.getLogger("determined")

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
            logger.warning(f"Unable to get {key} from nvidia-smi")
        return default


def _get_nvidia_gpus() -> List[GPU]:
    try:
        proc = subprocess.Popen(
            ["nvidia-smi", "--query-gpu=" + ",".join(gpu_fields), "--format=csv,noheader,nounits"],
            stdout=subprocess.PIPE,
            universal_newlines=True,
        )
    except FileNotFoundError:
        logger.info("detected 0 gpus (nvidia-smi not found)")
        # This case is expected if NVIDIA drivers are not available.
        return []
    except Exception as e:
        logger.warning(f"detected 0 gpus (error with nvidia-smi: {e})")
        return []

    gpus = []
    with proc:
        for field_list in csv.reader(proc.stdout):  # type: ignore
            if len(field_list) != len(gpu_fields):
                logger.warning(f"Ignoring unexpected nvidia-smi output: {field_list}")
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
                logger.warning(f"Ignoring GPU with unexpected nvidia-smi output: {fields}")
    if proc.returncode:
        logger.warning(f"`nvidia-smi` exited with failure status code {proc.returncode}")

    logger.info(f"detected {len(gpus)} gpus")
    return gpus


def _get_rocm_gpus() -> List[GPU]:
    try:
        output_json = subprocess.run(
            [
                "rocm-smi",
                "--showid",
                "--showuniqueid",
                "--json",
            ],
            check=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        ).stdout

    except FileNotFoundError:
        logger.info("rocm-smi not found")
        return []
    except Exception as e:
        logger.warning(f"rocm-smi error: {e}")
        return []

    try:
        output = json.loads(output_json)
    except Exception as e:
        logger.warning(f"failed to parse rocm-smi json output: {e}, content: {str(output_json)}")
        return []

    gpus = []
    for k, v in output.items():
        gpus.append(GPU(id=int(k[len("card") :]), uuid=v["Unique ID"], load=0, memoryUtil=0))

    logger.info(f"detected {len(gpus)} rocm gpus")
    return gpus


def get_gpus() -> Tuple[List[GPU], str]:
    result = _get_nvidia_gpus()
    if result:
        return result, "cuda"
    result = _get_rocm_gpus()
    if result:
        return result, "rocm"
    else:
        return [], ""


def get_gpu_uuids() -> List[str]:
    gpus, _ = get_gpus()
    if gpus:
        return [gpu.uuid for gpu in sorted(gpus, key=lambda gpu: gpu.id)]
    else:
        return []


class GPUProcess(NamedTuple):
    pid: int
    process_name: str
    gpu_uuid: str
    used_memory: str  # This is a string that includes units, e.g. "123 MiB"


def _get_nvidia_processes() -> List[GPUProcess]:
    try:
        proc = subprocess.Popen(
            [
                "nvidia-smi",
                "--query-compute-apps=" + ",".join(GPUProcess._fields),
                "--format=csv,noheader",
            ],
            stdout=subprocess.PIPE,
            universal_newlines=True,
        )
    except FileNotFoundError:
        logger.info("detected 0 gpu processes (nvidia-smi not found)")
        # This case is expected if NVIDIA drivers are not available.
        return []
    except Exception as e:
        logger.warning(f"detected 0 gpu processes (error with nvidia-smi: {e})")
        return []

    processes = []
    with proc:
        for field_list in csv.reader(proc.stdout):  # type: ignore
            if len(field_list) != len(GPUProcess._fields):
                logger.warning(f"Ignoring unexpected nvidia-smi output: {field_list}")
                continue
            fields = dict(zip(GPUProcess._fields, field_list))
            try:
                processes.append(
                    GPUProcess(
                        pid=int(fields["pid"]),
                        process_name=fields["process_name"].strip(),
                        gpu_uuid=fields["gpu_uuid"].strip(),
                        used_memory=fields["used_memory"].strip(),
                    )
                )
            except ValueError:
                logger.warning(f"Ignoring GPU process with unexpected nvidia-smi output: {fields}")
    if proc.returncode:
        logger.warning(f"nvidia-smi exited with failure status code {proc.returncode}")
    return processes


def get_gpu_processes() -> List[GPUProcess]:
    # TODO This extra layer of method calls is to match get_gpus above, in case we are later
    # interested in ROCm processes as well
    return _get_nvidia_processes()
