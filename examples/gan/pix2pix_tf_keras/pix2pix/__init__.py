"""
Implement Pix2Pix model based on: https://www.tensorflow.org/tutorials/generative/pix2pix
"""
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
        self.discriminator_optimizer = (
            discriminator_optimizer or make_discriminator_optimizer()
        )
        self.generator_optimizer = generator_optimizer or make_generator_optimizer()

    def call(self, inputs, training=None, mask=None):
        pass

    def train_step(self, data):
        input_image, real_image = data
        with tf.GradientTape() as gen_tape, tf.GradientTape() as disc_tape:
            gen_output = self.generator(input_image, training=True)

            disc_real = self.discriminator([input_image, real_image], training=True)
            disc_fake = self.discriminator([input_image, gen_output], training=True)

            g_loss, g_gan_loss, g_l1_loss = self.generator_loss(disc_fake, gen_output, real_image)
            d_loss = self.discriminator_loss(disc_real, disc_fake)

        generator_gradients = gen_tape.gradient(
            g_loss, self.generator.trainable_variables
        )
        discriminator_gradients = disc_tape.gradient(
            d_loss, self.discriminator.trainable_variables
        )

        self.generator_optimizer.apply_gradients(
            zip(generator_gradients, self.generator.trainable_variables)
        )
        self.discriminator_optimizer.apply_gradients(
            zip(discriminator_gradients, self.discriminator.trainable_variables)
        )
        return {"g_gan_loss": g_gan_loss, "g_l1_loss": g_l1_loss, "g_loss": g_loss, "d_loss": d_loss, "total_loss": g_loss + d_loss}

    def test_step(self, data):
        input_image, target = data
        gen_output = self.generator(input_image, training=False)
        disc_real = self.discriminator([input_image, target], training=False)
        disc_fake = self.discriminator([gen_output, target], training=False)
        g_loss, g_gan_loss, g_l1_loss = self.generator_loss(disc_fake, gen_output, target)
        d_loss = self.discriminator_loss(disc_real, disc_fake)
        return {"g_gan_loss": g_gan_loss, "g_l1_loss": g_l1_loss, "g_loss": g_loss, "d_loss": d_loss, "total_loss": g_loss + d_loss}
