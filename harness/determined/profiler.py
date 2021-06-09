import contextlib
import logging
import queue
import threading
import time
from datetime import datetime, timedelta, timezone
from types import TracebackType
from typing import Any, Callable, Dict, Iterator, List, Optional, Tuple, Type, Union, cast

import psutil

import determined as det
from determined.common import api, check
from determined.common.api import TrialProfilerMetricsBatch

SYSTEM_METRIC_TYPE_ENUM = "PROFILER_METRIC_TYPE_SYSTEM"
TIMING_METRIC_TYPE_ENUM = "PROFILER_METRIC_TYPE_TIMING"
MAX_COLLECTION_SECONDS = 300
LOG_NAMESPACE = "determined-profiler"


class PynvmlWrapperError(Exception):
    pass


class PynvmlWrapper:
    """
    Class to wrap pynvml. Handle checks around whether the nvidia management
    library is available, whether the pynvml bindings are installed, and
    whether actual utilization/memory operations are successful (they may not
    be in some cases, for example when using MIGs).
    """

    def __init__(self) -> None:
        self._pynvml = None  # type: Optional[Any]
        self._device_count = None  # type: Optional[int]
        self._index_to_uuid_map = {}  # type: Dict[int, str]
        try:
            import pynvml

            pynvml.nvmlInit()
            try:
                num_gpus = pynvml.nvmlDeviceGetCount()
                for i in range(num_gpus):
                    handle = pynvml.nvmlDeviceGetHandleByIndex(i)
                    uuid = pynvml.nvmlDeviceGetUUID(handle)
                    pynvml.nvmlDeviceGetMemoryInfo(handle)
                    pynvml.nvmlDeviceGetUtilizationRates(handle)
                    self._index_to_uuid_map[i] = uuid
                self._pynvml = pynvml
                self._device_count = num_gpus
            except Exception as e:
                logging.warning(
                    f"{LOG_NAMESPACE}: pynvml is functional, but failed to pass functionality "
                    f"test due to exception. Not collecting GPU metrics. Exception details: {e}"
                )
        except ModuleNotFoundError:
            logging.info(f"{LOG_NAMESPACE}: pynvml not found. Not collecting GPU metrics")
        except pynvml.NVMLError_LibraryNotFound:
            logging.info(
                f"{LOG_NAMESPACE}: pynvml LibraryNotFound error. Not collecting GPU metrics"
            )
        except Exception as e:
            logging.error(
                f"{LOG_NAMESPACE}: unexpected error while trying to set up pynvml. Not "
                f"collecting GPU metrics. Please report this error to "
                f"https://github.com/determined-ai/determined as it should not be "
                f"encountered by users. Error details: {e}"
            )

    @property
    def pynvml_is_available(self) -> bool:
        return self._pynvml is not None

    @property
    def device_count(self) -> int:
        self._safety_check()
        self._device_count = cast(int, self._device_count)
        return self._device_count

    def _safety_check(self) -> None:
        """Before calling any pynvml operations, raise an error if pynvml is not available"""
        if self._pynvml is None:
            raise PynvmlWrapperError(
                "Tried to call a pynvml operation but pynvml is either unavailable or not "
                "functional. Code should check pynvml_is_working before calling any operations."
            )

    def nvml_get_uuid_from_index(self, index: int) -> str:
        self._safety_check()

        if index not in self._index_to_uuid_map.keys():
            raise PynvmlWrapperError(
                f"Unrecognized index {index}. Current index to UUID mapping is: "
                f"{self._index_to_uuid_map}"
            )
        return self._index_to_uuid_map[index]

    def nvml_get_free_memory_by_index(self, index: int) -> float:
        self._safety_check()
        self._pynvml = cast(Any, self._pynvml)

        handle = self._pynvml.nvmlDeviceGetHandleByIndex(index)
        free_memory = self._pynvml.nvmlDeviceGetMemoryInfo(handle).free  # type: float
        return free_memory

    def nvml_get_gpu_utilization_by_index(self, index: int) -> float:
        self._safety_check()
        self._pynvml = cast(Any, self._pynvml)

        handle = self._pynvml.nvmlDeviceGetHandleByIndex(index)
        gpu_util = self._pynvml.nvmlDeviceGetUtilizationRates(handle).gpu  # type: float
        return gpu_util


class Measurement:
    def __init__(self, timestamp: datetime, batch_idx: int, value: float):
        self.timestamp = timestamp
        self.batch_idx = batch_idx
        self.measurement = value


class StartMessage:
    pass


class ShutdownMessage:
    pass


def profiling_metrics_exist(master_url: str, trial_id: str) -> bool:
    """
    Return True if there are already profiling metrics for the trial.
    """
    series_labels = api.get_trial_profiler_available_series(master_url, trial_id)
    return len(series_labels) > 0


SendBatchFnType = Callable[[str, List[TrialProfilerMetricsBatch]], None]
CheckDataExistsFnType = Callable[[str, str], bool]


class ProfilerAgent:
    """
    Agent that collects metrics and sends them to the master.

    The ProfilerAgent needs to be created at the beginning of training and it needs
    to be notified every time the batch_idx increases.

    It will collect System Metrics using a background thread and then batch them and send
    them to the master. You can also collect Timings through the ProfilerAgent with the
    record_timing() method. The timings will be batched and sent to the master.

    Profiling is only active between begin_on_batch and end_after_batch. It will also automatically
    shut down MAX_COLLECTION_SECONDS after starting. When profiling is not active, no system metrics
    are collected and the record_timing function is a no-op.

    Profiling is automatically disabled if profiling metrics already exist in the API. This would
    indicates that the harness restarted due to job failure or being descheduled. Picking up
    profiling in that case introduces issues around multiple data points for the same batch_idx,
    difficult-to-render graphs due to large time gaps, and misleading data due to GPU warmup.

    If is_enabled=False, every method in this class should be a no-op.

    send_batch_fn and check_data_exists_fn are the pieces of code that communicate with the
    master API. They can be replaced with dummy functions to enable testing without a master.
    """

    # dev note: We optimize this code by only creating threads if they will be used.
    # It is essential that any time you interact with a child thread, you gate the code
    # behind a check that the thread exists (using is_enabled, sysmetrics_is_enabled,
    # or timings_is_enabled)

    def __init__(
        self,
        trial_id: str,
        agent_id: str,
        master_url: str,
        profiling_is_enabled: bool,
        global_rank: int,
        local_rank: int,
        begin_on_batch: int,
        end_after_batch: Optional[int] = None,
        send_batch_fn: SendBatchFnType = api.post_trial_profiler_metrics_batches,
        check_data_exists_fn: CheckDataExistsFnType = profiling_metrics_exist,
    ):
        self.current_batch_idx = 0
        self.trial_id = trial_id
        self.agent_id = agent_id
        self.master_url = master_url
        self.profiling_is_enabled_in_experiment_config = profiling_is_enabled
        self.global_rank = global_rank
        self.local_rank = local_rank
        self.begin_on_batch = begin_on_batch
        self.end_after_batch = end_after_batch
        self.send_batch_fn = send_batch_fn
        self.check_data_already_exists_fn = check_data_exists_fn

        self.has_started = False
        self.has_finished = False
        self.disabled_due_to_preexisting_metrics = False

        self.shutdown_lock = threading.Lock()

        # If the ProfilingAgent is disabled, don't waste resources by creating useless threads
        # or making API calls
        if self.is_enabled:
            self.pynvml_wrapper = PynvmlWrapper()

            self.disabled_due_to_preexisting_metrics = self.check_data_already_exists_fn(
                self.master_url, self.trial_id
            )
            if self.disabled_due_to_preexisting_metrics and self.global_rank == 0:
                logging.warning(
                    f"{LOG_NAMESPACE}: ProfilerAgent is disabled because profiling data for "
                    f"this trial already exists. No additional profiling data is generated "
                    f"after a restart."
                )

            # Set up timer thread to stop collecting after MAX_COLLECTION_SECONDS
            self.shutdown_timer = PreemptibleTimer(MAX_COLLECTION_SECONDS, self._end_collection)

            self.send_queue = (
                queue.Queue()
            )  # type: """queue.Queue[Union[List[TrialProfilerMetricsBatch], ShutdownMessage]]"""

            num_producers = 0

            if self.sysmetrics_is_enabled:
                num_producers += 1
                self.sys_metric_collector_thread = SysMetricCollectorThread(
                    trial_id, agent_id, self.send_queue, self.pynvml_wrapper
                )

            if self.timings_is_enabled:
                num_producers += 1
                self.timings_batcher_queue = (
                    queue.Queue()
                )  # type: """queue.Queue[Union[Timing, StartMessage, ShutdownMessage]]"""
                self.timings_batcher_thread = TimingsBatcherThread(
                    trial_id, agent_id, self.timings_batcher_queue, self.send_queue
                )

            self.sender_thread = ProfilerSenderThread(
                self.send_queue, self.master_url, num_producers, self.send_batch_fn
            )

    @staticmethod
    def from_env(env: det.EnvContext, global_rank: int, local_rank: int) -> "ProfilerAgent":
        begin_on_batch, end_after_batch = env.experiment_config.profiling_interval()
        return ProfilerAgent(
            trial_id=env.det_trial_id,
            agent_id=env.det_agent_id,
            master_url=env.master_url,
            profiling_is_enabled=env.experiment_config.profiling_enabled(),
            global_rank=global_rank,
            local_rank=local_rank,
            begin_on_batch=begin_on_batch,
            end_after_batch=end_after_batch,
        )

    # Launch the children threads. This does not mean 'start collecting metrics'
    def start(self) -> None:
        if not self.is_enabled:
            return

        self.sender_thread.start()
        self.shutdown_timer.start()

        if self.sysmetrics_is_enabled:
            self.sys_metric_collector_thread.start()

        if self.timings_is_enabled:
            self.timings_batcher_thread.start()

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
        self.end()

    @property
    def is_enabled(self) -> bool:
        """
        Is the ProfilingAgent supposed to do anything at all?
        If this is false, the entire profiler is a no-op
        """
        if not self.profiling_is_enabled_in_experiment_config:
            return False
        if self.disabled_due_to_preexisting_metrics:
            return False
        return self.sysmetrics_is_enabled or self.timings_is_enabled

    @property
    def sysmetrics_is_enabled(self) -> bool:
        if not self.profiling_is_enabled_in_experiment_config:
            return False
        if self.disabled_due_to_preexisting_metrics:
            return False
        return self.local_rank == 0

    @property
    def timings_is_enabled(self) -> bool:
        if not self.profiling_is_enabled_in_experiment_config:
            return False
        if self.disabled_due_to_preexisting_metrics:
            return False
        return self.global_rank == 0

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

        if self.sysmetrics_is_enabled:
            self.sys_metric_collector_thread.update_batch_idx(self.current_batch_idx)

        # Check if we should start collecting metrics
        if not self.has_started and self.current_batch_idx >= self.begin_on_batch:
            self._begin_collection()

        # Check if we should stop collecting metrics due to batch idx being exceeded
        if (
            self.is_active
            and self.end_after_batch is not None
            and self.current_batch_idx > self.end_after_batch
        ):
            self._end_collection()
            self.shutdown_timer.send_shutdown_signal()

    @contextlib.contextmanager
    def record_timing(self, metric_name: str) -> Iterator[None]:
        if not self.is_enabled or not self.timings_is_enabled or not self.is_active:
            yield
            return

        timing = Timing(metric_name, self.current_batch_idx)
        timing.start()
        yield
        timing.end()
        self.timings_batcher_queue.put(timing)

    def cleanup_timer(self) -> None:
        if not self.is_enabled:
            return
        self.shutdown_timer.send_shutdown_signal()
        self.shutdown_timer.join()

    def _begin_collection(self) -> None:
        if not self.is_enabled:
            return

        # Note: due to its simplicity, sender_thread doesn't need to be activated
        if self.sysmetrics_is_enabled:
            self.sys_metric_collector_thread.activate()

        if self.timings_is_enabled:
            self.timings_batcher_thread.activate()

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

        with self.shutdown_lock:
            if self.has_finished:
                return

            self.shutdown_timer.send_shutdown_signal()

            if self.sysmetrics_is_enabled:
                self.sys_metric_collector_thread.send_shutdown_signal()
                self.sys_metric_collector_thread.join()

            if self.timings_is_enabled:
                self.timings_batcher_thread.send_shutdown_signal()
                self.timings_batcher_thread.join()

            self.sender_thread.join()

            self.has_finished = True


class Timing:
    def __init__(self, name: str, current_batch_idx: int) -> None:
        self.name = name
        self.current_batch_idx = current_batch_idx
        self.start_time = None  # type: Optional[float]
        self.dur = None  # type: Optional[float]

    def start(self) -> None:
        self.start_time = time.time()

    def end(self) -> None:
        check.is_not_none(
            self.start_time,
            "Timing has no start time and end() was called. You probably didn't "
            "run start() before end().",
        )
        self.start_time = cast(float, self.start_time)
        self.dur = time.time() - self.start_time

    def to_measurement(self) -> Measurement:
        check.is_not_none(
            self.start_time,
            "Timing has no start time and to_measurement() was called. You probably didn't "
            "run start() before to_measurement().",
        )
        check.is_not_none(
            self.dur,
            "Timing has no duration and to_measurement() was called. You probably didn't "
            "run end() before to_measurement().",
        )
        self.start_time = cast(float, self.start_time)
        start_time_dt = datetime.fromtimestamp(self.start_time, timezone.utc)
        self.dur = cast(float, self.dur)
        return Measurement(
            timestamp=start_time_dt, batch_idx=self.current_batch_idx, value=self.dur
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
    - GpuFreeMemory = Measured in Gigabytes
    """

    FLUSH_INTERVAL = 10  # How often to make API calls
    MEASUREMENT_INTERVAL = 0.1

    def __init__(
        self, trial_id: str, agent_id: str, send_queue: queue.Queue, pynvml_wrapper: PynvmlWrapper
    ):
        self.current_batch_idx = 0
        self.send_queue = send_queue
        self.control_queue: "queue.Queue[Union['StartMessage', 'ShutdownMessage']]" = queue.Queue()
        self.current_batch = SysMetricBatcher(trial_id, agent_id)
        self.pynvml_wrapper = pynvml_wrapper

        super().__init__(daemon=True)

    def activate(self) -> None:
        """Begin collecting System Metrics"""
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
        gpu_util_collector = GpuUtilCollector(self.pynvml_wrapper)
        gpu_memory_collection = GpuMemoryCollector(self.pynvml_wrapper)

        # Do nothing while we wait for a StartMessage
        msg = self.control_queue.get()
        if isinstance(msg, ShutdownMessage):
            self.send_queue.put(ShutdownMessage())
            return

        # Do initial measurement for rate-based collectors
        net_throughput_collector.reset()
        disk_collector.reset()

        batch_start_time = time.time()
        next_collection = time.time() + self.MEASUREMENT_INTERVAL

        while True:
            # This code is using a trick with the control_queue to sleep/block until the next
            # measurement should be taken, while still being able to respond to a shutdown
            # request immediately.
            now = time.time()
            if now < next_collection:
                sleep_time = next_collection - now
                # a negative timeout will lead to an exception when retrieving from the queue
                sleep_time = max(sleep_time, 0)
                try:
                    msg = self.control_queue.get(timeout=sleep_time)
                    if isinstance(msg, ShutdownMessage):
                        self.send_queue.put(self.current_batch.convert_to_post_format())
                        self.send_queue.put(ShutdownMessage())
                        return
                except queue.Empty:
                    pass

            next_collection += self.MEASUREMENT_INTERVAL

            cpu_util = cpu_util_collector.measure(self.current_batch_idx)
            self.current_batch.add_nongpu_measurement(
                SysMetricType.SIMPLE_CPU_UTIL_METRIC, cpu_util
            )

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

            gpu_util = gpu_util_collector.measure(self.current_batch_idx)
            for gpu_uuid in gpu_util.keys():
                self.current_batch.add_gpu_measurement(
                    SysMetricType.GPU_UTIL_METRIC, gpu_uuid, gpu_util[gpu_uuid]
                )

            gpu_memory = gpu_memory_collection.measure(self.current_batch_idx)
            for gpu_uuid in gpu_memory.keys():
                self.current_batch.add_gpu_measurement(
                    SysMetricType.GPU_FREE_MEMORY_METRIC, gpu_uuid, gpu_memory[gpu_uuid]
                )

            # Check if it is time to flush the batch and start a new batch
            if time.time() - batch_start_time > self.FLUSH_INTERVAL:
                self.send_queue.put(self.current_batch.convert_to_post_format())
                self.current_batch.clear()
                batch_start_time = time.time()


class TimingsBatcherThread(threading.Thread):
    """
    This is a thread that exists solely so that we can batch Timings and ship them to the
    SenderThread every FLUSH_INTERVAL seconds.
    """

    FLUSH_INTERVAL = 10  # How often to make API calls

    def __init__(
        self,
        trial_id: str,
        agent_id: str,
        inbound_queue: queue.Queue,
        send_queue: queue.Queue,
    ) -> None:
        self.inbound_queue = inbound_queue
        self.send_queue = send_queue
        self.current_batch = TimingsBatcher(trial_id, agent_id)
        super().__init__(daemon=True)

    def activate(self) -> None:
        """Begin collecting Timings"""
        self.inbound_queue.put(StartMessage())

    def send_shutdown_signal(self) -> None:
        self.inbound_queue.put(ShutdownMessage())

    def run(self) -> None:
        # Do nothing while we wait for a StartMessage
        while True:
            msg = self.inbound_queue.get()
            if isinstance(msg, StartMessage):
                break
            if isinstance(msg, ShutdownMessage):
                self.send_queue.put(ShutdownMessage())
                return
            else:
                # Ignore any Timings that are received before StartMessage
                pass

        batch_start_time = None  # type: Optional[float]
        while True:
            # Wait for the next Timing to arrive. If it doesn't arrive before the next flush
            # should happen, we stop waiting for the next Timing and go straight to flushing.
            timeout = None
            if batch_start_time is not None:
                time_since_flush = time.time() - batch_start_time
                timeout = self.FLUSH_INTERVAL - time_since_flush

            try:
                message = self.inbound_queue.get(timeout=timeout)
                if isinstance(message, ShutdownMessage):
                    self.send_queue.put(self.current_batch.convert_to_post_format())
                    self.send_queue.put(ShutdownMessage())
                    return
                elif isinstance(message, Timing):
                    if batch_start_time is None:
                        batch_start_time = time.time()
                    self.current_batch.add_timing(message)
                else:
                    logging.fatal(
                        f"ProfilerAgent.TimingsBatcherThread received a message "
                        f"of unexpected type '{type(message)}' from the "
                        f"inbound_queue. This should never happen - there must "
                        f"be a bug in the code."
                    )

            except queue.Empty:
                pass

            check.is_not_none(
                batch_start_time,
                "batch_start_time should never be None. The inbound_queue.get() "
                "should never return and proceed to this piece of code "
                "without batch_start_time being updated to a real timestamp. If "
                "batch_start_time is None, inbound_queue.get() timeout should be "
                "None and the get() should block until a Timing is received.",
            )
            batch_start_time = cast(float, batch_start_time)
            if time.time() - batch_start_time > self.FLUSH_INTERVAL:
                self.send_queue.put(self.current_batch.convert_to_post_format())
                self.current_batch.clear()
                batch_start_time = time.time()


class ProfilerSenderThread(threading.Thread):
    """
    This is a thread that exists solely so that we can make API calls without blocking.
    It has a Queue through which work is sent to the thread. It is aware of the number of
    upstream producers and exits whenever it receives a ShutdownMessage from each producer.
    """

    def __init__(
        self,
        inbound_queue: queue.Queue,
        master_url: str,
        num_producers: int,
        send_batch_fn: SendBatchFnType,
    ) -> None:
        self.master_url = master_url
        self.inbound_queue = inbound_queue
        self.num_producers = num_producers
        self.producers_shutdown = 0
        self.send_batch_fn = send_batch_fn
        super().__init__(daemon=True)

    def run(self) -> None:
        while True:
            message = self.inbound_queue.get()
            if isinstance(message, ShutdownMessage):
                self.producers_shutdown += 1
                if self.num_producers == self.producers_shutdown:
                    return
                else:
                    continue
            self.send_batch_fn(
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


def convert_to_timestamp_str(timestamp: datetime) -> str:
    """
    Convert a datetime object to the string format expected by the API. All timestamps must be
    timezone-aware datetime.datetimes in UTC.
    """
    # https://docs.python.org/3/library/datetime.html#determining-if-an-object-is-aware-or-naive
    assert (
        timestamp.tzinfo is not None and timestamp.tzinfo.utcoffset(timestamp) is not None
    ), "All datetime objects to be serialized must be timezone aware"
    utcoffset = cast(timedelta, timestamp.utcoffset())
    assert utcoffset.total_seconds() == 0, (
        f"All datetime objects to be serialized must be in UTC, but the utcoffset was "
        f"{utcoffset.total_seconds()}"
    )

    return timestamp.isoformat()


def to_post_format(
    measurements: List[Measurement], labels: Dict[str, Any]
) -> TrialProfilerMetricsBatch:
    values, batches, timestamps = [], [], []
    for m in measurements:
        values.append(m.measurement)
        batches.append(m.batch_idx)
        timestamps.append(convert_to_timestamp_str(m.timestamp))
    return TrialProfilerMetricsBatch(values, batches, timestamps, labels)


def make_labels(
    name: str, trial_id: str, agent_id: str, metric_type: str, gpu_uuid_label: str = ""
) -> Dict[str, Any]:
    return {
        "trialId": trial_id,
        "name": name,
        "agentId": agent_id,
        "gpuUuid": gpu_uuid_label,
        "metricType": metric_type,
    }


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

    def convert_to_post_format(self) -> List[TrialProfilerMetricsBatch]:

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
                        make_labels(
                            metric_name, self.trial_id, self.agent_id, SYSTEM_METRIC_TYPE_ENUM
                        ),
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
                                    metric_name,
                                    self.trial_id,
                                    self.agent_id,
                                    SYSTEM_METRIC_TYPE_ENUM,
                                    gpu_uuid_label=gpu_uuid,
                                ),
                            )
                        )

        return trial_profiler_metrics_batches


class TimingsBatcher:
    """
    Data structure to collect batches of Timings and then convert them to the format expected by
    the API
    """

    def __init__(self, trial_id: str, agent_id: str) -> None:
        self.trial_id = trial_id
        self.agent_id = agent_id
        self.clear()

    def clear(self) -> None:
        self.batch = {}  # type: Dict[str, List[Timing]]

    def add_timing(self, timing: Timing) -> None:
        if timing.name not in self.batch.keys():
            self.batch[timing.name] = []
        self.batch[timing.name].append(timing)

    def convert_to_post_format(self) -> List[TrialProfilerMetricsBatch]:
        trial_profiler_metrics_batches = []
        for metric_name in self.batch.keys():
            if len(self.batch[metric_name]) > 0:
                trial_profiler_metrics_batches.append(
                    to_post_format(
                        [timing.to_measurement() for timing in self.batch[metric_name]],
                        make_labels(
                            metric_name, self.trial_id, self.agent_id, TIMING_METRIC_TYPE_ENUM
                        ),
                    )
                )

        return trial_profiler_metrics_batches


GIGA = 1_000_000_000


class ThroughputTracker:
    def __init__(self, name: str, multiplier: float = 1.0):
        self.name = name
        self.multiplier = multiplier
        self.start_time = time.time()
        self.start_val = 0.0

    def add(self, new_val: float, batch_idx: int) -> Measurement:
        """
        Add a new value and return the throughput since the last measurement. The Measurement from
        the first call to add() is meaningless since the starting value is arbitrarily set to 0.
        """
        now = time.time()
        timestamp = datetime.fromtimestamp(now, timezone.utc)
        val_per_sec = (new_val - self.start_val) / (now - self.start_time)
        self.start_val = new_val
        self.start_time = now
        return Measurement(timestamp, batch_idx, val_per_sec * self.multiplier)


class SimpleCpuUtilCollector:
    def measure(self, batch_idx: int) -> Measurement:
        cpu_util = psutil.cpu_percent()
        timestamp = datetime.now(timezone.utc)
        return Measurement(timestamp, batch_idx, cpu_util)


class FreeMemoryCollector:
    def measure(self, batch_idx: int) -> Measurement:
        free_mem_bytes = psutil.virtual_memory().available
        timestamp = datetime.now(timezone.utc)
        return Measurement(timestamp, batch_idx, free_mem_bytes / GIGA)


class NetThroughputCollector:
    def __init__(self) -> None:
        self.sent_throughput = ThroughputTracker("Network Sent (Gbit/s)", multiplier=8 / GIGA)
        self.recv_throughput = ThroughputTracker("Network Recv (Gbit/s)", multiplier=8 / GIGA)

    def reset(self) -> None:
        # Discard initial batch that is meaningless
        net = psutil.net_io_counters()
        self.sent_throughput.add(net.bytes_sent, batch_idx=0)
        self.recv_throughput.add(net.bytes_recv, batch_idx=0)

    def measure(self, batch_idx: int) -> Tuple[Measurement, Measurement]:
        net = psutil.net_io_counters()
        sent = self.sent_throughput.add(net.bytes_sent, batch_idx=batch_idx)
        recv = self.recv_throughput.add(net.bytes_recv, batch_idx=batch_idx)
        return sent, recv


class DiskReadWriteRateCollector:
    def __init__(self) -> None:
        self.read_throughput_tracker = ThroughputTracker("Disk Read (bytes/s)")
        self.write_throughput_tracker = ThroughputTracker("Disk Write (bytes/s)")
        self.iops = ThroughputTracker("Disk IOPS")

    def reset(self) -> None:
        # Discard initial batch that is meaningless
        disk = psutil.disk_io_counters()
        self.read_throughput_tracker.add(disk.read_bytes, batch_idx=0)
        self.write_throughput_tracker.add(disk.write_bytes, batch_idx=0)
        self.iops.add(disk.read_count + disk.write_count, batch_idx=0)

    def measure(self, batch_idx: int) -> Tuple[Measurement, Measurement, Measurement]:
        """Return tuple of (Read, Write, IOPS) Measurements"""
        disk = psutil.disk_io_counters()
        read_throughput = self.read_throughput_tracker.add(disk.read_bytes, batch_idx=batch_idx)
        write_throughput = self.write_throughput_tracker.add(disk.write_bytes, batch_idx=batch_idx)
        iops = self.iops.add(disk.read_count + disk.write_count, batch_idx=batch_idx)
        return read_throughput, write_throughput, iops


class GpuUtilCollector:
    def __init__(self, pynvml_wrapper: PynvmlWrapper):
        self.pynvml_wrapper = pynvml_wrapper

    def measure(self, batch_idx: int) -> Dict[str, Measurement]:
        """
        Collect GPU utilization for each GPU. Returns empty dict if unable
        to measure GPU utilization
        """
        if not self.pynvml_wrapper.pynvml_is_available:
            return {}

        measurements = {}
        timestamp = datetime.now(timezone.utc)
        try:
            num_gpus = self.pynvml_wrapper.device_count
            for i in range(num_gpus):
                gpu_uuid = self.pynvml_wrapper.nvml_get_uuid_from_index(i)
                util = self.pynvml_wrapper.nvml_get_gpu_utilization_by_index(i)
                measurements[gpu_uuid] = Measurement(timestamp, batch_idx, util)
            return measurements

        except Exception as e:
            logging.warning(f"{LOG_NAMESPACE}: error while measuring GPU utilization: {e}")
            return {}


class GpuMemoryCollector:
    def __init__(self, pynvml_wrapper: PynvmlWrapper):
        self.pynvml_wrapper = pynvml_wrapper

    def measure(self, batch_idx: int) -> Dict[str, Measurement]:
        """
        Collect GPU memory for each GPU. Returns empty dict if unable to
        measure GPU memory
        """
        if not self.pynvml_wrapper.pynvml_is_available:
            return {}

        measurements = {}
        timestamp = datetime.now(timezone.utc)
        try:
            num_gpus = self.pynvml_wrapper.device_count
            for i in range(num_gpus):
                gpu_uuid = self.pynvml_wrapper.nvml_get_uuid_from_index(i)
                free_memory = self.pynvml_wrapper.nvml_get_free_memory_by_index(i)
                measurements[gpu_uuid] = Measurement(timestamp, batch_idx, free_memory)
            return measurements

        except Exception as e:
            logging.warning(f"{LOG_NAMESPACE}: error while measuring GPU memory: {e}")
            return {}
