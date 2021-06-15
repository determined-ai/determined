import pathlib

import numpy as np

import determined as det
import determined.common.types
from determined import constants, workload
from determined.tensorboard import SharedFSTensorboardManager, get_base_path, get_sync_path
from determined.tensorboard.metric_writers import util as metric_writers_util

BASE_PATH = pathlib.Path(__file__).resolve().parent.joinpath("fixtures")


def get_dummy_env() -> det.EnvContext:
    return det.EnvContext(
        master_addr="",
        master_port=0,
        use_tls=False,
        master_cert_file=None,
        master_cert_name=None,
        container_id="",
        experiment_config={"resources": {"slots_per_trial": 1, "native_parallel": False}},
        initial_workload=workload.Workload(
            workload.Workload.Kind.RUN_STEP,
            determined.common.types.ExperimentID(1),
            determined.common.types.TrialID(1),
            determined.common.types.StepID(1),
            constants.DEFAULT_SCHEDULING_UNIT,
            0,
        ),
        latest_checkpoint=None,
        use_gpu=False,
        container_gpus=[],
        slot_ids=[],
        debug=False,
        workload_manager_type="",
        hparams={"global_batch_size": 1},
        det_rendezvous_port="",
        det_trial_unique_port_offset=0,
        det_trial_runner_network_interface=constants.AUTO_DETECT_TRIAL_RUNNER_NETWORK_INTERFACE,
        det_trial_id="1",
        det_agent_id="1",
        det_experiment_id="1",
        det_task_token="",
        det_cluster_id="uuid-123",
        trial_seed=0,
        managed_training=True,
        test_mode=False,
        on_cluster=False,
    )


def test_is_not_numerical_scalar() -> None:
    # Invalid types
    assert not metric_writers_util.is_numerical_scalar("foo")
    assert not metric_writers_util.is_numerical_scalar(np.array("foo"))
    assert not metric_writers_util.is_numerical_scalar(object())

    # Invalid shapes
    assert not metric_writers_util.is_numerical_scalar([1])
    assert not metric_writers_util.is_numerical_scalar(np.array([3.14]))
    assert not metric_writers_util.is_numerical_scalar(np.ones(shape=(5, 5)))


def test_is_numerical_scalar() -> None:
    assert metric_writers_util.is_numerical_scalar(1)
    assert metric_writers_util.is_numerical_scalar(1.0)
    assert metric_writers_util.is_numerical_scalar(-3.14)
    assert metric_writers_util.is_numerical_scalar(np.ones(shape=()))
    assert metric_writers_util.is_numerical_scalar(np.array(1))
    assert metric_writers_util.is_numerical_scalar(np.array(-3.14))
    assert metric_writers_util.is_numerical_scalar(np.array([1.0])[0])


def test_list_tb_files(tmp_path: pathlib.Path) -> None:
    env = get_dummy_env()
    base_path = get_base_path({"base_path": BASE_PATH}, manager=True)
    sync_path = get_sync_path(env.det_cluster_id, env.det_experiment_id, env.det_trial_id)

    manager = SharedFSTensorboardManager(str(tmp_path), base_path, sync_path)
    test_files = [
        "79375caf89e9.kernel_stats.pb",
        "79375caf89e9.memory_profile.json.gz",
        "events.out.tfevents.example",
    ]

    test_filepaths = [BASE_PATH.joinpath("tensorboard", test_file) for test_file in test_files]
    tb_files = manager.list_tb_files(0)

    assert set(test_filepaths) == set(tb_files)


def test_list_tb_files_nonexistent_directory(tmp_path: pathlib.Path) -> None:
    env = get_dummy_env()
    base_path = pathlib.Path("/non-existent-directory")
    sync_path = get_sync_path(env.det_cluster_id, env.det_experiment_id, env.det_trial_id)
    manager = SharedFSTensorboardManager(str(tmp_path), base_path, sync_path)

    assert not pathlib.Path(base_path).exists()
    assert manager.list_tb_files(0) == []
