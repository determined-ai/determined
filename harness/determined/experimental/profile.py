"""
Useful tools for profiling.  This will hopefully be the basis of a public-facing profiling toolkit
for model development, but right now it is undocumented and only used internally.

.. warning::
   The code in this module should be considered totally unstable and may change or be removed at
   any time.
"""

import contextlib
import threading
import time
from typing import Any, Callable, Dict, Generator

import psutil


class ProfilingThread(threading.Thread):
    """
    Call a profiling function periodically on a separate thread (asynchronous collection).

    The _ProfilingThread also supports synchronous triggering, which is useful for collecting
    precise metrics around workload transitions.

        .. code-block:: python
           prof = Profiler()
           with ProfilingThread(5.0, lambda: logging.info(json.dumps(prof.metrics()))) as thread:
               ...
               # Profile before some important boundary.
               with thread.pause():
                   thread.trigger()
               ...
    """

    def __init__(self, period: float, profile_fn: Callable) -> None:
        self.period = period
        self.profile_fn = profile_fn
        self.quitting = False
        self.cond = threading.Condition()
        super().__init__()

    def run(self) -> None:
        """Asynchronously call a profiling function to collect metrics in a thread."""
        with self.cond:
            while True:
                self.cond.wait(timeout=self.period)
                if self.quitting:
                    break
                self.profile_fn()

    def __enter__(self) -> None:
        self.start()

    def __exit__(self, *arg: Any) -> None:
        self.quit()

    @contextlib.contextmanager
    def pause(self) -> Generator:
        """Provide a context during which the thread will not trigger asynchronously."""
        self.cond.acquire()
        try:
            yield
        finally:
            self.cond.release()

    def trigger(self) -> None:
        """
        Manually call profile_fn(). Only safe to call inside the context provided by self.pause().
        """
        self.profile_fn()

    def quit(self) -> None:
        self.quitting = True
        with self.cond:
            self.cond.notify_all()
        self.join()
        # Trigger one final profile.
        self.profile_fn()


class Profiler:
    """
    Stateful metric collection. It is nice to have as a separate object from the ProfilingLayer for
    flexibility in local training situations.
    """

    def __init__(self, system_level_metrics: bool, process_level_metrics: bool) -> None:
        self.system_level_metrics = system_level_metrics
        self.process_level_metrics = process_level_metrics

        self.last_time = time.time()

        # Bootstrap some stateful profiling calls.
        if self.system_level_metrics:
            _ = psutil.cpu_percent(percpu=True)

            d = psutil.disk_io_counters()
            self.sys_disk_read_bytes = d.read_bytes
            self.sys_disk_write_bytes = d.write_bytes

            n = psutil.net_io_counters()
            self.sys_net_bytes_recv = n.bytes_recv
            self.sys_net_bytes_sent = n.bytes_sent

        if self.process_level_metrics:
            self.process = psutil.Process()
            self.process.cpu_percent()

    def metrics(self) -> Dict:
        now = time.time()
        interval = now - self.last_time
        self.last_time = now

        metrics = {
            "time": now,
            "interval": interval,
        }

        if self.system_level_metrics:
            percpu = psutil.cpu_percent(percpu=True)
            metrics["sys_percpu_percent"] = percpu
            metrics["sys_cpu_percent"] = sum(percpu) / len(percpu) if percpu else -1.0

            d = psutil.disk_io_counters()
            metrics["sys_disk_read_bytes"] = d.read_bytes - self.sys_disk_read_bytes
            metrics["sys_disk_write_bytes"] = d.write_bytes - self.sys_disk_write_bytes
            self.sys_disk_read_bytes = d.read_bytes
            self.sys_disk_write_bytes = d.write_bytes

            n = psutil.net_io_counters()
            metrics["sys_net_bytes_recv"] = n.bytes_recv - self.sys_net_bytes_recv
            metrics["sys_net_bytes_sent"] = n.bytes_sent - self.sys_net_bytes_sent
            self.sys_net_bytes_recv = n.bytes_recv
            self.sys_net_bytes_sent = n.bytes_sent

            m = psutil.virtual_memory()
            metrics["sys_mem_percent"] = m.percent

        if self.process_level_metrics:
            with self.process.oneshot():
                metrics["proc_cpu_percent"] = self.process.cpu_percent()
                metrics["proc_mem_percent"] = self.process.memory_percent()

        return metrics
