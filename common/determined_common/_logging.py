import logging
import os
import sys
from typing import Any, Dict, Optional, Union


class _Loggers:
    def __init__(self) -> None:
        self._storage = logging.getLogger("det.storage")
        self._harness = logging.getLogger("det.harness")
        self._resources = logging.getLogger("det.resources")
        self._all_loggers = (self._storage, self._harness, self._resources)

    @property
    def storage(self) -> logging.Logger:
        return self._storage

    @property
    def harness(self) -> logging.Logger:
        return self._harness

    @property
    def resources(self) -> logging.Logger:
        return self._resources


# Loggers are normally global in python, and it would be a bit unexpected to behave differently.
log = _Loggers()


def read_level_from_config(level: Optional[str], default: int, limit: int = 0) -> int:
    """
    Read a level from the config using the built-in logging._checkLevel behavior.

    Args:
        level: The string specified in the config.
        default: The default value if level is None.
        limit: The highest log level allowable given additional factors (like worker rank).
    """
    return max(logging._checkLevel(level if level is not None else default), limit)  # type: ignore


class DebugConfig:
    def __init__(
        self,
        root_log_level: Optional[str] = None,
        storage_log_level: Optional[str] = None,
        harness_log_level: Optional[str] = None,
        debug_all_workers: bool = False,
        horovod_verbose: bool = False,
        nccl_debug: Optional[str] = None,
        nccl_debug_subsys: Optional[str] = None,
        resource_profile_period_sec: float = 0.0,
        stack_trace_period_sec: float = 0.0,
    ):
        self._root_log_level = root_log_level
        self._debug_all_workers = debug_all_workers
        self._horovod_verbose = horovod_verbose
        self._storage_log_level = storage_log_level
        self._harness_log_level = harness_log_level
        self._nccl_debug = nccl_debug
        self._nccl_debug_subsys = nccl_debug_subsys
        self._resource_profile_period_sec = resource_profile_period_sec
        self._stack_trace_period_sec = stack_trace_period_sec

        # The current workload, tracked here so we can emit it as part of the profiling output.
        self._workload = ""

    @classmethod
    def from_config(cls, config: Union[None, bool, Dict[str, Any]]) -> "DebugConfig":
        if config is None or config is False:
            return cls()

        if config is True:
            return cls(
                root_log_level="DEBUG",
                storage_log_level="INFO",
                harness_log_level="DEBUG",
                debug_all_workers=True,
                horovod_verbose=True,
                nccl_debug="INFO",
                nccl_debug_subsys=None,
                resource_profile_period_sec=10.0,
                stack_trace_period_sec=0.0,
            )

        assert isinstance(config, dict)
        return cls(**config)

    @classmethod
    def from_environ(cls, environ: Optional[Dict[str, str]] = None) -> "DebugConfig":
        if environ is None:
            environ = dict(os.environ)
        if environ.get("DET_DEBUG"):
            # If DET_DEBUG is set to a non-empty string, behave like `debug: true` in the config.
            return cls.from_config(True)
        return cls()

    def set_workload(self, workload: str) -> None:
        """Update the workload information to associate with debug messages."""
        self._workload = workload

    @property
    def stack_trace_period_sec(self) -> float:
        return self._stack_trace_period_sec

    @property
    def resource_profile_period_sec(self) -> float:
        return self._resource_profile_period_sec

    @property
    def nccl_debug(self) -> Optional[str]:
        return self._nccl_debug

    @property
    def nccl_debug_subsys(self) -> Optional[str]:
        return self._nccl_debug_subsys

    @property
    def horovod_verbose(self) -> bool:
        return self._horovod_verbose

    @property
    def debug_all_workers(self) -> bool:
        return self._debug_all_workers

    def set_loggers(self, is_chief: bool = True, handler: Optional[logging.Handler] = None) -> None:
        if not handler:
            handler = logging.StreamHandler(sys.stdout)
        formatter = logging.Formatter("%(levelname)s:%(name)s: %(message)s")
        handler.setFormatter(formatter)

        # Configure the root log handler for external libraries.
        root = logging.getLogger()
        root.setLevel(read_level_from_config(self._root_log_level, logging.WARNING))
        # Delete existing handlers before adding new ones.
        for h in root.handlers:
            root.removeHandler(h)
        root.addHandler(handler)

        # Configure the det.log handlers according to individual log levels and worker rank.
        default = logging.INFO
        limit = 0 if is_chief or self._debug_all_workers else logging.WARNING
        log.storage.setLevel(read_level_from_config(self._storage_log_level, default, limit))
        log.harness.setLevel(read_level_from_config(self._harness_log_level, default, limit))
        log.resources.setLevel(logging.DEBUG)
