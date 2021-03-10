    import numpy as np
    from tensorflow import keras
    from tensorflow.keras import layers

+   import determined as det
+   ctx = det.keras.init()  # also initialized random seeds
+   hparams = ctx.get_hparams()

    # Model / data parameters
    num_classes = 10
    input_shape = (28, 28, 1)

    # the data, split between train and test sets
    (x_train, y_train), (x_test, y_test) = keras.datasets.mnist.load_data()

    # Scale images to the [0, 1] range
    x_train = x_train.astype("float32") / 255
    x_test = x_test.astype("float32") / 255
    # Make sure images have shape (28, 28, 1)
    x_train = np.expand_dims(x_train, -1)
    x_test = np.expand_dims(x_test, -1)
    print("x_train shape:", x_train.shape)
    print(x_train.shape[0], "train samples")
    print(x_test.shape[0], "test samples")

    # convert class vectors to binary class matrices
    y_train = keras.utils.to_categorical(y_train, num_classes)
    y_test = keras.utils.to_categorical(y_test, num_classes)

    model = keras.Sequential(
        [
            keras.Input(shape=input_shape),
            layers.Conv2D(32, kernel_size=(3, 3), activation="relu"),
            layers.MaxPooling2D(pool_size=(2, 2)),
            layers.Conv2D(64, kernel_size=(3, 3), activation="relu"),
            layers.MaxPooling2D(pool_size=(2, 2)),
            layers.Flatten(),
            layers.Dropout(0.5),
            layers.Dense(num_classes, activation="softmax"),
        ]
    )

    model.summary()

    batch_size = 128
    epochs = 15

+   # TODO: wrap dataset somewhere
+   optimizer = ctx.keras.wrap_optimizer(...)
    model.compile(loss="categorical_crossentropy", optimizer=optimizer, metrics=["accuracy"])

    batch_size = 128
+   batch_size = ctx.distributed.per_slot_batch_size(batch_size)  # rename shard_batch_size()
+   cb = ctx.make_callback()

    model.fit(
        x_train,
        y_train,
        batch_size=batch_size,
        epochs=epochs,
        validation_split=0.1,
+       initial_epoch=ctx.get_initial_epoch(),
+       callbacks=[cb],
    )

    score = model.evaluate(
        x_test,
        y_test,
        verbose=0,
+       callbacks=[cb],
    )
    print("Test loss:", score[0])
    print("Test accuracy:", score[1])
