import json
from typing import Any

from determined import log, workload
from determined.experimental import profile


class ProfilingLayer(workload.Source):
    """
    ProfilingLayer is a passthru layer in the harness (it does not modify workloads at all) but
    it reads the state of the system by watching in order to log more useful information while it
    profiles the system. It is capable of profiling system-level metrics and process-level metrics.

    In non-distributed training, the main process should track both. Not counting data worker
    processes, that should effectively profile most of the training task.

    In distributed training, the main process should still track system-level and process-level
    metrics (to capture behaviors like the StorageManager), but the horovod workers will each
    track their own metrics.

    Presently, the metrics are emitted as part of the trial logs because that is the no-effort way
    to deliver metrics from all workers to an existing storage mechanism; it takes little
    development effort and zero user administration effort.
    """

    def __init__(
        self,
        workloads: workload.Stream,
        period: float,
        initial_workload_state: str,
        machine_rank: int,
        worker_rank: int,
        system_level_metrics: bool,
        process_level_metrics: bool,
    ) -> None:
        self.workloads = workloads
        self.period = period
        self.workload_state = initial_workload_state
        self.machine_rank = machine_rank
        self.worker_rank = worker_rank
        self.system_level_metrics = system_level_metrics
        self.process_level_metrics = process_level_metrics

        self.enabled = self.period > 0
        if not self.enabled:
            return

        self.profiler = profile.Profiler(system_level_metrics, process_level_metrics)

        # Start profiling immediately, to capture behavior of startup costs (importing tensorflow,
        # for instance).
        self.thread = profile.ProfilingThread(self.period, self.profile)
        self.thread.start()

    def __enter__(self) -> "ProfilingLayer":
        return self

    def __exit__(self, *arg: Any) -> None:
        self.close()

    def close(self) -> None:
        if self.enabled:
            self.thread.quit()
            self.profile()

    def profile(self) -> None:
        metrics = self.profiler.metrics()
        metrics.update(
            {
                "workload": self.workload_state,
                "worker_rank": self.worker_rank,
                "machine_rank": self.machine_rank,
            }
        )

        log.resources.info(json.dumps(metrics))

    def __iter__(self) -> workload.Stream:
        if not self.enabled:
            yield from self.workloads
            return

        first_workload = True
        for w, arg, respond in self.workloads:
            if first_workload:
                # Skip the first workload, which we started profiling right when we started.
                first_workload = False
            else:
                # Every incoming workload triggers a synchronous profiling event.  We trigger on
                # the incoming new events rather than the outgoing old metrics because in general
                # the pre-work that we execute is much lighter than the post-work (checkpoint or
                # tensorboard uploads, for instance).  As a result, the profiler output is less
                # sensitive to in which layer it is placed when we trigger on incoming new events.
                with self.thread.pause():
                    self.profile()
                    self.workload_state = str(w.kind)

            yield w, arg, respond
