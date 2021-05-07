import datetime
import logging
import queue
import threading
import time
from types import TracebackType
from typing import Any, Callable, Dict, List, Optional, Tuple, Type, Union

import psutil

from determined.common import api, check
from determined.common.api import TrialProfilerMetricsBatch

SYSTEM_METRIC_TYPE_ENUM = "PROFILER_METRIC_TYPE_SYSTEM"

LOG_NAMESPACE = "determined-profiler"

try:
    import pynvml

    pynvml.nvmlInit()
    SHOULD_PROFILE_GPUS = True
except ModuleNotFoundError:
    logging.info(f"{LOG_NAMESPACE} pynvml not found. Not collecting GPU metrics")
    SHOULD_PROFILE_GPUS = False
except pynvml.NVMLError_LibraryNotFound:
    logging.info(f"{LOG_NAMESPACE} pynvml LibraryNotFound error. Not collecting GPU metrics")
    SHOULD_PROFILE_GPUS = False
except Exception as e:
    raise RuntimeError(f"Could not set up pynvml, but it failed with an unexpected error: {e}")


class Measurement:
    def __init__(self, timestamp: datetime.datetime, batch_idx: int, value: float):
        self.timestamp = timestamp
        self.batch_idx = batch_idx
        self.measurement = value


class StartMessage:
    pass


class ShutdownMessage:
    pass


DEBUG = True


def debug_log(*args: Any) -> None:
    args_as_str = " ".join([str(arg) for arg in args])
    if DEBUG:
        logging.info(f"{LOG_NAMESPACE} (DEBUG) {args_as_str}")


class ProfilerAgent:
    """
    Agent that collects metrics and sends them to the master.

    The ProfilerAgent needs to be created at the beginning of training and it needs
    to be notified every time the batch_idx increases.

    It will collect System Metrics using a background thread and then batch them and send
    them to the master. You can also collect Timings through the ProfilerAgent with the
    record_timing() method. The timings will be batched and sent to the master.

    Profiling is only active between start_on_batch and end_after_batch. It will also automatically
    shut down 5 minutes after starting. When profiling is not active, no system metrics are
    collected and the record_timing function is a no-op.

    If is_enabled=False, every method in this class should be a no-op.
    """

    def __init__(
        self,
        trial_id: str,
        agent_id: str,
        master_url: str,
        profiling_is_enabled: bool,
        global_rank: int,
        local_rank: int,
        start_on_batch: int,
        end_after_batch: Optional[int] = None,
    ):

        debug_log("ProfilingAgent __init__")

        self.current_batch_idx = 0
        self.agent_id = agent_id
        self.trial_id = trial_id
        self.master_url = master_url
        self.start_on_batch = start_on_batch
        self.end_after_batch = end_after_batch
        self.local_rank = local_rank
        self.global_rank = global_rank

        self.profiling_is_enabled_in_experiment_config = profiling_is_enabled

        self.has_started = False
        self.has_finished = False

        self.shutdown_lock = threading.Lock()

        # If the ProfilingAgent is disabled, don't waste resources by creating useless threads
        if self.is_enabled:
            # Set up timer thread to stop collecting after 5 minutes
            self.max_collection_seconds = 300
            self.shutdown_timer = PreemptibleTimer(
                self.max_collection_seconds, self._end_collection
            )

            # Set up the thread responsible for making API calls
            self.send_queue = (
                queue.Queue()
            )  # type: """queue.Queue[Union[List[TrialProfilerMetricsBatch], ShutdownMessage]]"""
            self.sender_thread = ProfilerSenderThread(self.send_queue, self.master_url)

            if self.sysmetrics_is_enabled:
                self.sys_metric_collector_thread = SysMetricCollectorThread(
                    trial_id, agent_id, self.send_queue
                )

            # TODO [DET-5062]: Add data structure to batch timings and then send to SenderThread
            #       Does this need to be its own thread to flush correctly?
            # if self.timings_is_enabled:
            #     self.timings_batcher = TimingsBatcher()

    # Launch the children threads. This does not mean 'start collecting metrics'
    def start(self) -> None:
        if not self.is_enabled:
            return

        debug_log("ProfilerAgent.start")

        self.sender_thread.start()
        self.shutdown_timer.start()

        if self.sysmetrics_is_enabled:
            debug_log("ProfilerAgent.start - starting sys_metric_collector_thread")
            self.sys_metric_collector_thread.start()

    def end(self) -> None:
        if not self.is_enabled:
            return
        self._end_collection()
        self.cleanup_timer()

    def __enter__(self) -> "ProfilerAgent":
        self.start()
        return self

    def __exit__(
        self,
        exc_type: Optional[Type[BaseException]],
        exc_value: Optional[BaseException],
        traceback: Optional[TracebackType],
    ) -> None:
        debug_log("ProfilerAgent exited context manager")
        self.end()

    @property
    def is_enabled(self) -> bool:
        """
        Is the ProfilingAgent supposed to do anything at all?
        If this is false, the entire profiler is a no-op
        """
        if not self.profiling_is_enabled_in_experiment_config:
            return False
        return self.sysmetrics_is_enabled or self.timings_is_enabled

    @property
    def sysmetrics_is_enabled(self) -> bool:
        return self.profiling_is_enabled_in_experiment_config and self.local_rank == 0

    @property
    def timings_is_enabled(self) -> bool:
        return self.profiling_is_enabled_in_experiment_config and self.global_rank == 0

    @property
    def is_active(self) -> bool:
        """
        Is the ProfilingAgent actively collecting data and shipping to the API?
        """
        if not self.is_enabled:
            return False
        return self.has_started and not self.has_finished

    def update_batch_idx(self, new_batch_idx: int) -> None:
        if not self.is_enabled:
            return

        check.check_gt_eq(
            new_batch_idx, self.current_batch_idx, "Batch index should never decrease over time"
        )
        self.current_batch_idx = new_batch_idx
        self.sys_metric_collector_thread.update_batch_idx(self.current_batch_idx)

        # Check if we should start collecting metrics
        if not self.has_started and self.current_batch_idx >= self.start_on_batch:
            self._begin_collection()

        # Check if we should stop collecting metrics due to batch idx being exceeded
        if (
            self.is_active
            and self.end_after_batch is not None
            and self.current_batch_idx > self.end_after_batch
        ):
            debug_log(
                f"ProfilerAgent.update_batch_idx exceeded "
                f"end_after_batch ({self.end_after_batch}) and shutting down"
            )
            self._end_collection()
            self.shutdown_timer.send_shutdown_signal()

    def cleanup_timer(self) -> None:
        if not self.is_enabled:
            return
        self.shutdown_timer.send_shutdown_signal()
        self.shutdown_timer.join()

    def _begin_collection(self) -> None:
        if not self.is_enabled:
            return

        debug_log("ProfilerAgent._begin_collection")

        # Note: due to its simplicity, sender_thread doesn't need to be activated
        self.sys_metric_collector_thread.activate()
        # TODO [DET-5062]: Activate TimingBatcher as well
        self.shutdown_timer.activate()
        self.has_started = True

    def _end_collection(self) -> None:
        """
        Stop collecting data and shut down most child threads. This function can be invoked due to
        the max batch idx being exceeded, due to timeout or due to the ProfilingAgent shutting down
        as the harness exits, so the function needs to be threadsafe and idempotent.

        This cleans up all threads except for the shutdown timer, because this function might be
        invoked by the shutdown timer.
        """
        if not self.is_enabled:
            return

        debug_log("ProfilerAgent._end_collection")

        with self.shutdown_lock:
            if self.has_finished:
                debug_log(
                    "ProfilerAgent._end_collection - already finished, skipping any shut down"
                )
                return

            debug_log("ProfilerAgent._end_collection - is_enabled, shutting down threads")
            debug_log("ProfilerAgent._end_collection - ")

            self.shutdown_timer.send_shutdown_signal()
            debug_log("ProfilerAgent._end_collection - timer shutdown signal sent")

            if self.sysmetrics_is_enabled:
                debug_log(
                    "ProfilerAgent._end_collection - sysmetrics_is_enabled "
                    "shutting down sysmetrics collector"
                )
                self.sys_metric_collector_thread.send_shutdown_signal()
                debug_log("ProfilerAgent._end_collection - sysmetriccollector shutdown signal sent")
                self.sys_metric_collector_thread.join()
                debug_log("ProfilerAgent._end_collection - sysmetriccollector joined")

            # TODO [DET-5062]: Shut down TimingBatcher as well
            debug_log("ProfilerAgent._end_collection - shutting down sender thread")
            self.sender_thread.send_shutdown_signal()
            debug_log("ProfilerAgent._end_collection - sender thread shutdown signal sent")
            self.sender_thread.join()
            debug_log("ProfilerAgent._end_collection - sender thread joined")

            debug_log("ProfilerAgent._end_collection - setting has_finished to true")
            self.has_finished = True

    def record_timing(self, timing: float) -> None:
        if not self.is_enabled:
            return
        # TODO [DET-5062]: Add new timing to TimingBatcher


def create_no_op_profiler() -> ProfilerAgent:
    """
    Create a ProfilerAgent that is disabled. Utility function for testing, but also
    used by test_one_batch in native, so it can't be put in the tests folder.
    """
    return ProfilerAgent(
        trial_id="",
        agent_id="",
        master_url="",
        profiling_is_enabled=False,
        global_rank=0,
        local_rank=0,
        start_on_batch=0,
        end_after_batch=None,
    )


class PreemptibleTimer(threading.Thread):
    """
    Version of threading.Timer that can be cleaned up if the timer is no longer needed
    ```
    timer = CustomTimer(300, callback_fn)
    timer.start()  # start the thread, not the timer
    timer.begin_timer()

    # After 300 seconds, callback_fn will be executed

    # If we need to clean up the timer
    timer.send_shutdown_signal()  # This is idempotent and will work fine if the timer has gone off
    timer.join()
    ```
    """

    def __init__(self, duration: int, callback: Callable):
        self.duration = duration
        self.callback = callback
        self._timer_has_begun = False
        self.control_queue: "queue.Queue[Union[StartMessage, ShutdownMessage]]" = queue.Queue()

        super().__init__(daemon=True)

    def activate(self) -> None:
        if not self._timer_has_begun:
            self.control_queue.put(StartMessage())
            self._timer_has_begun = True

    def send_shutdown_signal(self) -> None:
        self.control_queue.put(ShutdownMessage())

    def run(self) -> None:
        msg = self.control_queue.get()
        if isinstance(msg, ShutdownMessage):
            return

        try:
            msg = self.control_queue.get(timeout=self.duration)
            if isinstance(msg, ShutdownMessage):
                return
        except queue.Empty:
            # Time is up!
            debug_log("PreemptibleTimer time expired, executing callback fn")
            self.callback()


class SysMetricCollectorThread(threading.Thread):
    """
    Background thread for collecting profiler metrics at a high granularity and shipping them to
    the master

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

    def __init__(self, trial_id: str, agent_id: str, send_queue: queue.Queue):

        self.current_batch_idx = 0
        self.send_queue = send_queue
        self.control_queue: "queue.Queue[Union['StartMessage', 'ShutdownMessage']]" = queue.Queue()
        self.current_batch = SysMetricBatcher(trial_id, agent_id)

        super().__init__(daemon=True)

    def activate(self) -> None:
        """Begin collecting System Metrics"""
        debug_log("SysMetricCollectorThread.activate()")
        self.control_queue.put(StartMessage())

    def send_shutdown_signal(self) -> None:
        self.control_queue.put(ShutdownMessage())

    def update_batch_idx(self, new_batch_idx: int) -> None:
        self.current_batch_idx = new_batch_idx

    def run(self) -> None:
        cpu_util_collector = SimpleCpuUtilCollector()
        net_throughput_collector = NetThroughputCollector()
        free_memory_collector = FreeMemoryCollector()
        disk_collector = DiskReadWriteRateCollector()

        if SHOULD_PROFILE_GPUS:
            gpu_util_collector = GpuUtilCollector()
            gpu_memory_collection = GpuMemoryCollector()

        # Do nothing while we wait for a StartMessage
        msg = self.control_queue.get()
        if isinstance(msg, ShutdownMessage):
            return

        debug_log("SysMetricCollectorThread.run - StartMessage received")

        # Do initial measurement for rate-based collectors
        net_throughput_collector.reset()
        disk_collector.reset()

        batch_start_time = time.time()
        next_collection = time.time() + self.MEASUREMENT_INTERVAL

        while True:
            debug_log("SysMetricCollectorThread.run - started new iteration of while loop")
            # This code is using a trick with the control_queue to sleep/block until the next
            # measurement should be taken, while still being able to respond to a shutdown
            # request immediately.
            now = time.time()
            if now < next_collection:
                sleep_time = next_collection - now
                # a negative timeout will lead to an exception when retrieving from the queue
                sleep_time = max(sleep_time, 0)
                try:
                    debug_log(
                        "SysMetricCollectorThread.run - waiting",
                        sleep_time,
                        "seconds for next collection",
                    )
                    msg = self.control_queue.get(timeout=sleep_time)
                    if isinstance(msg, ShutdownMessage):
                        debug_log(
                            "SysMetricCollectorThread.run - received shutdown message in while loop"
                        )
                        # Drop any partial batches if we receive a shutdown
                        return
                except queue.Empty:
                    pass

            debug_log("SysMetricCollectorThread.run - time for the next collection")

            next_collection += self.MEASUREMENT_INTERVAL

            cpu_util = cpu_util_collector.measure(self.current_batch_idx)
            self.current_batch.add_nongpu_measurement(
                SysMetricType.SIMPLE_CPU_UTIL_METRIC, cpu_util
            )
            debug_log("SysMetricCollectorThread.run - collected CPU metric and stored measurement")

            net_thru_sent, net_thru_recv = net_throughput_collector.measure(self.current_batch_idx)
            self.current_batch.add_nongpu_measurement(
                SysMetricType.NET_THRU_SENT_METRIC, net_thru_sent
            )
            self.current_batch.add_nongpu_measurement(
                SysMetricType.NET_THRU_RECV_METRIC, net_thru_recv
            )

            free_memory = free_memory_collector.measure(self.current_batch_idx)
            self.current_batch.add_nongpu_measurement(SysMetricType.FREE_MEM_METRIC, free_memory)

            disk_read_thru, disk_write_thru, iops = disk_collector.measure(self.current_batch_idx)
            self.current_batch.add_nongpu_measurement(
                SysMetricType.DISK_THRU_READ_METRIC, disk_read_thru
            )
            self.current_batch.add_nongpu_measurement(
                SysMetricType.DISK_THRU_WRITE_METRIC, disk_write_thru
            )
            self.current_batch.add_nongpu_measurement(SysMetricType.DISK_IOPS_METRIC, iops)

            if SHOULD_PROFILE_GPUS:
                gpu_util = gpu_util_collector.measure(self.current_batch_idx)
                for gpu_uuid in gpu_util.keys():
                    self.current_batch.add_gpu_measurement(
                        SysMetricType.GPU_UTIL_METRIC, gpu_uuid, gpu_util[gpu_uuid]
                    )

                gpu_memory = gpu_memory_collection.measure(self.current_batch_idx)
                for gpu_uuid in gpu_memory.keys():
                    self.current_batch.add_gpu_measurement(
                        SysMetricType.GPU_FREE_MEMORY_METRIC, gpu_uuid, gpu_util[gpu_uuid]
                    )

            debug_log("SysMetricCollectorThread.run - collected all metrics")
            # Check if it is time to flush the batch and start a new batch
            if time.time() - batch_start_time > self.FLUSH_INTERVAL:
                debug_log("SysMetricCollectorThread.run - decided to flush the batch")
                self.send_queue.put(self.current_batch.convert_to_post_format())
                self.current_batch.clear()
                batch_start_time = time.time()


class ProfilerSenderThread(threading.Thread):
    """
    This is a thread that exists solely so that we can make API calls without blocking.
    It has a Queue through which work is sent to the thread
    """

    def __init__(self, inbound_queue: queue.Queue, master_url: str) -> None:
        self.master_url = master_url
        self.inbound_queue = inbound_queue
        super().__init__(daemon=True)

    def send_shutdown_signal(self) -> None:
        self.inbound_queue.put(ShutdownMessage())

    def run(self) -> None:
        while True:
            message = self.inbound_queue.get()
            if isinstance(message, ShutdownMessage):
                return
            debug_log("ProfilerBatchToSend", self.master_url, message)
            api.post_trial_profiler_metrics_batches(
                self.master_url,
                message,
            )


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
    Data structure to collect batches of SysMetrics and then convert them to the format expected by
    the API
    """

    def __init__(self, trial_id: str, agent_id: str) -> None:
        self.trial_id = trial_id
        self.agent_id = agent_id
        self.clear()

    def clear(self) -> None:
        self.batch = {
            SysMetricType.GPU_UTIL_METRIC: {},
            SysMetricType.GPU_FREE_MEMORY_METRIC: {},
            SysMetricType.NET_THRU_SENT_METRIC: [],
            SysMetricType.NET_THRU_RECV_METRIC: [],
            SysMetricType.DISK_IOPS_METRIC: [],
            SysMetricType.DISK_THRU_READ_METRIC: [],
            SysMetricType.DISK_THRU_WRITE_METRIC: [],
            SysMetricType.FREE_MEM_METRIC: [],
            SysMetricType.SIMPLE_CPU_UTIL_METRIC: [],
        }  # type: Dict[str, Any]

    def add_nongpu_measurement(self, metric_type: str, measurement: Measurement) -> None:
        assert (
            metric_type in self.batch.keys()
        ), f"Tried to add unknown type of non-GPU metric: {metric_type}"
        self.batch[metric_type].append(measurement)

    def add_gpu_measurement(
        self, metric_type: str, gpu_uuid: str, measurement: Measurement
    ) -> None:
        assert (
            metric_type in self.batch.keys()
        ), f"Tried to add unknown type of GPU metric: {metric_type}"
        if gpu_uuid not in self.batch[metric_type].keys():
            self.batch[metric_type][gpu_uuid] = []
        self.batch[metric_type][gpu_uuid].append(measurement)

    def convert_to_timestamp_str(self, timestamp: datetime.datetime) -> str:
        return timestamp.isoformat() + "Z"

    def convert_to_post_format(self) -> List[TrialProfilerMetricsBatch]:
        def to_post_format(
            measurements: List[Measurement], labels: Dict[str, Any]
        ) -> TrialProfilerMetricsBatch:
            values, batches, timestamps = [], [], []
            for m in measurements:
                values.append(m.measurement)
                batches.append(m.batch_idx)
                timestamps.append(self.convert_to_timestamp_str(m.timestamp))
            return TrialProfilerMetricsBatch(values, batches, timestamps, labels)

        def make_labels(name: str, metric_type: str, gpu_uuid_label: str = "") -> Dict[str, Any]:
            return {
                "trialId": self.trial_id,
                "name": name,
                "agentId": self.agent_id,
                "gpuUuid": gpu_uuid_label,
                "metricType": metric_type,
            }

        trial_profiler_metrics_batches = []
        for metric_name in self.batch.keys():
            if (
                metric_name
                not in [SysMetricType.GPU_UTIL_METRIC, SysMetricType.GPU_FREE_MEMORY_METRIC]
                and len(self.batch[metric_name]) > 0
            ):
                trial_profiler_metrics_batches.append(
                    to_post_format(
                        self.batch[metric_name],
                        make_labels(metric_name, SYSTEM_METRIC_TYPE_ENUM),
                    )
                )

            # GPU Metrics need to be batched by GPU UUID
            if (
                metric_name in [SysMetricType.GPU_UTIL_METRIC, SysMetricType.GPU_FREE_MEMORY_METRIC]
                and len(self.batch[metric_name].keys()) > 0
            ):
                for gpu_uuid in self.batch[metric_name].keys():
                    if len(self.batch[metric_name][gpu_uuid]) > 0:
                        trial_profiler_metrics_batches.append(
                            to_post_format(
                                self.batch[metric_name][gpu_uuid],
                                make_labels(
                                    metric_name, SYSTEM_METRIC_TYPE_ENUM, gpu_uuid_label=gpu_uuid
                                ),
                            )
                        )

        return trial_profiler_metrics_batches


GIGA = 1_000_000_000


class SimpleCpuUtilCollector:
    def measure(self, batch_idx: int) -> Measurement:
        cpu_util = psutil.cpu_percent()
        timestamp = datetime.datetime.utcnow()
        return Measurement(timestamp, batch_idx, cpu_util)


class FreeMemoryCollector:
    def measure(self, batch_idx: int) -> Measurement:
        free_mem_bytes = psutil.virtual_memory().available
        timestamp = datetime.datetime.utcnow()
        return Measurement(timestamp, batch_idx, free_mem_bytes / GIGA)


class NetThroughputCollector:
    def __init__(self) -> None:
        self.reset()

    def reset(self) -> None:
        self.start_time = time.time()
        net = psutil.net_io_counters()
        self.start_sent = net.bytes_sent
        self.start_recv = net.bytes_recv

    def measure(self, batch_idx: int) -> Tuple[Measurement, Measurement]:
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

        sent_throughput_gigabits_per_second = sent_throughput_bytes_per_second * 8 / GIGA
        recv_throughput_gigabits_per_second = recv_throughput_bytes_per_second * 8 / GIGA

        timestamp = datetime.datetime.fromtimestamp(end_time)
        return Measurement(timestamp, batch_idx, sent_throughput_gigabits_per_second), Measurement(
            timestamp, batch_idx, recv_throughput_gigabits_per_second
        )


class DiskReadWriteRateCollector:
    def __init__(self) -> None:
        self.reset()

    def reset(self) -> None:
        self.start_time = time.time()
        disk = psutil.disk_io_counters()

        self.start_read_bytes = disk.read_bytes
        self.start_write_bytes = disk.write_bytes

        self.start_read_count = disk.read_count
        self.start_write_count = disk.write_count

    def measure(self, batch_idx: int) -> Tuple[Measurement, Measurement, Measurement]:
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
        iops = Measurement(
            timestamp, batch_idx, read_throughput_count_per_sec + write_throughput_count_per_sec
        )

        return read_throughput, write_throughput, iops


class GpuUtilCollector:
    def __init__(self) -> None:
        self.num_gpus = pynvml.nvmlDeviceGetCount()

    def measure(self, batch_idx: int) -> Dict[str, Measurement]:
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


class GpuMemoryCollector:
    def __init__(self) -> None:
        self.num_gpus = pynvml.nvmlDeviceGetCount()

    def measure(self, batch_idx: int) -> Dict[str, Measurement]:
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
