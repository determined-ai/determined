import datetime
import abc
from contextlib import contextmanager
from typing import Any, Optional
import threading
import psutil
import pynvml
import os
import logging
import queue
import threading
import time
from typing import Optional, List, Dict, Any

import psutil
import pynvml

from timeit import default_timer
import determined as det
from determined.common import api
from determined.common.api import TrialProfilerMetricsBatch

SYSTEM_METRIC_TYPE_ENUM = "PROFILER_METRIC_TYPE_SYSTEM"
TIMING_METRIC_TYPE_ENUM = "PROFILER_METRIC_TYPE_TIMING"
LOG_NAMESPACE = "determined-profiler"


def build_profiler_agent(env: det.EnvContext) -> 'ProfilerAgent':
    if env.experiment_config.profiling_enabled() and get_rank_if_horovod_process_else_return_zero() == 0:
        begin_on_batch, end_after_batch = env.experiment_config.profiling_interval()
        return SystemProfilerAgent(
            env.det_trial_id,
            env.det_agent_id,
            env.master_url,
            begin_on_batch,
            end_after_batch
        )
    return NoopProfilerAgent()


class ProfilerAgent(metaclass=abc.ABCMeta):
    """
    Base agent when profiling is disabled.
    """
    @abc.abstractmethod
    def update_batch_idx(self, new_batch_idx: int):
        pass

    @abc.abstractmethod
    def record_timing(self, metric_name: str):
        pass


class NoopProfilerAgent(ProfilerAgent):
    """
    Base agent when profiling is disabled.
    """
    def __enter__(self) -> 'NoopProfilerAgent':
        return self

    def __exit__(self):
        pass

    def update_batch_idx(self, new_batch_idx: int):
        pass

    @contextmanager
    def record_timing(self, metric_name: str):
        pass


class SystemProfilerAgent(ProfilerAgent):
    """
    Agent that collects metrics and sends them to the master. It has:
    - a thread to send data to the Master API (sender_thread)
    - a thread to collect System Metrics and periodically flush to sender_thread (sys_metric_collector_thread)
    - [UNIMPLEMENTED] something to batch Timings and periodically flush to sender_thread (timings_batcher)

    The ProfilerAgent needs to be created at the beginning of training and it needs
    to be notified every time the batch_idx increases. When it is created, it launches
    the sender_thread and the sys_metric_collector_thread. They will be cleaned up
    either once end_after_batch is finished or after 5 minutes have passed.

    You can also ship Timings through the ProfilerAgent with the record_timing() method. This
    functionality has not yet been implemented - we need to batch the timings and periodically
    flush them to the sender_thread.

    Profiling is only active between start_on_batch and end_after_batch. It will also automatically
    shut down 5 minutes after starting. When profiling is not active, no system metrics are collected
    and the record_timing function is a no-op.

    Usage:
    ```
    profiler_agent = ProfilerAgent(self, trial_id, agent_id, master_url, start_on_batch, end_after_batch)

    for batch_idx, batch in enumerate(batches):
        profiler_agent.update_batch_idx(batch_idx)

        # NOTE: Timing API has not been fully developed yet
        forward_pass_timing = Timing("forward_pass")
        forward_pass_timing.start()
        # Do forward pass
        forward_pass_timing.end()
        profiler_agent.record_timing(forward_pass_timing)
    ```
    """
    def __init__(self, trial_id: int, agent_id: str, master_url: str, start_on_batch: int, end_after_batch: Optional[int] = None):
        self.killed = False
        self.current_batch_idx = 0
        self.agent_id = agent_id
        self.trial_id = trial_id
        self.master_url = master_url
        self.start_on_batch = start_on_batch
        self.end_after_batch = end_after_batch

        # Track duration to stop collecting after 5 minutes. start_time also serves as
        # the indicator that collection has begun.
        self.start_time = None
        # TODO: Currently we only check for shutdown in update_batch_idx(). This is a problem when there
        #       is a long period of time between batch_idx being updated (such as during validation)
        #       because it won't shut down until validation completes. We probably need to create a
        #       new thread to enforce the timeout correctly.
        self.max_collection_seconds = 300
        self.has_finished = False

        # Set up the thread responsible for making API calls
        self.send_queue = queue.Queue()
        self.sender_thread = ProfilerSenderThread(self.send_queue, self.master_url)
        self.sender_thread.start()

        # Launch the system metric collecting thread, but not the actual collection of metrics
        self.sys_metric_collector_thread = SysMetricCollectorThread(trial_id, agent_id, self.send_queue)
        self.sys_metric_collector_thread.start()

        self.timings_batcher = TimingMetricBatcher(trial_id, agent_id)

    @property
    def is_enabled(self):
        # If the timer didn't start, collection hasn't been enabled
        has_started = self.start_time is not None
        return has_started and not self.has_finished

    def update_batch_idx(self, new_batch_idx: int):
        self.current_batch_idx = new_batch_idx
        self.sys_metric_collector_thread.update_batch_idx(self.current_batch_idx)
        self.send_queue.put(self.timings_batcher.convert_to_post_format())

        # Check if we should start collecting metrics
        if not self.is_enabled and not self.has_finished and self.current_batch_idx >= self.start_on_batch:
            self._begin_collection()
            self.start_time = time.time()

        # Check if we should stop collecting metrics.
        if self.is_enabled:
            exceeded_max_batch = False
            if self.end_after_batch is not None:
                exceeded_max_batch = self.current_batch_idx > self.end_after_batch
            exceeded_max_dur = time.time() - self.start_time > self.max_collection_seconds
            if exceeded_max_batch or exceeded_max_dur:
                self._end_collection()
                self.has_finished = True

    def __enter__(self) -> 'SystemProfilerAgent':
        return self

    def __exit__(self):
        self._end_collection()

    def _begin_collection(self):
        self.sys_metric_collector_thread.activate()
        # TODO: Start up TimingBatcher as well

    def _end_collection(self):
        if self.killed:
            return
        self.killed = True
        # Shut down in reverse creation order
        self.sys_metric_collector_thread.kill()
        self.sys_metric_collector_thread.join()

        self.sender_thread.kill()
        self.sender_thread.join()

    @contextmanager
    def record_timing(self, metric_name: str) -> None:
        # TODO(brad): Check within batch range.
        if not self.is_enabled:
            return
        start = default_timer()
        yield
        duration = default_timer() - start
        self.timings_batcher.add_measurement(
            metric_name,
            self.current_batch_idx,
            datetime.datetime.utcnow(),
            duration
        )


class Measurement:
    def __init__(self, timestamp: datetime.datetime, batch_idx: int, value: float):
        self.timestamp = timestamp
        self.batch_idx = batch_idx
        self.measurement = value


class SysMetricType:
    GPU_UTIL_METRIC = "gpu_util"
    GPU_FREE_MEMORY_METRIC = "gpu_free_memory"
    NET_THRU_SENT_METRIC = "net_throughput_sent"
    NET_THRU_RECV_METRIC = "net_throughput_recv"
    DISK_IOPS_METRIC = "disk_iops"
    DISK_THRU_READ_METRIC = "disk_throughput_read"
    DISK_THRU_WRITE_METRIC = "disk_throughput_write"
    FREE_MEM_METRIC = "free_memory"
    SIMPLE_CPU_UTIL_METRIC = "cpu_util_simple"


class SysMetricBatcher:
    """
    Data structure to collect batches of SysMetrics and then convert them to the format expected by the API
    """

    def __init__(self, trial_id, agent_id):
        self.trial_id = trial_id
        self.agent_id = agent_id
        self.clear()

    def clear(self):
        self.batch = {
            SysMetricType.GPU_UTIL_METRIC: {},
            SysMetricType.GPU_FREE_MEMORY_METRIC: [],
            SysMetricType.NET_THRU_SENT_METRIC: [],
            SysMetricType.NET_THRU_RECV_METRIC: [],
            SysMetricType.DISK_IOPS_METRIC: [],
            SysMetricType.DISK_THRU_READ_METRIC: [],
            SysMetricType.DISK_THRU_WRITE_METRIC: [],
            SysMetricType.FREE_MEM_METRIC: [],
            SysMetricType.SIMPLE_CPU_UTIL_METRIC: []
        }

    def add_nongpu_measurement(self, metric_type: str, measurement: Measurement):
        assert metric_type in self.batch.keys(), \
            f"Tried to add unknown type of non-GPU metric: {metric_type}"
        self.batch[metric_type].append(measurement)

    def add_gpu_measurement(self, metric_type: str, gpu_uuid: str, measurement: Measurement):
        assert metric_type in self.batch.keys(), \
            f"Tried to add unknown type of GPU metric: {metric_type}"
        if gpu_uuid not in self.batch[metric_type].keys():
            self.batch[metric_type][gpu_uuid] = []
        self.batch[metric_type][gpu_uuid].append(measurement)

    def convert_to_timestamp_str(self, timestamp: datetime.datetime):
        assert isinstance(timestamp, datetime.datetime), \
            f"Input to conversion function must be a datetime object. Instead got {type(timestamp)}"
        return timestamp.isoformat() + "Z"

    def convert_to_post_format(self) -> List[TrialProfilerMetricsBatch]:
        def to_post_format(measurements: List[Any], labels: Dict[str, Any]) -> TrialProfilerMetricsBatch:
            values, batches, timestamps = [], [], []
            for m in measurements:
                values.append(m.measurement)
                batches.append(m.batch_index)
                timestamps.append(self.convert_to_timestamp_str(m.timestamp))
            return TrialProfilerMetricsBatch(values, batches, timestamps, labels)

        def make_labels(name: str, metric_type: str, gpu_uuid_label: str = "") -> Dict[str, Any]:
            return {
                "trialId": self.trial_id,
                "name": name,
                "agentId": self.agent_id,
                "gpuUuid": gpu_uuid_label,
                "metricType": metric_type
            }

        trial_profiler_metrics_batches = []
        for metric_name in self.batch.keys():
            # TODO: Don't forget to include GPU Memory
            if metric_name != SysMetricType.GPU_UTIL_METRIC and len(self.batch[metric_name]) > 0:
                trial_profiler_metrics_batches.append(to_post_format(
                    self.batch[metric_name],
                    make_labels(metric_name, SYSTEM_METRIC_TYPE_ENUM),
                ))

            # GPU Metrics need to be grouped by GPU UUID
            # TODO: Don't forget to include GPU Memory
            if metric_name == SysMetricType.GPU_UTIL_METRIC and len(self.batch[metric_name].keys()) > 0:
                for gpu_uuid in self.batch[metric_name].keys():
                    trial_profiler_metrics_batches.append(to_post_format(
                        self.batch[metric_name][gpu_uuid],
                        make_labels(metric_name, SYSTEM_METRIC_TYPE_ENUM, gpu_uuid_label=gpu_uuid)
                    ))

        return trial_profiler_metrics_batches


class StartMessage:
    pass

class ShutdownMessage:
    pass

class TimingMetricBatcher:
    """
    Data structure to collect batches of SysMetrics and then convert them to the format expected by the API
    """

    def __init__(self, trial_id, agent_id):
        self.trial_id = trial_id
        self.agent_id = agent_id
        self.batch = {} # Dict[str, List['Measurement']]

    def add_measurement(self, metric_name: str, batch_idx: int, timestamp: datetime.datetime, value: float):
        if metric_name not in self.batch.keys():
            self.batch[metric_name] = []

        self.batch[metric_name].append(Measurement(
            timestamp,
            batch_idx,
            value
        ))

    def convert_to_timestamp_str(self, timestamp: datetime.datetime):
        return timestamp.isoformat() + "Z"

    def convert_to_post_format(self):
        post_formatted = []
        for metric_type in self.batch.keys():
            batches, timestamps, values = [], [], []
            for m in self.batch[metric_type]:
                batches.append(m.batch_index)
                timestamps.append(self.convert_to_timestamp_str(m.timestamp))
                values.append(m.value)
            single_metric_batch = {
                "values": values,
                "batches": batches,
                "timestamps":timestamps,
                "labels": {
                    "trialId": self.trial_id,
                    "name": metric_type,
                    "agentId": self.agent_id,
                    "metricType": TIMING_METRIC_TYPE_ENUM
                }
            }
            post_formatted.append(single_metric_batch)
        self.batch = {}
        return post_formatted


class SysMetricCollectorThread(threading.Thread):
    """
    Background thread for collecting profiler metrics at a high granularity and shipping them to the master

    - SimpleCpuUtilization = Measured in percent
    - FreeMemory = Measured in Gigabytes
    - NetworkSentThroughput = Measured in Gigabit/s
    - NetworkRecvThroughput = Measured in Gigabit/s
    - DiskIops
    - DiskReadThroughput = Measured in bytes/second
    - DiskWriteThroughput = Measured in bytes/second
    - GpuUtilization = Measured in percent
    """

    FLUSH_INTERVAL = 10  # How often to make API calls
    MEASUREMENT_INTERVAL = 0.1

    def __init__(self, trial_id: int, agent_id: str, send_queue: queue.Queue):

        self.current_batch_idx = 0
        self.send_queue = send_queue
        self.control_queue = queue.Queue()
        self.current_batch = SysMetricBatcher(trial_id, agent_id)
        self.current_batch.clear()

        super().__init__(daemon=True)

    def activate(self):
        self.control_queue.put(StartMessage())

    def kill(self):
        self.control_queue.put(ShutdownMessage())

    def update_batch_idx(self, new_batch_idx: int):
        self.current_batch_idx = new_batch_idx

    def run(self) -> None:
        last_measurement_time = None
        batch_start_time = None
        cpu_util_collector = SimpleCpuUtilCollector()
        gpu_util_collector = GpuUtilCollector()
        gpu_memory_collection = GpuMemoryCollector()
        network_throughput_collector = NetThroughputCollector()
        free_memory_collector = FreeMemoryCollector()
        disk_collector = DiskReadWriteRateCollector()

        msg = self.control_queue.get()
        if isinstance(msg, ShutdownMessage):
            return

        next_collection = time.time()

        while True:
            now = time.time()
            if now < next_collection:
                # check for quit message while we wait
                sleep_time = next_collection - now
                try:
                    msg = self.control_queue.get(timeout=sleep_time)
                    if isinstance(msg, ShutdownMessage):
                        # Drop any partial batches
                        return
                except queue.Empty:
                    pass

            next_collection += self.MEASUREMENT_INTERVAL

            # One-time initialization
            if last_measurement_time is None:
                last_measurement_time = time.time()
                batch_start_time = time.time()
                network_throughput_collector.reset()
                disk_collector.reset()

            # Check if it is time to take a new measurement
            if time.time() - last_measurement_time > self.MEASUREMENT_INTERVAL:
                immutable_batch_idx = self.current_batch_idx
                cpu_util = cpu_util_collector.measure(immutable_batch_idx)
                gpu_util = gpu_util_collector.measure(immutable_batch_idx)
                gpu_memory = gpu_memory_collection.measure(immutable_batch_idx)
                net_thru_sent, net_thru_recv = network_throughput_collector.measure(immutable_batch_idx)
                free_memory = free_memory_collector.measure(immutable_batch_idx)
                disk_read_thru, disk_write_thru, iops = disk_collector.measure(immutable_batch_idx)

                for gpu_uuid in gpu_util.keys():
                    self.current_batch.add_gpu_measurement(SysMetricType.GPU_UTIL_METRIC, gpu_uuid, gpu_util[gpu_uuid])

                for gpu_uuid in gpu_memory.keys():
                    self.current_batch.add_gpu_measurement(SysMetricType.GPU_FREE_MEMORY_METRIC, gpu_uuid, gpu_util[gpu_uuid])

                self.current_batch.add_nongpu_measurement(SysMetricType.NET_THRU_SENT_METRIC, net_thru_sent)
                self.current_batch.add_nongpu_measurement(SysMetricType.NET_THRU_RECV_METRIC, net_thru_recv)
                self.current_batch.add_nongpu_measurement(SysMetricType.DISK_IOPS_METRIC, iops)
                self.current_batch.add_nongpu_measurement(SysMetricType.DISK_THRU_READ_METRIC, disk_read_thru)
                self.current_batch.add_nongpu_measurement(SysMetricType.DISK_THRU_WRITE_METRIC, disk_write_thru)
                self.current_batch.add_nongpu_measurement(SysMetricType.FREE_MEM_METRIC, free_memory)
                self.current_batch.add_nongpu_measurement(SysMetricType.SIMPLE_CPU_UTIL_METRIC, cpu_util)
                last_measurement_time = time.time()

            # Check if it is time to flush the batch and start a new batch
            if time.time() - batch_start_time > self.FLUSH_INTERVAL:
                self.send_queue.put(self.current_batch.convert_to_post_format())
                self.current_batch.clear()
                batch_start_time = time.time()


# This is a thread that exists solely so that we can make API calls without blocking
# It has a Queue through which work is sent to the thread
class ProfilerSenderThread(threading.Thread):
    def __init__(self, inbound_queue: queue.Queue, master_url: str) -> None:
        self.master_url = master_url
        self.inbound_queue = inbound_queue
        super().__init__(daemon=True)

    def kill(self):
        self.inbound_queue.put(ShutdownMessage())

    def run(self) -> None:
        while True:
            message = self.inbound_queue.get()
            if isinstance(message, ShutdownMessage):
                return
            self.send_batch(message)

    # This is a blocking operation that must handle all exceptions gracefully
    def send_batch(self, post_bodies: List[TrialProfilerMetricsBatch]):
        # TODO: In the case of repeated failures, this can take a long time
        #       and we have no mechanism to handle the inbound queue filling up

        # TODO: Convert to work with new multiple batch API
        max_attempts = 2
        for post_body in post_bodies:
            for i in range(max_attempts):
                try:
                    api.post_trial_profiler_metrics(
                        self.master_url,
                        post_body["values"],
                        post_body["batches"],
                        post_body["timestamps"],
                        post_body["labels"],
                    )
                    break
                except Exception as e:
                    # TODO: We could handle specific API errors differently as some are non-recoverable
                    logging.warning(f"Failed to post metrics with labels {post_body['labels']} to the master: {e}")
                    time.sleep(1)


GIGA = 1_000_000_000


class SimpleCpuUtilCollector:
    def measure(self, batch_idx):
        cpu_util = psutil.cpu_percent()
        timestamp = datetime.datetime.utcnow()
        return Measurement(timestamp, batch_idx, cpu_util)


class FreeMemoryCollector:
    def measure(self, batch_idx):
        free_mem_bytes = psutil.virtual_memory().available
        timestamp = datetime.datetime.utcnow()
        return Measurement(timestamp, batch_idx, free_mem_bytes * GIGA)


class NetThroughputCollector:
    def __init__(self):
        self.reset()

    def reset(self):
        self.start_time = time.time()
        net = psutil.net_io_counters()
        self.start_sent = net.bytes_sent
        self.start_recv = net.bytes_recv

    def measure(self, batch_idx):
        net = psutil.net_io_counters()
        end_time = time.time()

        delta_sent_bytes = net.bytes_sent - self.start_sent
        delta_recv_bytes = net.bytes_recv - self.start_recv

        time_delta = end_time - self.start_time

        self.start_time = end_time
        self.start_sent = net.bytes_sent
        self.start_recv = net.bytes_recv

        sent_throughput_bytes_per_second = delta_sent_bytes / time_delta
        recv_throughput_bytes_per_second = delta_recv_bytes / time_delta

        sent_throughput_gigabits_per_second = sent_throughput_bytes_per_second * 8 * GIGA
        recv_throughput_gigabits_per_second = recv_throughput_bytes_per_second * 8 * GIGA

        timestamp = datetime.datetime.fromtimestamp(end_time)
        return Measurement(timestamp, batch_idx, sent_throughput_gigabits_per_second), \
               Measurement(timestamp, batch_idx, recv_throughput_gigabits_per_second)


class DiskReadWriteRateCollector:
    def __init__(self):
        self.reset()

    def reset(self):
        self.start_time = time.time()
        disk = psutil.disk_io_counters()

        self.start_read_bytes = disk.read_bytes
        self.start_write_bytes = disk.write_bytes

        self.start_read_count = disk.read_count
        self.start_write_count = disk.write_count

    def measure(self, batch_idx):
        disk = psutil.disk_io_counters()
        end_time = time.time()

        delta_read_bytes = disk.read_bytes - self.start_read_bytes
        delta_write_bytes = disk.write_bytes - self.start_write_bytes

        delta_read_count = disk.read_count - self.start_read_count
        delta_write_count = disk.write_count - self.start_write_count

        delta_time = end_time - self.start_time

        self.start_time = end_time
        self.start_read_bytes = disk.read_bytes
        self.start_write_bytes = disk.write_bytes
        self.start_read_count = disk.read_count
        self.start_write_count = disk.write_count

        read_throughput_bytes_per_sec = delta_read_bytes / delta_time
        write_throughput_bytes_per_sec = delta_write_bytes / delta_time

        read_throughput_count_per_sec = delta_read_count / delta_time
        write_throughput_count_per_sec = delta_write_count / delta_time

        timestamp = datetime.datetime.fromtimestamp(end_time)
        read_throughput = Measurement(timestamp, batch_idx, read_throughput_bytes_per_sec)
        write_throughput = Measurement(timestamp, batch_idx, write_throughput_bytes_per_sec)
        iops = Measurement(timestamp, batch_idx, read_throughput_count_per_sec + write_throughput_count_per_sec)

        return read_throughput, write_throughput, iops


class GpuUtilCollector:
    def __init__(self):
        pynvml.nvmlInit()
        self.num_gpus = pynvml.nvmlDeviceGetCount()

    def measure(self, batch_idx):
        measurements = {}
        timestamp = datetime.datetime.utcnow()
        for i in range(self.num_gpus):
            handle = pynvml.nvmlDeviceGetHandleByIndex(i)
            try:
                util = pynvml.nvmlDeviceGetUtilizationRates(handle)
                measurements[handle] = Measurement(timestamp, batch_idx, util.gpu)
            except pynvml.NVMLError as e:
                logging.info(f"{LOG_NAMESPACE}: failed to sample GPU utilization for GPU {i}: {e}")
        return measurements


class GpuMemoryCollector():
    def __init__(self):
        pynvml.nvmlInit()
        self.num_gpus = pynvml.nvmlDeviceGetCount()

    def measure(self, batch_idx):
        measurements = {}
        timestamp = datetime.datetime.utcnow()
        for i in range(self.num_gpus):
            handle = pynvml.nvmlDeviceGetHandleByIndex(i)
            try:
                info = pynvml.nvmlDeviceGetMemoryInfo(handle)
                measurements[handle] = Measurement(timestamp, batch_idx, info.free)
            except pynvml.NVMLError as e:
                logging.info(f"{LOG_NAMESPACE}: failed to sample GPU memory for GPU {i}: {e}")
        return measurements


def get_rank_if_horovod_process_else_return_zero() -> Optional[int]:
    return int(os.getenv("HOROVOD_LOCAL_RANK", 0))