import torch.nn as nn
from determined.pytorch import DataLoader, PyTorchTrial, PyTorchTrialContext, LRScheduler, samplers

# from efficientdet_files.utils import *

from PIL import Image
import requests
import matplotlib.pyplot as plt
# %config InlineBackend.figure_format = 'retina'
import pickle, os
import numpy as np

import torch
import timm
import torchvision
import torchvision.transforms as T

from timm.optim import create_optimizer
from timm.scheduler import create_scheduler
from timm.utils import NativeScaler, get_state_dict, ModelEma, accuracy
from timm.models import create_model

from timm.data.constants import IMAGENET_DEFAULT_MEAN, IMAGENET_DEFAULT_STD

from attrdict import AttrDict
from datasets import build_dataset
import torch.distributed as dist
import models
from losses import DistillationLoss

from typing import Any, Dict, Sequence, Tuple, Union, cast

torch.set_grad_enabled(False);

TorchData = Union[Dict[str, torch.Tensor], Sequence[torch.Tensor], torch.Tensor]

def unpickle(file):
    with open(file, 'rb') as fo:
        dict = pickle.load(fo)
    return dict

def load_databatch(data_folder, idx, img_size=32):
    data_file = os.path.join(data_folder, 'train_data_batch_')

    d = unpickle(data_file + str(idx))
    x = d['data']
    y = d['labels']
    mean_image = d['mean']

    x = x/np.float32(255)
    mean_image = mean_image/np.float32(255)

    # Labels are indexed from 1, shift it so that indexes start at 0
    y = [i-1 for i in y]
    data_size = x.shape[0]

    x -= mean_image

    img_size2 = img_size * img_size

    x = np.dstack((x[:, :img_size2], x[:, img_size2:2*img_size2], x[:, 2*img_size2:]))
    x = x.reshape((x.shape[0], img_size, img_size, 3)).transpose(0, 3, 1, 2)

    # create mirrored images
    X_train = x[0:data_size, :, :, :]
    Y_train = y[0:data_size]
    X_train_flip = X_train[:, :, :, ::-1]
    Y_train_flip = Y_train
    X_train = np.concatenate((X_train, X_train_flip), axis=0)
    Y_train = np.concatenate((Y_train, Y_train_flip), axis=0)

    return dict(
        X_train=X_train,
        Y_train=Y_train.astype('int32'),
        mean=mean_image)


def is_dist_avail_and_initialized():
    if not dist.is_available():
        return False
    if not dist.is_initialized():
        return False
    return True


def get_world_size():
    if not is_dist_avail_and_initialized():
        return 1
    return dist.get_world_size()


def get_rank():
    if not is_dist_avail_and_initialized():
        return 0
    return dist.get_rank()

class DeitTrial(PyTorchTrial):
    def __init__(self, context: PyTorchTrialContext):

        self.context = context

        # Create a unique download directory for each rank so they don't overwrite each
        # other when doing distributed training.
        self.download_directory = f"/tmp/data-rank{self.context.distributed.get_rank()}"
        self.args = AttrDict(self.context.get_hparams())

        self.args.data_path = self.download_directory

        _, nb_classes = build_dataset(is_train=True, args=self.args)

        model = create_model(
            'deit_base_patch16_224',
            pretrained=False,
            num_classes=nb_classes,
            drop_rate=self.args['drop'],
            drop_path_rate=self.args['drop_path'],
            drop_block_rate=None,
        )
        self.model = self.context.wrap_model(model)

        n_parameters = sum(p.numel() for p in self.model.parameters() if p.requires_grad)
        print('number of params:', n_parameters)

        model_without_ddp = self.model

        optimizer = create_optimizer(self.args, model_without_ddp)
        self.optimizer = self.context.wrap_optimizer(optimizer)

        criterion = torch.nn.CrossEntropyLoss()
        self.criterion = DistillationLoss(
            criterion, None, 'none', 0.5, 1.0
        )

    
        
    def train_batch(self, batch: TorchData, epoch_idx: int, batch_idx: int):
        samples, targets = batch
        with torch.cuda.amp.autocast():
            outputs = self.model(samples)
            loss = self.criterion(samples, outputs, targets)

        loss_scaler = NativeScaler()
        is_second_order = hasattr(self.optimizer, 'is_second_order') and self.optimizer.is_second_order
        loss_scaler(loss, self.optimizer, clip_grad=0,
                    parameters=self.model.parameters(), create_graph=is_second_order)
        # self.context.backward(loss)
        self.context.step_optimizer(self.optimizer)
        return {"loss": loss.item()}

    def evaluate_batch(self, batch: TorchData):
        self.model.eval()
        images, target = batch

        with torch.cuda.amp.autocast():
            output = self.model(images)
            loss = self.criterion(output, target)

        acc1, acc5 = accuracy(output, target, topk=(1, 5))
        return {"loss": loss, 'top1': acc1[0], 'top5': acc5[0]}


    def build_training_data_loader(self):
        dataset_train, self.args.nb_classes = build_dataset(is_train=True, args=self.args)
        num_tasks = get_world_size()
        global_rank = get_rank()
        sampler_train = torch.utils.data.DistributedSampler(
                dataset_train, num_replicas=num_tasks, rank=global_rank, shuffle=True)
        return DataLoader(
            dataset_train, sampler=sampler_train,
            batch_size=self.context.get_per_slot_batch_size(),
            num_workers=self.context.get_hparam("workers"),
            pin_memory=True,
            drop_last=True,
        )


    def build_validation_data_loader(self):
        dataset_val, _ = build_dataset(is_train=False, args=self.args)
        num_tasks = get_world_size()
        global_rank = get_rank()
        sampler_val = torch.utils.data.DistributedSampler(
                dataset_val, num_replicas=num_tasks, rank=global_rank, shuffle=False)
        return DataLoader(
            dataset_val, sampler=sampler_val,
            batch_size=int(1.5 * self.context.get_per_slot_batch_size()),
            num_workers=self.context.get_hparam("workers"),
            pin_memory=True,
            drop_last=False
        )