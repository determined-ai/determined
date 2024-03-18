import pytest

from tests import api_utils
from tests import config as conf
from tests import experiment as exp


@pytest.mark.distributed
@pytest.mark.gpu_required
@pytest.mark.e2e_slurm_gpu
def test_pytorch2_hf_language_modeling_distributed() -> None:
    sess = api_utils.user_session()
    test_dir = "hf_language_modeling"

    config = conf.load_config(conf.hf_trainer_examples_path(f"{test_dir}/distributed.yaml"))
    config = conf.set_pt2_image(config)
    config = conf.set_slots_per_trial(config, 4)
    config["searcher"]["max_length"]["batches"] = 50

    # Our hardware GPUs have only 16gb memory, lower memory use with smaller batches.
    config = conf.set_entrypoint(
        config,
        config["entrypoint"]
        .replace("--per_device_train_batch_size 8", "--per_device_train_batch_size 2")
        .replace("--per_device_eval_batch_size 8", "--per_device_eval_batch_size 2"),
    )

    exp.run_basic_test_with_temp_config(sess, config, conf.hf_trainer_examples_path(test_dir), 1)
