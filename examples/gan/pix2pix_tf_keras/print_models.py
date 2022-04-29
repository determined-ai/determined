import matplotlib.pyplot as plt

import tensorflow as tf

from data import _load, _PATH
from pix2pix import (
    downsample,
    upsample,
    generator,
    discriminator,
)


def main():
    inp, _ = _load(str(_PATH / "train/100.jpg"))

    down_model = downsample(3, 4)
    down_result = down_model(tf.expand_dims(inp, 0))
    print(down_result.shape)

    up_model = upsample(3, 4)
    up_result = up_model(down_result)
    print(up_result.shape)

    tf.keras.utils.plot_model(
        generator, show_shapes=True, dpi=64, to_file="generator.png"
    )

    gen_output = generator(inp[tf.newaxis, ...], training=False)
    plt.imshow(gen_output[0, ...])
    plt.show()

    tf.keras.utils.plot_model(
        discriminator, show_shapes=True, dpi=64, to_file="discriminator.png"
    )

    disc_out = discriminator([inp[tf.newaxis, ...], gen_output], training=False)
    plt.imshow(disc_out[0, ..., -1], vmin=-20, vmax=20, cmap="RdBu_r")
    plt.colorbar()
    plt.show()

    from plotting import generate_images

    from data import test_dataset

    for example_input, example_target in test_dataset.take(1):
        generate_images(generator, example_input, example_target)


if __name__ == "__main__":
    main()
