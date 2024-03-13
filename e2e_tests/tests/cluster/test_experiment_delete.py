import pathlib
import time

import pytest

from determined.common import api
from tests import api_utils
from tests import config as conf
from tests import detproc
from tests import experiment as exp


@pytest.mark.e2e_cpu
def test_delete_experiment_removes_tensorboard_files() -> None:
    """
    Start a random experiment, delete the experiment and verify that TensorBoard files are deleted.
    """
    sess = api_utils.user_session()
    config_obj = conf.load_config(conf.fixtures_path("no_op/single-medium-train-step.yaml"))
    experiment_id = exp.run_basic_test_with_temp_config(
        sess, config_obj, conf.fixtures_path("no_op"), 1
    )

    # Check if Tensorboard files are created
    path = (
        config_obj["checkpoint_storage"]["host_path"]
        + "/"
        + config_obj["checkpoint_storage"]["storage_path"]
    )
    tb_path = sorted(pathlib.Path(path).glob("*/tensorboard"))[0]
    tb_path = tb_path / "experiment" / str(experiment_id)
    assert pathlib.Path(tb_path).exists()

    command = ["det", "-m", conf.make_master_url(), "e", "delete", str(experiment_id), "--yes"]
    detproc.check_call(sess, command)

    ticks = 60
    for i in range(ticks):
        try:
            state = exp.experiment_state(sess, experiment_id)
            if i % 5 == 0:
                print(f"experiment in state {state} waiting to be deleted")
            time.sleep(1)
        except api.errors.NotFoundException:
            # Check if Tensorboard files are deleted
            assert not pathlib.Path(tb_path).exists()
            return

    pytest.fail(f"experiment failed to be deleted after {ticks} seconds")
