import torch
from torch import nn
import pytorch_lightning as ptl
from torch.utils.data import random_split
from determined.pytorch import DataLoader
from torch.nn import functional as F
from torchvision.datasets import MNIST
from torchvision import datasets, transforms
from include.adapter import GH, DETLightningDataModule
from typing import Optional
import os


class LightningMNISTClassifier(ptl.LightningModule):

    # TODO expect determined config.
    def __init__(self, get_hparam: GH = None):
        super().__init__()
        self.get_hparam = get_hparam

        # mnist images are (1, 28, 28) (channels, width, height) 
        self.layer_1 = torch.nn.Linear(28 * 28, 128)
        self.layer_2 = torch.nn.Linear(128, 256)
        self.layer_3 = torch.nn.Linear(256, 10)

    def forward(self, x):
        batch_size, channels, width, height = x.size()

        # (b, 1, 28, 28) -> (b, 1*28*28)
        x = x.view(batch_size, -1)

        # layer 1 (b, 1*28*28) -> (b, 128)
        x = self.layer_1(x)
        x = torch.relu(x)

        # layer 2 (b, 128) -> (b, 256)
        x = self.layer_2(x)
        x = torch.relu(x)

        # layer 3 (b, 256) -> (b, 10)
        x = self.layer_3(x)

        # probability distribution over labels
        x = torch.log_softmax(x, dim=1)

        return x

    # CHANGE: define loss fn. TODO a hyperparam?
    def _loss_fn(self, logits, labels):
        return F.nll_loss(logits, labels)

    def training_step(self, train_batch, batch_idx):
        x, y = train_batch
        logits = self.forward(x)
        loss = self._loss_fn(logits, y)
        self.log('train_loss', loss)
        return {'loss': loss}

    def validation_step(self, val_batch, batch_idx=None):
        x, y = val_batch
        logits = self.forward(x)
        loss = self._loss_fn(logits, y)
        self.log('val_loss', loss)

        pred = logits.argmax(dim=1, keepdim=True)
        accuracy = pred.eq(y.view_as(pred)).sum().item() / len(x)
        return {'val_loss': loss, 'accuracy': accuracy}

    def configure_optimizers(self):
        optimizer = torch.optim.Adam(self.parameters(),
                                     lr=self.get_hparam('learning_rate'))
        return optimizer
    
    # TODO audit other available LightningModule hooks


# CHANGE to DETLightningDataModule
class MNISTDataModule(DETLightningDataModule):
    def __init__(self):
        super().__init__()

    # rank (id of proc across all machines) and local(machine) rank. horovard. 1proc per gpu
    # def prepare_data(self):
    #     # download, split, etc...
    #     # only called on 1 GPU/TPU in distributed
    #     return super().prepare_data()

    def setup(self, stage: Optional[str] = None):
        # make assignments here (val/train/test split)
        # called on every process in DDP
        # transforms for images
        transform = transforms.Compose([transforms.ToTensor(), 
                                        transforms.Normalize((0.1307,), (0.3081,))])

        # prepare transforms standard to MNIST
        self.mnist_train = MNIST(os.getcwd(), train=True, download=True, transform=transform)
        self.mnist_val = MNIST(os.getcwd(), train=False, download=True, transform=transform)

    def train_det_dataloader(self):
        return DataLoader(self.mnist_train, batch_size=64, num_workers=12)
    def val_det_dataloader(self):
        return DataLoader(self.mnist_val, batch_size=64)
    # def test_dataloader(self):
    #     pass

if __name__ == '__main__':
    # train
    # CHANGE: define hyperparameters to be used
    def get_hparam(key: str):
        params = {
            'learning_rate': 1e-3,
        }
        return params[key]

    # CHANGE: provide the hyperparameters
    model = LightningMNISTClassifier(get_hparam)
    trainer = ptl.Trainer(max_epochs=2)

    dm = MNISTDataModule()
    trainer.fit(model, datamodule=dm)
