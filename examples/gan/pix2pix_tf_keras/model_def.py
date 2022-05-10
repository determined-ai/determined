import tensorflow as tf

from determined.keras import InputData, TFKerasTrial, TFKerasTrialContext

from pix2pix import Pix2Pix, make_discriminator_optimizer, make_generator_optimizer
from data import download, get_dataset


class Pix2PixTrial(TFKerasTrial):
    def __init__(self, context: TFKerasTrialContext) -> None:
        self.context = context
        self.path = download(
            self.context.get_data_config()["base"],
            self.context.get_data_config()["dataset"],
        )

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

    def _get_wrapped_dataset(self, set_, batch_size) -> InputData:
        ds = get_dataset(
            self.path,
            self.context.get_data_config()["height"],
            self.context.get_data_config()["width"],
            set_,
            self.context.get_hparam("jitter"),
            self.context.get_hparam("mirror"),
            batch_size,
        )
        ds = self.context.wrap_dataset(ds)
        return ds

    def build_training_data_loader(self) -> InputData:
        return self._get_wrapped_dataset("train", self.context.get_per_slot_batch_size())

    def build_validation_data_loader(self) -> InputData:
        return self._get_wrapped_dataset("test", self.context.get_per_slot_batch_size())
