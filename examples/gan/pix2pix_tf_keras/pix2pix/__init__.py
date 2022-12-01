"""
Implement Pix2Pix model based on: https://www.tensorflow.org/tutorials/generative/pix2pix
"""
from typing import Tuple

import tensorflow as tf

from .discriminator import (
    make_discriminator_model,
    loss as discriminator_loss,
    make_optimizer as make_discriminator_optimizer,
)
from .generator import (
    make_generator_model,
    loss as generator_loss,
    make_optimizer as make_generator_optimizer,
)


class Pix2Pix(tf.keras.Model):
    def __init__(self):
        super(Pix2Pix, self).__init__()

        self.generator = make_generator_model()
        self.generator_loss = generator_loss

        self.discriminator = make_discriminator_model()
        self.discriminator_loss = discriminator_loss

    def compile(self, discriminator_optimizer=None, generator_optimizer=None):
        super(Pix2Pix, self).compile()
        self.discriminator_optimizer = discriminator_optimizer or make_discriminator_optimizer()
        self.generator_optimizer = generator_optimizer or make_generator_optimizer()

    def call(self, input_images, training=False):
        gen_outputs = self.generator(input_images, training=training)
        return gen_outputs

    def train_step(self, batch: Tuple[tf.Tensor, tf.Tensor], verbose=False):
        input_images, real_images = batch
        if verbose:
            print(f"Shape of input_images in train_step is {input_images.shape}")
            print(f"Shape of real_images in train_step is  {real_images.shape}")
        with tf.GradientTape() as gen_tape, tf.GradientTape() as disc_tape:
            gen_outputs = self.generator(input_images, training=True)

            disc_reals = self.discriminator([input_images, real_images], training=True)
            disc_fakes = self.discriminator([input_images, gen_outputs], training=True)

            g_loss, g_gan_loss, g_l1_loss = self.generator_loss(
                disc_fakes, gen_outputs, real_images
            )
            d_loss = self.discriminator_loss(disc_reals, disc_fakes)

        generator_gradients = gen_tape.gradient(g_loss, self.generator.trainable_variables)
        discriminator_gradients = disc_tape.gradient(d_loss, self.discriminator.trainable_variables)

        self.generator_optimizer.apply_gradients(
            zip(generator_gradients, self.generator.trainable_variables)
        )
        self.discriminator_optimizer.apply_gradients(
            zip(discriminator_gradients, self.discriminator.trainable_variables)
        )
        return {
            "g_gan_loss": g_gan_loss,
            "g_l1_loss": g_l1_loss,
            "g_loss": g_loss,
            "d_loss": d_loss,
            "total_loss": g_loss + d_loss,
        }

    def test_step(self, batch: Tuple[tf.Tensor, tf.Tensor], verbose=False):
        input_images, real_images = batch
        if verbose:
            print(f"Shape of input_images in test_step is {input_images.shape}")
            print(f"Shape of  real_images in test_step is {real_images.shape}")
        gen_outputs = self.generator(input_images, training=False)
        disc_reals = self.discriminator([input_images, real_images], training=False)
        disc_fakes = self.discriminator([gen_outputs, real_images], training=False)
        g_loss, g_gan_loss, g_l1_loss = self.generator_loss(disc_fakes, gen_outputs, real_images)
        d_loss = self.discriminator_loss(disc_reals, disc_fakes)
        return {
            "g_gan_loss": g_gan_loss,
            "g_l1_loss": g_l1_loss,
            "g_loss": g_loss,
            "d_loss": d_loss,
            "total_loss": g_loss + d_loss,
        }
