import contextlib
import enum
import json
import logging
import os
import pathlib
import uuid
from datetime import datetime, timezone
from typing import Any, Dict, Iterator, Optional, Tuple, Union

from determined import core, tensorboard
from determined.common import api, storage
from determined.common.api import bindings
from determined import profiler

logger = logging.getLogger("determined.core")


class ProfilerContext:
    """
    ``ProfilerContext`` gives access to Determined-profiler-related features of a Determined cluster.
    """

    def __init__(
        self,
        dist: core.DistributedContext,
        trial_id: str,
        agent_id: str,
        master_url: str,
        enabled: bool,
        begin_on_batch: int,
        sync_timings: bool,
        end_after_batch: Optional[int] = None,
    ) -> None:
        self.profiler = profiler.ProfilerAgent(
            trial_id=trial_id,
            agent_id=agent_id,
            master_url=master_url,
            profiling_is_enabled=enabled,
            global_rank=dist.rank,
            local_rank=dist.local_rank,
            begin_on_batch=begin_on_batch,
            end_after_batch=end_after_batch,
            sync_timings=sync_timings
        )

    def record_metric(self, metric_name: str, value: float) -> None:
        self.profiler.record_metric(metric_name, value)

    @contextlib.contextmanager
    def record_timing(
        self, metric_name: str, accumulate: bool = False, requires_sync: bool = True
    ) -> Iterator[None]:
        yield self.profiler.record_timing(metric_name, accumulate, requires_sync)

    def start(self) -> None:
        self.profiler.start()
        self.profiler.set_training(True)

    def end(self) -> None:
        self.profiler.end()
