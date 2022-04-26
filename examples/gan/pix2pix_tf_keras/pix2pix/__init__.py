"""
Implement Pix2Pix model based on: https://www.tensorflow.org/tutorials/generative/pix2pix
"""
import os.path

import tensorflow as tf

from .discriminator import (
    Discriminator,
    loss as discriminator_loss,
    optimizer as discriminator_optimizer,
)
from .generator import (
    Generator,
    loss as generator_loss,
    optimizer as generator_optimizer,
)
from .sampling import downsample, upsample


generator = Generator()
discriminator = Discriminator()


checkpoint_dir = "./training_checkpoints"
checkpoint_prefix = os.path.join(checkpoint_dir, "ckpt")
checkpoint = tf.train.Checkpoint(
    generator_optimizer=generator_optimizer,
    discriminator_optimizer=discriminator_optimizer,
    generator=generator,
    discriminator=discriminator,
)
