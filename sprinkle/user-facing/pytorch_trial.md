# "Trainer API" for PyTorchTrial

## Features

PyTorchTrial from Determined is already a way to define your PyTorch models in
a structured way that gives you an automatic training loop with support for:

* no-effort distributed training
* low-effort AMP support
* performant and flexible custom metric reduction
* all the benefits of the Determined platform (metrics, checkpoints,
scheduling, distributed training, hyperparameter search, etc)

That's not news; PyTorchTrial already has all of those features.  But the
difference between a PyTorchTrial and a keras model today is simply that there
does not exist a way to manually invoke training on your PyTorchTrial.  You are
only able to train it by submitting it to a Determined master.  That sucks.

The proposed "Trainer API" for PyTorchTrial follows the pattern of keras'
model.fit(), pytorch\_lightning's trainer.fit(), or transformers'
trainer.train() (a pattern we refer to as "trainer APIs").

## Checklist

* [x] Start by organizing your code into a PyTorchTrial (as usual).
* [x] Get a pytorch context object with `context = det.pytorch.init()`
* [x] Create your trial class with `trial = context.build_trial(MyTrialClass)`
* [x] Invoke training with `context.fit(trial, ...)`
* [x] Optionally you can override the your trial's data loaders with arguments
      to `context.fit()`

## Training Example

Here is an example training script, which can either be submitted to the
cluster or it can be invoked to train locally right on your laptop.

```python
# train.py
"""
trains a trivial model to briefly show the PyTorchTrial API
and also the new proposed Trainer API.
"""

import torch
from torch import nn

import determined as det
import determined.pytorch

## Data Definition ##

class OnesDataset(torch.utils.data.Dataset):
    def __len__(self):
        return 64

    def __getitem__(self, index):
        # return data, labels
        return torch.Tensor([1.0]), torch.Tensor([1.0])

## Model Definition ##

class OneVarPytorchTrial(det.pytorch.PyTorchTrial):
    def __init__(self, context):
        # Watch out! This is a trial context, not a trainer context like below!
        self.context = context

        # Wrap any models with context.wrap_model()
        self.model = context.wrap_model(nn.Linear(1, 1, False))

        lr = context.hparams["learning_rate"]

        # Wrap any optimizers with context.wrap_optimizer()
        self.opt = context.wrap_optimizer(
            torch.optim.SGD(self.model.parameters(), lr=lr)
        )

    def train_batch(self, batch, epoch_idx, batch_idx):
        """
        Tell us how to calculate the loss/metrics for a single batch.
        """
        data, labels = batch
        preds = self.model(batch)
        loss = torch.nn.MSELoss()(preds, labels)
        # use context.backward() and context.step_optimizer()
        self.context.backward(loss)
        self.context.step_optimizer(self.opt)
        return {"loss": loss}

    def evaluate_batch(self, batch: pytorch.TorchData):
        """
        Tell us how to calculate validation metrics for a single batch.
        """
        data, labels = batch
        preds = self.model(data)
        loss = torch.nn.MSELoss()(preds, labels)
        return {"loss": loss}

    # you can also define build_training_data_loader() or
    # build_validation_data_loader() as methods of your PyTorchTrial,
    # but in this example we show how to pass the data loaders in via
    # the trainer API.

## Training script ##

if __name__ == "__main__":

    # Watch out! This is a trainer context, not a trial context like above!
    # (this step will also initialize random seeds automatically)
    context = det.pytorch.init()

    # Access your hparams from the trainer context.
    hparams = context.training.hparams
    global_batch_size = context.training.hparams["global_batch_size"]

    # Optional for distributed training: cacluate this worker's
    # batch_size so that the effective batch_size across all workers
    # remains constant.  If you choose not to use this, the
    # effective batch_size will be multiplied by the number slots_per_trial.
    batch_size = context.distributed.shard_batch_size(global_batch_size)

    # det.pytorch.DataLoader is a drop-in replacement for
    # torch.utils.data.DataLoader for non-iterable Datasets.
    train_data = det.pytorch.DataLoader(OnesDataset(), batch_size)
    val_data = det.pytorch.DataLoader(OnesDataset(), batch_size)

    # Create your Trial class.
    my_trial = context.build_trial(OneVarPytorchTrial)

    # now train your Trial class!
    metrics = context.train(
        my_trial,
        train_data,
        val_data,

        # configure the trainer's behavior settings
        min_validation_period=det.Epochs(1),
        min_checkpoint_period=det.Epochs(1),

        # ignored by cluster training, but honored used when training locally
        max_length=det.Epochs(10),
    )
```

### Configure and Run Training

Point your config at your training script, configure a searcher and some
hyperparameters:

```yaml
# grid.yaml
entrypoint_script: python3 train.py
hyperparameters:
    global_batch_size:
        type: categorical
        vals: [16, 32, 64]
searcher:
    name: "grid"
    max_length:
        epochs: 20
```

Launching experiment on the cluster:

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

from train.py import OneVarPytorchTrial, OnesDataset

import torch
from torch import nn

import determined as det
import determined.pytorch

context = det.pytorch.init()

# you can access values from your config's .data field
# which is just a dictionary of arbitrary user data
trial_id = context.config_data["trial_id"]

# load a trial from a checkpoint
my_trial = context.load_checkpoint_trial(OneVarPytorchTrial, trial_id=trial_id)

# You are still free to shard your batch_size but since we are not training
# there isn't any particular need to.
batch_size = context.config_data["batch_size"]

# build a dataset, same as for training
pred_data = det.pytorch.DataLoader(OnesDataset(), batch_size=batch_size)

predictions = context.predict(my_trial, pred_data)

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

Launching is similar to creating experiments, but uses `det job run` instead of
`det experiment create`

```sh
det job run batch_inference.yaml .
```

## PyTorchTrial API Reference

The [PyTorchTrial API docs](
https://docs.determined.ai/latest/reference/api/pytorch.html#determined-pytorch-pytorchtrial
) docs are very thorough and don't need repeating here.

## Trainer API Reference

TODO.
