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


generator = make_generator_model()
discriminator = make_discriminator_model()


@tf.function
def train_step(input_image, target):
    with tf.GradientTape() as gen_tape, tf.GradientTape() as disc_tape:
        gen_output = generator(input_image, training=True)

        disc_real = discriminator([input_image, target], training=True)
        disc_fake = discriminator([input_image, gen_output], training=True)

        g_loss, _, _ = generator_loss(disc_fake, gen_output, target)
        d_loss = discriminator_loss(disc_real, disc_fake)

    generator_gradients = gen_tape.gradient(g_loss, generator.trainable_variables)
    discriminator_gradients = disc_tape.gradient(
        d_loss, discriminator.trainable_variables
    )

    generator_optimizer.apply_gradients(
        zip(generator_gradients, generator.trainable_variables)
    )
    discriminator_optimizer.apply_gradients(
        zip(discriminator_gradients, discriminator.trainable_variables)
    )

    return {"d_loss": d_loss, "g_loss": g_loss}
