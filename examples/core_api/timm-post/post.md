(This post references unreleased features, namely https://github.com/determined-ai/determined/pull/3859 and https://github.com/determined-ai/determined/pull/3807.  To run examples, you'll need to create a local branch with those two PRs merged.)

(TODO: Replace github references with doc references where possible.)

(TODO: Are the initial paragraphs the right story to tell about Core API?  Could completely replace them.)

Users of Determined have always faced a small but annoying barrier to entry - porting their model training scripts to use one of Determined's Trial APIs.  Similar to popular libraries like PyTorch Lightning, our Trial APIs provide user-definable hooks for each step in a standard model training workflow.  Our behind-the-scenes execution harness then handles the messy details of providing scalable, distributed, preemptible training with full access to features like hyperparameter search and checkpoint management.

Over time, we've realized that ML teams often have extensive script libraries and custom execution harnesses of their own that make this porting process non-trivial.  To address this, we designed *Core API*: an API with fewer built-in assumptions that can be plugged into existing code with minimal fuss.  We're releasing it as part of Determined 0.17.XX.

In this post, we'll walk through converting an existing training script to use the full set of Core API features.  We'll proceed in steps, to illustrate how Core API enables incremental adoption of features.  And instead of working with a toy example, we'll instead use the training script from Ross Wightman's popular `timm`  library for image models.  TODO: More background on `timm`?

To follow along, all you'll need is the Determined CLI and a Determined cluster with at least 2 GPUs.  Check out https://docs.determined.ai/latest/getting-started.html for details.  Scripts and configuration files for this post can be downloaded at (TODO).

# Step 0: Running the Script
To start off, we'll show how to run the script on a single GPU with no modifications.  We copied the script unchanged from https://github.com/rwightman/pytorch-image-models/blob/master/train.py to `step0-run.py`.  Changes in 0.17.XX allow us to run arbitrary executables in a Determined experiment by setting them as our  `entrypoint`, so we can use the following configuration to train on CIFAR10:
```name: step0-run.yaml
entrypoint: python step0-run.py --dataset=torch/cifar10 --dataset-download data --input-size 3 32 32 --epochs 5
max_restarts: 0
searcher:
   name: single
   max_length: 1
   metric: val_loss
```
We'll talk more about `searcher` in Step 4, but for now we can safely ignore `max_length` and `metric` .  Setting the searcher to `single` means this experiment will create only one trial.

We also need to make sure `timm` gets installed in each task container.  We'll use the following `startup-hook.sh` file, which will run in each container before running our `entrypoint`:
```name: startup-hook.sh
pip install timm
```

We can now launch training using `det experiment create step0.yaml .` and monitor training logs through the Web UI.

# Step 1: Distributed Training

The `timm` training script uses PyTorch Distributed to do distributed data parallel training, with the essential bit of configuration being this call:
```
torch.distributed.init_process_group(backend="nccl", init_method="env://")
```
[init_process_group](https://pytorch.org/docs/stable/distributed.html#torch.distributed.init_process_group) will then pull information from environment variables about how to connect with other processes and coordinate distributed training.

When we set `slots_per_trial` to something bigger than 1 in our experiment configuration, the Determined master will take care of starting up multiple copies of our entrypoint in containers distributed appropriately across our cluster -- we just need to set those environment variables correctly in each container.  In 0.17.XX, we've provided a [launch script](https://github.com/determined-ai/determined/tree/master/harness/determined/launch) that wraps an entrypoint and handles this environment setup, along with similar scripts for DeepSpeed and Horovod.  Our main step toward distributed training is to modify our `entrypoint` with the appropriate launch script.  While we're at it, we'll bump `slots_per_trial` to 2 to tell the Determined master we want to use 2 GPUs:
```name: step1-distributed.yaml
name: core-api-timm-step1
entrypoint: >-
  python -m determined.launch.torch_distributed
  python step1-distributed.py --dataset=torch/cifar10 --dataset-download data --input-size 3 32 32 --epochs 5
max_restarts: 0
resources:
  slots_per_trial: 2
searcher:
   name: single
   max_length: 1
   metric: val_loss

```

We also need a small edit during initialization to set `args.local_rank` appropriately instead of getting it from the command line:
```
if "LOCAL_RANK" in os.environ:
    args.local_rank = int(os.environ["LOCAL_RANK"])
else:
    args.local_rank = 0
```

And that's it!  If we launch this updated configuration, we can now confirm in the logs that we're training across 2 GPUs:
```
[rank=0] || Training in distributed mode with multiple processes, 1 GPU per process. Process 0, total 2.
[rank=1] || Training in distributed mode with multiple processes, 1 GPU per process. Process 1, total 2.
```

Looking under the hood a bit, the `determined.launch.torch_distributed` launcher uses the `determined.get_cluster_info()` Core API call to get rendezvous information for each task container before passing the hard work on to `torchrun`.  Advanced users can follow a similar pattern to write their own launch scripts as needed.

# Step 2: Collecting Metrics
Currently, our experiment overview page is blank:
< TODO: Blank Metrics Image >
Let's report our metrics to the Determined master, so we can see some nice graphs instead!  First, we'll initialize a Core API context and pass it into our `main` function:
```
# We need to add determined as an import at the top of the file.
import determined as det

...

distributed = det.core.DistributedContext.from_torch_distributed()
with det.core.init(distributed=distributed) as core_context:
    main(core_context)

```
The `DistributedContext` we create here provides simple synchronization primitives used to coordinate Core API functionality across workers.  If you're running with one of our supported launchers, you can use the correspondng `from_` helper function to create one from the corresponding environment variables -- otherwise, you can [initialize one manually](https://github.com/determined-ai/determined/blob/master/harness/determined/_core/_distributed.py).

Reporting a dictionary of metrics is now a single API call:
```
# The original script already takes care of aggregating metrics across workers.
train_metrics = train_one_epoch(...)

# Metrics can only be reported on rank 0, to avoid duplicate reports.
if args.rank == 0:
    core_context.train.report_training_metrics(latest_batch=latest_batch, metrics=train_metrics)
```
And similarly for validation metrics:
```
if args.rank == 0:
	# Prefix metrics with val_ to distinguish from training metrics.
    core_context.train.report_validation_metrics(
        latest_batch=latest_batch,
        metrics={"val_" + k: v for k, v in eval_metrics.items()},
    )
```


If we rerun our experiment and check the overview page, we can now see our metrics nicely presented:

<TODO: Metrics Image>


# Step 3: Checkpointing
The `timm` training script already has it's own built-in checkpointing system.  Rather than fully replace it, we'll instead see how we can gracefully layer Determined's checkpointing on top of it.  In a real use case, this would avoid breaking any integrations that rely on the existing functionality.

First, we'll pull some metadata from `get_cluster_info()`:

```
info = det.get_cluster_info()
# If running in local mode, cluster info will be None.
if info is not None:
    latest_checkpoint = info.latest_checkpoint
    trial_id = info.trial.trial_id
else:
    latest_checkpoint = None
    trial_id = -1
```

To save checkpoints, we'll hook into `timm`'s `CheckpointSaver` class with our own derived version:
```
class DeterminedCheckpointSaver(CheckpointSaver):
    def __init__(self, trial_id, epoch_length, *args, **kwargs):
        self.trial_id = trial_id
        self.epoch_length = epoch_length
        super().__init__(*args, **kwargs)

    def _save(self, save_path, epoch, metric=None):
        super()._save(save_path, epoch, metric)
        checkpoint_metadata = {
            "latest_batch": self.epoch_length * (epoch + 1),
            "trial_id": self.trial_id,
        }
        with core_context.checkpoint.store_path(checkpoint_metadata) as (path, _):
            shutil.copy2(save_path, path.joinpath("data"))
```
The essential Core API call here is `core_context.checkpoint.store_path`, which returns a context manager that:
- On entry, finds or creates a directory for us to write checkpoint files to, returning the path and a unique checkpoint ID.
- On exit, uploads the files to checkpoint storage (if necessary) and notifies the master.
After running the parent logic to save a checkpoint to `save_path` , we just `shutil.copy2` the full contents of `save_path` to the Determined checkpoint directory without needing to know anything specific about the `timm` checkpoint format.

Now we just need to substitute `DeterminedCheckpointSaver` in for the original:
```
saver = DeterminedCheckpointSaver(...)
```
And we can now see that checkpoints are being saved to storage:
< TODO: Checkpoint image >

Restoring from a checkpoint is only slightly more complicated:
```
if latest_checkpoint is not None:
    restore_path_context = core_context.checkpoint.restore_path(latest_checkpoint)
else:
    restore_path_context = contextlib.nullcontext()
with restore_path_context as restore_path:
    if restore_path is not None:
        checkpoint_path = restore_path.joinpath("data")
    else:
        checkpoint_path = args.resume
    if checkpoint_path:
        resume_epoch = resume_checkpoint(...)
        metadata = core_context.checkpoint.get_metadata(latest_checkpoint)
        prev_trial_id = metadata["trial_id"]
        if trial_id != prev_trial_id:
            resume_epoch = 0
```

We'll break this down piece by piece. First:
```
if latest_checkpoint is not None:
    restore_path_context = core_context.checkpoint.restore_path(latest_checkpoint)
else:
    restore_path_context = contextlib.nullcontext()
with restore_path_context as restore_path:
```
The centerpiece is `core_context.checkpoint.restore_path`, which returns another context manager.  This one:
- On entry, downloads the Determined checkpoint files if necessary and returns a path to the directory containing them.  A separate download won't be necessary if checkpoints are stored in a shared filesystem.
- On exit, cleans up the files if they were downloaded.

Because the checkpoint files are only guaranteed to exist inside the `with` block, we manage the slightly awkward control flow by using `nullcontext` as a stand in when there's no checkpoint to resume from.

Next:
```
if restore_path is not None:
    checkpoint_path = restore_path.joinpath("data")
else:
    checkpoint_path = args.resume
```
We continue to support the `--resume` script argument, but only if there's no Determined checkpoint to continue from.  Depending on your use case, you might want to flip which one takes priority.

And last:
```
if checkpoint_path:
    resume_epoch = resume_checkpoint(...)
    metadata = core_context.checkpoint.get_metadata(latest_checkpoint)
    prev_trial_id = metadata["trial_id"]
    if trial_id != prev_trial_id:
        resume_epoch = 0
```
This addresses the two ways we might restore a Determined checkpoint:
1. Continue an existing trial, resuming at the epoch where we left off.  This corresponds to the Pause/Resume button in the Web UI.
2. Start a new trial, using the checkpont weights for initialization but starting training from epoch 0.  This corresponds to the Continue Trial button in the Web UI.

As a finishing touch, you might've noticed that pausing our experiments in the Web UI doesn't actually pause them!  That's because pausing an experiment (as opposed to killing one) is a voluntary activity.  For instance, we may want to hold off on pausing until we can finish a checkpoint.  To correctly support pausing and other forms of preemption such as scheduler prioritization, we need to add the following after each epoch:
```
if core_context.preempt.should_preempt():
	# Exit the process.  When unpausing, the process will be restarted and resumed from the latest checkpoint.
    return
```

# Step 4: Hyperparameter Tuning
To make use of Determined's [Hyperparameter Tuning](https://docs.determined.ai/latest/training-hyperparameter/index.html) functionality, we need to define our hyperparameter search space in the `hyperparameters` section of our experiment configuration.  For demonstration, we'll perform a five point grid search for learning rate on a logarithmic scale from 0.01 to 0.1:
```
name: core-api-timm-step4
entrypoint: >-
  python -m determined.launch.torch_distributed
  python step4-hyperparameters.py --dataset=torch/cifar10 --dataset-download data --input-size 3 32 32 --epochs 5
max_restarts: 0
hyperparameters:
  lr:
    type: log
    base: 10
    minval: -2
    maxval: -1
    count: 5
resources:
  slots_per_trial: 2
searcher:
   name: grid
   max_length: 5
   metric: val_loss
```

(Notice that we've also set `searcher.name` to `grid` and `searcher.max_length` to 5 to indicate that we'd like to run each trial for at most 5 epochs -- more on that momentarily.)

The `timm` training script accepts hyperparameters and other configuration through the command line via Python's `ArgumentParser`.  We can inject our hyperparameters into this process in the `namespace` argument to `parse_args` as follows:
```
def _parse_args(namespace=None):
	...
    args = parser.parse_args(remaining, namespace=namespace)
    ...
    return args, args_text

def main(core_context):
    info = det.get_cluster_info()
    ...
    hparams = argparse.Namespace(**info.trial.hparams)
    args, args_text = _parse_args(hparams)
    _logger.info(f"Arguments and hyperparameters: {args_text}")
    ...
```

This results in command line parameters taking priority over experiment configuration hyperparameters.  We can now see that we're performing a grid search, and our learning rate changes are displayed in the trial logs.

< TODO: Experiment overview page >

Back to `searcher.max_length`: right now, we're specifying training length through the `--epochs` argument.  This doesn't allow us to correctly make use of hyperparameter search algorithms like [Adaptive Asha](https://docs.determined.ai/latest/training-hyperparameter/hp-adaptive-asha.html) which dynamically adjust training length.  If that's fine for your use case, you can stop here.

If we instead want to respect the `searcher`'s opinion on how long we should train for, we can do the following:
```
next_epoch = start_epoch
for op in core_context.searcher.operations():
    for epoch in range(next_epoch, op.length):
	    ... # train for one epoch
	    if args.rank == 0:
            op.report_progress(epoch)
        ...
    next_epoch = op.length
    if args.rank == 0:
        op.report_completed(best_metric)
```
Here, `op.length` specifies the epoch we should train up to.  Note that we're still relying on `--epochs` for setting learning rate schedules.  We could instead use `searcher.max_length` for this if we wanted DRYness at the expense of flexibility.

# TODO: Outro, any CTAs?
