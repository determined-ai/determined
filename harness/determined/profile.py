import json
import os
import threading
import time
from typing import Any, Dict, Optional, TextIO


def _get_base_info(name: str, category: Optional[str] = None) -> Dict[str, Any]:
    return {
        "name": name,
        "pid": os.getpid(),
        "tid": threading.current_thread().ident,
        "cat": category,
    }


def time_usec() -> int:
    return int(round(1e6 * time.time()))


def _log_event(profiler_file: TextIO, **kwargs: Any) -> None:
    profiler_file.write(json.dumps(kwargs) + os.linesep)


def log_start(name: str, profiler_file: TextIO, **kwargs: Any) -> int:
    start_time = time_usec()
    # Log beginning event in the chrome://tracing format.
    _log_event(profiler_file, ph="B", ts=start_time, **_get_base_info(name), **kwargs)
    return start_time


def log_end(name: str, profiler_file: TextIO, start_time: int, **kwargs: Any) -> None:
    if type(start_time) is not int:
        raise AssertionError(
            "Start time must be an int representing microseconds since epoch, but got {}".format(
                start_time
            )
        )
    end_time = time_usec()
    # Log end event in the chrome://tracing format.
    _log_event(
        profiler_file,
        ph="E",
        ts=end_time,
        duration=end_time - start_time,
        **_get_base_info(name),
        **kwargs
    )
