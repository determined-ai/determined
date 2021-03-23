import torch
import pytorch_lightning as pl
from torch.nn import functional as F
from data import MNISTDataModule
from typing import Dict, Any


class LightningMNISTClassifier(pl.LightningModule):

    def __init__(self, *args, lr: float, **kwargs):
        super().__init__(*args, **kwargs)

        self.save_hyperparameters()
        # mnist images are (1, 28, 28) (channels, width, height)
        self.layer_1 = torch.nn.Linear(28 * 28, 128)
        self.layer_2 = torch.nn.Linear(128, 256)
        self.layer_3 = torch.nn.Linear(256, 10)
        # self.dims is returned when you call dm.size()
        # Setting default dims here because we know them.
        # Could optionally be assigned dynamically in dm.setup()
        self.dims = (1, 28, 28)


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

    def training_step(self, batch, *args, **kwargs):
        x, y = batch
        logits = self.forward(x)
        loss = self._loss_fn(logits, y)
        self.log('train_loss', loss)
        return {'loss': loss}

    def validation_step(self, batch, batch_idx, *args, **kwargs):
        x, y = batch
        logits = self.forward(x)
        loss = self._loss_fn(logits, y)
        self.log('val_loss', loss)

        pred = logits.argmax(dim=1, keepdim=True)
        accuracy = pred.eq(y.view_as(pred)).sum().item() / len(x)
        return {'val_loss': loss, 'accuracy': accuracy}

    def configure_optimizers(self):
        optimizer = torch.optim.Adam(self.parameters(),
                                     lr=self.hparams.lr)
        return optimizer

    def on_load_checkpoint(self, checkpoint: Dict[str, Any]) -> None:
        assert checkpoint['test'] == True

    def on_save_checkpoint(self, checkpoint: Dict[str, Any]) -> None:
        checkpoint['test'] = True


if __name__ == '__main__':
    model = LightningMNISTClassifier(lr=1e-3)
    trainer = pl.Trainer(max_epochs=2, default_root_dir='/tmp/lightning')

    dm = MNISTDataModule('https://s3-us-west-2.amazonaws.com/determined-ai-test-data/pytorch_mnist.tar.gz')
    trainer.fit(model, datamodule=dm)
