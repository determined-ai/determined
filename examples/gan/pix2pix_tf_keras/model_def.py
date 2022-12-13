import tensorflow as tf
from data import download, load_dataset
from pix2pix import Pix2Pix, make_discriminator_optimizer, make_generator_optimizer

from determined.keras import InputData, TFKerasTrial, TFKerasTrialContext


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

    def _get_wrapped_dataset(self, set_) -> InputData:
        ds = load_dataset(
            self.path,
            self.context.get_data_config()["height"],
            self.context.get_data_config()["width"],
            set_,
            self.context.get_hparam("jitter"),
            self.context.get_hparam("mirror"),
        )
        ds = self.context.wrap_dataset(ds)
        return ds

    def build_training_data_loader(self) -> InputData:
        train_dataset = (
            self._get_wrapped_dataset("train")
            .cache()
            .shuffle(self.context.get_data_config().get("BUFFER_SIZE"))
            .batch(self.context.get_per_slot_batch_size())
            .repeat()
            .prefetch(buffer_size=tf.data.experimental.AUTOTUNE)
        )
        return train_dataset

    def build_validation_data_loader(self) -> InputData:
        test_dataset = self._get_wrapped_dataset("test").batch(
            self.context.get_per_slot_batch_size()
        )
        return test_dataset
