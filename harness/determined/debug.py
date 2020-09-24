import contextlib
import faulthandler
import threading
import time
from typing import Any, Callable, Dict, Generator

import psutil


@contextlib.contextmanager
def stack_trace_thread(stack_trace_period_sec: float) -> Generator:
    """If enabled, emit stack traces periodically."""
    if stack_trace_period_sec <= 0.0:
        yield
        return

    faulthandler.dump_traceback_later(stack_trace_period_sec, repeat=True)
    try:
        yield
    finally:
        faulthandler.cancel_dump_traceback_later()


class ProfilingThread(threading.Thread):
    """
    Call a profiling function periodically on a separate thread (asynchronous collection).

    The _ProfilingThread also supports synchronous triggering, which is useful for collecting
    precise metrics around workload transitions.
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
        """Guarantee that the thread will not trigger asynchronously."""
        self.cond.acquire()
        try:
            yield
        finally:
            self.cond.release()

    def trigger(self) -> None:
        """
        Allow the profile_fn() to be triggered externally.  Only safe to call inside the context
        manager provided by this self object.
        """
        self.profile_fn()

    def quit(self) -> None:
        self.quitting = True
        with self.cond:
            self.cond.notify_all()


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
            metrics["sys_cpu_percent"] = psutil.cpu_percent(percpu=True)

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
