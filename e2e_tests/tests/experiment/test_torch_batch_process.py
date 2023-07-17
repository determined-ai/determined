import os
import shutil
import tempfile

import pytest

from tests import config as conf
from tests import experiment as exp


@pytest.mark.distributed
@pytest.mark.gpu_required
def test_torch_batch_process_generate_embedding() -> None:
    config = conf.load_config(
        conf.torch_batch_process_examples_path(
            "batch_inference/generate_embedding/distributed.yaml"
        )
    )

    with tempfile.TemporaryDirectory() as tmpdir:
        copy_destination = os.path.join(tmpdir, "example")
        shutil.copytree(
            conf.torch_batch_process_examples_path("batch_inference/generate_embedding"),
            copy_destination,
        )
        exp.run_basic_test_with_temp_config(config, copy_destination, 1)
