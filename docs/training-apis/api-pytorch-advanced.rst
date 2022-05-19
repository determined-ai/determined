################
 Advanced Usage
################

*******************
 Gradient Clipping
*******************

Users need to pass a gradient clipping function to
:meth:`~determined.pytorch.PyTorchTrialContext.step_optimizer`.

.. _pytorch-custom-reducers:

******************
 Reducing Metrics
******************

Determined supports proper reduction of arbitrary training and validation metrics, even during
distributed training, by allowing users to define custom reducers. Custom reducers can be either a
function or an implementation of the :class:`determined.pytorch.MetricReducer` interface. See
:meth:`determined.pytorch.PyTorchTrialContext.wrap_reducer` for more details.

.. _pytorch-reproducible-dataset:

************************************
 Customizing A Reproducible Dataset
************************************

.. note::

   Normally, using :class:`determined.pytorch.DataLoader` is required and handles all of the below
   details without any special effort on your part (see :ref:`pytorch-data-loading`). When
   :class:`determined.pytorch.DataLoader` is not suitable (especially in the case of
   ``IterableDatasets``), you may disable this requirement by calling
   :meth:`context.experimental.disable_dataset_reproducibility_checks()
   <determined.pytorch.PyTorchExperimentalContext.disable_dataset_reproducibility_checks>` in your
   Trial's ``__init__()`` method. Then you may choose to follow the below guidelines for ensuring
   dataset reproducibility on your own.

Achieving a reproducible dataset that is able to pause and continue (sometimes called "incremental
training") is easy if you follow a few rules.

-  Even if you are going to ultimately return an IterableDataset, it is best to use PyTorch's
   Sampler class as the basis for choosing the order of records. Operations on Samplers are quick
   and cheap, while operations on data afterwards are expensive. For more details, see the
   discussion of random vs sequential access `here <https://yogadl.readthedocs.io>`_. If you don't
   have a custom sampler, start with a simple one:

   ..
      code::python

      sampler = torch.utils.data.SequentialSampler(my_dataset)

-  **Shuffle first**: Always use a reproducible shuffle when you shuffle. Determined provides two
   shuffling samplers for this purpose; the ``ReproducibleShuffleSampler`` for operating on records
   and the ``ReproducibleShuffleBatchSampler`` for operating on batches. You should prefer to
   shuffle on records (use the ``ReproducibleShuffleSampler``) whenever possible, to achieve the
   highest-quality shuffle.

-  **Repeat when training**: In Determined, you always repeat your training dataset and you never
   repeat your validation datasets. Determined provides a RepeatSampler and a RepeatBatchSampler to
   wrap your sampler or batch_sampler. For your training dataset, make sure that you always repeat
   AFTER you shuffle, otherwise your shuffle will hang.

-  **Always shard, and not before a repeat**: Use Determined's DistributedSampler or
   DistributedBatchSampler to provide a unique shard of data to each worker based on your sampler or
   batch_sampler. It is best to always shard your data, and even when you are not doing distributed
   training, because in non-distributed-training settings, the sharding is nearly zero-cost, and it
   makes distributed training seamless if you ever want to use it in the future.

   It is generally important to shard after you repeat, unless you can guarantee that each shard of
   the dataset will have the same length. Otherwise, differences between the epoch boundaries for
   each worker can grow over time, especially on small datasets. If you shard after you repeat, you
   can change the number of workers arbitrarily without issue.

-  **Skip when training, and always last**: In Determined, training datasets should always be able
   to start from an arbitrary point in the dataset. This allows for advanced hyperparameter searches
   and responsive preemption for training on spot instances in the cloud. The easiest way to do
   this, which is also very efficient, is to apply a skip to the sampler.

   Determined provides a SkipBatchSampler that you can apply to your batch_sampler for this purpose.
   There is also a SkipSampler that you can apply to your sampler, but you should prefer to skip on
   batches unless you are confident that your dataset always yields identical size batches, where
   the number of records to skip can be reliably calculatd from the number of batches already
   trained.

   Always skip AFTER your repeat, so that the skip only happens once, and not on every epoch.

   Always skip AFTER your shuffle, to preserve the reproducibility of the shuffle.

Here is some example code that follows each of these rules that you can use as a starting point if
you find that the built-in context.DataLoader() does not support your use case.

.. code:: python

   def make_batch_sampler(
     sampler_or_dataset,
     mode,  # mode="training" or mode="validation"
     shuffle_seed,
     num_workers,
     rank,
     batch_size,
     skip,
   ):
       if isinstance(sampler_or_dataset, torch.utils.data.Sampler):
           sampler = sampler_or_dataset
       else:
           # Create a SequentialSampler if we started with a Dataset.
           sampler = torch.utils.data.SequentialSampler(sampler_or_dataset)

       if mode == "training":
           # Shuffle first.
           sampler = samplers.ReproducibleShuffleSampler(sampler, shuffle_seed)

           # Repeat when training.
           sampler = samplers.RepeatSampler(sampler)

       # Always shard, and not before a repeat.
       sampler = samplers.DistributedSampler(sampler, num_workers=num_workers, rank=rank)

       # Batch before skip, because Determined counts batches, not records.
       batch_sampler = torch.utils.data.BatchSampler(sampler, batch_size, drop_last=False)

       if mode == "training":
           # Skip when training, and always last.
           batch_sampler = samplers.SkipBatchSampler(batch_sampler, skip)

       return batch_sampler

   class MyPyTorchTrial(det.pytorch.PyTorchTrial):
       def __init__(self, context):
           context.experimental.disable_dataset_reproducibility_checks()

       def build_training_data_loader(self):
           my_dataset = ...

           batch_sampler = make_batch_sampler(
               dataset=my_dataset,
               mode="training",
               seed=self.context.get_trial_seed(),
               num_workers=self.context.distributed.get_size(),
               rank=self.distributed.get_rank(),
               batch_size=self.context.get_per_slot_batch_size(),
               skip=self.context.get_initial_batch(),
           )

           return torch.utils.data.DataLoader(my_dataset, batch_sampler=batch_sampler)

See the :mod:`determined.pytorch.samplers` for details.

***************************
 Automatic Mixed Precision
***************************

Determined supports using the PyTorch Automatic Mixed Precision (AMP) library, which enables a more
compact representation of models in GPU memory, and for larger batches to be trained.

In typical PyTorch usage, a scaler object, typically an instance of ``torch.cuda.amp.GradScaler``,
is created and used to scale the loss for the backward pass and step optimizers during training. The
scaler itself should be updated on each training iteration. The forward pass is wrapped in a
``torch.cuda.amp.autocast`` context manager, which is done both in training and when doing
operations such as evaluation and inference. See the `PyTorch AMP package documentation
<https://pytorch.org/docs/stable/amp.html>`_ and the `AMP recipe
<https://pytorch.org/tutorials/recipes/recipes/amp_recipe.html>`_ for full details and examples.

In the Determined PyTorchTrial API, this usage requires two modifications:

-  The scaler should be wrapped before use with ``PyTorchTrial.wrap_scaler()``.
-  Instead of calling ``scaler.step()``, directly, pass the scaler to
   ``PyTorchTrialContext.step_optimizer()``, which calls ``scaler.step()``.

For example:

.. code:: python

   from torch.cuda.amp import GradScaler, autocast
   from determined.pytorch import PyTorchTrial

   class AMPTrial(PyTorchTrial):

       def __init__(self, context):
           # Other initialization

           self.scaler = context.wrap_scaler(GradScaler())
           super().__init__(context)


       def train_batch(self, ...):
           with autocast():
               # Normal forward pass to get loss

           self.context.backward(self.scaler.scale(loss))
           self.context.step_optimizer(self.optimizer, scaler=self.scaler)
           self.scaler.update()
           return {"loss": loss}


       def evaluate_batch(self, ...):
           with autocast():
               # Normal forward pass to get loss

           return {"validation_loss": loss}

If your model invokes the features in ``torch.cuda.amp`` using only the pattern shown in the
examples and recipes, you might be able to use an experimental feature that has all the above
functionality without the need for additional code changes. To use the experimental feature, enable
the feature during model initialization, as shown here:

.. code:: python

   from determined.pytorch import PyTorchTrial

   class AMPTrial(PyTorchTrial):

       def __init__(self, context):
           # Other initialization

          context.experimental.use_amp()
          super().__init__(context)
