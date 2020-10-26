"""
This example demonstrates how to train a GAN with Determined's TF Keras API.

The Determined TF Keras API support using a subclassed `tf.keras.Model` which
defines a custom `train_step()` and `test_step()`.
"""

import tensorflow as tf

from determined.keras import InputData, TFKerasTrial, TFKerasTrialContext

from dc_gan import DCGan
from data import get_train_dataset, get_validation_dataset


class DCGanTrial(TFKerasTrial):
    def __init__(self, context: TFKerasTrialContext) -> None:
        self.context = context

    def build_model(self) -> tf.keras.models.Model:
        model = DCGan(
            batch_size=self.context.get_per_slot_batch_size(),
            noise_dim=self.context.get_hparam("noise_dim")
        )

        # Wrap the model.
        model = self.context.wrap_model(model)

        # Create and wrap the optimizers.
        g_optimizer = tf.keras.optimizers.Adam(
            learning_rate=self.context.get_hparam("generator_lr"))
        g_optimizer = self.context.wrap_optimizer(g_optimizer)

        d_optimizer = tf.keras.optimizers.Adam(
            learning_rate=self.context.get_hparam("discriminator_lr"))
        d_optimizer = self.context.wrap_optimizer(d_optimizer)

        model.compile(
            discriminator_optimizer=d_optimizer,
            generator_optimizer=g_optimizer,
        )

        return model

    def build_training_data_loader(self) -> InputData:
        ds = get_train_dataset(self.context.distributed.get_rank())

        # Wrap the training dataset.
        ds = self.context.wrap_dataset(ds)
        ds = ds.batch(self.context.get_per_slot_batch_size())
        return ds

    def build_validation_data_loader(self) -> InputData:
        ds = get_validation_dataset(self.context.distributed.get_rank())

        # Wrap the validation dataset.
        ds = self.context.wrap_dataset(ds)
        ds = ds.batch(self.context.get_per_slot_batch_size())
        return ds
