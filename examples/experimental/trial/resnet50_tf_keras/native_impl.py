import argparse
import pathlib

from official.vision.image_classification import common
from official.vision.image_classification import imagenet_preprocessing
from official.vision.image_classification import resnet_model

import determined as det
from determined import experimental
from determined.experimental import keras

import data


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--mode", dest="mode", help="Specifies test mode or submit mode.", default="submit"
    )
    args = parser.parse_args()

    config = {
        "description": "Resnet50 Imagenet TF Keras",
        "searcher": {
            "name": "single",
            "metric": "val_loss",
            "max_length": {
                "batches": 100,
            },
            "smaller_is_better": True,
        },
        "validation_period": {
            "batches": 100,
        },
        "hyperparameters": {
            "global_batch_size": det.Constant(value=32),
            "learning_rate": det.Constant(value=0.1),
        },
    }
    ctx = keras.init(
        config=config, mode=experimental.Mode(args.mode), context_dir=str(pathlib.Path.cwd())
    )

    lr_schedule = ctx.get_hparam("learning_rate")
    if ctx.get_data_config().get("use_tensor_lr", False):
        lr_schedule = common.PiecewiseConstantDecayWithWarmup(
            batch_size=ctx.get_per_slot_batch_size(),
            epoch_size=imagenet_preprocessing.NUM_IMAGES["train"],
            warmup_epochs=common.LR_SCHEDULE[0][1],
            boundaries=[p[1] for p in common.LR_SCHEDULE[1:]],
            multipliers=[p[0] for p in common.LR_SCHEDULE],
            compute_lr_on_cpu=True,
        )
    optimizer = common.get_optimizer(lr_schedule)

    model = resnet_model.resnet50(num_classes=imagenet_preprocessing.NUM_CLASSES)
    model = ctx.wrap_model(model)

    model.compile(
        loss="sparse_categorical_crossentropy",
        optimizer=optimizer,
        metrics=(["sparse_categorical_accuracy"]),
    )

    data_shape = (
        ctx.get_per_slot_batch_size(),
        imagenet_preprocessing.DEFAULT_IMAGE_SIZE,
        imagenet_preprocessing.DEFAULT_IMAGE_SIZE,
        imagenet_preprocessing.NUM_CHANNELS,
    )
    labels_shape = (ctx.get_per_slot_batch_size(),)
    model.fit(
        data.SyntheticData(ctx.get_per_slot_batch_size(), data_shape, labels_shape),
        epochs=1,
        steps_per_epoch=1,
        validation_steps=1,
        validation_data=data.SyntheticData(ctx.get_per_slot_batch_size(), data_shape, labels_shape),
        verbose=2,
    )
