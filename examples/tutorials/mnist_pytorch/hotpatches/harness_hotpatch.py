"""
The entrypoint for the isolated environment we use to run trials.

Basic workflow is:
  * Agent launches a new container that has this script as its
    entrypoint. The agent passes along various parameters (e.g., master
    address, workload) via environment variables.

  * The script establishes a WebSocket connection back to the master,
    and sends a TRIAL_RUNNER_STARTUP message including the container's
    initial workload. We then start running the specified workload.

  * When the initial workload is complete, the trial runner notifies the
    master via a WORKLOAD_COMPLETED message.

  * The master sends a RUN_WORKLOAD message to the trial runner to ask
    it to do some work, e.g., run a step of the current trial,
    checkpoint the model to persistent storage, or compute the model's
    current validation metrics. This can happen many times to run multiple
    steps of the same trial in a row.

  * Eventually, the master asks the trial runner to exit via a TERMINATE
    message.

"""
import contextlib
import distutils.util
import faulthandler
import json
import logging
import os
import pathlib
import sys
from typing import Any, Dict, Iterator, Optional
import threading

import simplejson

import determined as det
import determined_common
from determined import gpu, horovod, layers, load, workload
from determined_common import constants, storage

print("HOTPATCH - harness hotpatch is working")

ENVIRONMENT_VARIABLE_KEYS = {
    "DET_MASTER_ADDR",
    "DET_MASTER_PORT",
    "DET_AGENT_ID",
    "DET_SLOT_IDS",
    "DET_CONTAINER_ID",
    "DET_USE_GPU",
    "DET_EXPERIMENT_ID",
    "DET_TRIAL_ID",
    "DET_TRIAL_SEED",
    "DET_EXPERIMENT_CONFIG",
    "DET_HPARAMS",
    "DET_INITIAL_WORKLOAD",
    "DET_LATEST_CHECKPOINT",
    "DET_WORKLOAD_MANAGER_TYPE",
    "DET_RENDEZVOUS_PORTS",
    "DET_TRIAL_RUNNER_NETWORK_INTERFACE",
}


@contextlib.contextmanager
def maybe_load_checkpoint(
        storage_mgr: storage.StorageManager, checkpoint: Optional[Dict[str, Any]]
) -> Iterator[Optional[pathlib.Path]]:
    """
    Either wrap a storage_mgr.restore_path() context manager, or be a noop
    context manager if there is no checkpoint to load.
    """

    if checkpoint is None:
        yield None

    else:
        metadata = storage.StorageMetadata.from_json(checkpoint)
        logging.info("Restoring trial from checkpoint {}".format(metadata.storage_id))

        with storage_mgr.restore_path(metadata) as path:
            yield pathlib.Path(path)


def build_and_run_training_pipeline(env: det.EnvContext) -> None:
    import gc
    gc.disable()
    metrics_thread = SystemMetricsThread()
    metrics_thread.start()

    # Create the socket manager. The socket manager will connect to the master and read messages
    # until it receives the rendezvous_info.
    #
    # TODO(ryan): Pull profiler hooks out of SocketManager and into their own layer.
    with layers.SocketManager(env) as socket_mgr:

        # Create the storage manager. This is used to download the initial checkpoint here in
        # build_training_pipeline and also used by the workload manager to create and store
        # checkpoints during training.
        storage_mgr = storage.build(
            env.experiment_config["checkpoint_storage"],
            container_path=constants.SHARED_FS_CONTAINER_PATH,
        )

        [tensorboard_mgr, tensorboard_writer] = load.prepare_tensorboard(
            env, constants.SHARED_FS_CONTAINER_PATH
        )

        metrics_thread = SystemMetricsThread()
        metrics_thread.run()

        # Create the workload manager. The workload manager will receive workloads from the
        # socket_mgr, and augment them with some additional arguments. Additionally, the
        # workload manager is responsible for some generic workload hooks for things like timing
        # workloads, preparing checkpoints, and uploading completed checkpoints.  Finally, the
        # workload manager does some sanity checks on response messages that originate from the
        # trial.
        #
        # TODO(ryan): Refactor WorkloadManager into separate layers that do each separate task.
        workload_mgr = layers.build_workload_manager(
            env,
            iter(socket_mgr),
            socket_mgr.get_rendezvous_info(),
            storage_mgr,
            tensorboard_mgr,
            tensorboard_writer,
        )

        workloads = iter(workload_mgr)
        hvd_config = horovod.HorovodContext.from_configs(
            env.experiment_config, socket_mgr.get_rendezvous_info(), env.hparams
        )
        logging.info(f"Horovod config: {hvd_config.__dict__}.")

        # Load the checkpoint, if necessary. Any possible sinks to this pipeline will need access
        # to this checkpoint.
        with maybe_load_checkpoint(storage_mgr, env.latest_checkpoint) as load_path:

            # Horovod distributed training is done inside subprocesses.
            if hvd_config.use:
                subproc = layers.SubprocessLauncher(
                    env, workloads, load_path, socket_mgr.get_rendezvous_info(), hvd_config
                )
                subproc.run()
            else:
                if env.experiment_config.debug_enabled():
                    faulthandler.dump_traceback_later(30, repeat=True)

                with det._catch_sys_exit():
                    with det._catch_init_invalid_hp(workloads):
                        controller = load.prepare_controller(
                            env,
                            workloads,
                            load_path,
                            socket_mgr.get_rendezvous_info(),
                            hvd_config,
                        )
                    controller.run()


def main() -> None:
    for k in ENVIRONMENT_VARIABLE_KEYS:
        if k not in os.environ:
            sys.exit("Environment not set: missing " + k)

    experiment_config = simplejson.loads(os.environ["DET_EXPERIMENT_CONFIG"])
    debug = experiment_config.get("debug", False)
    determined_common.set_logger(debug)

    master_addr = os.environ["DET_MASTER_ADDR"]
    master_port = int(os.environ["DET_MASTER_PORT"])
    use_tls = distutils.util.strtobool(os.environ.get("DET_USE_TLS", "false"))
    master_cert_file = os.environ.get("DET_MASTER_CERT_FILE")
    master_cert_name = os.environ.get("DET_MASTER_CERT_NAME")
    agent_id = os.environ["DET_AGENT_ID"]
    container_id = os.environ["DET_CONTAINER_ID"]
    hparams = simplejson.loads(os.environ["DET_HPARAMS"])
    initial_work = workload.Workload.from_json(simplejson.loads(os.environ["DET_INITIAL_WORKLOAD"]))

    with open(os.environ["DET_LATEST_CHECKPOINT"], "r") as f:
        latest_checkpoint = json.load(f)

    use_gpu = distutils.util.strtobool(os.environ.get("DET_USE_GPU", "false"))
    slot_ids = json.loads(os.environ["DET_SLOT_IDS"])
    workload_manager_type = os.environ["DET_WORKLOAD_MANAGER_TYPE"]
    det_rendezvous_ports = os.environ["DET_RENDEZVOUS_PORTS"]
    det_trial_unique_port_offset = int(os.environ["DET_TRIAL_UNIQUE_PORT_OFFSET"])
    det_trial_runner_network_interface = os.environ["DET_TRIAL_RUNNER_NETWORK_INTERFACE"]
    det_trial_id = os.environ["DET_TRIAL_ID"]
    det_experiment_id = os.environ["DET_EXPERIMENT_ID"]
    det_cluster_id = os.environ["DET_CLUSTER_ID"]
    trial_seed = int(os.environ["DET_TRIAL_SEED"])

    gpu_uuids = gpu.get_gpu_uuids_and_validate(use_gpu, slot_ids)

    env = det.EnvContext(
        master_addr,
        master_port,
        use_tls,
        master_cert_file,
        master_cert_name,
        container_id,
        experiment_config,
        hparams,
        initial_work,
        latest_checkpoint,
        use_gpu,
        gpu_uuids,
        slot_ids,
        debug,
        workload_manager_type,
        det_rendezvous_ports,
        det_trial_unique_port_offset,
        det_trial_runner_network_interface,
        det_trial_id,
        det_experiment_id,
        det_cluster_id,
        trial_seed,
    )

    logging.info(
        f"New trial runner in (container {container_id}) on agent {agent_id}: {env.__dict__}."
    )

    try:
        storage.validate_config(
            env.experiment_config["checkpoint_storage"],
            container_path=constants.SHARED_FS_CONTAINER_PATH,
        )
    except Exception as e:
        logging.error("Checkpoint storage validation failed: {}".format(e))
        sys.exit(1)

    try:
        build_and_run_training_pipeline(env)
    except det.InvalidHP:
        logging.info("InvalidHP detected, gracefully exiting trial")
        pass


import psutil
import pynvml

def humanize_float(num): return "{0:,.2f}".format(num)


class QuickTimer:
    def __init__(self, name):
        self.name = name
        self.start = time.time()

    def stop(self):
        end = time.time()
        print(f"[TIMER] {self.name}: {humanize_float(end-self.start)}s")

##########################################################################################
##                              Measurements                                            ##
##########################################################################################

# Measured in percent
class SimpleCpuUtilization:
    def __init__(self, timestamp, batch_idx, util_percent):
        self.timestamp = timestamp
        self.batch_idx = batch_idx
        self.util_percent = util_percent


# Measured in Gigabytes
class FreeMemory:
    def __init__(self, timestamp, batch_idx, memory_free):
        self.timestamp = timestamp
        self.batch_idx = batch_idx
        self.memory_free = memory_free


# Measured in Gigabit/s
class NetworkSentThroughput:
    def __init__(self, timestamp, batch_idx, throughput):
        self.timestamp = timestamp
        self.batch_idx = batch_idx
        self.throughput = throughput


# Measured in Gigabit/s
class NetworkRecvThroughput:
    def __init__(self, timestamp, batch_idx, throughput):
        self.timestamp = timestamp
        self.batch_idx = batch_idx
        self.throughput = throughput


class DiskIops:
    def __init__(self, timestamp, batch_idx, iops):
        self.timestamp = timestamp
        self.batch_idx = batch_idx
        self.iops = iops


# Measured in bytes/second
class DiskReadThroughput:
    def __init__(self, timestamp, batch_idx, throughput):
        self.timestamp = timestamp
        self.batch_idx = batch_idx
        self.throughput = throughput


# Measured in bytes/second
class DiskWriteThroughput:
    def __init__(self, timestamp, batch_idx, throughput):
        self.timestamp = timestamp
        self.batch_idx = batch_idx
        self.throughput = throughput


# Measured in percent
class GpuUtilization:
    def __init__(self, timestamp, batch_idx, gpu_uuid, utilization):
        self.timestamp = timestamp
        self.batch_idx = batch_idx
        self.gpu_uuid = gpu_uuid
        self.utilization = utilization


##########################################################################################
##                              Metric Collectors                                       ##
##########################################################################################
GIGA = 1_000_000_000

class SimpleCpuUtilCollector:
    def measure(self, batch_idx):
        timer = QuickTimer("SimpleCpuUtilCollector")
        cpu_util = psutil.cpu_percent()
        timestamp = time.time()
        timer.stop()
        return SimpleCpuUtilization(timestamp, batch_idx, cpu_util)


class FreeMemoryCollector:
    # We choose to report free memory instead of available memory because it is useful to
    # be able to see memory usage for cached files, but we could change to available instead
    # https://psutil.readthedocs.io/en/latest/#psutil.virtual_memory
    def measure(self, batch_idx):
        timer = QuickTimer("FreeMemoryCollector")
        free_mem_bytes = psutil.virtual_memory().free
        timestamp = time.time()
        timer.stop()
        # print(free_mem_bytes)
        # print(free_mem_bytes * GIGA)
        return FreeMemory(timestamp, batch_idx, free_mem_bytes * GIGA)



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
        return NetworkSentThroughput(end_time, batch_idx, sent_throughput_gigabits_per_second), \
               NetworkRecvThroughput(end_time, batch_idx, recv_throughput_gigabits_per_second)


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

        time_delta_ns = end_time - self.start_time

        self.start_time = end_time
        self.start_read_bytes = disk.read_bytes
        self.start_write_bytes = disk.write_bytes
        self.start_read_count = disk.read_count
        self.start_write_count = disk.write_count

        read_throughput_bytes_per_second = read_bytes_delta / time_delta_ns
        write_throughput_bytes_per_second = write_bytes_delta / time_delta_ns

        read_throughput_count_per_second = read_count_delta / time_delta_ns
        write_throughput_count_per_second = write_count_delta / time_delta_ns

        read_throughput = DiskReadThroughput(end_time, batch_idx, read_throughput_bytes_per_second)
        write_throughput = DiskWriteThroughput(end_time, batch_idx, write_throughput_bytes_per_second)
        iops = DiskIops(end_time, batch_idx, read_throughput_count_per_second + write_throughput_count_per_second)

        timer.stop()
        return read_throughput, write_throughput, iops


import pynvml


class GpuUtilCollector:
    def __init__(self):
        pynvml.nvmlInit()
        self.num_gpus = pynvml.nvmlDeviceGetCount()

    def measure(self, batch_idx):
        timer = QuickTimer("GpuUtilCollector")
        measurements = []
        timestamp = time.time()
        for i in range(self.num_gpus):
            handle = pynvml.nvmlDeviceGetHandleByIndex(i)
            try:
                util = pynvml.nvmlDeviceGetUtilizationRates(handle)
                gpu_util = util.gpu
            except pynvml.NVMLError as err:
                # TODO: Is this how we want to communicate error in metric collection?
                gpu_util = -1
            measurement = GpuUtilization(timestamp, batch_idx, handle, gpu_util)
            measurements.append(measurement)
        timer.stop()
        return measurements


# TODO: Haven't figured out how to collect GPU memory usage yet
class GpuMemory:
    pass


# The psutil way of measuring this is to query by a path. Should we just query /?
class DiskFree:
    pass


# System Metrics Design

# The Master thread/process is instantiated at the beginning of the harness, if it is local_rank=0
# The Master thread/process has three methods:
# Begin
# UpdateCurrentBatch
# Finish
# The Master thread has two state variable
# is_active
# current_batch
# There is at least one subthread
# Metric sending thread
# We may need one or more metric collecting threads (TBD)
# Configuration settings
# Flush interval
# Master API details
# Metrics to detail
# If not is_active, sleep for 1 second
# If is active
#   spend 1 second retrieving information for each metric and saving into list
#   This may require spawning multiple thread (investigate exact mechanism ASAP).
#   After the 1 second,
#       reformat the results into a batch and dispatch the results to the sender Thread


# Open Questions:
#   - System metrics will differ based on hardware. Do we automatically detect that?
#     For now, the list of metrics is hardcoded and if we fail to get GPU data, we return null info

GPU_UTIL_METRIC = "GPU_UTIL"
NET_THRU_SENT_METRIC = "NET_THRU_SENT"
NET_THRU_RECV_METRIC = "NET_THRU_RECV"
DISK_IOPS_METRIC = "DISK_IOPS"
DISK_THRU_READ_METRIC = "DISK_THRU_READ"
DISK_THRU_WRITE_METRIC = "DISK_THRU_WRITE"
FREE_MEM_METRIC = "FREE_MEM"
SIMPLE_CPU_UTIL_METRIC = "SIMPLE_CPU_UTIL"

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

    def __init__(self) -> None:

        self.verbose = True
        self.log("Creating SystemMetricsThread")

        self.is_active = True
        self.current_batch = 1

        self.dispatch_queue = queue.Queue()
        self.sending_thread = SystemMetricsSendingThread(self.dispatch_queue)
        self.sending_thread.start()

        self.current_metric_batch = {
            GPU_UTIL_METRIC: [],
            # "GPU_MEM": [],
            NET_THRU_SENT_METRIC: [],
            NET_THRU_RECV_METRIC: [],
            # "DISK_FREE": [],
            DISK_IOPS_METRIC: [],
            DISK_THRU_READ_METRIC: [],
            DISK_THRU_WRITE_METRIC: [],
            FREE_MEM_METRIC: [],
            SIMPLE_CPU_UTIL_METRIC: []
        }

        self.quitting = False
        super().__init__()

    def log(self, *s):
        if self.verbose:
            print("[SystemMetricsThread]", *s)

    # TODO: Add flushing at intervals
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

            # One-time initialization
            if last_measurement_time is None:
                last_measurement_time = time.time()
                batch_start_time = time.time()
                network_throughput_collector.reset()
                disk_collector.reset()




            # Check if it is time to take a new measurement
            if time.time() - last_measurement_time > self.MEASUREMENT_INTERVAL:
                self.log("Taking new measurement")
                immutable_batch_idx = self.current_batch
                self.log("Taking new measurement - cpu")
                cpu_util_measurement = cpu_util_collector.measure(immutable_batch_idx)
                self.log("Taking new measurement - gpu")
                gpu_util_measurement = gpu_util_collector.measure(immutable_batch_idx)
                self.log("Taking new measurement - network")
                net_thru_sent_measurement, net_thru_recv_measurement = network_throughput_collector.measure(immutable_batch_idx)
                self.log("Taking new measurement - memory")
                free_memory_measurement = free_memory_collector.measure(immutable_batch_idx)
                self.log("Taking new measurement - disk")
                disk_read_thru_measurement, disk_write_thru_measurement, iops_measurement = disk_collector.measure(immutable_batch_idx)

                self.current_metric_batch[GPU_UTIL_METRIC].extend(gpu_util_measurement)
                self.current_metric_batch[NET_THRU_SENT_METRIC].append(net_thru_sent_measurement)
                self.current_metric_batch[NET_THRU_RECV_METRIC].append(network_throughput_collector)
                self.current_metric_batch[DISK_IOPS_METRIC].append(iops_measurement)
                self.current_metric_batch[DISK_THRU_READ_METRIC].append(disk_read_thru_measurement)
                self.current_metric_batch[DISK_THRU_WRITE_METRIC].append(disk_write_thru_measurement)
                self.current_metric_batch[FREE_MEM_METRIC].append(free_memory_measurement)
                self.current_metric_batch[SIMPLE_CPU_UTIL_METRIC].append(cpu_util_measurement)
                self.log("Finished taking measurement")


            # Check if it is time to flush the batch and start a new batch
            if time.time() - batch_start_time > self.FLUSH_INTERVAL:
                self.log("Completed a batch")
                self.enqueue_for_async_send(self.current_metric_batch)
                self.current_metric_batch = {
                    GPU_UTIL_METRIC: [],
                    NET_THRU_SENT_METRIC: [],
                    NET_THRU_RECV_METRIC: [],
                    DISK_IOPS_METRIC: [],
                    DISK_THRU_READ_METRIC: [],
                    DISK_THRU_WRITE_METRIC: [],
                    FREE_MEM_METRIC: [],
                    SIMPLE_CPU_UTIL_METRIC: []
                }
                batch_start_time = time.time()



    def update_current_batch(self, new_current_batch):
        self.current_batch = new_current_batch

    def enqueue_for_async_send(self, metric_batch):
        # This method can theoretically raise a FULL error, but SimpleQueues are unbounded so
        # I don't think it will ever happen (https://docs.python.org/3/library/queue.html#queue.Queue.put)
        self.log("Enqueuing metric batch", metric_batch)

        # TODO work on sending logic
        self.dispatch_queue.put_nowait(metric_batch)

    def __enter__(self) -> "SystemMetricsThread":
        self.start()
        return self

    def __exit__(self, *arg: Any) -> None:
        self.quitting = True


class MetricsBatch:
    def as_string(self):
        pass


import queue
import time


# This is a thread that exists solely so that we can make API calls without blocking
# It has a SimpleQueue through which work is sent to the thread
class SystemMetricsSendingThread(threading.Thread):
    def __init__(self, inbound_queue: queue.Queue) -> None:
        self.POLL_INTERVAL_SECS = 0.5
        self.inbound_queue = inbound_queue

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

            self.send_batch(batch_to_send)

    # This is a blocking operation (that handles retries?) that must handle all exceptions gracefully
    def send_batch(self, batch: MetricsBatch):
        print(batch)
        pass

    def quit(self):
        self.quitting = True


if __name__ == "__main__":
    main()
