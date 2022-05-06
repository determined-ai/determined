"""
Various utility functions for using mmdetection in Determined that may be useful
even if not using the provided MMDetTrial.

build_fp16_loss_scaler is large derived from the original mmcv code at
https://github.com/open-mmlab/mmcv/blob/master/mmcv/runner/hooks/optimizer.py
mmcv is covered by the Apache 2.0 License.  Copyright (c) OpenMMLab. All rights reserved.
"""
import os
from typing import Any, Dict, Tuple

import mmcv
import torch

import model_hub.utils


def get_config_pretrained_url_mapping() -> Dict[str, str]:
    """
    Walks the MMDETECTION_CONFIG_DIR and creates a mapping of configs
    to urls for pretrained checkpoints. The url for pretrained checkpoints
    are parsed from the README files in each of the mmdetection config folders.

    MMDETECTION_CONFIG_DIR is set to /mmdetection/configs in the default
    determinedai/model-hub-mmdetection docker image.
    """
    models = {}
    config_dir = os.getenv("MMDETECTION_CONFIG_DIR")
    if config_dir:
        for root, _, files in os.walk(config_dir):
            for f in files:
                if "README" in f:
                    with open(os.path.join(root, f), "r") as readme:
                        lines = readme.readlines()
                        for line in lines:
                            if "[config]" in line:
                                start = line.find("[config]")
                                end = line.find(".py", start)
                                start = line.rfind("/", start, end)
                                config_name = line[start + 1 : end + 3]
                                start = line.find("[model]")
                                end = line.find(".pth", start)
                                ckpt_name = line[start + 8 : end + 4]
                                models[config_name] = ckpt_name
    return models


CONFIG_TO_PRETRAINED = get_config_pretrained_url_mapping()


def get_pretrained_ckpt_path(download_directory: str, config_file: str) -> Tuple[Any, Any]:
    """
    If the config_file has an associated pretrained checkpoint,
    return path to downloaded checkpoint and preloaded checkpoint

    Arguments:
        download_directory: path to download checkpoints to
        config_file: mmdet config file path for which to find and load pretrained weights
    Returns:
        checkpoint path, loaded checkpoint
    """
    config_file = config_file.split("/")[-1]
    if config_file in CONFIG_TO_PRETRAINED:
        ckpt_path = model_hub.utils.download_url(
            download_directory, CONFIG_TO_PRETRAINED[config_file]
        )
        return ckpt_path, torch.load(ckpt_path)  # type: ignore
    return None, None


def build_fp16_loss_scaler(loss_scale: mmcv.Config) -> Any:
    """
    This function is derived from mmcv, which is coverd by the Apache 2.0 License.
    Copyright (c) OpenMMLab. All rights reserved.

    Arguments:
        loss_scale (float | str | dict): Scale factor configuration.
                    If loss_scale is a float, static loss scaling will be used with
                    the specified scale. If loss_scale is a string, it must be
                    'dynamic', then dynamic loss scaling will be used.
                    It can also be a dict containing arguments of GradScalar.
                    Defaults to 512. For PyTorch >= 1.6, mmcv uses official
                    implementation of GradScaler. If you use a dict version of
                    loss_scale to create GradScaler, please refer to:
                    https://pytorch.org/docs/stable/amp.html#torch.cuda.amp.GradScaler
                    for the parameters.
    Examples:
        >>> loss_scale = dict(
        ...     init_scale=65536.0,
        ...     growth_factor=2.0,
        ...     backoff_factor=0.5,
        ...     growth_interval=2000
        ... )
    """
    if loss_scale == "dynamic":
        loss_scaler = torch.cuda.amp.GradScaler()  # type: ignore
    elif isinstance(loss_scale, float):
        loss_scaler = torch.cuda.amp.GradScaler(init_scale=loss_scale)  # type: ignore
    elif isinstance(loss_scale, dict):
        loss_scaler = torch.cuda.amp.GradScaler(**loss_scale)  # type: ignore
    else:
        raise Exception(
            "Cannot parse fp16 configuration.  Expected cfg to be str(dynamic), float or dict."
        )
    return loss_scaler
