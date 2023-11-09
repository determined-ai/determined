import pathlib
import subprocess
import time

import pytest

from determined.common import api
from tests import config as conf
from tests import experiment as exp


@pytest.mark.e2e_cpu
def test_delete_experiment_removes_tensorboard_files() -> None:
    """
    Start a random experiment, delete the experiment and verify that TensorBoard files are deleted.
    """
    config_obj = conf.load_config(conf.tutorials_path("mnist_pytorch/const.yaml"))
    experiment_id = exp.run_basic_test_with_temp_config(
        config_obj, conf.tutorials_path("mnist_pytorch"), 1
    )

    command = ["det", "e", "delete", str(experiment_id), "--yes"]
    subprocess.run(command, universal_newlines=True, stdout=subprocess.PIPE, check=True)

    ticks = 120
    for i in range(ticks):
        try:
            state = exp.experiment_state(experiment_id)
            if i % 5 == 0:
                print(f"experiment in state {state} waiting to be deleted")
            time.sleep(1)
        except api.errors.NotFoundException:
            return

    pytest.fail(f"experiment failed to be deleted after {ticks} seconds")

    # Check if Tensorboard files are deleted
    tb_path = sorted(pathlib.Path("/tmp/determined-cp/").glob("*/tensorboard"))[0]
    tb_path = tb_path / "experiment" / str(experiment_id)
    assert not pathlib.Path(tb_path).exists()
