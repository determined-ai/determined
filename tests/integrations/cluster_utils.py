import functools
import os
import subprocess
from typing import Any, Callable, Dict, List

import pytest

CONTAINER_ETC_ROOT = "/etc/determined"


def get_num_gpus() -> int:
    try:
        cmd = ["nvidia-smi", "--query-gpu=index,name,uuid", "--format=csv,noheader"]
        output = subprocess.check_output(cmd, encoding="utf-8").rstrip()
        gpu_info = [line.split(", ") for line in output.split(os.linesep)]
        return len(gpu_info)
    except (OSError, subprocess.CalledProcessError):
        return 0


def skip_if_not_enough_gpus(num_gpus: int) -> None:
    available_gpus = get_num_gpus()
    if available_gpus < num_gpus:
        pytest.skip(
            "Not enough GPUs, requested {} but there are only {} available".format(
                num_gpus, available_gpus
            )
        )


def skip_test_if_not_enough_gpus(num_gpus: int) -> Callable:
    def decorator(func: Callable) -> Callable:
        @functools.wraps(func)
        def wrapper(*args: List, **kwargs: Dict) -> Any:
            skip_if_not_enough_gpus(num_gpus)
            return func(*args, **kwargs)

        return wrapper

    return decorator
