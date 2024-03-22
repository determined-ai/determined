from typing import Any, Dict, List, NamedTuple
from unittest import mock

import psutil
import pytest

from determined.core import _profiler as profiler


@mock.patch("determined.core._profiler.psutil.net_io_counters")
@mock.patch("time.time")
def test_sample_metrics_network(mock_time: mock.MagicMock, mock_psutil: mock.MagicMock) -> None:
    """Test that ``Network.sample_metrics()`` collects metrics to the expected format.

    Mocks `psutil` and `time` dependencies used to calculate network throughput metrics and
    verifies that the resulting dict appended to ``Network.metric_samples`` contains the expected
    metric names and values.
    """
    mock_start_time = 0.0
    mock_time.return_value = mock_start_time
    mock_start_bytes_sent = 100
    mock_start_bytes_recv = 200
    mock_start_net_io = psutil._common.snetio(
        mock_start_bytes_sent, mock_start_bytes_recv, 10, 20, 0, 0, 0, 0
    )
    mock_psutil.return_value = mock_start_net_io

    net_collector = profiler._Network()
    mock_psutil.assert_called_once()
    assert len(net_collector.metric_samples) == 0

    mock_end_time = mock_start_time + 1
    mock_time.return_value = mock_end_time
    mock_bytes_sent = 200
    mock_bytes_recv = 300
    mock_net_io = psutil._common.snetio(mock_bytes_sent, mock_bytes_recv, 10, 20, 0, 0, 0, 0)
    mock_psutil.return_value = mock_net_io

    exp_thru_sent = (mock_bytes_sent - mock_start_bytes_sent) / (mock_end_time - mock_start_time)
    exp_thru_recv = (mock_bytes_recv - mock_start_bytes_recv) / (mock_end_time - mock_start_time)
    expected_metrics = {
        "net_throughput_sent": exp_thru_sent,
        "net_throughput_recv": exp_thru_recv,
    }
    net_collector.sample_metrics()
    assert len(net_collector.metric_samples) == 1
    assert net_collector.metric_samples[0] == expected_metrics


@mock.patch("determined.core._profiler.psutil.disk_io_counters")
@mock.patch("determined.core._profiler.psutil.disk_usage")
@mock.patch("time.time")
def test_sample_metrics_disk(
    mock_time: mock.MagicMock,
    mock_psutil_disk_usage: mock.MagicMock,
    mock_psutil_disk_io: mock.MagicMock,
) -> None:
    """Test that ``Disk.sample_metrics()`` collects metrics to the expected format.

    Mocks `psutil` and `time` dependencies used to calculate disk throughput metrics and verifies
    that the resulting dict appended to ``Disk.metric_samples`` contains the expected metric names
    and values.
    """
    mock_start_time = 0.0
    mock_time.return_value = mock_start_time
    mock_start_disk_io = psutil._common.sdiskio(
        read_count=10,
        write_count=20,
        read_bytes=100,
        write_bytes=200,
        read_time=0,
        write_time=0,
    )
    mock_psutil_disk_io.return_value = mock_start_disk_io

    disk_collector = profiler._Disk()
    mock_psutil_disk_io.assert_called_once()
    assert mock_psutil_disk_usage.call_count == 2
    assert len(disk_collector.metric_samples) == 0

    mock_end_time = mock_start_time + 1
    mock_time.return_value = mock_end_time
    mock_disk_io = psutil._common.sdiskio(
        read_count=20,
        write_count=30,
        read_bytes=200,
        write_bytes=300,
        read_time=0,
        write_time=0,
    )
    mock_psutil_disk_io.return_value = mock_disk_io

    mock_disk_usage_paths = {}
    mock_disk_usage = psutil._common.sdiskusage(total=100, used=10, free=90, percent=10.0)
    for dp in disk_collector._disk_paths:
        mock_disk_usage_paths[dp] = mock_disk_usage
    mock_psutil_disk_usage.return_value = mock_disk_usage

    exp_thru_read = (mock_disk_io.read_bytes - mock_start_disk_io.read_bytes) / (
        mock_end_time - mock_start_time
    )
    exp_thru_write = (mock_disk_io.write_bytes - mock_start_disk_io.write_bytes) / (
        mock_end_time - mock_start_time
    )
    exp_iops = (
        (mock_disk_io.read_count + mock_disk_io.write_count)
        - (mock_start_disk_io.read_count + mock_start_disk_io.write_count)
    ) / (mock_end_time - mock_start_time)

    expected_metrics = {
        "disk_iops": exp_iops,
        "disk_throughput_read": exp_thru_read,
        "disk_throughput_write": exp_thru_write,
    }
    for p, u in mock_disk_usage_paths.items():
        expected_metrics[p] = {"disk_util": u.percent}

    disk_collector.sample_metrics()
    assert len(disk_collector.metric_samples) == 1
    assert disk_collector.metric_samples[0] == expected_metrics


@mock.patch("determined.core._profiler.psutil.virtual_memory")
def test_sample_metrics_memory(
    mock_psutil: mock.MagicMock,
) -> None:
    cpu_collector = profiler._Memory()
    assert len(cpu_collector.metric_samples) == 0
    MockMemoryInfo = NamedTuple("MockMemoryInfo", [("available", int)])

    mock_psutil.return_value = MockMemoryInfo(available=900)

    expected_metrics = {
        "memory_free": 900,
    }

    cpu_collector.sample_metrics()
    assert len(cpu_collector.metric_samples) == 1
    assert cpu_collector.metric_samples[0] == expected_metrics


@mock.patch("determined.core._profiler.psutil.cpu_percent")
def test_sample_metrics_cpu(
    mock_psutil: mock.MagicMock,
) -> None:
    cpu_collector = profiler._CPU()
    assert len(cpu_collector.metric_samples) == 0

    mock_cpu_util = 20.0
    mock_psutil.return_value = mock_cpu_util

    expected_metrics = {
        "cpu_util_simple": mock_cpu_util,
    }

    cpu_collector.sample_metrics()
    assert len(cpu_collector.metric_samples) == 1
    assert cpu_collector.metric_samples[0] == expected_metrics


@pytest.mark.skipif(profiler.pynvml is None, reason="pynvml required for test")
@mock.patch("determined.core._profiler.pynvml.nvmlInit")
@mock.patch("determined.core._profiler.pynvml.nvmlDeviceGetCount")
@mock.patch("determined.core._profiler.pynvml.nvmlDeviceGetHandleByIndex")
@mock.patch("determined.core._profiler.pynvml.nvmlDeviceGetUUID")
@mock.patch("determined.core._profiler.pynvml.nvmlDeviceGetMemoryInfo")
@mock.patch("determined.core._profiler.pynvml.nvmlDeviceGetUtilizationRates")
def test_sample_metrics_gpu(
    mock_pynvml_device_util: mock.MagicMock,
    mock_pynvml_device_memory: mock.MagicMock,
    mock_pynvml_device_uuid: mock.MagicMock,
    mock_pynvml_device_handle: mock.MagicMock,
    mock_pynvml_device_count: mock.MagicMock,
    mock_pynvml_init: mock.MagicMock,
) -> None:
    """Test that ``GPU.sample_metrics()`` collects metrics to the expected format.

    Calls to the `pynvml` dependency used to get GPU metrics have been mocked. This test verifies
    that given expected `pynvml` output values, ``sample_metrics()`` generates a properly-formatted
    metrics dict that is appended to ``GPU.metric_samples``.
    """

    class MockNVMLDeviceHandle:
        pass

    MockMemoryInfo = NamedTuple("MockMemoryInfo", [("free", float)])
    MockGPUInfo = NamedTuple("MockGPUInfo", [("gpu", float)])

    mock_gpu_uuids = ["GPU-UUID-1", "GPU-UUID-2"]
    mock_gpu_devices = [MockNVMLDeviceHandle() for _ in mock_gpu_uuids]

    mock_gpu_free_memory = [1.0e10, 1.5e10]
    mock_gpu_util = [10.0, 20.0]

    def mock_device_handle_by_index(index: int) -> MockNVMLDeviceHandle:
        return mock_gpu_devices[index]

    def mock_device_get_uuid(handle: Any) -> str:
        return mock_gpu_uuids[mock_gpu_devices.index(handle)]

    def mock_device_get_memory(handle: Any) -> MockMemoryInfo:
        return MockMemoryInfo(mock_gpu_free_memory[mock_gpu_devices.index(handle)])

    def mock_device_get_util(handle: Any) -> MockGPUInfo:
        return MockGPUInfo(mock_gpu_util[mock_gpu_devices.index(handle)])

    mock_pynvml_device_count.return_value = len(mock_gpu_devices)
    mock_pynvml_device_handle.side_effect = mock_device_handle_by_index
    mock_pynvml_device_uuid.side_effect = mock_device_get_uuid
    mock_pynvml_device_util.side_effect = mock_device_get_util
    mock_pynvml_device_memory.side_effect = mock_device_get_memory

    gpu_collector = profiler._GPU()
    mock_pynvml_init.assert_called_once()
    mock_pynvml_device_count.assert_called_once()
    assert mock_pynvml_device_uuid.call_count == len(mock_gpu_devices)
    assert mock_pynvml_device_handle.call_count == len(mock_gpu_devices)
    assert mock_pynvml_device_memory.call_count == len(mock_gpu_devices)
    assert mock_pynvml_device_util.call_count == len(mock_gpu_devices)

    assert len(gpu_collector.metric_samples) == 0

    expected_metrics = {}
    for i, uuid in enumerate(mock_gpu_uuids):
        expected_metrics[uuid] = {
            "gpu_free_memory": mock_gpu_free_memory[i],
            "gpu_util": mock_gpu_util[i],
        }

    gpu_collector.sample_metrics()
    assert len(gpu_collector.metric_samples) == 1
    assert gpu_collector.metric_samples[0] == expected_metrics


def test_average_metric_samples_flat() -> None:
    test_metrics_flat = [
        {
            "name1": 0,
            "name2": 1,
        },
        {
            "name1": 2,
            "name2": 3,
        },
    ]

    result = profiler._average_metric_samples_depth_one(metric_samples=test_metrics_flat)
    assert result == {"name1": (0 + 2) / 2, "name2": (1 + 3) / 2}


def test_average_metric_samples_depth_one() -> None:
    test_metrics_single_nested = [
        {
            "label1": {
                "name1": 0,
                "name2": 1,
            }
        },
        {
            "label1": {
                "name1": 2,
                "name2": 3,
            }
        },
    ]
    result = profiler._average_metric_samples_depth_one(metric_samples=test_metrics_single_nested)
    assert result == {"label1": {"name1": (0 + 2) / 2, "name2": (1 + 3) / 2}}


def test_average_metric_samples_depth_two() -> None:
    test_metrics_twice_nested: List[Dict[str, Any]] = [
        {
            "label1": {
                "name1": 0,
                "name2": 1,
                "label2": {
                    "name3": 1,
                },
            }
        },
        {
            "label1": {
                "name1": 2,
                "name2": 3,
            }
        },
    ]
    with pytest.raises(ValueError):
        profiler._average_metric_samples_depth_one(metric_samples=test_metrics_twice_nested)
