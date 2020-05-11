import threading
import time
from typing import Any, Dict, List, Tuple

import matplotlib.pyplot as plt
import psutil
import simplejson

import determined.gpu

MeasurementHistory = List[Tuple[float, Any]]


class Measurement(object):
    """
    Tracks the history of a scalar measurement.
    """

    def __init__(self, display_name: str, multiplier: float = 1.0) -> None:
        self._display_name = display_name
        self._multiplier = multiplier
        self._history = []  # type: MeasurementHistory

    def add_measurement(self, measurement: Any) -> None:
        pt = (time.time(), measurement * self._multiplier)
        self._history.append(pt)

    def history(self) -> MeasurementHistory:
        return self._history

    def display_name(self) -> str:
        return self._display_name


class ThroughputMeasurement(Measurement):
    """
    Tracks the history of a scalar measurement per second.
    """

    def __init__(
        self, display_name: str, initial_measurement: float, multiplier: float = 1.0
    ) -> None:
        super().__init__(display_name, multiplier)
        self._prev_measurement = initial_measurement
        self._prev_time = time.time()

    def add_measurement(self, measurement: float) -> None:
        now = time.time()
        super().add_measurement((measurement - self._prev_measurement) / (now - self._prev_time))

        self._prev_measurement = measurement
        self._prev_time = now


class HarnessProfiler(object):
    """Monitors utilization of the process in a seperate thread."""

    def __init__(self, interval: float = 0.1, use_gpu: bool = False) -> None:
        self._use_gpu = use_gpu
        self._interval = interval
        self._stop_signal = threading.Event()
        self._process = psutil.Process()
        self._monitor_thread = threading.Thread(
            target=self._monitor, name="DeterminedProfileMonitor"
        )

    def _initialize_measurements(self) -> None:
        self._cpu_percent = Measurement("CPU Utilization (%)")
        self._memory_utilization = Measurement("Physical Memory (KB)", multiplier=1.0 / 1000.0)

        disk_stats = psutil.disk_io_counters()
        self._disk_read = ThroughputMeasurement(
            "Disk Read (KB/s)", disk_stats.read_bytes, multiplier=1.0 / 1000.0
        )
        self._disk_write = ThroughputMeasurement(
            "Disk Write (KB/s)", disk_stats.write_bytes, multiplier=1.0 / 1000.0
        )

        net_stats = psutil.net_io_counters()
        self._net_read = ThroughputMeasurement(
            "Network Read (KB/s)", net_stats.bytes_recv, multiplier=1.0 / 1000.0
        )
        self._net_write = ThroughputMeasurement(
            "Network Write (KB/s)", net_stats.bytes_sent, multiplier=1.0 / 1000.0
        )

        process_io_stats = self._process.io_counters()
        self._process_read = ThroughputMeasurement(
            "Process Read (KB/s)", process_io_stats.read_bytes, multiplier=1.0 / 1000.0
        )
        self._process_read_chars = ThroughputMeasurement(
            "Process Read (char/s)", process_io_stats.read_chars
        )
        self._process_write = ThroughputMeasurement(
            "Process Write (KB/s)", process_io_stats.write_bytes, multiplier=1.0 / 1000.0
        )
        self._process_write_chars = ThroughputMeasurement(
            "Process Write (char/s)", process_io_stats.write_chars
        )

        if self._use_gpu:
            gpu_list = determined.gpu.get_gpus()
            self._gpu_loads = {g.id: Measurement("GPU {} Load (%)".format(g.id)) for g in gpu_list}
            self._gpu_utilizations = {
                g.id: Measurement("GPU {} Memory Utilization (%)".format(g.id)) for g in gpu_list
            }

    def _monitor(self) -> None:
        self._initialize_measurements()
        while not self._stop_signal.is_set():
            time.sleep(self._interval)

            self._cpu_percent.add_measurement(self._process.cpu_percent())
            self._memory_utilization.add_measurement(self._process.memory_info().rss)

            disk_stats = psutil.disk_io_counters()
            self._disk_read.add_measurement(disk_stats.read_bytes)
            self._disk_write.add_measurement(disk_stats.write_bytes)

            net_stats = psutil.net_io_counters()
            self._net_read.add_measurement(net_stats.bytes_recv)
            self._net_write.add_measurement(net_stats.bytes_sent)

            process_io_stats = self._process.io_counters()
            self._process_read.add_measurement(process_io_stats.read_bytes)
            self._process_write.add_measurement(process_io_stats.write_bytes)
            self._process_read_chars.add_measurement(process_io_stats.read_chars)
            self._process_write_chars.add_measurement(process_io_stats.write_chars)

            if self._use_gpu:
                for g in determined.gpu.get_gpus():
                    self._gpu_loads[g.id].add_measurement(g.load)
                    self._gpu_utilizations[g.id].add_measurement(g.memoryUtil)

    def start(self) -> None:
        self._monitor_thread.start()

    def stop(self) -> None:
        self._stop_signal.set()
        self._monitor_thread.join()

    def results(self) -> Dict[str, MeasurementHistory]:
        measurements = [
            self._cpu_percent,
            self._memory_utilization,
            self._disk_read,
            self._disk_write,
            self._net_read,
            self._net_write,
            self._process_read,
            self._process_write,
            self._process_read_chars,
            self._process_write_chars,
        ]

        if self._use_gpu:
            measurements.extend(self._gpu_loads.values())
            measurements.extend(self._gpu_utilizations.values())

        return {m.display_name(): m.history() for m in measurements}

    def serialize_raw_results(self, path: str) -> None:
        with open(path, "w") as f:
            simplejson.dump(self.results(), f)

    def serialize_graph(self, path: str, figsize: Tuple[int, int] = (20, 40)) -> None:
        results = self.results()

        plt.figure(figsize=figsize)
        for idx, (name, history) in enumerate(results.items()):
            if not history:
                continue

            times, values = zip(*history)
            plt.subplot(len(results), 1, idx + 1)
            plt.plot(times, values)
            plt.title(name)

        # Increase the vertical space between subplots (default is 0.2)
        # before serializing to disk.
        plt.subplots_adjust(hspace=0.4)
        plt.savefig(path)
