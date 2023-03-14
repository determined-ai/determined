"""
Determined training loop for mmdetection
mmdetection: https://github.com/open-mmlab/mmdetection.

This Determined trial definition makes use of mmcv and mmdet libraries.
The license for mmcv and mmdet is reproduced below.

Copyright (c) OpenMMLab. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
"""

import logging
import os
from typing import Any, Dict, List

import attrdict
import mmcv
import mmcv.parallel
import mmcv.runner
import mmdet.core
import mmdet.datasets
import mmdet.models
import numpy as np
import torch

import determined.pytorch as det_torch
from determined.common import set_logger
from model_hub.mmdetection import _callbacks as callbacks
from model_hub.mmdetection import _data as data
from model_hub.mmdetection import _data_backends as data_backends
from model_hub.mmdetection import utils as utils


class MMDetTrial(det_torch.PyTorchTrial):
    """
    This trial serves as the trainer for MMDetection models.  It replaces the
    `mmcv runner used by MMDetection
    <https://github.com/open-mmlab/mmdetection/blob/master/mmdet/apis/train.py>`_.

    For nearly all use cases, you can just use this trial definition and control behavior
    by changing the MMDetection config.  If you want to customize the trial further, you
    can use this trial as the starting point.
    """

    def __init__(self, context: det_torch.PyTorchTrialContext) -> None:
        self.context = context
        self.hparams = attrdict.AttrDict(context.get_hparams())
        self.data_config = attrdict.AttrDict(context.get_data_config())
        self.cfg = self.build_mmdet_config()
        # We will control how data is moved to GPU.
        self.context.experimental.disable_auto_to_device()

        # Build model and make sure it's compatible with horovod.
        self.model = mmdet.models.build_detector(self.cfg.model)

        # Initialize model
        self.model.init_weights()

        # If use_pretrained, try loading pretrained weights for the mmcv config if available.
        if self.hparams.use_pretrained:
            ckpt_path, ckpt = utils.get_pretrained_ckpt_path("/tmp", self.hparams.config_file)
            if ckpt_path is not None:
                logging.info("Loading from pretrained weights.")
                if "state_dict" in ckpt:
                    self.model.load_state_dict(ckpt["state_dict"])
                else:
                    self.model.load_state_dict(ckpt)

        # If fp16 is specified in the mmdet config, we will use torch native amp.
        fp16_cfg = self.cfg.get("fp16", None)
        if fp16_cfg is not None:
            self.setup_torch_amp(fp16_cfg)

        self.model = self.context.wrap_model(self.model)

        self.optimizer = self.context.wrap_optimizer(
            mmcv.runner.build_optimizer(self.model, self.cfg.optimizer)
        )
        self.model.zero_grad()

        self.clip_grads_fn = None
        if self.cfg.optimizer_config.grad_clip is not None:
            self.clip_grads_fn = lambda x: torch.nn.utils.clip_grad_norm_(
                x,
                self.cfg.optimizer_config.grad_clip.max_norm,
                self.cfg.optimizer_config.grad_clip.norm_type,
            )

        # mmdet sets loggers in the package that interrupt with Determined logging.
        # We reset the root logger after mmdet models are initialized.
        set_logger(bool(self.context.env.experiment_config.get("debug", False)))

    def build_mmdet_config(self) -> mmcv.Config:
        """
        Apply overrides to the mmdet config according to the following experiment config fields:
        - data.file_client_args
        - hyperparameters.merge_config
        - hyperparameters.override_mmdet_config.

        Returns:
            overridden mmdet config
        """
        config_file = self.hparams.config_file
        if not os.path.exists(config_file):
            config_dir = os.getenv("MMDETECTION_CONFIG_DIR")
            if config_dir is not None:
                config_file = os.path.join(config_dir, config_file)
            if config_dir is None or not os.path.exists(config_file):
                raise OSError(f"Config file {self.hparams.config_file} not found.")
        cfg = mmcv.Config.fromfile(config_file)
        cfg.data.val.test_mode = True

        # If a backend is specified, we will the backend used in all occurrences of
        # LoadImageFromFile in the mmdet config.
        if self.data_config.file_client_args is not None:
            data_backends.sub_backend(self.data_config.file_client_args, cfg)
        if self.hparams.merge_config is not None:
            override_config = mmcv.Config.fromfile(self.hparams.merge_config)
            new_config = mmcv.Config._merge_a_into_b(override_config, cfg._cfg_dict)
            cfg = mmcv.Config(new_config, cfg._text, cfg._filename)

        if "override_mmdet_config" in self.hparams:
            cfg.merge_from_dict(self.hparams.override_mmdet_config)

        cfg.data.val.pipeline = mmdet.datasets.replace_ImageToTensor(cfg.data.val.pipeline)
        cfg.data.test.pipeline = mmdet.datasets.replace_ImageToTensor(cfg.data.test.pipeline)

        # Save and log the resulting config.
        if "save_cfg" in self.hparams and self.hparams.save_cfg:
            save_dir = self.hparams.save_dir if "save_dir" in self.hparams else "/tmp"
            extension = cfg._filename.split(".")[-1]
            cfg.dump(os.path.join(save_dir, f"final_config.{extension}"))
        logging.info(cfg)
        return cfg

    def setup_torch_amp(self, fp16_cfg: mmcv.Config) -> None:
        """
        Build the torch amp gradient scaler according to the fp16_cfg.
        Please refer to :meth:`model_hub.mmdetection.build_fp16_loss_scaler` function
        to see how to configure fp16 training.
        """
        mmcv.runner.wrap_fp16_model(self.model)
        loss_scaler = utils.build_fp16_loss_scaler(fp16_cfg.loss_scale)
        self.loss_scaler = self.context.wrap_scaler(loss_scaler)
        self.context.experimental._auto_amp = True

    def build_callbacks(self) -> Dict[str, det_torch.PyTorchCallback]:
        self.lr_updater = None
        hooks = {}  # type: Dict[str, det_torch.PyTorchCallback]
        if "lr_config" in self.cfg:
            logging.info("Adding lr updater callback.")
            self.lr_updater = callbacks.LrUpdaterCallback(
                self.context, lr_config=self.cfg.lr_config
            )
            hooks["lr_updater"] = self.lr_updater
        return hooks

    def train_batch(self, batch: Any, epoch_idx: int, batch_idx: int) -> Dict[str, torch.Tensor]:
        batch = self.to_device(batch)
        if self.lr_updater is not None:
            self.lr_updater.on_batch_start()
        batch = {key: batch[key].data[0] for key in batch}

        losses = self.model(**batch)
        loss, log_vars = self.model._parse_losses(losses)
        self.model.zero_grad()
        self.context.backward(loss)
        self.context.step_optimizer(
            self.optimizer, clip_grads=self.clip_grads_fn, auto_zero_grads=False
        )

        lr = self.optimizer.param_groups[0]["lr"]
        metrics = {"loss": loss, "lr": lr}
        metrics.update(log_vars)
        return metrics

    def evaluate_batch(self, batch: Any, batch_idx: int) -> Dict[str, Any]:
        batch = self.to_device(batch)
        batch = {key: batch[key][0].data for key in batch}
        with torch.no_grad():  # type: ignore
            result = self.model(return_loss=False, rescale=True, **batch)
        if isinstance(result[0], tuple):
            result = [
                (bbox_results, mmdet.core.encode_mask_results(mask_results))
                for bbox_results, mask_results in result
            ]
        self.reducer.update(([b["idx"] for b in batch["img_metas"][0]], result))
        return {}

    def build_training_data_loader(self) -> det_torch.DataLoader:
        dataset, dataloader = data.build_dataloader(
            self.cfg.data,
            "train",
            self.context,
            True,
        )
        self.model.CLASSES = dataset.CLASSES  # type: ignore
        return dataloader

    def build_validation_data_loader(self) -> det_torch.DataLoader:
        dataset, dataloader = data.build_dataloader(
            self.cfg.data,
            "val",
            self.context,
            False,
        )

        def evaluate_fn(results: List[Any]) -> Any:
            # Determined's distributed batch sampler interleaves shards on each GPU slot so
            # sample i goes to worker with rank i % world_size.  Therefore, we need to re-sort
            # all the samples once we gather the predictions before computing the validation metric.
            inds, results = zip(*results)
            inds = [ind for sub_ind in inds for ind in sub_ind]
            results = [res for result in results for res in result]
            sorted_inds = np.argsort(inds)
            results = [results[i] for i in sorted_inds]

            eval_kwargs = self.cfg.evaluation

            for key in ["interval", "tmpdir", "start", "gpu_collect"]:
                eval_kwargs.pop(key, None)

            metrics = dataset.evaluate(results, **eval_kwargs)  # type: ignore
            if not len(metrics):
                return {"bbox_mAP": 0}
            return metrics

        self.reducer = self.context.wrap_reducer(
            evaluate_fn, for_training=False, for_validation=True
        )
        return dataloader

    def get_batch_length(self, batch: Any) -> int:
        if isinstance(batch["img"], mmcv.parallel.data_container.DataContainer):
            length = len(batch["img"].data[0])
        else:
            # The validation data has a different format so we have separate handling below.
            length = len(batch["img"][0].data[0])
        return length

    def to_device(self, batch: Any) -> Dict[str, Any]:
        new_data = {}
        for k, item in batch.items():
            if isinstance(item, mmcv.parallel.data_container.DataContainer) and not item.cpu_only:
                new_data[k] = mmcv.parallel.data_container.DataContainer(
                    self.context.to_device(item.data),
                    item.stack,
                    item.padding_value,
                    item.cpu_only,
                    item.pad_dims,
                )
            # The validation data has a different format so we have separate handling below.
            elif (
                isinstance(item, list)
                and len(item) == 1
                and isinstance(item[0], mmcv.parallel.data_container.DataContainer)
                and not item[0].cpu_only
            ):
                new_data[k] = [
                    mmcv.parallel.data_container.DataContainer(
                        self.context.to_device(item[0].data),
                        item[0].stack,
                        item[0].padding_value,
                        item[0].cpu_only,
                        item[0].pad_dims,
                    )
                ]
            else:
                new_data[k] = item
        return new_data
