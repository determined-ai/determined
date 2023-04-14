import logging
import math
import sys
import time
from copy import deepcopy
from typing import Any, Dict, Sequence, Tuple, Union, cast

import apex
import numpy as np
import torch
import yaml
from apex import amp
from effdet import create_dataset, create_loader, create_model
from effdet.anchors import AnchorLabeler, Anchors
from effdet.data import SkipSubset, resolve_input_config
from effdet.data.loader import DetectionFastCollate
from effdet.data.transforms import *
from efficientdet_files.evaluator import *
from efficientdet_files.modelema import ModelEma
from efficientdet_files.utils import *
from horovod.torch.sync_batch_norm import SyncBatchNorm
from timm.optim import create_optimizer
from timm.scheduler import create_scheduler

from determined import pytorch
from determined.experimental import Determined
from determined.pytorch import DataLoader, LRScheduler, PyTorchTrial, PyTorchTrialContext

TorchData = Union[Dict[str, torch.Tensor], Sequence[torch.Tensor], torch.Tensor]


class EffDetTrial(PyTorchTrial):
    def __init__(self, context: PyTorchTrialContext) -> None:
        self.context = context
        self.hparam = self.context.get_hparam
        self.args = DotDict(self.context.get_hparams())
        # Create a unique download directory for each rank so they don't overwrite each other.
        self.download_directory = f"/tmp/data-rank{self.context.distributed.get_rank()}"
        self.num_slots = int(self.context.get_experiment_config()["resources"]["slots_per_trial"])

        if self.args.sync_bn and self.num_slots == 1:
            print(
                "Can not use sync_bn with one slot. Either set sync_bn to False or use distributed training."
            )
            sys.exit()
        self.args.pretrained_backbone = not self.args.no_pretrained_backbone
        self.args.prefetcher = not self.args.no_prefetcher

        tmp = []
        for arg in self.args.lr_noise.split(" "):
            tmp.append(float(arg))
        self.args.lr_noise = tmp

        self.model = create_model(
            self.args.model,
            bench_task="train",
            num_classes=self.args.num_classes,
            pretrained=self.args.pretrained,
            pretrained_backbone=self.args.pretrained_backbone,
            redundant_bias=self.args.redundant_bias,
            label_smoothing=self.args.smoothing,
            new_focal=self.args.new_focal,
            jit_loss=self.args.jit_loss,
            bench_labeler=self.args.bench_labeler,
            checkpoint_path=self.args.initial_checkpoint,
        )
        self.model_config = self.model.config
        self.input_config = resolve_input_config(self.args, model_config=self.model_config)
        print("h: ", self.args.model, sum([m.numel() for m in self.model.parameters()]))

        if self.args.sync_bn:
            print("creating batch sync model")
            if self.args.model_ema:
                print("creating batch sync ema model")

                self.model_ema = self.context.wrap_model(deepcopy(self.model))
            self.model = self.convert_syncbn_model(self.model)

        self.model = self.context.wrap_model(self.model)
        print(
            "Model created, param count:",
            self.args.model,
            sum([m.numel() for m in self.model.parameters()]),
        )

        self.optimizer = self.context.wrap_optimizer(create_optimizer(self.args, self.model))
        print("Created optimizer: ", self.optimizer)

        if self.args.amp:
            print("using amp")
            if self.args.sync_bn and self.args.model_ema:
                print("using sync_bn and model_ema when creating apex_amp")
                (self.model, self.model_ema), self.optimizer = self.context.configure_apex_amp(
                    [self.model, self.model_ema],
                    self.optimizer,
                    min_loss_scale=self.hparam("min_loss_scale"),
                )
            else:
                self.model, self.optimizer = self.context.configure_apex_amp(
                    self.model, self.optimizer, min_loss_scale=self.hparam("min_loss_scale")
                )

        if self.args.model_ema:
            print("using model ema")
            if self.args.sync_bn:
                print("using model ema batch syn")
                self.model_ema = ModelEma(
                    self.model_ema, context=self.context, decay=self.args.model_ema_decay
                )
            else:
                # Important to create EMA model after cuda(), DP wrapper, and AMP but before SyncBN and DDP wrapper
                self.model_ema = ModelEma(
                    self.model, context=self.context, decay=self.args.model_ema_decay
                )

        self.lr_scheduler, self.num_epochs = create_scheduler(self.args, self.optimizer)
        self.lr_scheduler = self.context.wrap_lr_scheduler(
            self.lr_scheduler, LRScheduler.StepMode.MANUAL_STEP
        )

        self.cur_epoch = 0
        self.num_updates = 0 * self.cur_epoch

        if self.args.prefetcher:
            self.train_mean, self.train_std, self.train_random_erasing = self.calculate_means(
                mean=self.input_config["mean"],
                std=self.input_config["std"],
                re_prob=self.args.reprob,
                re_mode=self.args.remode,
                re_count=self.args.recount,
            )

            self.val_mean, self.val_std, self.val_random_erasing = self.calculate_means(
                self.input_config["mean"], self.input_config["std"]
            )

        self.val_reducer = self.context.wrap_reducer(self.validation_reducer, for_training=False)

    def build_callbacks(self):
        return {"model": self.model_ema.callback_object()}

    def calculate_means(
        self,
        mean=IMAGENET_DEFAULT_MEAN,
        std=IMAGENET_DEFAULT_STD,
        re_prob=0.0,
        re_mode="pixel",
        re_count=1,
    ):
        # We need to precalculate the prefetcher.
        mean = torch.tensor([x * 255 for x in mean]).cuda().view(1, 3, 1, 1)
        std = torch.tensor([x * 255 for x in std]).cuda().view(1, 3, 1, 1)

        if re_prob > 0.0:
            random_erasing = RandomErasing(probability=re_prob, mode=re_mode, max_count=re_count)
        else:
            random_erasing = None

        return mean, std, random_erasing

    def _create_loader(
        self,
        dataset,
        input_size,
        batch_size,
        is_training=False,
        use_prefetcher=True,
        re_prob=0.0,
        re_mode="pixel",
        re_count=1,
        interpolation="bilinear",
        fill_color="mean",
        mean=IMAGENET_DEFAULT_MEAN,
        std=IMAGENET_DEFAULT_STD,
        num_workers=4,
        distributed=False,
        pin_mem=False,
        anchor_labeler=None,
    ):
        if isinstance(input_size, tuple):
            img_size = input_size[-2:]
        else:
            img_size = input_size

        if is_training:
            transform = transforms_coco_train(
                img_size,
                interpolation=interpolation,
                use_prefetcher=use_prefetcher,
                fill_color=fill_color,
                mean=mean,
                std=std,
            )
        else:
            transform = transforms_coco_eval(
                img_size,
                interpolation=interpolation,
                use_prefetcher=use_prefetcher,
                fill_color=fill_color,
                mean=mean,
                std=std,
            )

        dataset.transform = transform

        sampler = None

        collate_fn = DetectionFastCollate(anchor_labeler=anchor_labeler)
        loader = DataLoader(
            dataset,
            batch_size=self.context.get_per_slot_batch_size(),
            shuffle=False,  # sampler is None and is_training,
            num_workers=num_workers,
            sampler=sampler,
            pin_memory=pin_mem,
            collate_fn=collate_fn,
        )

        return loader

    def build_training_data_loader(self):
        if self.context.get_hparam("fake_data"):
            dataset_train = FakeBackend()
            self.dataset_eval = dataset_train

        else:
            dataset_train, self.dataset_eval = create_dataset(self.args.dataset, self.args.root)

        self.labeler = None
        if not self.args.bench_labeler:
            self.labeler = AnchorLabeler(
                Anchors.from_config(self.model_config),
                self.model_config.num_classes,
                match_threshold=0.5,
            )

        loader_train = self._create_loader(
            dataset_train,
            input_size=self.input_config["input_size"],
            batch_size=self.context.get_per_slot_batch_size(),
            is_training=True,
            use_prefetcher=self.args.prefetcher,
            re_prob=self.args.reprob,
            re_mode=self.args.remode,
            re_count=self.args.recount,
            # color_jitter=self.args.color_jitter,
            # auto_augment=self.args.aa,
            interpolation=self.args.train_interpolation or self.input_config["interpolation"],
            fill_color=self.input_config["fill_color"],
            mean=self.input_config["mean"],
            std=self.input_config["std"],
            num_workers=1,  # self.args.workers,
            distributed=self.args.distributed,
            pin_mem=self.args.pin_mem,
            anchor_labeler=self.labeler,
        )

        if (
            not self.context.get_hparam("fake_data")
            and self.model_config.num_classes < loader_train.dataset.parser.max_label
        ):
            logging.error(
                f"Model {self.model_config.num_classes} has fewer classes than dataset {loader_train.dataset.parser.max_label}."
            )
            sys.exit(1)
        if (
            not self.context.get_hparam("fake_data")
            and self.model_config.num_classes > loader_train.dataset.parser.max_label
        ):
            logging.warning(
                f"Model {self.model_config.num_classes} has more classes than dataset {loader_train.dataset.parser.max_label}."
            )

        self.data_length = len(loader_train)

        return loader_train

    def build_validation_data_loader(self):
        if self.args.val_skip > 1:
            self.dataset_eval = SkipSubset(self.dataset_eval, self.args.val_skip)
        self.loader_eval = self._create_loader(
            self.dataset_eval,
            input_size=self.input_config["input_size"],
            batch_size=self.context.get_per_slot_batch_size(),
            is_training=False,
            use_prefetcher=self.args.prefetcher,
            interpolation=self.input_config["interpolation"],
            fill_color=self.input_config["fill_color"],
            mean=self.input_config["mean"],
            std=self.input_config["std"],
            num_workers=self.args.workers,
            distributed=self.args.distributed,
            pin_mem=self.args.pin_mem,
            anchor_labeler=self.labeler,
        )

        self.evaluator = create_evaluator(
            self.args.dataset, self.loader_eval.dataset, pred_yxyx=False, context=self.context
        )

        return self.loader_eval

    def clip_grads(self, params):
        torch.nn.utils.clip_grad_norm_(amp.master_params(self.optimizer), self.args.clip_grad)

    def train_batch(self, batch: TorchData, epoch_idx: int, batch_idx: int):
        if epoch_idx != self.cur_epoch and self.lr_scheduler is not None:
            self.cur_epoch = epoch_idx
            self.num_updates = epoch_idx * self.data_length

            self.lr_scheduler.step(self.cur_epoch)
            lrl2 = [param_group["lr"] for param_group in self.optimizer.param_groups]
            print(self.cur_epoch, "new lr: ", lrl2, batch_idx)

        input, target = batch

        if self.args.prefetcher:
            input = input.float().sub_(self.train_mean).div_(self.train_std)
            if self.train_random_erasing is not None:
                input = self.train_random_erasing(input, target)

        if self.args.channels_last:
            input = input.contiguous(memory_format=torch.channels_last)

        output = self.model(input, target)
        loss = output["loss"]

        self.context.backward(loss)
        self.context.step_optimizer(self.optimizer, self.clip_grads)

        if self.model_ema is not None:
            self.model_ema.update(self.model)

        self.num_updates += 1
        if self.lr_scheduler is not None:
            self.lr_scheduler.step_update(num_updates=self.num_updates)
            lrl = [param_group["lr"] for param_group in self.optimizer.param_groups]

        return {"loss": loss}

    def evaluate_batch(self, batch: TorchData):
        input, target = batch
        if self.args.prefetcher:
            input = input.float().sub_(self.val_mean).div_(self.val_std)

        if self.val_random_erasing is not None:
            input = self.val_random_erasing(input, target)

        input = self.context.to_device(input)
        target = self.context.to_device(target)

        output = self.model_ema.ema(input, target)
        loss = output["loss"]

        reduced_loss = loss.data.item()

        if reduced_loss is np.nan or math.isnan(reduced_loss):
            # The original implementation is sensitive to the hyperparameters and configurations.
            # This is used to help debug.
            if not self.context.get_hparam("fake_data"):
                for name, p in self.model.named_parameters():
                    print(
                        "Nan occurred: ",
                        name,
                        "norm: ",
                        p.grad.norm().item(),
                        "sum: ",
                        p.grad.sum().item(),
                        "max: ",
                        p.grad.min().item(),
                        "min: ",
                        p.grad.min().item(),
                    )
            reduced_loss = 0

        if self.evaluator is not None:
            self.evaluator.add_predictions(output["detections"], target)

        vals = (self.evaluator.img_indices, self.evaluator.predictions)
        self.val_reducer.update(vals)
        self.evaluator.reset()

        return {"val_loss": reduced_loss}

    def validation_reducer(self, values):
        concat_imgs, concat_pred = zip(*values)

        new_concat_imgs = []
        for val in concat_imgs:
            new_concat_imgs.extend(val)

        new_concat_pred = []
        for val in concat_pred:
            new_concat_pred.extend(val)

        self.evaluator.img_indices = new_concat_imgs
        self.evaluator.predictions = new_concat_pred
        metrics_map = float(self.evaluator.evaluate())
        self.evaluator.reset()
        return {"map": metrics_map}

    def convert_syncbn_model(self, module, process_group=None, channel_last=False):
        """
        This function is apex's convert_syncbn_model; however, we instead use horovod's sync batch norm.
        """
        mod = module
        if isinstance(module, torch.nn.modules.instancenorm._InstanceNorm):
            return module
        if isinstance(module, torch.nn.modules.batchnorm._BatchNorm):
            mod = SyncBatchNorm(
                module.num_features,
                module.eps,
                module.momentum,
                module.affine,
                module.track_running_stats,
            )  # , process_group, channel_last=channel_last
            mod.running_mean = module.running_mean
            mod.running_var = module.running_var
            if module.affine:
                mod.weight.data = module.weight.data.clone().detach()
                mod.bias.data = module.bias.data.clone().detach()
        for name, child in module.named_children():
            mod.add_module(
                name,
                self.convert_syncbn_model(
                    child, process_group=process_group, channel_last=channel_last
                ),
            )
        # TODO(jie) should I delete model explicitly?
        del module
        return mod
