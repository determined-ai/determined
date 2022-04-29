"""
Implement Pix2Pix model based on: https://www.tensorflow.org/tutorials/generative/pix2pix
"""
import tensorflow as tf

from .discriminator import (
    make_discriminator_model,
    loss as discriminator_loss,
    optimizer as discriminator_optimizer,
)
from .generator import (
    make_generator_model,
    loss as generator_loss,
    optimizer as generator_optimizer,
)
from .sampling import downsample, upsample


generator = make_generator_model()
discriminator = make_discriminator_model()
