import pathlib
from typing import Tuple

import filelock
import tensorflow as tf


def download(base, dataset) -> str:
    filename = f"{dataset}.tar.gz"
    url = f"{base}/{filename}"
    with filelock.FileLock(f"{filename}.lock"):
        path_to_zip = tf.keras.utils.get_file(filename, origin=url, extract=True)
        path_to_zip = pathlib.Path(path_to_zip)

        PATH = path_to_zip.parent / dataset
        return PATH


def _load(image_file: str) -> Tuple[tf.Tensor, tf.Tensor]:
    # Read and decode an image file to a uint8 tensor
    image = tf.io.read_file(image_file)
    image = tf.io.decode_jpeg(image)

    # Split each image tensor into two tensors:
    # - one with a real building facade image
    # - one with an architecture label image
    w = tf.shape(image)[1]
    w = w // 2
    input_image = image[:, w:, :]
    real_image = image[:, :w, :]

    # Convert both images to float32 tensors
    input_image = tf.cast(input_image, tf.float32)
    real_image = tf.cast(real_image, tf.float32)

    return input_image, real_image


def _resize(input_image, real_image, height, width) -> Tuple[tf.Tensor, tf.Tensor]:
    input_image = tf.image.resize(
        input_image, [height, width], method=tf.image.ResizeMethod.NEAREST_NEIGHBOR
    )
    real_image = tf.image.resize(
        real_image, [height, width], method=tf.image.ResizeMethod.NEAREST_NEIGHBOR
    )

    return input_image, real_image


def _random_crop(input_image, real_image, height, width) -> Tuple[tf.Tensor, tf.Tensor]:
    stacked_image = tf.stack([input_image, real_image], axis=0)
    cropped_image = tf.image.random_crop(stacked_image, size=[2, height, width, 3])

    return cropped_image[0], cropped_image[1]


# Normalizing the images to [-1, 1]
def _normalize(input_image, real_image) -> Tuple[tf.Tensor, tf.Tensor]:
    input_image = (input_image / 127.5) - 1
    real_image = (real_image / 127.5) - 1

    return input_image, real_image


@tf.function()
def _random_jitter(
    input_image,
    real_image,
    height,
    width,
    jitter=0,
    mirror=False,
) -> Tuple[tf.Tensor, tf.Tensor]:
    if jitter > 0:
        # Resizing to 286x286
        input_image, real_image = _resize(input_image, real_image, height + jitter, width + jitter)

        # Random cropping back to 256x256
        input_image, real_image = _random_crop(input_image, real_image, height, width)
    else:
        input_image, real_image = _resize(input_image, real_image, height, width)
    if mirror and (tf.random.uniform(()) > 0.5):
        # Random mirroring
        input_image = tf.image.flip_left_right(input_image)
        real_image = tf.image.flip_left_right(real_image)

    return input_image, real_image


def _preprocess_images(
    image_filename,
    height,
    width,
    jitter=0,
    mirror=False,
) -> Tuple[tf.Tensor, tf.Tensor]:
    input_image, real_image = _load(image_filename)
    input_image, real_image = _random_jitter(
        input_image,
        real_image,
        height,
        width,
        jitter,
        mirror,
    )
    input_image, real_image = _normalize(input_image, real_image)

    return input_image, real_image


def load_dataset(path, height, width, set_="train", jitter=0, mirror=False):
    """Load the images into memory and preprocess them."""
    ds = tf.data.Dataset.list_files(str(path / f"{set_}/*.jpg"))
    if set_ != "train":
        jitter = 0
        mirror = False

    def _prep(i):
        return _preprocess_images(i, height, width, jitter, mirror)

    ds = ds.map(_prep, num_parallel_calls=tf.data.experimental.AUTOTUNE)
    return ds


def main():
    import matplotlib.pyplot as plt
    import yaml

    config = yaml.load(open("const.yaml", "r"), Loader=yaml.BaseLoader)
    path, dataset_name = config["data"]["base"], config["data"]["dataset"]
    path = download(path, dataset_name)

    inp, re = _load(str(path / "train/100.jpg"))
    plt.figure(figsize=(6, 6))
    for i in range(4):
        rj_inp, _ = _random_jitter(inp, re, 256, 256, 30, True)
        plt.subplot(2, 2, i + 1)
        plt.imshow(rj_inp / 255.0)
        plt.axis("off")
    plt.show()


if __name__ == "__main__":
    main()
