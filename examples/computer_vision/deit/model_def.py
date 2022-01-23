import torch.nn as nn
from determined.pytorch import DataLoader, PyTorchTrial, PyTorchTrialContext, LRScheduler

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
from timm.utils import NativeScaler, get_state_dict, ModelEma

from timm.data.constants import IMAGENET_DEFAULT_MEAN, IMAGENET_DEFAULT_STD

from attrdict import AttrDict

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
class DeitTrial(PyTorchTrial):
    def __init__(self, context: PyTorchTrialContext):

        # Initialize the trial class and wrap the models, optimizers, and LR schedulers.
         # Store trial context for later use.
        self.context = context

        # Create a unique download directory for each rank so they don't overwrite each
        # other when doing distributed training.
        self.download_directory = f"/tmp/data-rank{self.context.distributed.get_rank()}"
        self.data_downloaded = False

        self.args = AttrDict(self.context.get_hparams())

        self.model = self.context.wrap_model(torch.hub.load('facebookresearch/deit:main', 'deit_base_patch16_224', pretrained=False))
        print('args: ', self.args)
        self.optimizer = self.context.wrap_optimizer(create_optimizer(self.args, self.model))
        
        self.lr_scheduler, self.num_epochs = create_scheduler(self.args, self.optimizer)
        self.lr_scheduler = self.context.wrap_lr_scheduler(self.lr_scheduler, LRScheduler.StepMode.MANUAL_STEP)

        self.transform = T.Compose([
            T.Resize(256, interpolation=3),
            T.CenterCrop(224),
            T.ToTensor(),
            T.Normalize(IMAGENET_DEFAULT_MEAN, IMAGENET_DEFAULT_STD),
        ])
    
        
    def train_batch(self, batch: TorchData, epoch_idx: int, batch_idx: int):
        # Run forward passes on the models and backward passes on the optimizers.
        batch = cast(Tuple[torch.Tensor, torch.Tensor], batch)
        print('batch: ', batch)
        x_data, y_data, mean = batch

        # Define the training forward pass and calculate loss.
        output = self.model(x_data)
        loss = torch.nn.functional.nll_loss(output, y_data)

        # Define the training backward pass and step the optimizer.
        self.context.backward(loss)
        self.context.step_optimizer(self.optimizer)

        return {"loss": loss}

    def evaluate_batch(self, batch: TorchData):
        # Define how to evaluate the model by calculating loss and other metrics
        # for a batch of validation data.
        batch = cast(Tuple[torch.Tensor, torch.Tensor], batch)
        data, labels = batch

        output = self.model(data)
        validation_loss = torch.nn.functional.nll_loss(output, labels).item()

        pred = output.argmax(dim=1, keepdim=True)
        accuracy = pred.eq(labels.view_as(pred)).sum().item() / len(data)

        return {"validation_loss": validation_loss, "accuracy": accuracy}

    def build_training_data_loader(self):
        # Create the training data loader.
        # This should return a determined.pytorch.Dataset.
        # url = 'http://images.cocodataset.org/val2017/000000039769.jpg'
        # im = Image.open(requests.get(url, stream=True).raw)
        # img = self.transform(im).unsqueeze(0)
        data = load_databatch('data/Imagenet8_train', 1, 8)
        return DataLoader([data], batch_size=1)

    def build_validation_data_loader(self):
        # Create the validation data loader.
        # This should return a determined.pytorch.Dataset.
        data = load_databatch('data/Imagenet8_train', 1, 8)
        return DataLoader([data], batch_size=1)