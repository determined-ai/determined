import tensorflow as tf
from tensorflow import keras
from tensorflow.keras import layers
from determined.keras import TFKerasTrial, TFKerasTrialContext
from data import DataInitializer


class MultiTextClassificationTrial(TFKerasTrial):

    def __init__(self, context: TFKerasTrialContext) -> None:
        self.context = context
        self.data_loader = DataInitializer(self.context.distributed.get_rank())

    def build_model(self):
        model = tf.keras.Sequential([
            self.data_loader.create_vectorization_layer(),
            layers.Embedding(10000, self.context.get_hparam("embedding_dim")),
            layers.Dropout(0.2),
            layers.GlobalAveragePooling1D(),
            layers.Dropout(0.2),
            layers.Dense(self.context.get_hparam("dense1"))
        ])
        model = self.context.wrap_model(model)
        optimizer = tf.keras.optimizers.Adam()
        optimizer = self.context.wrap_optimizer(optimizer)
        model.compile(loss=keras.losses.SparseCategoricalCrossentropy(from_logits=True),
                      optimizer=optimizer,
                      metrics=[tf.metrics.SparseCategoricalCrossentropy(from_logits=True),
                               tf.metrics.SparseCategoricalAccuracy()
                               ]
                      )
        return model

    def build_training_data_loader(self) -> tf.data.Dataset:
        return self.context.wrap_dataset(self.data_loader.load_training_data())

    def build_validation_data_loader(self) -> tf.data.Dataset:
        return self.context.wrap_dataset(self.data_loader.load_testing_data())
