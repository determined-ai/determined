import torch
import pytorch_lightning as ptl
from torch.utils.data import DataLoader
from torch.nn import functional as F
from torchvision import datasets, transforms
from torchvision.datasets import MNIST
from pathlib import Path
from determined.experimental.pytorch_lightning import HyperparamsProvider
from typing import Optional
import os


# CHANGE exnted DETLightningModule
class LightningMNISTClassifier(ptl.LightningModule):

    def __init__(self, *args, get_hparam: HyperparamsProvider, **kwargs):
        super().__init__(*args, **kwargs)

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

    def _loss_fn(self, logits, labels):
        return F.nll_loss(logits, labels)

    def training_step(self, batch, batch_idx):
        x, y = batch
        logits = self.forward(x)
        loss = self._loss_fn(logits, y)
        self.log('train_loss', loss)
        return {'loss': loss}

    def validation_step(self, batch, batch_idx=None):
        x, y = batch
        logits = self.forward(x)
        loss = self._loss_fn(logits, y)
        self.log('val_loss', loss)

        pred = logits.argmax(dim=1, keepdim=True)
        accuracy = pred.eq(y.view_as(pred)).sum().item() / len(x)
        return {'val_loss': loss, 'accuracy': accuracy}

    def configure_optimizers(self):
        # CHANGE read externally provided hyperparameters
        optimizer = torch.optim.Adam(self.parameters(),
                                     lr=self.get_hparam('learning_rate'))
        return optimizer


class MNISTDataModule(ptl.LightningDataModule):
    def __init__(self):
        super().__init__()

    def setup(self, stage: Optional[str] = None):
        transform = transforms.Compose([transforms.ToTensor(), 
                                        transforms.Normalize((0.1307,), (0.3081,))])

        # prepare transforms standard to MNIST
        self.mnist_train = MNIST(str(Path('/tmp/MNIST')), train=True, download=True, transform=transform)
        self.mnist_val = MNIST(str(Path('/tmp/MNIST')), train=False, download=True, transform=transform)

    def train_dataloader(self):
        return DataLoader(self.mnist_train, batch_size=64, num_workers=12)
    def val_dataloader(self):
        return DataLoader(self.mnist_val, batch_size=64)

if __name__ == '__main__':
    # train
    # CHANGE: define hyperparameters to be used
    def get_hparam(key: str):
        params = {
            'learning_rate': 1e-3,
        }
        return params[key]

    # CHANGE: provide the hyperparameters
    model = LightningMNISTClassifier(get_hparam=get_hparam)
    trainer = ptl.Trainer(max_epochs=2)

    dm = MNISTDataModule()
    trainer.fit(model, datamodule=dm)
