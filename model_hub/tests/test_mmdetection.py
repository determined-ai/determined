import os
import shutil
from typing import Generator, cast

import attrdict
import git
import pytest
import torch

import determined.pytorch as det_torch
import model_hub.mmdetection as mh_mmdet
import model_hub.mmdetection._callbacks as callbacks
import model_hub.utils as mh_utils
from determined.common import util


def cleanup_dir(directory: str) -> None:
    for root, dirs, files in os.walk(directory, topdown=False):
        for name in files:
            os.remove(os.path.join(root, name))
        for name in dirs:
            os.rmdir(os.path.join(root, name))
    os.rmdir(directory)


@pytest.fixture(scope="module")
def mmdet_config_dir() -> Generator[str, None, None]:
    git.Repo.clone_from("https://github.com/open-mmlab/mmdetection", "/tmp/mmdetection")
    mmdet_config_dir = "/tmp/mmdetection/configs"
    os.environ["MMDETECTION_CONFIG_DIR"] = mmdet_config_dir
    yield mmdet_config_dir

    # cleanup
    cleanup_dir("/tmp/mmdetection")


@pytest.fixture(scope="module")
def context(mmdet_config_dir: str) -> det_torch.PyTorchTrialContext:
    config_file = "./tests/fixtures/maskrcnn.yaml"
    with open(config_file, "rb") as f:
        config = util.safe_load_yaml_with_exceptions(f)
    context = det_torch.PyTorchTrialContext.from_config(config)
    context = cast(det_torch.PyTorchTrialContext, context)
    return context


@pytest.fixture(scope="module")
def trial(context: det_torch.PyTorchTrialContext) -> mh_mmdet.MMDetTrial:
    trial = mh_mmdet.MMDetTrial(context)
    return trial


@pytest.fixture(scope="module")
def dataloader(trial: mh_mmdet.MMDetTrial) -> Generator[torch.utils.data.DataLoader, None, None]:
    mh_utils.download_url(
        "/tmp", "http://images.cocodataset.org/annotations/annotations_trainval2017.zip"
    )
    shutil.unpack_archive("/tmp/annotations_trainval2017.zip", "/tmp")
    det_data_loader = trial.build_training_data_loader()
    data_loader = det_data_loader.get_data_loader()
    trial.context._current_batch_idx = 0
    trial.context._epoch_len = len(data_loader)
    yield data_loader

    # cleanup
    os.remove("/tmp/annotations_trainval2017.zip")
    cleanup_dir("/tmp/annotations")


# _callbacks.py
def test_fake_runner(trial: mh_mmdet.MMDetTrial, dataloader: torch.utils.data.DataLoader) -> None:
    runner = callbacks.FakeRunner(trial.context)
    assert len(runner.optimizer) == 1
    assert len(runner.data_loader) == len(dataloader)  # type: ignore
    assert runner.iter == 0
    assert runner.epoch == 0
    assert runner.max_iters == 200


# _data.py
def test_group_sampler(dataloader: torch.utils.data.DataLoader) -> None:
    dataset = dataloader.dataset
    sampler = mh_mmdet.GroupSampler(dataset, 2, 1)
    flags = [dataset.flag[i] for i in sampler]  # type: ignore
    test = [flags[i] == flags[i + 1] for i in range(0, len(flags), 2)]
    assert all(test)


# utils.py
def test_get_pretrained_weights(
    mmdet_config_dir: None, context: det_torch.PyTorchTrialContext
) -> None:
    mh_mmdet.utils.CONFIG_TO_PRETRAINED = mh_mmdet.utils.get_config_pretrained_url_mapping()
    path, ckpt = mh_mmdet.get_pretrained_ckpt_path("/tmp", context.get_hparam("config_file"))
    assert path is not None
    assert ckpt is not None


# _trial.py
class TestMMDetTrial:
    def test_merge_config(
        self, context: det_torch.PyTorchTrialContext, trial: mh_mmdet.MMDetTrial
    ) -> None:
        hparams = context.get_hparams()
        hparams["merge_config"] = "./tests/fixtures/merge_config.py"
        trial.hparams = attrdict.AttrDict(hparams)
        new_cfg = trial.build_mmdet_config()
        assert new_cfg.optimizer.type == "AdamW"
        assert new_cfg.optimizer_config.grad_clip.max_norm == 0.1

    def test_override_mmdet_config(
        self, context: det_torch.PyTorchTrialContext, trial: mh_mmdet.MMDetTrial
    ) -> None:
        hparams = context.get_hparams()
        hparams["override_mmdet_config"] = {
            "optimizer_config._delete_": True,
            "optimizer_config.grad_clip.max_norm": 35,
            "optimizer_config.grad_clip.norm_type": 2,
        }
        trial.hparams = attrdict.AttrDict(hparams)
        new_cfg = trial.build_mmdet_config()
        assert new_cfg.optimizer_config.grad_clip.max_norm == 35
        assert new_cfg.optimizer_config.grad_clip.norm_type == 2
