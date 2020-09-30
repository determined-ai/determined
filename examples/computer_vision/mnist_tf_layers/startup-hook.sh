#!/bin/sh

# Download the dataset before starting training.
python -c "import tensorflow_datasets as tfds; tfds.image.MNIST().download_and_prepare()"
