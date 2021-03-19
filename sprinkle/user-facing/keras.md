# New Keras API

## Features

The New Keras API from Determined lets you run your keras model with
automatic support for:

* running locally or on a managed cluster
* running on one gpu, or across many gpus
* pausing & resuming
* checkpoint tracking
* metrics tracking & visualizations
* advanced hyperparameter search
* no-effort distributed training
* reproducibility

The New Keras API is meant to replace `TFKerasTrial` with something more
natural.

## Checklist

* [x] Get a keras context object with `context = det.keras.init()`
* [x] Access any hyperparameters via `context.training.hparams["my_hparam"]`
* [x] Wrap your training and validation data each with `data = context.wrap_data(data)`
* [x] Wrap your optimizer with `optimizer = context.wrap_optimizer(optimizer)`
* [x] Optionally calculate your per-shard batch size with `batch_size =
context.distributed.shard_batch_size(global_batch_size)` to keep the same effective
batch size regardless of number of workers.
* [x] Train and evaluate in a single step with `context.fit(model, ...)`

## Training Example

Here is an example training script, which can either be submitted to the
cluster or it can be invoked to train locally right on your laptop.

```diff
    # train.py
    # taken from keras mnist example, with modifications indicated

    import numpy as np
    from tensorflow import keras
    from tensorflow.keras import layers

+   import determined as det
+   import determined.keras

+   # this step will also initialize random seeds automatically
+   context = det.keras.init()

+   # get the hyperparameters for this trial
+   hparams = context.training.hparams

    # model code is straight from keras mnist example
    num_classes = 10
    input_shape = (28, 28, 1)

    (x_train, y_train), (x_test, y_test) = keras.datasets.mnist.load_data()

    x_train = x_train.astype("float32") / 255
    x_test = x_test.astype("float32") / 255
    x_train = np.expand_dims(x_train, -1)
    x_test = np.expand_dims(x_test, -1)

    y_train = keras.utils.to_categorical(y_train, num_classes)
    y_test = keras.utils.to_categorical(y_test, num_classes)

+   # calling context.wrap_data() will ensure your data is sharded for distributed
+   # training, and it is basically a noop when you are on a single gpu
+   x_train, y_train = context.wrap_data(x_train, y_train)
+   x_test, y_test = context.wrap_data(x_test, y_test)

    model = keras.Sequential(
        [
            keras.Input(shape=input_shape),
            layers.Conv2D(32, kernel_size=(3, 3), activation="relu"),
            layers.MaxPooling2D(pool_size=(2, 2)),
            layers.Conv2D(64, kernel_size=(3, 3), activation="relu"),
            layers.MaxPooling2D(pool_size=(2, 2)),
            layers.Flatten(),
-           layers.Dropout(0.5),
+           # example: suppose you wanted to tune dropout as a hyperparameter:
+           layers.Dropout(hparams["dropout"])
            layers.Dense(num_classes, activation="softmax"),
        ]
    )

    model.summary()

-   optimizer = keras.optimizers.SGD(0.0001)
+   # learning rate is commonly set as a hyperparameter
+   optimizer = keras.optimizers.SGD(hparams["learning_rate"])

+   # calling context.wrap_optimizer() does two things:
+   #   - ensures the optimizer state is saved in checkpoints
+   #   - wraps the keras optimizer in a Horovod optimizer
+   #     (only during distributed training)
+   optimizer = context.wrap_optimizer(optimizer)

    model.compile(
        loss="categorical_crossentropy",
        optimizer=optimizer,
        metrics=["accuracy"]
    )


-   global_batch_size = 128
+   # batch_size is a commonly set as a hyperparameter
+   global_batch_size = hparams["global_batch_size"]

+   # Optional for distributed training: cacluate this worker's
+   # batch_size so that the effective batch_size across all workers
+   # remains constant.  If you choose not to use this, the
+   # effective batch_size will be multiplied by the number slots_per_trial.
+   batch_size = context.distributed.shard_batch_size(global_batch_size)

+   # Our keras support requires that you pass validation data into fit()
+   # rather than using separate fit() and evaluate() calls.
+   hist = context.fit(
+       model,
        x=x_train,
        y=y_train,
        batch_size=batch_size,

-       epochs=epochs,         # ignored/automatically set for cluster training
-       initial_epoch=...,     # ignored/automatically set for cluster training

+       validation_split=...,  # during a training job,
+       validation_data=...,   #   some form of validation data
+       x_val=...,             #   must be set
+       y_val=...,             #   for context.fit()
    )

    print("Test loss:", hist.history['val_loss'][-1])
    print("Test accuracy:", hist.score['val_acc'][-1])
```

### Configure and Run Training

The experiment config is very similar to current Determined experiment configs.
The notable difference is that we are defining an `entrypoint_script` that
points at a training script, rather than setting an `entrypoint` which points
to a `module:TrialClass` like before:

```yaml
# const.yaml
hyperparameters:
    dropout: 0.5
    global_batch_size: 128
entrypoint_script: python3 train.py
```

Launching the experiment is also very similar:

```sh
det experiment create const.yaml .
```

## Batch Inference Example

Batch Inference is just like training with a couple minor differences:

* You launch it as a generic job (not as a training experient), i.e. `det job
run ...`
* `context.training` is not available (since you are not training)
* There is no fancy tracking or visualizations of batch inference (yet).

```python
    # batch_inference.py

    import numpy as np
    from tensorflow import keras
    from tensorflow.keras import layers

    import determined as det
    import determined.keras

    context = det.keras.init()

    # you can access values from your config's .data field
    # which is just a dictionary of arbitrary user data
    trial_id = context.config_data["trial_id"]

    # load a model you trained in Determined
    model = context.load_checkpoint_model(trial_id=trial_id)

    data = ...
    data = context.wrap_data(data)

    # You are still free to shard your batch_size but since we are not training
    # there isn't any particular need to.
    batch_size = context.config_data["batch_size"]

    # simple wrapper around model.predict()
    predictions = context.predict(model, data)

    # you'll want to upload your predictions to somewhere persistent
    my_upload_fn(predictions)
```

### Configure and Run Batch Inference

Don't look too closely at this config; we really haven't thought too hard
about it just yet.

```yaml
# batch_inference.yaml
entrypoint_script: python3 batch_inference.py
data:
    trial_id: 777
    batch_size: 64
resources:
    slots: 8
```

Launching is similar to creating experiments:

```sh
det job run batch_inference.yaml .
```

## API Reference

* `context = det.keras.init()`

  Initialize the `context` object which exposes most of the rest of the API.
  The context knows if you are running on the cluster or locally (basically by
  reading environment variables), and it knows if multiple nodes are
  participating in this workflow or not enabled or not, so that calls like
  `context.wrap_optimizer` can do the right thing.

* `context.training.hparams`

  A dictionary containing the hyperparameters selected for this trial.  The
  keys of the dictionary are just the hyperparameters configured in your
  experiment config.

* `data = context.wrap_data(data)`

  A helper funtion for sharding your data for distributed training.  You should
  always shard your datasets, both training and validation.  Wrap data can take
  the same data types as the normal `model.fit()` call:

  ```python
  # keras.utils.data.Sequence:
  seq = context.data.wrap(seq)

  # numpy arrays:
  x, y = context.data.wrap(x, y)
  # or if you have weights:
  x, y, w = context.data.wrap(x, y, w)

  # tf.data.Dataset:
  # (note that this is using tf.data.Datset.shard(), which is best
  # to use as early as possible in the construction of your dataset)
  ds = context.data.wrap(ds)
  ```
* `optimizer = context.wrap_optimizer(optimizer)`

  A helper function which performs two tasks:

  * track the optimizer state so it is reliably saved/restored in checkpoints
  * wrap the optimizer with a Horovod DistributedOptimizer (only during
    distributed training)

  If you have an advance model with multiple optimizers, no problem.  Just wrap
  each of them individually.

* `context.fit(model, ...)`

  A wrapper function for `model.fit()`, with the following differences:

  * It is required that you pass validation data to `context.fit`, rather than
    having a separate `fit()` and `evaluate()` calls.
  * When training on the cluster, the `epochs` and `initial_epoch` parameters
    to `context.fit()` will be ignored and automatic values will be passed
    instead.  `epochs` and `initial_epochs` will still be honored when training
    off-cluster.
  * Additional callbacks will be injected into the training loop to handle:
    * restoring from previous checkpoints on startup
    * saving checkpoints and reporting them to the Determined master
    * reporting training and validation metrics to the master
    * executing hyperparameter searches

* `model = context.load_checkpoint_model(trial_id=None, checkpoint_uuid=None)`

  Load a keras model (weights and architecture) from a checkpoint; either the
  last checkpoint for a given trial, or from a particular uuid.  This is never
  necessary during cluster traning, but it is commonly used for running batch
  inference or test-set metrics in follow-on jobs.

* `context.predict(model, ...)`

  Lightweight wrapper around `model.predict()`.  Commonly used for running
  batch inference in follow-on jobs.

* `context.evaluate(model, ...)`

  Lightweight wrapper around `model.evaluate()`.  In training jobs, this will
  also report validation metrics to the Determined master, but since
  `context.fit()` requires validation data, it will more likely be called to
  evaluate test-set metrics in follow-on jobs.
