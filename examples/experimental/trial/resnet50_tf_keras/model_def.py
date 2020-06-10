"""
This example shows how to use an existing tf.keras model with Determined.
The flags and configurations can be under const.yaml. For more infomation
about optional flags view the original script linked below.

Based off:
https://github.com/tensorflow/models/blob/master/official/vision/image_classification/resnet_imagenet_main.py

"""
import tensorflow as tf

import tensorflow_files.imagenet_preprocessing as imagenet_preprocessing
from determined.keras import (
    TFKerasTensorBoard,
    TFKerasTrial,
    TFKerasTrialContext,
    InputData,
)
from tensorflow_files.common import (
    LR_SCHEDULE,
    LearningRateBatchScheduler,
    PiecewiseConstantDecayWithWarmup,
    get_optimizer,
    learning_rate_schedule,
)
from tensorflow_files.resnet_model import resnet50
from tensorflow_files.trivial_model import trivial_model

import data


class ResNetModel(TFKerasTrial):
    def __init__(self, context: TFKerasTrialContext) -> None:
        self.context = context

    def build_training_data_loader(self) -> InputData:
        """
        Create synthetic data loaders for testing and benchmarking purposes.
        """
        batch_size = self.context.get_per_slot_batch_size()
        image_size = imagenet_preprocessing.DEFAULT_IMAGE_SIZE

        data_shape = (batch_size, image_size, image_size, imagenet_preprocessing.NUM_CHANNELS)
        labels_shape = (batch_size,)

        return data.SyntheticData(50, data_shape, labels_shape)

    def build_validation_data_loader(self) -> InputData:
        """
        Create synthetic data loaders for testing and benchmarking purposes.
        """
        batch_size = self.context.get_per_slot_batch_size()
        image_size = imagenet_preprocessing.DEFAULT_IMAGE_SIZE

        data_shape = (batch_size, image_size, image_size, imagenet_preprocessing.NUM_CHANNELS)
        labels_shape = (batch_size,)

        return data.SyntheticData(50, data_shape, labels_shape)

    def set_learning_rate(self):
        lr_schedule = PiecewiseConstantDecayWithWarmup(
            batch_size=self.context.get_per_slot_batch_size(),
            epoch_size=imagenet_preprocessing.NUM_IMAGES["train"],
            warmup_epochs=LR_SCHEDULE[0][1],
            boundaries=[p[1] for p in LR_SCHEDULE[1:]],
            multipliers=[p[0] for p in LR_SCHEDULE],
            compute_lr_on_cpu=True,
        )
        return lr_schedule

    def build_model(self):
        """
        Required Method that build the model
        Returns: Sequential
        """
        lr_schedule = self.context.get_hparam("learning_rate")
        if self.context.get_data_config()["use_tensor_lr"]:
            lr_schedule = self.set_learning_rate()

        optimizer = get_optimizer(lr_schedule)
        if self.context.get_data_config()["fp16_implementation"] == "graph_rewrite":
            # Note: when flags_obj.fp16_implementation == "graph_rewrite", dtype as
            # determined by flags_core.get_tf_dtype(flags_obj) would be 'float32'
            # which will ensure tf.compat.v2.keras.mixed_precision and
            # tf.train.experimental.enable_mixed_precision_graph_rewrite do not double
            # up.
            optimizer = tf.train.experimental.enable_mixed_precision_graph_rewrite(optimizer)

        if self.context.get_data_config()["use_trivial_model"]:
            model = trivial_model(imagenet_preprocessing.NUM_CLASSES)
        else:
            model = resnet50(num_classes=imagenet_preprocessing.NUM_CLASSES)
        model = self.context.wrap_model(model)
        model.compile(
            loss="sparse_categorical_crossentropy",
            optimizer=optimizer,
            metrics=(["sparse_categorical_accuracy"]),
            run_eagerly=self.context.get_data_config()["run_eagerly"],
        )

        return model

    def keras_callbacks(self):
        """
        Returns: List[tf.keras.callbacks.Callback]
        """
        callbacks = [TFKerasTensorBoard(update_freq="batch", profile_batch=0, histogram_freq=1)]
        callbacks.append(
            LearningRateBatchScheduler(
                learning_rate_schedule, self.context.get_global_batch_size(), 1000
            )
        )
        return callbacks
