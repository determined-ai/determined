import tensorflow as tf

from determined.keras import InputData, TFKerasTrial, TFKerasTrialContext

from pix2pix import Pix2Pix, make_discriminator_optimizer, make_generator_optimizer
from data import download, get_dataset


class Pix2PixTrial(TFKerasTrial):
    def __init__(self, context: TFKerasTrialContext) -> None:
        self.context = context
        self.path = download(self.context.distributed.get_rank())

    def build_model(self) -> tf.keras.models.Model:
        model = Pix2Pix()

        # Wrap the model
        model = self.context.wrap_model(model)

        # Create and wrap the optimizers
        g_optimizer = self.context.wrap_optimizer(
            make_generator_optimizer(
                lr=self.context.get_hparam("generator_lr"),
                beta_1=self.context.get_hparam("generator_beta_1"),
            )
        )
        d_optimizer = self.context.wrap_optimizer(
            make_discriminator_optimizer(
                lr=self.context.get_hparam("discriminator_lr"),
                beta_1=self.context.get_hparam("discriminator_beta_1"),
            )
        )

        model.compile(
            discriminator_optimizer=d_optimizer,
            generator_optimizer=g_optimizer,
        )

        return model

    def build_training_data_loader(self) -> InputData:
        ds = get_dataset(
            self.path, batch_size=1
        )  # self.context.get_per_slot_batch_size())
        # Wrap the training dataset.
        ds = self.context.wrap_dataset(ds)
        # ds = ds.batch(self.context.get_per_slot_batch_size())
        return ds

    def build_validation_data_loader(self) -> InputData:
        ds = get_dataset(
            self.path, "val", batch_size=1
        )  # self.context.get_per_slot_batch_size())
        # Wrap the validation dataset.
        ds = self.context.wrap_dataset(ds)
        # ds = ds.batch(self.context.get_per_slot_batch_size())
        return ds
