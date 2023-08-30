import logging
from typing import Any, Dict, Iterator, Optional, Tuple, Union, cast

import data
import deepspeed
import torch
import torch.nn as nn
import torch.utils.data
import torchvision
from attrdict import AttrDict
from gan_model import Discriminator, Generator, weights_init

from determined.pytorch import DataLoader, TorchData
from determined.pytorch.deepspeed import (
    DeepSpeedTrial,
    DeepSpeedTrialContext,
    overwrite_deepspeed_config,
)

REAL_LABEL = 1
FAKE_LABEL = 0


class DCGANTrial(DeepSpeedTrial):
    def __init__(self, context: DeepSpeedTrialContext) -> None:
        self.context = context
        self.hparams = AttrDict(self.context.get_hparams())
        self.data_config = AttrDict(self.context.get_data_config())
        self.logger = self.context.get_tensorboard_writer()
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
        self.gradient_accumulation_steps = generator.gradient_accumulation_steps()
        # Manually perform gradient accumulation.
        if self.gradient_accumulation_steps > 1:
            logging.info("Disabling automatic gradient accumulation.")
            self.context.disable_auto_grad_accumulation()

    def _get_noise(self, dtype: torch.dtype) -> torch.Tensor:
        return cast(
            torch.Tensor,
            self.context.to_device(
                torch.randn(
                    self.context.train_micro_batch_size_per_gpu,
                    self.hparams.noise_length,
                    1,
                    1,
                    dtype=dtype,
                )
            ),
        )

    def _get_label_constants(
        self, batch_size: int, dtype: torch.dtype
    ) -> Tuple[torch.Tensor, torch.Tensor]:
        real_label = cast(
            torch.Tensor,
            self.context.to_device(torch.full((batch_size,), REAL_LABEL, dtype=dtype)),
        )
        fake_label = cast(
            torch.Tensor,
            self.context.to_device(torch.full((batch_size,), FAKE_LABEL, dtype=dtype)),
        )
        return real_label, fake_label

    def train_batch(
        self, iter_dataloader: Optional[Iterator[TorchData]], epoch_idx: int, batch_idx: int
    ) -> Union[torch.Tensor, Dict[str, Any]]:
        assert iter_dataloader is not None
        if self.fp16:
            dtype = torch.float16
        else:
            dtype = torch.float32
        real_label, fake_label = self._get_label_constants(
            self.context.train_micro_batch_size_per_gpu, dtype
        )
        ############################
        # (1) Update D network: maximize log(D(x)) + log(1 - D(G(z)))
        ###########################
        self.discriminator.zero_grad()

        real_sample_count = 0
        errD_real_sum = 0.0
        errD_fake_sum = 0.0
        D_x = 0.0
        D_G_z1 = 0.0
        fake_sample_count = (
            self.context.train_micro_batch_size_per_gpu * self.gradient_accumulation_steps
        )

        for i in range(self.gradient_accumulation_steps):
            # Note: at end of epoch, may receive a batch of size smaller than train_micro_batch_size_per_gpu.
            # In that case, we end up training on more fake examples than real examples.
            # train with real
            real, _ = self.context.to_device(next(iter_dataloader))
            real = cast(torch.Tensor, real)
            actual_batch_size = real.shape[0]
            real_sample_count += actual_batch_size
            if self.fp16:
                real = real.half()
            output = self.discriminator(real)
            # For edge-case small batches, must cut real_label size to match.
            errD_real = self.criterion(output, real_label[:actual_batch_size])
            self.discriminator.backward(errD_real)
            # Undo averaging so we can re-average at end when reporting metrics.
            errD_real_sum += errD_real * actual_batch_size
            D_x += output.sum().item()
            # train with fake
            noise = self._get_noise(dtype)
            fake = self.generator(noise)
            output = self.discriminator(fake.detach())
            errD_fake = self.criterion(output, fake_label)
            self.discriminator.backward(errD_fake)
            errD_fake_sum += errD_fake * self.context.train_micro_batch_size_per_gpu
            D_G_z1 += output.sum().item()
            # update
            self.discriminator.step()
        D_x /= real_sample_count
        D_G_z1 /= fake_sample_count
        errD = (errD_real_sum / real_sample_count) + (errD_fake_sum / fake_sample_count)
        ############################
        # (2) Update G network: maximize log(D(G(z)))
        ###########################
        self.generator.zero_grad()
        D_G_z2_sum = 0.0
        errG_sum = 0.0
        for i in range(self.gradient_accumulation_steps):
            if i > 0:
                # Must repeat forward pass of generator for accumulation steps beyond the first.
                noise = self._get_noise(dtype)
                fake = self.generator(noise)
            output = self.discriminator(fake)
            errG = self.criterion(output, real_label)  # fake labels are real for generator cost
            self.generator.backward(errG)
            errG_sum += errG * self.context._train_micro_batch_size_per_gpu
            D_G_z2_sum += output.sum().item()
            self.generator.step()

        if batch_idx % 100 == 0:
            fake = self.generator(self.fixed_noise)
            denormalized_real = (real + 1) / 2
            denormalized_fake = (fake + 1) / 2
            self.logger.add_image(
                "real_images", torchvision.utils.make_grid(denormalized_real), batch_idx
            )
            self.logger.add_image(
                "fake_images", torchvision.utils.make_grid(denormalized_fake), batch_idx
            )

        return {
            "errD": errD,
            "errG": errG_sum / fake_sample_count,
            "D_x": D_x,
            "D_G_z1": D_G_z1,
            "D_G_z2": D_G_z2_sum / fake_sample_count,
        }

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
