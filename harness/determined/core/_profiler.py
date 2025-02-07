import abc
import collections
import datetime
import logging
import threading
import time
from typing import Any, Callable, Dict, List, Optional, Union, cast

import psutil

from determined import core
from determined.common import constants

try:
    import pynvml
except ImportError:
    pynvml = None

logger = logging.getLogger("determined.core")


class ProfilerContext:
    """Gives access to the system profiling feature within Determined.

    It is responsible for collecting system metrics at specified time intervals and reporting
    them to the master. When it is turned on, it spawns two threads that run in the background
    that collect and send profiling metrics to a Determined master, which are cleaned up when
    the ``core.Context`` exits or the profiler is turned off.

    This class is automatically created when the ``core.Context`` is initialized and can be
    turned on/off as such:

    .. code::

        with det.core.init() as core_context:
            core_context.profiler.on()
            ...
            core_context.profiler.off()

    """

    def __init__(
        self,
        agent_id: str,
        metrics: core._MetricsContext,
        distributed: core.DistributedContext,
    ) -> None:
        self._metrics = metrics
        self._agent_id = agent_id
        self._distributed = distributed
        self._on = False

        self._collector: Optional[_Collector] = None

    def on(self, sampling_interval: int = 1, samples_per_report: int = 10) -> None:
        """Turns system profiling functionality on.

        This method creates two threads, one that collects system metrics at specified time
        intervals, and another that ships them to the master.

        These metrics are persisted in the database and can be viewed in the Determined web UI
        for the associated trial.

        .. note::

            This method is idempotent: if profiling is already on, this method is effectively a
            no-op.

        Arguments:
            sampling_interval: time (in seconds) between each metric collection.
            samples_per_report: number of samples to collect before aggregating for report.

        """
        if self._on:
            return

        if sampling_interval < 0.1:
            raise ValueError(f"Sampling interval must be > 0.1, got {sampling_interval}.")

        if not isinstance(samples_per_report, int) or samples_per_report < 1:
            raise ValueError(
                f"Samples per report specifies the number of samples to aggregate before "
                f"reporting the metric. It must be an int > 1, but was specified as "
                f"{samples_per_report}."
            )

        # Currently, metrics collected are scoped at the machine level, so we only collect metrics
        # on the chief worker of each node.
        if self._distributed.local_rank != 0:
            return

        logger.info("Starting system metrics profiling.")

        self._collector = _Collector(
            metrics=self._metrics,
            agent_id=self._agent_id,
            sampling_interval=sampling_interval,
            aggregation_period=samples_per_report,
        )

        self._collector.start()
        self._on = True

    def off(self) -> None:
        """Turns off profiling.

        Sets the internal state of this class and stops any threads that where created.

        .. note::

            This method is idempotent: if profiling is already off, this method is effectively a
            no-op.
        """
        if not self._on:
            return

        logger.info("Stopping system metrics profiling.")
        self._close()
        self._on = False

    def _close(self) -> None:
        """Shuts down any threads that were created."""
        if self._collector:
            self._collector.stop()


class DummyProfilerContext(ProfilerContext):
    """Drop-in replacement of ``ProfilerContext``.

    Used by the ``core.Context`` for cases when profiling cannot run.
    """

    def __init__(self) -> None:
        pass

    def on(self, sampling_interval: int = 1, samples_per_report: int = 10) -> None:
        pass

    def off(self) -> None:
        pass

    def _close(self) -> None:
        pass


# Define the types for reading and collection
TimestampedValue = collections.namedtuple("TimestampedValue", ["timestamp", "value"])
Reading = Dict[str, Union[float, "Reading"]]
Collection = Dict[str, Union[List[TimestampedValue], "Collection"]]


def _add_reading(collection: Collection, reading: Reading, timestamp: float) -> None:
    """Recursively adds samples from a Reading dict to a Collection dict.

    This function traverses a nested dictionary of readings, where leaves are float
    samples, and replicates the structure in a collection dictionary, appending the
    samples to lists at the corresponding leaves.

    Args:
        reading (Reading): A nested dictionary with arbitrary depth where the leaves are
            float samples.
        collection (Collection): A nested dictionary to which the samples from the reading
            will be added. The structure mirrors that of the reading, but the leaves are
            lists of floats.

    Raises:
        ValueError: If an unsupported value type is encountered in the reading dictionary.
    """
    for k, v in reading.items():
        if isinstance(v, float):
            if k not in collection:
                collection[k] = []
            reading_list = collection[k]
            # the collection should have the same structure as the reading
            assert isinstance(reading_list, list)
            reading_list.append(TimestampedValue(timestamp, v))
        elif isinstance(v, dict):
            if k not in collection:
                collection[k] = {}
            child_collection = collection[k]
            assert isinstance(child_collection, dict)
            cast(Collection, child_collection)
            _add_samples(child_collection, v, timestamp)
        else:
            raise ValueError(f"Unsupported value '{v}' found of type {type(v)}")


def _agg_collection(
    collection: Collection, aggregator: Callable[[List[TimestampedValue]], float]
) -> Reading:
    """Applies an aggregation function to each leaf in the collection and returns a new dict.

    Traverses a nested dictionary (collection) where each leaf is a list of floats. Applies
    the passed aggregator function to each list, replacing it with a single float in the
    resulting dictionary, which mirrors the structure of the input collection.

    Args:
        collection: A nested dictionary where the leaves are lists of floats.
        aggregator: The aggregation function to apply to each list of floats.

    Returns:
        Reading: A new nested dictionary with the same structure as collection, where each
        list of floats has been replaced by a single float, the result of applying agg_vals.
    """
    aggregated = {}
    for k, v in collection.items():
        if isinstance(v, dict):
            aggregated[k] = _agg_collection(v, aggregator)
        elif isinstance(v, list):
            aggregated[k] = aggregator(v)
    return aggregated


class _MetricGroupCollector(metaclass=abc.ABCMeta):
    """Abstract class that samples and collects groups of system metrics.

    This class is subclassed by metric groups that implement their respective metrics collection
    logic.
    """

    def __init__(self) -> None:
        self.metric_samples: Collection = {}

    @property
    @abc.abstractmethod
    def group(self) -> str:
        pass

    @abc.abstractmethod
    def aggregate_values(self, values: List[TimestampedValue]) -> float:
        """Aggregate a list of values into a single value depending on the metric."""
        pass

    @abc.abstractmethod
    def take_reading(self) -> Reading:
        """Sample all metrics for this group.

        Records metrics as a dictionary mapping each metric name to its metric value and appends
        this value to the samples on this class (``self.metric_samples``).

        Certain metrics may be associated with additional labels (i.e. GPU UUIDs) in which case
        the recorded dictionary should be nested with the label as keys. For example:

        .. code:: python
            {
                "GPU-UUID-1": {
                    "gpu_util": 0.12,
                    "gpu_free_memory": 123.45,
                },
                "GPU-UUID-2": {
                    "gpu_util": 0.23,
                    "gpu_free_memory": 234.56,
                }
            }
        """
        pass

    def aggregate(self) -> Reading:
        """Merge the list of `self.metric_samples` into a single dictionary with aggregate values.

        This method should return a single dictionary where the values represent meaningful
        aggregation for this metric group.

        By default, this averages all the values across `self.metric_samples` by keys. This should
        be the aggregation method for most if not all metrics, but individual metric group
        collectors should override this method should they need an alternate aggregation method.
        """
        return _agg_collection(self.metric_samples, self.aggregate_values)

    def reset(self) -> None:
        self.metric_samples = {}

    def add_reading(self) -> None:
        """Take a single Reading and add it to a Collection of them."""
        _add_reading(self.metric_samples, self.take_reading(), time.time())


class _Network(_MetricGroupCollector):
    @property
    def group(self) -> str:
        return "network"

    def aggregate_values(self, list_of_values: List[TimestampedValue]) -> float:
        """rate between first and last value in the list of values."""
        sorted_values = sorted(list_of_values, key=lambda x: x.timestamp)
        interval_len = sorted_values[-1].timestamp - sorted_values[0].timestamp
        val_diff = sorted_values[-1].value - sorted_values[0].value
        return val_diff / interval_len

    def take_reading(self) -> Reading:
        vals = psutil.net_io_counters()

        return {
            "bytes_sent": vals.bytes_sent,
            "bytes_recv": vals.bytes_recv,
        }


class _Disk(_MetricGroupCollector):
    _disk_paths = ["/", constants.SHARED_FS_CONTAINER_PATH]

    def __init__(self) -> None:
        # Set initial values for throughput calculations.
        self._interval_start_ts = time.time()
        self._interval_start_vals = psutil.disk_io_counters()

        # Initialize accessible disk paths.
        self._paths = []
        for path in self._disk_paths:
            try:
                psutil.disk_usage(path)
                self._paths.append(path)
            except Exception:
                pass

        super().__init__()

    @property
    def group(self) -> str:
        return "disk"

    def aggregate_values(self, list_of_values: List[TimestampedValue]) -> float:
        """rate between first and last value in the list of values."""
        sorted_values = sorted(list_of_values, key=lambda x: x.timestamp)
        interval_len = sorted_values[-1].timestamp - sorted_values[0].timestamp
        val_diff = sorted_values[-1].value - sorted_values[0].value
        return val_diff / interval_len

    def take_reading(self) -> Reading:
        vals = psutil.disk_io_counters()
        assert vals

        metrics = {
            "disk_iops": vals.read_count + vals.write_count,
            "disk_throughput_read": vals.read_bytes,
            "disk_throughput_write": vals.write_bytes,
        }

        for path in self._paths:
            disk_usage = psutil.disk_usage(path)
            metrics.update({path: {"disk_util": disk_usage.percent}})
        return metrics


class _Memory(_MetricGroupCollector):
    # TODO: Actually... maybe we shouldn't record a reading when an object is instantiated.

    @property
    def group(self) -> str:
        return "memory"

    def aggregate_values(self, list_of_values: List[TimestampedValue]) -> float:
        """Average the values in the list of values."""
        return sum([v.value for v in list_of_values]) / len(list_of_values)

    def take_reading(self) -> Reading:
        free_mem_bytes = psutil.virtual_memory().available
        return {
            "memory_free": free_mem_bytes / 1e9,
        }


class _CPU(_MetricGroupCollector):
    @property
    def group(self) -> str:
        return "memory"

    def aggregate_values(self, list_of_values: List[TimestampedValue]) -> float:
        """Average the values in the list of values."""
        return sum([v.value for v in list_of_values]) / len(list_of_values)

    def take_reading(self) -> Reading:
        cpu_util = psutil.cpu_percent()
        return {
            "cpu_util_simple": cpu_util,
        }


class _GPU(_MetricGroupCollector):
    def __init__(self) -> None:
        self._pynvml_device_handles: Dict[str, Any] = {}
        if pynvml:
            self._init_pynvml()
        else:
            logging.warning("pynvml module not found. GPU metrics will not be collected.")

        super().__init__()

    @property
    def group(self) -> str:
        return "gpu"

    def aggregate_values(self, list_of_values: List[TimestampedValue]) -> float:
        """Average the values in the list of values."""
        return sum([v.value for v in list_of_values]) / len(list_of_values)

    def _init_pynvml(self) -> None:
        """Initialize the pynvml library and validate methods.

        Call the NVML library methods we'll be using to validate that they are accessible.
        Sometimes NVML initializes successfully but individual device methods will fail.

        If any NVML method fails for any GPU device, no GPU metrics will be collected for
        all GPU devices.
        """
        assert pynvml
        try:
            pynvml.nvmlInit()
            num_gpus = pynvml.nvmlDeviceGetCount()
            for i in range(num_gpus):
                handle = pynvml.nvmlDeviceGetHandleByIndex(i)
                uuid = pynvml.nvmlDeviceGetUUID(handle)
                pynvml.nvmlDeviceGetMemoryInfo(handle)
                pynvml.nvmlDeviceGetUtilizationRates(handle)
                self._pynvml_device_handles[str(uuid)] = handle
        except pynvml.NVMLError as ne:
            self._pynvml_device_handles = {}
            logging.info(f"Error accessing NVML {ne}. GPU metrics will not be collected.")

    def take_reading(self) -> Reading:
        metrics = {}

        for uuid, handle in self._pynvml_device_handles.items():
            assert pynvml
            gpu_util = pynvml.nvmlDeviceGetUtilizationRates(handle).gpu
            free_memory = pynvml.nvmlDeviceGetMemoryInfo(handle).free
            metrics.update(
                {
                    uuid: {
                        "gpu_util": gpu_util,
                        "gpu_free_memory": free_memory,
                    }
                }
            )
        return metrics


class _Collector(threading.Thread):
    """Samples metrics from a set list of ``_MetricGroupCollector``s.

    Collects the sampled metrics and puts them into a queue to be consumed by the ``_Shipper``.
    """

    def __init__(
        self,
        metrics: core._MetricsContext,
        agent_id: str,
        sampling_interval: int = 1,
        aggregation_period: int = 10,
    ):
        self._sampling_interval = sampling_interval
        self._aggregation_period = aggregation_period
        self._metrics = metrics
        self._agent_id = agent_id
        self._metric_collectors = [
            _GPU(),
            _CPU(),
            _Memory(),
            _Disk(),
            _Network(),
        ]
        self._should_exit = threading.Event()

        super().__init__(daemon=True)

    def run(self) -> None:
        try:
            while not self._should_exit.is_set():
                # Collect number of samples the aggregation period calls for across all groups.
                for _ in range(self._aggregation_period):
                    next_collection_ts = time.time() + self._sampling_interval
                    for collector in self._metric_collectors:
                        collector.add_sample()
                    wait_ts = max(next_collection_ts - time.time(), 0)
                    self._should_exit.wait(timeout=wait_ts)

                self._aggregate_metrics()
        finally:
            self.stop()

    def stop(self) -> None:
        self._should_exit.set()

    def _aggregate_metrics(self) -> None:
        """Aggregate metrics across groups and put in outbound queue."""
        for collector in self._metric_collectors:
            aggregated_metrics = collector.aggregate()
            if not aggregated_metrics:
                continue
            timestamp = datetime.datetime.now(datetime.timezone.utc)
            # Append agent ID to all metrics before reporting.
            aggregated_metrics = {self._agent_id: aggregated_metrics}

            self._metrics.report(
                group=collector.group,
                metrics=aggregated_metrics,
                report_time=timestamp,
            )

            # Reset the aggregated metrics on each group collector for the next iteration.
            collector.reset()
