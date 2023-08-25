import tensorflow as tf
from pix2pix import make_discriminator_model, make_generator_model


def main():
    generator = make_generator_model()
    tf.keras.utils.plot_model(generator, show_shapes=True, dpi=64, to_file="generator.png")
    discriminator = make_discriminator_model()
    tf.keras.utils.plot_model(discriminator, show_shapes=True, dpi=64, to_file="discriminator.png")


if __name__ == "__main__":
    main()
