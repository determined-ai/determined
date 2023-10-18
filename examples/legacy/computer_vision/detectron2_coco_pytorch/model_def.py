import logging
import os
from collections import OrderedDict
from typing import Any, Dict, Sequence, Tuple, Union, cast

import torch
from detectron2.checkpoint import DetectionCheckpointer
from detectron2.config import get_cfg
from detectron2.modeling import build_model
from detectron2.solver import build_lr_scheduler, build_optimizer
from detectron2.utils.events import EventStorage
from detectron2_files.common import *
from detectron2_files.data import *
from detectron2_files.evaluator import *
from torch import nn

from determined.pytorch import DataLoader, LRScheduler, PyTorchTrial, PyTorchTrialContext

TorchData = Union[Dict[str, torch.Tensor], Sequence[torch.Tensor], torch.Tensor]


class DetectronTrial(PyTorchTrial):
    def __init__(self, context: PyTorchTrialContext):
        self.context = context

        self.cfg = self.setup_cfg()
        model = build_model(self.cfg)

        checkpointer = DetectionCheckpointer(model, self.cfg.OUTPUT_DIR)
        checkpointer.resume_or_load(self.cfg.MODEL.WEIGHTS, resume=False)
        self.model = self.context.wrap_model(checkpointer.model)

        optimizer = build_optimizer(self.cfg, self.model)
        self.optimizer = self.context.wrap_optimizer(optimizer)

        self.scheduler = build_lr_scheduler(self.cfg, self.optimizer)
        self.scheduler = self.context.wrap_lr_scheduler(
            self.scheduler, LRScheduler.StepMode.STEP_EVERY_BATCH
        )

        self.dataset_name = self.cfg.DATASETS.TEST[0]
        self.evaluators = get_evaluator(
            self.cfg,
            self.dataset_name,
            self.context.get_hparam("output_dir"),
            self.context.get_hparam("fake_data"),
        )
        self.val_reducer = self.context.wrap_reducer(
            EvaluatorReducer(self.evaluators), for_training=False
        )

        self.context.experimental.disable_dataset_reproducibility_checks()

    def setup_cfg(self):
        cfg = get_cfg()
        cfg.merge_from_file(self.context.get_hparam("model_yaml"))
        cfg.SOLVER.IMS_PER_BATCH = self.context.get_per_slot_batch_size()
        cfg.freeze()
        return cfg

    def build_training_data_loader(self):
        seed = self.context.get_trial_seed()
        rank = self.context.distributed.get_rank()
        bs = self.context.get_per_slot_batch_size()
        size = self.context.distributed.get_size()

        dataloader = build_detection_train_loader(
            self.cfg, per_gpu_bs=bs, seed=seed, rank=rank, world_size=size, context=self.context
        )
        return dataloader

    def build_validation_data_loader(self):
        data_loader = build_detection_test_loader(self.cfg, self.dataset_name, context=self.context)
        return data_loader

    def train_batch(self, batch: TorchData, epoch_idx: int, batch_idx: int):
        with EventStorage():
            loss_dict = self.model(batch)
        losses = sum(loss_dict.values())
        losses_reduced = sum(loss for loss in loss_dict.values())

        loss_dict["lr"] = self.optimizer.param_groups[0]["lr"]
        loss_dict["loss"] = losses
        loss_dict["total_loss"] = losses_reduced

        self.context.backward(losses)
        self.context.step_optimizer(self.optimizer)

        return loss_dict

    def evaluate_batch(self, batch: TorchData):
        outputs = self.model(batch)
        preds = self.evaluators.process(batch, outputs)

        # results will be generated with validation_reducer
        if preds is not None:
            self.val_reducer.update(preds)

        return {}
