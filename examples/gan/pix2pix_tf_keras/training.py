import time

import tensorflow as tf

from data import test_dataset, train_dataset

from pix2pix import (
    discriminator,
    discriminator_loss,
    discriminator_optimizer,
    generator,
    generator_loss,
    generator_optimizer,
)
from plotting import generate_images


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


def fit(train_ds, test_ds, steps, preview=0):
    example_input, example_target = next(iter(test_ds.take(1)))
    start = time.time()

    for step, (input_image, target) in train_ds.repeat().take(steps).enumerate():
        if preview and ((step) % preview == 0):
            # display.clear_output(wait=True)

            if step != 0:
                print(f"Time taken for {preview} steps: {time.time()-start:.2f} sec\n")

            start = time.time()

            generate_images(generator, example_input, example_target)
            print(f"Step: {step}")

        train_step(input_image, target)

        # Training step
        if (step + 1) % 10 == 0:
            print(".", end="", flush=True)


def main():
    fit(train_dataset, test_dataset, steps=200, preview=100)


if __name__ == "__main__":
    main()
