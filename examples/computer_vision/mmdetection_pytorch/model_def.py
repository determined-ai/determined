"""
Port of mmdetection repo at https://github.com/open-mmlab/mmdetection.
"""

from typing import Any, Dict, Sequence, Union

import torch
from torch.optim.lr_scheduler import MultiStepLR

from mmcv import Config
from mmcv.runner import build_optimizer
from mmcv import ProgressBar
from mmdet.models import build_detector
from mmdet.core import encode_mask_results

# from mmdet.datasets import replace_ImageToTensor

from determined.pytorch import DataLoader, PyTorchTrial, LRScheduler
import determined as det

from utils.data import build_dataloader, sub_backend
from utils.lr_schedulers import WarmupWrapper


TorchData = Union[Dict[str, torch.Tensor], Sequence[torch.Tensor], torch.Tensor]


class MMDetTrial(PyTorchTrial):
    def __init__(self, context: det.TrialContext) -> None:
        self.context = context
        self.hparams = context.get_hparams()
        self.data_config = context.get_data_config()
        self.cfg = Config.fromfile(self.hparams["config_file"])

        self.cfg.data.train.ann_file = self.data_config["train_ann_file"]
        self.cfg.data.val.ann_file = self.data_config["val_ann_file"]
        self.cfg.data.val.test_mode = True
        self.cfg.data.workers_per_gpu = self.data_config["workers_per_gpu"]

        if self.data_config["backend"] in ["gcs", "fake"]:
            sub_backend(self.data_config["backend"], self.cfg)

        print(self.cfg)

        self.model = self.context.wrap_model(
            build_detector(
                self.cfg.model, train_cfg=self.cfg.train_cfg, test_cfg=self.cfg.test_cfg
            )
        )

        self.optimizer = self.context.wrap_optimizer(
            build_optimizer(self.model, self.cfg.optimizer)
        )

        scheduler_cls = WarmupWrapper(MultiStepLR)
        scheduler = scheduler_cls(
            self.hparams["warmup"],  # warmup schedule
            self.hparams["warmup_iters"],  # warmup_iters
            self.hparams["warmup_ratio"],  # warmup_ratio
            self.optimizer,
            [self.hparams["step1"], self.hparams["step2"]],  # milestones
            self.hparams["gamma"],  # gamma
        )
        self.scheduler = self.context.wrap_lr_scheduler(
            scheduler, step_mode=LRScheduler.StepMode.MANUAL_STEP
        )

        self.clip_grads_fn = (
            lambda x: torch.nn.utils.clip_grad_norm_(x, self.hparams["clip_grads_norm"])
            if self.hparams["clip_grads"]
            else None
        )

    def train_batch(
        self, batch: TorchData, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        # TODO: improve this so that we can pass to device as part of dataloader.
        batch = {key: batch[key][0] for key in batch}
        losses = self.model.forward_train(**batch)
        loss, log_vars = self.model._parse_losses(losses)
        self.context.backward(loss)
        self.context.step_optimizer(self.optimizer, clip_grads=self.clip_grads_fn)
        self.scheduler.step()

        metrics = {"loss": loss}
        metrics.update(log_vars)
        return metrics

    def evaluate_full_dataset(
        self, data_loader: torch.utils.data.DataLoader
    ) -> Dict[str, Any]:
        if self.data_config["backend"]=="fake":
            return {'bbox_mAP': 0}

        # Will need custom reducer to do this across gpus
        prog_bar = ProgressBar(len(data_loader.dataset))

        results = []
        for i, batch in enumerate(data_loader):
            # TODO: modify this to use cpu_only field of DataContainers.
            batch["img"] = [self.context.to_device(batch["img"])]
            batch = {key: batch[key][0] for key in batch}
            with torch.no_grad():
                result = self.model(return_loss=False, rescale=True, **batch)
            if isinstance(result[0], tuple):
                result = [
                    (bbox_results, encode_mask_results(mask_results))
                    for bbox_results, mask_results in result
                ]
            batch_size = len(result)
            results.extend(result)

            for _ in range(batch_size):
                prog_bar.update()

        eval_kwargs = self.cfg.evaluation

        for key in ["interval", "tmpdir", "start", "gpu_collect"]:
            eval_kwargs.pop(key, None)

        metrics = data_loader.dataset.evaluate(results, **eval_kwargs)
        return metrics

    def build_training_data_loader(self) -> DataLoader:
        dataset, dataloader = build_dataloader(
            self.cfg.data.train,
            self.context.get_per_slot_batch_size(),
            self.context.distributed.get_size(),
            self.cfg.data.workers_per_gpu,
            True,
        )
        self.model.CLASSES = dataset.CLASSES
        return dataloader

    def build_validation_data_loader(self) -> DataLoader:
        # For now only support eval with batch size 1.
        # if self.context.distributed.get_size() > 1:
        #    self.cfg.data.val.pipeline = replace_ImageToTensor(self.cfg.data.test.pipeline)

        dataset, dataloader = build_dataloader(self.cfg.data.val, 1, 1, 8, False)
        return dataloader
