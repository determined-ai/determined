import pathlib

import pytest

import determined as det
from determined import tensorboard
from determined.tensorboard import SharedFSTensorboardManager

BASE_PATH = pathlib.Path(__file__).resolve().parent.joinpath("fixtures")


def get_dummy_env() -> det.EnvContext:
    return det.EnvContext(
        master_url="",
        master_cert_file=None,
        master_cert_name=None,
        experiment_config={"resources": {"slots_per_trial": 1, "native_parallel": False}},
        latest_checkpoint=None,
        steps_completed=0,
        use_gpu=False,
        container_gpus=[],
        slot_ids=[],
        debug=False,
        hparams={"global_batch_size": 1},
        det_trial_id="1",
        det_agent_id="1",
        det_experiment_id="1",
        det_cluster_id="uuid-123",
        trial_seed=0,
        trial_run_id=1,
        allocation_id="",
        managed_training=True,
        test_mode=False,
        on_cluster=False,
    )


def test_list_tb_files(tmp_path: pathlib.Path) -> None:
    env = get_dummy_env()
    base_path = tensorboard.get_base_path({"base_path": BASE_PATH})
    sync_path = tensorboard.get_sync_path(
        env.det_cluster_id, env.det_experiment_id, env.det_trial_id
    )

    manager = SharedFSTensorboardManager(str(tmp_path), base_path, sync_path)
    test_files = [
        "no_show.txt",
        "79375caf89e9.kernel_stats.pb",
        "79375caf89e9.memory_profile.json.gz",
        "events.out.tfevents.example",
    ]

    test_filepaths = [BASE_PATH.joinpath("tensorboard--0", test_file) for test_file in test_files]
    tb_files = manager.list_tb_files(0, lambda _: True)

    assert set(test_filepaths) == set(tb_files)


def test_list_tb_files_nonexistent_directory(tmp_path: pathlib.Path) -> None:
    env = get_dummy_env()
    base_path = pathlib.Path("/non-existent-directory")
    sync_path = tensorboard.get_sync_path(
        env.det_cluster_id, env.det_experiment_id, env.det_trial_id
    )
    manager = SharedFSTensorboardManager(str(tmp_path), base_path, sync_path)

    assert not pathlib.Path(base_path).exists()
    assert manager.list_tb_files(0, lambda _: True) == []


test_data = [
    (
        "/home/bob/tensorboard/the-host-name.memory_profile.json.gz",
        3,
        "/home/bob/tensorboard/the-host-name#3.memory_profile.json.gz",
    ),
    (
        "/home/bob/tensorboard/the-host-name.some-extension.gz",
        2,
        "/home/bob/tensorboard/the-host-name.some-extension.gz",
    ),
    # Pytorch profiler file with timestamp and ends with pt.trace.json
    (
        (
            "/tmp/tensorboard-39.ff54aea9-0a94-4ce7-bf38-e8b3e69cc944.1-0/"
            "aa1f87508336_37.1674696139174.pt.trace.json"
        ),
        1,
        (
            "/tmp/tensorboard-39.ff54aea9-0a94-4ce7-bf38-e8b3e69cc944.1-0/"
            "aa1f87508336_37#1.1674696139174.pt.trace.json"
        ),
    ),
    # Pytorch profiler file without timestamp and ends with pt.trace.json
    (
        (
            "/tmp/tensorboard-39.ff54aea9-0a94-4ce7-bf38-e8b3e69cc944.1-0/"
            "aa1f87508336_37.pt.trace.json"
        ),
        1,
        (
            "/tmp/tensorboard-39.ff54aea9-0a94-4ce7-bf38-e8b3e69cc944.1-0/"
            "aa1f87508336_37#1.pt.trace.json"
        ),
    ),
    # Pytorch profiler file with timestamp and ends with pt.trace.json.gz
    (
        (
            "/tmp/tensorboard-39.ff54aea9-0a94-4ce7-bf38-e8b3e69cc944.1-0/"
            "aa1f87508336_37.1674696139174.pt.trace.json.gz"
        ),
        1,
        (
            "/tmp/tensorboard-39.ff54aea9-0a94-4ce7-bf38-e8b3e69cc944.1-0/"
            "aa1f87508336_37#1.1674696139174.pt.trace.json.gz"
        ),
    ),
    # Pytorch profiler file without timestamp and ends with pt.trace.json.gz
    (
        (
            "/tmp/tensorboard-39.ff54aea9-0a94-4ce7-bf38-e8b3e69cc944.1-0/"
            "aa1f87508336_37.pt.trace.json.gz"
        ),
        1,
        (
            "/tmp/tensorboard-39.ff54aea9-0a94-4ce7-bf38-e8b3e69cc944.1-0/"
            "aa1f87508336_37#1.pt.trace.json.gz"
        ),
    ),
    # Pytorch profiler file (only file name) with timestamp and ends with pt.trace.json.gz
    (
        "aa1f87508336_37.1674696139174.pt.trace.json.gz",
        1,
        "aa1f87508336_37#1.1674696139174.pt.trace.json.gz",
    ),
    # Pytorch profiler file (only file name) without timestamp and ends with pt.trace.json.gz
    (
        "aa1f87508336_37.pt.trace.json.gz",
        1,
        "aa1f87508336_37#1.pt.trace.json.gz",
    ),
]


@pytest.mark.parametrize("path,rank,expected", test_data)
def test_get_rank_aware_path(path: str, rank: int, expected: str) -> None:
    actual = tensorboard.util.get_rank_aware_path(pathlib.Path(path), rank)
    assert pathlib.Path(expected) == actual, (expected, actual)
