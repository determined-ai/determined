# Generic API (the "low-level" API)

## Features

The Generic API lets you train arbitrary models in Determined, but in a way
that you can integrate seamlessly with the entire Determined platform.
Technically, no interaction with the Generic API is required to run arbitrary
models on Determined, in which case you will still already have access to the
following platform features:

* scheduling single-gpu jobs on a Determined cluster (on-premise or on-cloud)
* tracking/viewing logs from training jobs in the webui or cli
* tracking of experiment configurations and model definitions
* tracking of random seeds
* interactve labeling of experiments

With a small amount of code changes, you can also integrate with each of the
following features of the Determined platform:

* metrics tracking & visualizations
* checkpoint tracking
* basic hyperparameter search
* advanced hyperparameter search
* pausing & resuming jobs interactively from the webui
* performant spot-instance support
* running on one gpu, or across many gpus

The Generic API is used behind-the-scenes to implement each of the higher-level
APIs Determined offers (PyTorchTrial, Keras, Estimator, Lightning, etc), so
with the Generic API, you really have the full capabilities of the Determined
platform at your fingertips!

## Starting Point

Let's pretend you have a very simple training script built around PyTorch.
In real life, you should use our PyTorchTrial, Determined's high-level API for
integrating with PyTorch, but this author is most familiar with PyTorch, so
that's what the Generic API tutorial is in right now).

We'll split our training into a few different files:
* model.py: defines the model and how to train/evaluate it
* data.py: defines the dataset
* train.py: entrypoint for training the model

### `model.py`

```python
# model.py

import torch

def build_model(learning_rate):
    # the model is silly, the real focus is on the Generic API
    model = nn.Linear(1, 1, False)

    loss_fn = torch.nn.MSELoss()

    opt = torch.optim.SGD(model.parameters(), lr=learning_rate)

    return model, loss_fn, opt

def train_model(model, loss_fn, opt, train_data):
    losses = []
    for batch in train_data:
        data, labels = batch
        # forward pass
        pred = model(data)
        loss = loss_fn(pred, labels)
        losses.append(loss)
        # backward pass
        loss.backward()
        opt.step()
        opt.zero()
    # reduce training metrics, simple average
    train_loss = sum(losses)/len(losses)
    return train_loss

@torch.no_grad
def eval_model(model, loss_fn, eval_data):
    # Validate model.
    losses = []
    for batch in eval_data:
        data, labels = batch
        pred = model(data)
        loss = loss_fn(pred, labels)
        losses.append(loss)
    val_loss = sum(losses)/len(losses)
    return val_loss
```

### `data.py`

```python
# data.py

import torch

class OnesDataset(torch.utils.data.Dataset):
    def __len__(self):
        return 2048

    def __getitem__(self, index):
        # return data, labels
        return torch.Tensor([1.0]), torch.Tensor([1.0])

def build_datasets(batch_size):
    train_data = torch.utils.data.DataLoader(
        OnesDataset(), batch_size=batch_size
    )
    eval_data = torch.utils.data.DataLoader(
        OnesDataset(), batch_size=batch_size
    )
    return train_data, eval_data
```

### `train.py`

```python
# train.py

import torch
from model import build_model, train_model, eval_model
from data import build_data

# some configurations
BATCH_SIZE=32
LR=0.00001
EPOCHS=50

if __name__ == "__main__":
    model, loss_fn, opt = build_model(LR)
    train_data, eval_data = build_dataset(BATCH_SIZE)

    for epoch in range(config.EPOCHS):
        # train model
        train_loss = model.train_model(model, loss_fn, opt, train_data)
        print(f"after epoch {epoch}, train_loss={train_loss}")

        # evaluate model
        eval_loss = model.eval_model(model, loss_fn, train_data)
        print(f"after epoch {epoch}, eval_loss={eval_loss}")

    # checkpoint model
    path="checkpoint_dir/my_checkpoint"
    torch.save(model.state_dict(), path)
```

At this point, by simply submitting your script to the cluster, you already
have all of the following features:

* scheduling single-gpu jobs on a Determined cluster (on-premise or on-cloud)
* tracking/viewing logs from training jobs in the webui or cli
* tracking of experiment configurations and model definitions
* tracking of random seeds
* interactve labeling of experiments

To submit to the cluster, you would configure an experiment config with
something like this:

```yaml
# my_config.yaml
entrypoint_script: python3 train.py

searcher:
    # we're not doing hp search yet so just use the 'single' searcher
    name: "single"
    # we have to configure the searcher for the master even though
    # our training script is going to totally ignore it.
    max_length:
        epochs: 1

# note that for the script to save the checkpoint
# to a persistent location, you would have to set
# up a bind mound so that the checkpoint created
# inside the container appears on the host file
# system.  There's easier ways to do this in
# Determined; we'll get to those shortly.
bind_mounts:
  - host_path: /my/path/to/checkpoints
    container_path: ./checkpoint_dir
```

Then you can submit the script to the cluster like this:

```sh
det experiment create my_config.yaml . -f
```

Note that the `-f` means "follow logs" (they will print right to your terminal)
and the `.` is the model directory that gets passed to the cluster.  To use `.`
you of course have to run this from the same directory as `train.py`, etc.

## Step 1: Metrics and Checkpoint tracking

Now you are running on a Determined cluster, but the webui doesn't have any
useful information about your training metrics or checkpoints, and also that
`bind_mounts` setting was obnoxious.

Let's look at how to get the following API features:

* metrics tracking & visualizations in the webui
* checkpoint tracking

All you have to do is:

* [x] Initialize a Generic API context: `context = det.generic.init()`
* [x] Report training metrics: `context.training.report_training_metrics()`
* [x] Report validation metrics: `context.training.report_validation_metrics()`
* [x] Report checkpoints to the Checkpoint API: `with
context.checkpoint.save_path() as path: ...`

You don't have to report all metrics and checkpoints in one go, but we will in
this demo.

### Updates to `train.py`

```diff
    # train.py

    import torch
    from model import build_model, train_model, eval_model
    from data import build_data

    # some configurations
    BATCH_SIZE=32
    LR=0.00001
    EPOCHS=50

    if __name__ == "__main__":
+       # initialize a context object for accessing the Generic API
+       import determined
+       context = det.generic.init()

        model, loss_fn, opt = build_model(LR)
        train_data, eval_data = build_dataset(BATCH_SIZE)

+       # keep track of how many batches we have trained
+       batches_trained = 0

        for epoch in range(config.EPOCHS):
            # train model
            train_loss = model.train_model(model, loss_fn, opt, train_data)
            print(f"after epoch {epoch}, train_loss={train_loss}")

+           # report training metrics to the master
+           batches_trained += len(train_data)
+           context.training.report_training_metrics(
+               batches_trained=batches_trained,
+               metrics={"loss": train_loss},
+           )

            # evaluate model
            eval_loss = model.eval_model(model, loss_fn, train_data)
            print(f"after epoch {epoch}, eval_loss={eval_loss}")

+           # report validation metrics to the master
+           context.training.report_validation_metrics(
+               batches_trained=batches_trained,
+               metrics={"loss": eval_loss},
+           )

        # checkpoint model
-       path="checkpoint_dir/my_checkpoint"
-       torch.save(model.state_dict(), path)
+       with context.checkpoint.save_path() as path:
+           torch.save(model.state_dict(), f"{path]/my_checkpoint")
```

### Updates to `my_config.yaml`

Notice that the `context.checkpoint.save_path()` from the Generic API was easy
to use, and now we can rely on the cluster's checkpoint configuration, rather
than hacking something together with bindmounts:

```diff
    # my_config.yaml
    entrypoint_script: python3 train.py

    searcher:
        # we're not doing hp search yet so just use the 'single' searcher
        name: "single"
        # we have to configure the searcher for the master even though
        # our training script is going to totally ignore it.
        max_length:
            epochs: 50
-
-   # since we are using context.checkpoint.save_path(),
-   # we don't need bind_mounts anymore
-   bind_mounts:
-     - host_path: /my/path/to/checkpoints
-       container_path: ./checkpoint_dir
```

## Step 2: Basic HP Search + best practices

(rb: after writing out this API, I am not actually that crazy about it, so
don't look to closely at the searcher API itself)

You already have many great benefits of Determined with very little work.  Now,
let's add support for basic hyperparameter searches.  After this, you'll be
able to run random searches, grid searches, and even hyperband searches, using
the "adaptive" searcher, Determined's state-of-the-art implementation of the
hyperband algorithm!

Note that with only basic HP support, adaptive searches will be rougly 20% less
efficient than with advanced HP support, but they will still far outperform
random or grid searches.

As we introduce HP search, we are also going to implement some best practices
for model definitions, including:

* seed our RNGs with the Determined-tracked seed for an experiment,
  significantly improving our ability to reproducibility training results
* separate configuration from code, to be able to easily browse configs and
  even configure new experiments interactively!

### Updates to `train.py`

```diff
    # train.py

    import torch
    from model import build_model, train_model, eval_model
    from data import build_data

-   # prefer to configure via the experiment config
-   BATCH_SIZE=32
-   LR=0.00001
-   EPOCHS=50

    if __name__ == "__main__":
        # initialize a context object for accessing the Generic API
        import determined
        context = det.generic.init()

+       # seed RNGs based on context-provided seed
+       seed = context.training.trial_seed
+       random.seed(seed)
+       torch.random.manual_seed(seed)
+       # ... also seed any other RNGs you need to

+       # read hyperparameters from the context object
+       hparams = context.training.hparams

-       model, loss_fn, opt = build_model(LR)
+       model, loss_fn, opt = build_model(hparams["learning_rate"])

-       train_data, eval_data = build_dataset(BATCH_SIZE)
+       train_data, eval_data = build_dataset(hparams["batch_size"])

        # keep track of how many batches we have trained
        batches_trained = 0

+       # basic searcher API: just find out how long to train for
+       searcher_op = context.training.basic_search()

-       for epoch in range(config.EPOCHS):
+       for epoch in range(searcher_op.epochs):
            # train model
            ...

            # report training metrics to the master
            ...

            # evaluate model
            ...

            # report validation metrics to the master
            ...

+           if epoch == searcher_op.epochs - 1:
+               # on the last epoch, report your final metric
+               searcher_op.complete(searcher)
+           else:
+               # totally optional: tell the searcher about your intermediate
+               # progress so the webui can estimate overall progress
+               searcher_op.report_progress(epoch)

        # checkpoint model
        with context.checkpoint.save_path() as path:
            torch.save(model.state_dict(), f"{path]/my_checkpoint")
```


### Updates to `my_config.yaml`

As we move more information to the configuration, we are improving our own
development velocity, allowing us to quickly reconfigure new models with all
the controls in one place:

```diff
    # my_config.yaml
    entrypoint_script: python3 train.py

+   # define a hyperparameter search space
+   hyperparameters:
+       batch_size:
+           type: int
+           minval: 1
+           maxval: 64
+       learning_rate:
+           type: log
+           base: 10
+           minval: -5  # 10^-5 = .00001
+           maxval: -3  # 10^-5 = .001

    searcher:
-       name: "single"
+       # use a real hp search instead of the single searcher
+       name: "random"
        max_length:
-           epochs: 1
+           # we are now honoring max_length in train.py
+           epochs: 50
+
+       max_trials: 10
```

## Step 3: Fault-Tolerance

Do you have training that take hours, days, or even weeks to complete?

Don't let power outages, network outages, or even slow memory leaks cause you
to have to restart.

When you use the Generic API, can checkpoint your progress as often as you like.
Determined will restart your job with the last checkpoint you reported, and
all you have to do for fault tolerance is load that checkpoint and go!

```diff
    model, loss_fn, opt = build_model(hparams["learning_rate"])

+   last_epoch = -1
    batches_trained = 0

+   # read state from the last checkpoint
+   if context.latest_checkpoint is not None:
+       with context.latest_checkpoint as path:
+           state = torch.load(f"{path]/my_checkpoint")
+       model.load_state(state["model"])
+       last_epoch = state["epoch"]
+       batches_trained = state["batches_trained"]

    searcher_op = context.training.basic_search()

-   for epoch in range(searcher_op.epochs):
+   # Only train for the amount that remains
+   for epoch in range(last_epoch, searcher_op.epochs):
        # train model
        ...

        # report training metrics to the master
        ...

        # evaluate model
        ...

        # report validation metrics to the master
        ...

        # report searcher progress or complete
        ...

+       # checkpoint every epoch for high fault-tolerance
+       # also track how far we have trained
+       state = {
+           "model": model.state_dict(),
+           "epoch": epoch,
+           "batches_trained": batches_trained,
+       }
+       with context.checkpoint.save_path() as path:
+           torch.save(state, f"{path]/my_checkpoint")

-   with context.checkpoint.save_path() as path:
-       torch.save(model.state_dict(), f"{path]/my_checkpoint")
```

## Advanced Topics:

### Pause and Resume

Simply check `context.should_preempt()` periodically, and shut down when it
returns true.  Checking every epoch is easy but when you hit the pause button
in the webui you will have to wait up to a full epoch to finish before the
training job shuts down.

You can have a more responsive experience by choosing to check every batch, in
which case you should use `context.should_preempt(block=False)`.  When it
returns true, you may choose to save a checkpoint in a partially-complete epoch
or you may choose to just exit immediately.  Saving a checkpoint minimizes lost
progress, but keeping reproducibility even when checkpoints can occur mid-epoch
can be extremely difficult, depending in particular on your dataset and training
loop complexity.

Here we show the simple, per-epoch version:

```diff
    for epoch in range(last_epoch, searcher_op.epochs):
+       if context.should_preempt():
+           break

        # train model
        ...
```

### Distributed training

Have you ever felt like the hardest part of distributed training is just
setting up all of the machines and getting them so they can talk to each other?

Determined exposes a "launch layer" so to solve that problem.  Let Determined
coordinate all of the compute nodes in a multi-node distributed training job,
and you just kick of training when everything's ready.

The way it works is:

* You schedule a multi-gpu training job with a custom `launch_layer` setting in
  the experiment config.  The `launch_layer` should be an executable script
  with arguments
* Determined starts up all the containers with their assigned gpus
* Determined gathers all of ip addresses and assigned gpus for every compute
  node in the training job (the "rendezvous info" for the job)
* Determined calls your `launch_layer` on every compute node, with the
  rendezvous info exposed as both a python API and as environment variables
* Your launch layer starts the distributed training job on the assigned compute
  nodes however it sees fit.
* For common distributed training solutions (horovod, torch.distributed,
  pytorch lightning) you can actually just use a pre-made launch layer.

Example: Launch a Detectron2 model (based on torch.distributed) with
Determined's built-in torch.distributed support:

```yaml
launch_layer: python3 -m determined.launch.torch_distributed
entrypoint_script: python3 train_detectron.py
```

Example: Train a model that uses horovod directly:

```yaml
launch_layer: python3 -m determined.launch.horovod
entrypoint_script: python3 horovod_model.py
```

TODO: decide on an API for writing your own launch layer from scratch and
include examples.

### Advanced Hyperparameter Search

This API isn't finalized yet, sorry.

### API reference

TODO
