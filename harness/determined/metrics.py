import threading
import psutil
import pynvml
import datetime
from typing import Any, Dict, Iterator, Optional
import queue
import time
from determined.common import api

def humanize_float(num): return "{0:,.2f}".format(num)


class QuickTimer:
    def __init__(self, name):
        self.name = name
        self.start = time.time()

    def stop(self):
        end = time.time()
        # print(f"[TIMER] {self.name}: {humanize_float(end-self.start)}s")





SYSTEM_METRIC_TYPE_ENUM = "PROFILER_METRIC_TYPE_SYSTEM"

class Measurement:
    def __init__(self, timestamp: datetime.datetime, batch_idx, value):
        self.timestamp = timestamp
        self.batch_idx = batch_idx
        self.measurement = value


# SimpleCpuUtilization = Measured in percent
# FreeMemory = Measured in Gigabytes
# NetworkSentThroughput = Measured in Gigabit/s
# NetworkRecvThroughput = Measured in Gigabit/s
# DiskIops
# DiskReadThroughput = Measured in bytes/second
# DiskWriteThroughput = Measured in bytes/second
# GpuUtilization = Measured in percent




class MetricsHolder:
    # TODO: Change these constants to match ERD/what backend expects
    GPU_UTIL_METRIC = "GPU_UTIL"
    NET_THRU_SENT_METRIC = "NET_THRU_SENT"
    NET_THRU_RECV_METRIC = "NET_THRU_RECV"
    DISK_IOPS_METRIC = "DISK_IOPS"
    DISK_THRU_READ_METRIC = "DISK_THRU_READ"
    DISK_THRU_WRITE_METRIC = "DISK_THRU_WRITE"
    FREE_MEM_METRIC = "FREE_MEM"
    SIMPLE_CPU_UTIL_METRIC = "SIMPLE_CPU_UTIL"


    def __init__(self, trial_id, agent_id):
        self.trial_id = trial_id
        self.agent_id = agent_id
        self.reset()

    def reset(self):
        self.current_measurements = {
            MetricsHolder.GPU_UTIL_METRIC: {},
            # "GPU_MEM": [],
            MetricsHolder.NET_THRU_SENT_METRIC: [],
            MetricsHolder.NET_THRU_RECV_METRIC: [],
            # "DISK_FREE": [],
            MetricsHolder.DISK_IOPS_METRIC: [],
            MetricsHolder.DISK_THRU_READ_METRIC: [],
            MetricsHolder.DISK_THRU_WRITE_METRIC: [],
            MetricsHolder.FREE_MEM_METRIC: [],
            MetricsHolder.SIMPLE_CPU_UTIL_METRIC: []
        }

    def add_nongpu_measurement(self, metric_type, measurement):
        assert metric_type in self.current_measurements.keys(), f"Tried to add unknown type of metric: {metric_type}"
        self.current_measurements[metric_type].append(measurement)

    def add_gpu_measurement(self, metric_type, gpu_uuid, measurement):
        assert metric_type in self.current_measurements.keys(), f"Tried to add unknown type of metric: {metric_type}"
        if gpu_uuid not in self.current_measurements[metric_type].keys():
            self.current_measurements[metric_type][gpu_uuid] = []
        self.current_measurements[metric_type][gpu_uuid].append(measurement)

    def to_post_timestamp(self, timestamp: datetime.datetime):
        assert isinstance(timestamp, datetime.datetime), \
            f"Input to conversion function must be a datetime object. Instead got {type(timestamp)}"
        return timestamp.isoformat() + "Z"

    def convert_to_post_format(self):
        post_formatted = []
        for metric_type in self.current_measurements.keys():
            if metric_type != MetricsHolder.GPU_UTIL_METRIC and len(self.current_measurements[metric_type]) > 0:
                single_metric_batch = {
                    "values": [],
                    "batches": [],
                    "timestamps": [],
                    "labels": {
                        "trialId": self.trial_id,
                        "name": metric_type,
                        "agentId": self.agent_id,
                        "gpuUuid": "",
                        "metricType": SYSTEM_METRIC_TYPE_ENUM
                    }
                }
                for measurement in self.current_measurements[metric_type]:
                    single_metric_batch["values"].append(measurement.measurement)
                    single_metric_batch["batches"].append(measurement.batch_index)
                    single_metric_batch["timestamp"].append(self.to_post_timestamp(measurement.timestamp))
                post_formatted.append(single_metric_batch)

            # GPU Metrics need to be grouped by GPU UUID
            if metric_type == MetricsHolder.GPU_UTIL_METRIC and len(self.current_measurements[metric_type].keys()) > 0:
                for gpu_uuid in self.current_measurements[metric_type].keys():
                    single_metric_batch = {
                        "values": [],
                        "batches": [],
                        "timestamps": [],
                        "labels": {
                            "trialId": self.trial_id,
                            "name": metric_type,
                            "agentId": self.agent_id,
                            "gpuUuid": gpu_uuid,
                            "metricType": SYSTEM_METRIC_TYPE_ENUM
                        }
                    }
                    for measurement in self.current_measurements[metric_type][gpu_uuid]:
                        single_metric_batch["values"].append(measurement.measurement)
                        single_metric_batch["batches"].append(measurement.batch_index)
                        single_metric_batch["timestamp"].append(self.to_post_timestamp(measurement.timestamp))
                    post_formatted.append(single_metric_batch)
        return post_formatted




class SystemMetricsThread(threading.Thread):
    """
    Background thread for collecting system metrics at a high granularity and shipping them to the master
    """
    # master_address = None
    # master_port = None
    # use_tls = None
    # scheme = "https" if use_tls else "http"
    # self.master_url = f"{scheme}://{master_address}:{master_port}"

    ACTIVE_POLL_INTERVAL = 1  # Check if metric collection has been turned on/off every 1 second
    FLUSH_INTERVAL = 5  # Send batched info every 5 seconds
    MEASUREMENT_INTERVAL = 0.1


    def __init__(self, master_url: str):

        self.verbose = True
        self.log("Creating SystemMetricsThread")

        self.is_active = True
        self.current_batch = 1

        self.dispatch_queue = queue.Queue()

        # TODO: Correctly extract these values
        TRIAL_ID = 0
        AGENT_ID = 0

        self.sending_thread = SystemMetricsSendingThread(self.dispatch_queue, master_url)
        self.sending_thread.start()


        self.current_metrics = MetricsHolder(TRIAL_ID, AGENT_ID)

        self.quitting = False
        super().__init__()

    def log(self, *s):
        if self.verbose:
            print("[SystemMetricsThread]", *s)

    def run(self) -> None:
        last_measurement_time = None
        batch_start_time = None
        cpu_util_collector = SimpleCpuUtilCollector()
        gpu_util_collector = GpuUtilCollector()
        network_throughput_collector = NetThroughputCollector()
        free_memory_collector = FreeMemoryCollector()
        disk_collector = DiskReadWriteRateCollector()

        while True:
            if self.quitting:
                break

            if not self.is_active:
                time.sleep(1)
                continue

            # TODO: Check if we should shut down due to max duration exceeded or max batch idx exceeded


            # One-time initialization
            if last_measurement_time is None:
                last_measurement_time = time.time()
                batch_start_time = time.time()
                network_throughput_collector.reset()
                disk_collector.reset()




            # Check if it is time to take a new measurement
            if time.time() - last_measurement_time > self.MEASUREMENT_INTERVAL:
                immutable_batch_idx = self.current_batch
                cpu_util_measurement = cpu_util_collector.measure(immutable_batch_idx)
                gpu_util_measurements = gpu_util_collector.measure(immutable_batch_idx)
                net_thru_sent_measurement, net_thru_recv_measurement = network_throughput_collector.measure(immutable_batch_idx)
                free_memory_measurement = free_memory_collector.measure(immutable_batch_idx)
                disk_read_thru_measurement, disk_write_thru_measurement, iops_measurement = disk_collector.measure(immutable_batch_idx)

                for gpu_uuid in gpu_util_measurements.keys():
                    self.current_metrics.add_gpu_measurement(MetricsHolder.GPU_UTIL_METRIC, gpu_uuid, gpu_util_measurements[gpu_uuid])

                self.current_metrics.add_nongpu_measurement(MetricsHolder.NET_THRU_SENT_METRIC, net_thru_sent_measurement)
                self.current_metrics.add_nongpu_measurement(MetricsHolder.NET_THRU_RECV_METRIC, net_thru_recv_measurement)
                self.current_metrics.add_nongpu_measurement(MetricsHolder.DISK_IOPS_METRIC, iops_measurement)
                self.current_metrics.add_nongpu_measurement(MetricsHolder.DISK_THRU_READ_METRIC, disk_read_thru_measurement)
                self.current_metrics.add_nongpu_measurement(MetricsHolder.DISK_THRU_WRITE_METRIC, disk_write_thru_measurement)
                self.current_metrics.add_nongpu_measurement(MetricsHolder.FREE_MEM_METRIC, free_memory_measurement)
                self.current_metrics.add_nongpu_measurement(MetricsHolder.SIMPLE_CPU_UTIL_METRIC, cpu_util_measurement)

                last_measurement_time = time.time()


            # Check if it is time to flush the batch and start a new batch
            if time.time() - batch_start_time > self.FLUSH_INTERVAL:
                self.enqueue_for_async_send(self.current_metrics.convert_to_post_format())
                self.current_metrics.reset()
                batch_start_time = time.time()

            time.sleep(0.02)

    def update_current_batch(self, new_current_batch):
        self.current_batch = new_current_batch

    def enqueue_for_async_send(self, metric_batch):
        # This method can theoretically raise a FULL error, but SimpleQueues are unbounded so
        # I don't think it will ever happen (https://docs.python.org/3/library/queue.html#queue.Queue.put)
        # self.log("Enqueuing metric batch", metric_batch)
        self.dispatch_queue.put_nowait(metric_batch)

    def __enter__(self) -> "SystemMetricsThread":
        self.start()
        return self

    def __exit__(self, *arg: Any) -> None:
        self.quitting = True







# This is a thread that exists solely so that we can make API calls without blocking
# It has a SimpleQueue through which work is sent to the thread
class SystemMetricsSendingThread(threading.Thread):
    def __init__(self, inbound_queue: queue.Queue, master_url: str) -> None:
        print("[SystemMetricsSendingThread] Creating SystemMetricsSendingThread")
        self.master_url = master_url
        self.POLL_INTERVAL_SECS = 0.5
        self.inbound_queue = inbound_queue

        self.short_circuit_counter = 0
        self.all = []

        self.quitting = False
        super().__init__()

    def run(self) -> None:
        while True:
            if self.quitting:
                break

            try:
                batch_to_send = self.inbound_queue.get_nowait()
            except queue.Empty:
                time.sleep(self.POLL_INTERVAL_SECS)
                continue

            # print(f"[SystemMetricsSendingThread] Sending a batch. {humanize_float(time.time())}")
            self.send_batch(batch_to_send)

    # This is a blocking operation that must handle all exceptions gracefully
    def send_batch(self, post_bodies):
        # TODO: Handle exceptions and automatically retry X times and then log and drop the batch
        # TODO: Change API so we don't have to make so many independent API calls

        for post_body in post_bodies:
            api.post_trial_profiler_metrics(
                self.master_url,
                post_body["values"],
                post_body["batches"],
                post_body["timestamps"],
                post_body["labels"],
            )

    def quit(self):
        self.quitting = True





GIGA = 1_000_000_000

class SimpleCpuUtilCollector:
    def measure(self, batch_idx):
        timer = QuickTimer("SimpleCpuUtilCollector")
        cpu_util = psutil.cpu_percent()
        timestamp = datetime.datetime.utcnow()
        timer.stop()
        return Measurement(timestamp, batch_idx, cpu_util)



class FreeMemoryCollector:
    # We choose to report free memory instead of available memory because it is useful to
    # be able to see memory usage for cached files, but we could change to available instead
    # https://psutil.readthedocs.io/en/latest/#psutil.virtual_memory
    def measure(self, batch_idx):
        timer = QuickTimer("FreeMemoryCollector")
        free_mem_bytes = psutil.virtual_memory().free
        timestamp = datetime.datetime.utcnow()
        timer.stop()
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
        timer = QuickTimer("NetThroughputCollector")
        net = psutil.net_io_counters()
        end_time = time.time()

        sent_bytes_delta = net.bytes_sent - self.start_sent
        recv_bytes_delta = net.bytes_recv - self.start_recv

        time_delta = end_time - self.start_time

        self.start_time = end_time
        self.start_sent = net.bytes_sent
        self.start_recv = net.bytes_recv

        sent_throughput_bytes_per_second = sent_bytes_delta / time_delta
        recv_throughput_bytes_per_second = recv_bytes_delta / time_delta

        sent_throughput_gigabits_per_second = sent_throughput_bytes_per_second * 8 * GIGA
        recv_throughput_gigabits_per_second = recv_throughput_bytes_per_second * 8 * GIGA
        timer.stop()

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
        timer = QuickTimer("DiskReadWriteRateCollector")
        disk = psutil.disk_io_counters()
        end_time = time.time()

        read_bytes_delta = disk.read_bytes - self.start_read_bytes
        write_bytes_delta = disk.write_bytes - self.start_write_bytes

        read_count_delta = disk.read_count - self.start_read_count
        write_count_delta = disk.write_count - self.start_write_count

        time_delta = end_time - self.start_time

        self.start_time = end_time
        self.start_read_bytes = disk.read_bytes
        self.start_write_bytes = disk.write_bytes
        self.start_read_count = disk.read_count
        self.start_write_count = disk.write_count

        read_throughput_bytes_per_second = read_bytes_delta / time_delta
        write_throughput_bytes_per_second = write_bytes_delta / time_delta

        read_throughput_count_per_second = read_count_delta / time_delta
        write_throughput_count_per_second = write_count_delta / time_delta

        timestamp = datetime.datetime.fromtimestamp(end_time)
        read_throughput = Measurement(timestamp, batch_idx, read_throughput_bytes_per_second)
        write_throughput = Measurement(timestamp, batch_idx, write_throughput_bytes_per_second)
        iops = Measurement(timestamp, batch_idx, read_throughput_count_per_second + write_throughput_count_per_second)

        timer.stop()
        return read_throughput, write_throughput, iops



class GpuUtilCollector:
    def __init__(self):
        pynvml.nvmlInit()
        self.num_gpus = pynvml.nvmlDeviceGetCount()

    def measure(self, batch_idx):
        timer = QuickTimer("GpuUtilCollector")
        measurements = {}
        timestamp = datetime.datetime.utcnow()
        for i in range(self.num_gpus):
            handle = pynvml.nvmlDeviceGetHandleByIndex(i)
            try:
                util = pynvml.nvmlDeviceGetUtilizationRates(handle)
                gpu_util = util.gpu

            except pynvml.NVMLError as err:
                continue
            measurements[handle] = Measurement(timestamp, batch_idx, gpu_util)
        timer.stop()
        return measurements


# TODO: Haven't figured out how to collect GPU memory usage yet
class GpuMemory:
    pass


# The psutil way of measuring this is to query by a path. Should we just query /?
class DiskFree:
    pass
