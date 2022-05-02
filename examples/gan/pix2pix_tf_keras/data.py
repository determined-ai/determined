import pathlib

from typing import Union

import tensorflow as tf

DATASET_NAME = "facades"
HEIGHT, WIDTH = 256, 256
# The facade training set consists of 400 images
BUFFER_SIZE = 400


def download(worker_rank: Union[None, int] = None):
    URL = f"http://efrosgans.eecs.berkeley.edu/pix2pix/datasets/{DATASET_NAME}.tar.gz"

    fname = (
        f"{DATASET_NAME}-{worker_rank}.tar.gz"
        if worker_rank is not None
        else f"{DATASET_NAME}.tzr.gz"
    )
    path_to_zip = tf.keras.utils.get_file(fname, origin=URL, extract=True)

    path_to_zip = pathlib.Path(path_to_zip)

    PATH = path_to_zip.parent / DATASET_NAME
    return PATH


def _load(image_file):
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


def _resize(input_image, real_image, height, width):
    input_image = tf.image.resize(
        input_image, [height, width], method=tf.image.ResizeMethod.NEAREST_NEIGHBOR
    )
    real_image = tf.image.resize(
        real_image, [height, width], method=tf.image.ResizeMethod.NEAREST_NEIGHBOR
    )

    return input_image, real_image


def _random_crop(input_image, real_image, height, width):
    stacked_image = tf.stack([input_image, real_image], axis=0)
    cropped_image = tf.image.random_crop(stacked_image, size=[2, height, width, 3])

    return cropped_image[0], cropped_image[1]


# Normalizing the images to [-1, 1]
def _normalize(input_image, real_image):
    input_image = (input_image / 127.5) - 1
    real_image = (real_image / 127.5) - 1

    return input_image, real_image


@tf.function()
def _random_jitter(input_image, real_image, height, width, jitter=30):
    if jitter > 0:
        # Resizing to 286x286
        input_image, real_image = _resize(
            input_image, real_image, height + jitter, width + jitter
        )

        # Random cropping back to 256x256
        input_image, real_image = _random_crop(input_image, real_image, height, width)

    if tf.random.uniform(()) > 0.5:
        # Random mirroring
        input_image = tf.image.flip_left_right(input_image)
        real_image = tf.image.flip_left_right(real_image)

    return input_image, real_image


def _load_train_images(image_file, height=HEIGHT, width=WIDTH, jitter=30):
    input_image, real_image = _load(image_file)
    input_image, real_image = _random_jitter(
        input_image, real_image, height, width, jitter
    )
    input_image, real_image = _normalize(input_image, real_image)

    return input_image, real_image


def _load_test_images(image_file, height=HEIGHT, width=WIDTH):
    input_image, real_image = _load(image_file)
    input_image, real_image = _resize(input_image, real_image, height, width)
    input_image, real_image = _normalize(input_image, real_image)

    return input_image, real_image


def get_dataset(path, set_="train", batch_size=0):
    ds = tf.data.Dataset.list_files(str(path / f"{set_}/*.jpg"))
    ds = ds.map(_load_train_images if set_ == "train" else _load_test_images)
    if set_ == "train":
        ds = ds.shuffle(BUFFER_SIZE)
    if batch_size:
        ds = ds.batch(batch_size)
    #    test_dataset = tf.data.Dataset.from_tensor_slices(test_dataset).shuffle(50000)
    return ds


def main():
    import matplotlib.pyplot as plt

    PATH = download()

    inp, re = _load(str(PATH / "train/100.jpg"))
    plt.figure(figsize=(6, 6))
    for i in range(4):
        rj_inp, _ = _random_jitter(inp, re)
        plt.subplot(2, 2, i + 1)
        plt.imshow(rj_inp / 255.0)
        plt.axis("off")
    plt.show()


if __name__ == "__main__":
    main()
