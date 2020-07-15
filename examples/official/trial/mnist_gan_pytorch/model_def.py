"""
This example shows how to interact with the new Determined PyTorch interface with multiple models,
optimizers, and LR schedulers support to build a GAN network.

The functions of the new interface starts with underscores and we will remove these underscores
when the new interface is public.

In the new interface, you need to instantiate your models, optimizers, and LR schedulers in
__init__ and run forward and backward passes and step the optimizer in train_batch. By doing so,
you could be flexible in building your own training workflows.
"""

from typing import Any, Dict, Union, Sequence

import numpy as np
import torch
import torch.nn as nn
import torch.nn.functional as F
import torchvision
from torch.optim.lr_scheduler import LambdaLR

from determined.pytorch import PyTorchTrial, PyTorchTrialContext, DataLoader, LRScheduler
from determined.tensorboard.metric_writers.pytorch import TorchWriter

import data

TorchData = Union[Dict[str, torch.Tensor], Sequence[torch.Tensor], torch.Tensor]


class Generator(nn.Module):
    def __init__(self, latent_dim, img_shape):
        super().__init__()
        self.img_shape = img_shape

        def block(in_feat, out_feat, normalize=True):
            layers = [nn.Linear(in_feat, out_feat)]
            if normalize:
                layers.append(nn.BatchNorm1d(out_feat, 0.8))
            layers.append(nn.LeakyReLU(0.2, inplace=True))
            return layers

        self.model = nn.Sequential(
            *block(latent_dim, 128, normalize=False),
            *block(128, 256),
            *block(256, 512),
            *block(512, 1024),
            nn.Linear(1024, int(np.prod(img_shape))),
            nn.Tanh()
        )

    def forward(self, z):
        img = self.model(z)
        img = img.view(img.size(0), *self.img_shape)
        return img


class Discriminator(nn.Module):
    def __init__(self, img_shape):
        super().__init__()

        self.model = nn.Sequential(
            nn.Linear(int(np.prod(img_shape)), 512),
            nn.LeakyReLU(0.2, inplace=True),
            nn.Linear(512, 256),
            nn.LeakyReLU(0.2, inplace=True),
            nn.Linear(256, 1),
            nn.Sigmoid(),
        )

    def forward(self, img):
        img_flat = img.view(img.size(0), -1)
        validity = self.model(img_flat)

        return validity


class GANTrial(PyTorchTrial):
    def __init__(self, trial_context: PyTorchTrialContext) -> None:
        self.context = trial_context
        self.logger = TorchWriter()

        # Create a unique download directory for each rank so they don't overwrite each other.
        self.download_directory = f"/tmp/data-rank{self.context.distributed.get_rank()}"
        self.data_downloaded = False

        # Initialize the models.
        mnist_shape = (1, 28, 28)
        self.generator = self.context.Model(Generator(latent_dim=self.context.get_hparam("latent_dim"), img_shape=mnist_shape))
        self.discriminator = self.context.Model(Discriminator(img_shape=mnist_shape))

        # Initialize the optimizers and learning rate scheduler.
        lr = self.context.get_hparam("lr")
        b1 = self.context.get_hparam("b1")
        b2 = self.context.get_hparam("b2")
        self.opt_g = self.context.Optimizer(torch.optim.Adam(self.generator.parameters(), lr=lr, betas=(b1, b2)))
        self.opt_d = self.context.Optimizer(torch.optim.Adam(self.discriminator.parameters(), lr=lr, betas=(b1, b2)))
        self.lr_g = self.context.LRScheduler(
            lr_scheduler=LambdaLR(self.opt_g, lr_lambda=lambda epoch: 0.95 ** epoch),
            step_mode=LRScheduler.StepMode.STEP_EVERY_EPOCH,
        )

    def build_training_data_loader(self) -> DataLoader:
        if not self.data_downloaded:
            self.download_directory = data.download_dataset(
                download_directory=self.download_directory,
                data_config=self.context.get_data_config(),
            )
            self.data_downloaded = True

        train_data = data.get_dataset(self.download_directory, train=True)
        return DataLoader(train_data, batch_size=self.context.get_per_slot_batch_size())

    def build_validation_data_loader(self) -> DataLoader:
        if not self.data_downloaded:
            self.download_directory = data.download_dataset(
                download_directory=self.download_directory,
                data_config=self.context.get_data_config(),
            )
            self.data_downloaded = True

        validation_data = data.get_dataset(self.download_directory, train=False)
        return DataLoader(validation_data, batch_size=self.context.get_per_slot_batch_size())

    def train_batch(
        self, batch: TorchData, model: nn.Module, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        imgs, _ = batch

        # Train generator.
        # Set the requires_grad to only update parameters on the generator.
        self.generator.requires_grad_(True)
        self.discriminator.requires_grad_(False)

        # Sample noise and generator images.
        # Note that you need to map the generated data to the device specified by Determined.
        z = torch.randn(imgs.shape[0], self.context.get_hparam("latent_dim"))
        z = self.context.to_device(z)
        generated_imgs = self.generator(z)

        # Log sampled images to Tensorboard.
        sample_imgs = generated_imgs[:6]
        grid = torchvision.utils.make_grid(sample_imgs)
        self.logger.writer.add_image(f'generated_images_epoch_{epoch_idx}', grid, batch_idx)

        # Calculate generator loss.
        valid = torch.ones(imgs.size(0), 1)
        valid = self.context.to_device(valid)
        g_loss = F.binary_cross_entropy(self.discriminator(generated_imgs), valid)

        # Run backward pass and step the optimizer for the generator.
        self.context.backward(g_loss)
        self.context.step_optimizer(self.opt_g)


        # Train discriminator
        # Set the requires_grad to only update parameters on the discriminator.
        self.generator.requires_grad_(False)
        self.discriminator.requires_grad_(True)

        # Calculate discriminator loss with a batch of real images and a batch of fake images.
        valid = torch.ones(imgs.size(0), 1)
        valid = self.context.to_device(valid)
        real_loss = F.binary_cross_entropy(self.discriminator(imgs), valid)
        fake = torch.zeros(generated_imgs.size(0), 1)
        fake = self.context.to_device(fake)
        fake_loss = F.binary_cross_entropy(self.discriminator(generated_imgs.detach()), fake)
        d_loss = (real_loss + fake_loss) / 2

        # Run backward pass and step the optimizer for the generator.
        self.context.backward(d_loss)
        self.context.step_optimizer(self.opt_d)

        return {
            'loss': d_loss,
            'g_loss': g_loss,
            'd_loss': d_loss,
        }

    def evaluate_batch(self, batch: TorchData, model: nn.Module) -> Dict[str, Any]:
        imgs, _ = batch
        valid = torch.ones(imgs.size(0), 1)
        valid = self.context.to_device(valid)
        loss = F.binary_cross_entropy(self.discriminator(imgs), valid)
        return {"loss": loss}
