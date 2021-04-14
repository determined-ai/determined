import numpy as np
import tensorflow as tf

from determined.keras import TFKerasTrial, TFKerasTrialContext

tf.compat.v1.disable_eager_execution()
tf.compat.v1.disable_v2_behavior()


class NoopKerasTrial(TFKerasTrial):
    def __init__(self, context: TFKerasTrialContext):
        self.context = context

    def build_model(self):
        model = tf.keras.Sequential(
            [
                tf.keras.layers.Dense(
                    8,
                    input_shape=(
                        8,
                        8,
                    ),
                )
            ]
        )
        model = self.context.wrap_model(model)
        optimizer = self.context.wrap_optimizer(tf.keras.optimizers.SGD())
        model.compile(
            loss=tf.keras.losses.MeanSquaredError(),
            optimizer=optimizer,
            metrics=[],
        )
        return model

    def build_training_data_loader(self):
        x_train = np.ones((64, 8, 8))
        y_train = np.ones((64, 8, 8))
        return (x_train, y_train)

    def build_validation_data_loader(self):
        x_val = np.ones((64, 8, 8))
        y_val = np.ones((64, 8, 8))
        return (x_val, y_val)
