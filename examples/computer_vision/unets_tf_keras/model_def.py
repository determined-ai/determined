"""
This example is to show how to use an existing Tensorflow Image Segmentation model with Determined.
The flags and configurations can be found under const.yaml. For more information
regarding the optional flags view the original script linked below.
This implementation is based on:
https://github.com/tensorflow/docs/blob/master/site/en/tutorials/images/segmentation.ipynb
"""
import tensorflow as tf
from tensorflow_examples.models.pix2pix import pix2pix
import tensorflow_datasets as tfds
import numpy as np

import urllib.request
import os

from determined.keras import TFKerasTrial
from tensorflow import keras


class UNetsTrial(TFKerasTrial):
    def __init__(self, context):
        self.context = context

        # Create a unique download directory for each rank so they don't overwrite each other.
        self.download_directory = f"/tmp/data-rank{self.context.distributed.get_rank()}"
        self.data_downloaded = False

    def normalize(self, input_image, input_mask):
        input_image = tf.cast(input_image, tf.float32) / 255.0
        input_mask -= 1
        return input_image, input_mask

    def unet_model(self, output_channels):
        inputs = tf.keras.layers.Input(shape=[128, 128, 3])
        x = inputs

        # Downsampling through the model
        skips = self.down_stack(x)
        x = skips[-1]
        skips = reversed(skips[:-1])

        # Upsampling and establishing the skip connections
        for up, skip in zip(self.up_stack, skips):
            x = up(x)
            concat = tf.keras.layers.Concatenate()
            x = concat([x, skip])

        # This is the last layer of the model
        last = tf.keras.layers.Conv2DTranspose(
            output_channels, 3, strides=2,
            padding='same')  #64x64 -> 128x128

        x = last(x)

        model = tf.keras.Model(inputs=inputs, outputs=x)
        return model

    def download_weights(self):
        weights_dir = self.download_directory + '/weights/'
        data_file = self.context.get_data_config()['data_file']
        mobilenet_link = 'https://storage.googleapis.com/tensorflow/keras-applications/mobilenet_v2/' + data_file
        os.mkdir(weights_dir)

        urllib.request.urlretrieve(mobilenet_link,weights_dir + data_file)
        return weights_dir + data_file

    def build_model(self):
        model_weights_loc =  self.download_weights()

        base_model = tf.keras.applications.MobileNetV2(input_shape=[128, 128, 3], include_top=False, weights=model_weights_loc)

        # Use the activations of these layers
        layer_names = [
            'block_1_expand_relu',   # 64x64
            'block_3_expand_relu',   # 32x32
            'block_6_expand_relu',   # 16x16
            'block_13_expand_relu',  # 8x8
            'block_16_project',      # 4x4
        ]
        layers = [base_model.get_layer(name).output for name in layer_names]

        # Create the feature extraction model
        self.down_stack = tf.keras.Model(inputs=base_model.input, outputs=layers)

        self.down_stack.trainable = False

        self.up_stack = [
        pix2pix.upsample(512, 3),  # 4x4 -> 8x8
        pix2pix.upsample(256, 3),  # 8x8 -> 16x16
        pix2pix.upsample(128, 3),  # 16x16 -> 32x32
        pix2pix.upsample(64, 3),   # 32x32 -> 64x64
        ]

        model = self.unet_model(self.context.get_hparam("OUTPUT_CHANNELS"))

        # Wrap the model.
        model = self.context.wrap_model(model)

        # Create and wrap optimizer.
        optimizer = tf.keras.optimizers.Adam()
        optimizer = self.context.wrap_optimizer(optimizer)

        model.compile(
            optimizer=optimizer,
            loss=tf.keras.losses.SparseCategoricalCrossentropy(from_logits=True),
            metrics=[tf.keras.metrics.SparseCategoricalAccuracy(name="accuracy")]
        )
        return model

    def build_training_data_loader(self):
        dataset = tfds.load(
            'oxford_iiit_pet:3.*.*',
            split="train",
            with_info=False,
            data_dir=self.download_directory,
            download=not self.data_downloaded,
        )

        def load_image_train(datapoint):
            input_image = tf.image.resize(datapoint['image'], (128, 128))
            input_mask = tf.image.resize(datapoint['segmentation_mask'], (128, 128))

            if np.random.uniform(()) > 0.5:
                input_image = tf.image.flip_left_right(input_image)
                input_mask = tf.image.flip_left_right(input_mask)

            input_image, input_mask = self.normalize(input_image, input_mask)
            return input_image, input_mask

        train = dataset.map(load_image_train, num_parallel_calls=tf.data.experimental.AUTOTUNE)
        train = self.context.wrap_dataset(train)
        train_dataset = train.cache().shuffle(self.context.get_data_config().get("BUFFER_SIZE")).batch(self.context.get_per_slot_batch_size()).repeat()
        train_dataset = train_dataset.prefetch(buffer_size=tf.data.experimental.AUTOTUNE)

        return train_dataset

    def build_validation_data_loader(self):
        dataset, info = tfds.load(
            'oxford_iiit_pet:3.*.*',
            split="test",
            with_info=True,
            data_dir=self.download_directory,
            download=not self.data_downloaded,
        )
        def load_image_test(datapoint):
            input_image = tf.image.resize(datapoint['image'], (128, 128))
            input_mask = tf.image.resize(datapoint['segmentation_mask'], (128, 128))

            input_image, input_mask = self.normalize(input_image, input_mask)

            return input_image, input_mask

        test = dataset.map(load_image_test)
        test = self.context.wrap_dataset(test)
        test_dataset = test.batch(self.context.get_per_slot_batch_size())

        return test_dataset
