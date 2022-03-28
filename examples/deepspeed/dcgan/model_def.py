from attrdict import AttrDict

import torch
import torch.nn as nn
import torch.utils.data
import torchvision
from typing import Any, Dict, Iterator, Optional, cast

import data
from gan_model import Generator, Discriminator, weights_init
from determined.pytorch import DataLoader, TorchData
from determined.pytorch.deepspeed import (
    DeepSpeedTrial,
    DeepSpeedTrialContext,
    overwrite_deepspeed_config,
)
from determined.tensorboard.metric_writers.pytorch import TorchWriter

import deepspeed

REAL_LABEL = 1
FAKE_LABEL = 0


class DCGANTrial(DeepSpeedTrial):
    def __init__(self, context: DeepSpeedTrialContext) -> None:
        self.context = context
        self.hparams = AttrDict(self.context.get_hparams())
        self.data_config = AttrDict(self.context.get_data_config())
        self.logger = TorchWriter()
        num_channels = data.CHANNELS_BY_DATASET[self.data_config.dataset]
        gen_net = Generator(
            self.hparams.generator_width_base, num_channels, self.hparams.noise_length
        )
        gen_net.apply(weights_init)
        disc_net = Discriminator(self.hparams.discriminator_width_base, num_channels)
        disc_net.apply(weights_init)
        gen_parameters = filter(lambda p: p.requires_grad, gen_net.parameters())
        disc_parameters = filter(lambda p: p.requires_grad, disc_net.parameters())
        ds_config = overwrite_deepspeed_config(
            self.hparams.deepspeed_config, self.hparams.get("overwrite_deepspeed_args", {})
        )
        generator, _, _, _ = deepspeed.initialize(
            model=gen_net, model_parameters=gen_parameters, config=ds_config
        )
        discriminator, _, _, _ = deepspeed.initialize(
            model=disc_net, model_parameters=disc_parameters, config=ds_config
        )

        self.generator = self.context.wrap_model_engine(generator)
        self.discriminator = self.context.wrap_model_engine(discriminator)
        self.fixed_noise = self.context.to_device(
            torch.randn(
                self.context.train_micro_batch_size_per_gpu, self.hparams.noise_length, 1, 1
            )
        )
        self.criterion = nn.BCELoss()
        # TODO: Test fp16
        self.fp16 = generator.fp16_enabled()
        if self.generator.gradient_accumulation_steps() > 1:
            # The intermixed activation pattern requires zeroing gradients midway through batch, so
            # we can't support gradient accumulation.
            # One solution would be to disable automatic accumulation and run the generator
            # separately for discriminator and generator training.
            raise Exception("Gradient accumulation steps > 1 not supported.")

    def train_batch(
        self, iter_dataloader: Optional[Iterator[TorchData]], epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        assert iter_dataloader is not None
        real, _ = self.context.to_device(next(iter_dataloader))
        real = cast(torch.Tensor, real)
        if self.fp16:
            real = real.half()
        noise = self.context.to_device(
            torch.randn(
                self.context.train_micro_batch_size_per_gpu, self.hparams.noise_length, 1, 1
            )
        )
        ############################
        # (1) Update D network: maximize log(D(x)) + log(1 - D(G(z)))
        ###########################
        self.discriminator.zero_grad()
        # train with real
        batch_size = self.context.train_micro_batch_size_per_gpu
        label = cast(
            torch.Tensor,
            self.context.to_device(torch.full((batch_size,), REAL_LABEL, dtype=real.dtype)),
        )
        output = self.discriminator(real)
        errD_real = self.criterion(output, label)
        self.discriminator.backward(errD_real)
        D_x = output.mean().item()

        # train with fake
        noise = self.context.to_device(torch.randn(batch_size, self.hparams.noise_length, 1, 1))
        fake = self.generator(noise)
        label.fill_(FAKE_LABEL)
        output = self.discriminator(fake.detach())
        errD_fake = self.criterion(output, label)
        self.discriminator.backward(errD_fake)
        self.discriminator.step()
        D_G_z1 = output.mean().item()
        errD = errD_real + errD_fake

        ############################
        # (2) Update G network: maximize log(D(G(z)))
        ###########################
        self.generator.zero_grad()
        label.fill_(REAL_LABEL)  # fake labels are real for generator cost
        output = self.discriminator(fake)
        errG = self.criterion(output, label)
        self.generator.backward(errG)
        self.generator.step()
        D_G_z2 = output.mean().item()

        if batch_idx % 100 == 0:
            fake = self.generator(self.fixed_noise)
            self.logger.writer.add_image(
                "real_images", torchvision.utils.make_grid(real), batch_idx
            )
            self.logger.writer.add_image(
                "fake_images", torchvision.utils.make_grid(fake), batch_idx
            )

        return {"errD": errD, "errG": errG, "D_x": D_x, "D_G_z1": D_G_z1, "D_G_z2": D_G_z2}

    def evaluate_batch(
        self, iter_dataloader: Optional[Iterator[TorchData]], batch_idx: int
    ) -> Dict[str, Any]:
        # TODO: We could add an evaluation metric like FID here.
        assert iter_dataloader is not None
        next(iter_dataloader)
        return {"no_validation_metric": 0.0}

    def build_training_data_loader(self) -> Any:
        dataset = data.get_dataset(self.data_config)
        return DataLoader(
            dataset,
            batch_size=self.context.train_micro_batch_size_per_gpu,
            shuffle=True,
            num_workers=int(self.hparams.data_workers),
        )

    def build_validation_data_loader(self) -> Any:
        dataset = data.get_dataset(self.data_config)
        # Since we're not doing validation, limit to single batch.
        dataset = torch.utils.data.Subset(
            dataset,
            list(
                range(
                    self.context.train_micro_batch_size_per_gpu
                    * self.context.distributed.get_size()
                )
            ),
        )
        return DataLoader(dataset, batch_size=self.context.train_micro_batch_size_per_gpu)
