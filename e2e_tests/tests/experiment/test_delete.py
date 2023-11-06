import pathlib
import subprocess
import pytest

from determined.common import util
from tests import config as conf
from tests import experiment as exp

@pytest.mark.e2e_cpu
def test_delete_experiment_removes_tensorboard_files(tmp_path: pathlib.Path) -> None:
    config_obj = conf.load_config(conf.fixtures_path("no_op/single-one-short-step.yaml"))
    config_obj["checkpoint_storage"] = {
        "type": "directory",
        "container_path": "/tmp/somepath",
    }
    tb_config = {}
    tb_config["bind_mounts"] = config_obj["bind_mounts"] = [
        {
            "host_path": "/tmp/",
            "container_path": "/tmp/somepath",
        }
    ]

    tb_config_path = tmp_path / "tb.yaml"
    with tb_config_path.open("w") as fout:
        util.yaml_safe_dump(tb_config, fout)

    experiment_id = exp.run_basic_test_with_temp_config(config_obj, conf.fixtures_path("no_op"), 1)

    command = ["det", "e", "delete", str(experiment_id), "--yes"]
    subprocess.run(command, universal_newlines=True, stdout=subprocess.PIPE, check=True)

    # Check if Tensorboard files are deleted
    tb_path = sorted(pathlib.Path("/tmp/determined-cp/").glob("*/tensorboard"))[0]
    tb_path = tb_path / "experiment" / str(experiment_id)
    assert not pathlib.Path(tb_path).exists()