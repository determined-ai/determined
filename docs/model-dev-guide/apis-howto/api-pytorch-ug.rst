#############
 PyTorch API
#############

.. meta::
   :description: Learn how to train a PyTorch model in Determined. This user guide covers everything from PyTorch's tensor operations, data loading, and preprocessing techniques, to how to train and evaluate your models using Determined AI's PyTorch Trial and PyTorch Trainer.

In this guide, you'll learn how to use :ref:`pytorch_trial_ug` and :ref:`pytorch_trainer_ug`.

+---------------------------------------------------------------------+
| Visit the API reference                                             |
+=====================================================================+
| :ref:`pytorch_api_ref`                                              |
+---------------------------------------------------------------------+

.. _pytorch_trial_ug:

***************
 PyTorch Trial
***************

To train a PyTorch model in Determined, you need to implement a trial class that inherits from
:class:`~determined.pytorch.PyTorchTrial` and specify it as the entrypoint in the :ref:`experiment
configuration <experiment-config-reference>`.

To implement a :class:`~determined.pytorch.PyTorchTrial`, you need to override specific functions
that represent the components that are used in the training procedure. It is helpful to work off of
a skeleton to keep track of what is still required. A good starting template can be found below:

.. code:: python

   from typing import Any, Dict, Union, Sequence
   from determined.pytorch import DataLoader, PyTorchTrial, PyTorchTrialContext

   TorchData = Union[Dict[str, torch.Tensor], Sequence[torch.Tensor], torch.Tensor]

   class MyTrial(PyTorchTrial):
       def __init__(self, context: PyTorchTrialContext) -> None:
           self.context = context

       def build_training_data_loader(self) -> DataLoader:
           return DataLoader()

       def build_validation_data_loader(self) -> DataLoader:
           return DataLoader()

       def train_batch(self, batch: TorchData, epoch_idx: int, batch_idx: int)  -> Dict[str, Any]:
           return {}

       def evaluate_batch(self, batch: TorchData) -> Dict[str, Any]:
           return {}

To learn more about the PyTorch API, you can start by reading the trial definitions from the
following examples:

-  :download:`cifar10_pytorch.tgz </examples/cifar10_pytorch.tgz>`
-  :download:`mnist_pytorch.tgz </examples/mnist_pytorch.tgz>`
-  :download:`fasterrcnn_coco_pytorch.tgz </examples/fasterrcnn_coco_pytorch.tgz>`

For tips on debugging, see :ref:`model-debug`.

.. _pytorch-downloading-data:

Distributed Backend
===================

By default, PyTorch Trial uses Horovod as the backend. You can choose to use ``torch.distributed``
and ``DistributedDataParallel`` as your distributed backend, by following :ref:`PyTorch Distributed
Launcher <pytorch-dist-launcher>`.

Download Data
=============

.. note::

   Before continuing, read how to :ref:`prepare-data` to understand how to work with different
   sources of data.

There are two ways to download your dataset in the PyTorch API:

#. Download the data in the :ref:`startup-hook.sh <startup-hooks>`.
#. Download the data in the constructor function :meth:`~determined.pytorch.PyTorchTrial.__init__`
   of :class:`~determined.pytorch.PyTorchTrial`.

If you are running a distributed training experiment, we suggest you to use the second approach.
During distributed training, a trial needs running multiple processes on different containers. In
order for all the processes to have access to the data and to prevent multiple download download
processes (one process per GPU) from conflicting with one another, the data should be downloaded to
unique directories for different ranks.

See the following code as an example:

.. code:: python

   def __init__(self, context) -> None:
       self.context = context

       # Create a unique download directory for each rank so they don't overwrite each
       # other when doing distributed training.
       self.download_directory = f"/tmp/data-rank{self.context.distributed.get_rank()}"
       self.download_directory = download_data(
          download_directory=self.download_directory,
          url=self.context.get_data_config()["url"],
       )

.. _pytorch-data-loading:

Load Data
=========

Loading data into :class:`~determined.pytorch.PyTorchTrial` models is done by defining two
functions, :meth:`~determined.pytorch.PyTorchTrial.build_training_data_loader` and
:meth:`~determined.pytorch.PyTorchTrial.build_validation_data_loader`. Each function should return
an instance of :class:`determined.pytorch.DataLoader`.

The :class:`determined.pytorch.DataLoader` class behaves the same as ``torch.utils.data.DataLoader``
and is a drop-in replacement in most cases. It handles distributed training with
:class:`~determined.pytorch.PyTorchTrial`.

Each :class:`determined.pytorch.DataLoader` will return batches of data, which will be fed directly
to the :meth:`~determined.pytorch.PyTorchTrial.train_batch` and
:meth:`~determined.pytorch.PyTorchTrial.evaluate_batch` functions. The batch size of the data loader
will be set to the per-slot batch size, which is calculated based on ``global_batch_size`` and
``slots_per_trial`` as defined in the :ref:`experiment configuration <experiment-config-reference>`.

See the following code as an example:

.. code:: python

   def build_training_data_loader(self):
       traindir = os.path.join(self.download_directory, 'train')
       self.normalize = transforms.Normalize(mean=[0.485, 0.456, 0.406],
                                   std=[0.229, 0.224, 0.225])

       train_dataset = datasets.ImageFolder(
           traindir,
           transforms.Compose([
               transforms.RandomResizedCrop(224),
               transforms.RandomHorizontalFlip(),
               transforms.ToTensor(),
               self.normalize,
           ]))

       train_loader = determined.pytorch.DataLoader(
           train_dataset,
           batch_size=self.context.get_per_slot_batch_size(),
           shuffle=True,
           num_workers=self.context.get_hparam("workers", pin_memory=True),
       )
       return train_loader

The output :meth:`~determined.pytorch.PyTorchTrial.train_batch` returns a batch of data in one of
the following formats:

.. code:: python

   # A numpy array
   batch: np.ndarray = np.array([0, 0], [0, 0]])
   # A PyTorch tensor
   batch: torch.Tensor = torch.Tensor([[0, 0], [0, 0]])
   # A tuple of arrays or tensors
   batch: Tuple[np.ndarray] = (np.array([0, 0]), np.array([0, 0]))
   batch: Tuple[torch.Tensor] = (torch.Tensor([0, 0]), torch.Tensor([0, 0]))
   # A list of arrays or tensors
   batch: List[np.ndarray] = [np.array([0, 0]), np.array([0, 0])]
   batch: List[torch.Tensor] = [torch.Tensor([0, 0]), torch.Tensor([0, 0])]
   # A dictionary mapping strings to arrays or tensors
   batch: Dict[str, np.ndarray] = {"data": np.array([0, 0]), "label": np.array([0, 0])}
   batch: Dict[str, torch.Tensor] = {"data": torch.Tensor([0, 0]), "label": torch.Tensor([0, 0])}
   # A combination of the above
   batch = {
       "data": [
           {"sub_data1": torch.Tensor([[0, 0], [0, 0]])},
           {"sub_data2": torch.Tensor([0, 0])},
       ],
       "label": (torch.Tensor([0, 0]), torch.Tensor([[0, 0], [0, 0]])),
   }

Initialize Objects
==================

You need to initialize the objects that will be used in training in the constructor
:meth:`~determined.pytorch.PyTorchTrial.__init__` of :class:`determined.pytorch.PyTorchTrial` using
the provided ``context``: these objects include the model(s), optimizer(s), learning rate
scheduler(s), and custom loss and metric functions. See
:meth:`~determined.pytorch.PyTorchTrial.__init__` for details.

.. warning::

   Be sure to wrap your objects! You may see metrics for trials that are paused and later continued
   that are significantly different from trials that are not paused if some of your models,
   optimizers, and learning rate schedulers are not wrapped. The reason is that the model's state
   may not be restored accurately or completely from the checkpoint, which is saved to a checkpoint
   and then later loaded into the trial during resumed training. When using PyTorch, this can
   sometimes happen if the PyTorch API is not used correctly.

Optimizers
----------

You need to call the :meth:`~determined.pytorch.PyTorchTrialContext.wrap_optimizer` method of the
:class:`~determined.pytorch.PyTorchTrialContext` to wrap your instantiated optimizers in the
:meth:`~determined.pytorch.PyTorchTrial.__init__` constructor. For example,

.. code:: python

   def __init__(self, context: PyTorchTrialContext):
       self.context = context

       ...
       optimizer = torch.optim.SGD(
            self.model.parameters(),
            self.context.get_hparam("lr"),
            momentum=self.context.get_hparam("momentum"),
            weight_decay=self.context.get_hparam("weight_decay"),
        )
       self.optimizer = self.context.wrap_optimizer(optimizer)

Then you need to step your optimizer in the :meth:`~determined.pytorch.PyTorchTrial.train_batch`
(see :ref:`pytorch-optimization-step` below).

Learning Rate Schedulers
------------------------

Determined has a few ways of managing the learning rate. Determined can automatically update every
batch or epoch, or you can manage it yourself.

You need to call the :meth:`~determined.pytorch.PyTorchTrialContext.wrap_lr_scheduler` method of the
:class:`~determined.pytorch.PyTorchTrialContext` to wrap your instantiated learning rate schedulers
in the :meth:`~determined.pytorch.PyTorchTrial.__init__` constructor. For example,

.. code:: python

   def __init__(self, context: PyTorchTrialContext):
       self.context = context

       ...
       lr_sch = torch.optim.lr_scheduler.StepLR(self.optimizer, gamma=.1, step_size=2)
       self.lr_sch = self.context.wrap_lr_scheduler(
           lr_sch,
           step_mode=LRScheduler.StepMode.STEP_EVERY_EPOCH,
       )

If your learning rate scheduler uses the manual step mode, you will need to step your learning rate
scheduler in the :meth:`~determined.pytorch.PyTorchTrial.train_batch` method of
:class:`~determined.pytorch.PyTorchTrial` by calling:

.. code:: python

   def train_batch(self, batch: pytorch.TorchData, epoch_idx: int, batch_idx: int)
       ...
       self.lr_sch.step()
       ...

Define the Training Loop
========================

.. _pytorch-optimization-step:

Optimization Step
-----------------

You need to implement the :meth:`~determined.pytorch.PyTorchTrial.train_batch` method of your
``PyTorchTrial`` subclass.

Typically when training with native PyTorch, you write a training loop, which iterates through the
dataloader to access and train your model one batch at a time. You can usually identify this code by
finding the common code snippet: ``for batch in dataloader``. In Determined,
:meth:`~determined.pytorch.PyTorchTrial.train_batch` also works with one batch at a time.

Take `this script implemented with the native PyTorch
<https://github.com/pytorch/examples/blob/master/imagenet/main.py>`_ as an example. It has the
following code for the training loop.

.. code:: python

   for i, (images, target) in enumerate(train_loader):
       # measure data loading time
       data_time.update(time.time() - end)

       # move data to the same device as model
       images = images.to(device, non_blocking=True)
       target = target.to(device, non_blocking=True)

       # compute output
       output = model(images)
       loss = criterion(output, target)

       # measure accuracy and record loss
       acc1, acc5 = accuracy(output, target, topk=(1, 5))
       losses.update(loss.item(), images.size(0))
       top1.update(acc1[0], images.size(0))
       top5.update(acc5[0], images.size(0))

       # compute gradient and do SGD step
       optimizer.zero_grad()
       loss.backward()
       optimizer.step()

       # measure elapsed time
       batch_time.update(time.time() - end)
       end = time.time()

       if i % args.print_freq == 0:
           progress.display(i + 1)

Notice that this pure-PyTorch loop manages the per-batch metrics. With Determined, metrics returned
by :meth:`~determined.pytorch.PyTorchTrial.train_batch` are automatically averaged and displayed, so
we do not need to do this ourselves.

Next, we will convert some PyTorch functions to use Determined’s equivalents. We need to change
``optimizer.zero_grad()``, ``loss.backward()``, and ``optimizer.step()``. The ``self.context``
object will be used to call ``loss.backwards`` and handle zeroing and stepping the optimizer.

The final :meth:`~determined.pytorch.PyTorchTrial.train_batch` will look like:

.. code:: python

   def train_batch(self, batch: TorchData, epoch_idx: int, batch_idx: int):
       images, target = batch
       output = self.model(images)
       loss = self.criterion(output, target)
       acc1, acc5 = self.accuracy(output, target, topk=(1, 5))

       self.context.backward(loss)
       self.context.step_optimizer(self.optimizer)

       return {"loss": loss.item(), "top1": acc1[0], "top5": acc5[0]}

Checkpointing
-------------

A checkpoint includes the model definition (Python source code), experiment configuration file,
network architecture, and the values of the model's parameters (i.e., weights) and hyperparameters.
When using a stateful optimizer during training, checkpoints will also include the state of the
optimizer (i.e., learning rate). You can also embed arbitrary metadata in checkpoints via a
:ref:`Python SDK <store-checkpoint-metadata>`.

PyTorch trials are checkpointed as a ``state-dict.pth`` file. This file is created in a similar
manner to the procedure described in the `PyTorch documentation
<https://pytorch.org/tutorials/beginner/saving_loading_models.html#saving-loading-a-general-checkpoint-for-inference-and-or-resuming-training>`__,
but instead of the fields in that documentation, the dictionary will have four keys:
``models_state_dict``, ``optimizers_state_dict``, ``lr_schedulers_state_dict``, and ``callbacks``,
which are the ``state_dict`` of the models, optimizers, LR schedulers, and callbacks respectively.

Define the Validation Loop
==========================

You need to implement either the :meth:`~determined.pytorch.PyTorchTrial.evaluate_batch` or
:meth:`~determined.pytorch.PyTorchTrial.evaluate_full_dataset` method. To load data into the
validation loop, define :meth:`~determined.pytorch.PyTorchTrial.build_validation_data_loader`. To
define reducing metrics, define :meth:`~determined.pytorch.PyTorchTrial.evaluation_reducer`.

For example,

.. code:: python

   def evaluate_batch(self, batch: TorchData):
       images, target = batch
       output = self.model(images)
       validation_loss = self.criterion(output, target)
       return {"validation_loss": loss.item()}

Callbacks
=========

To execute arbitrary Python code during the lifecycle of a
:class:`~determined.pytorch.PyTorchTrial`, implement the
:class:`~determined.pytorch.PyTorchCallback` and supply them to the
:class:`~determined.pytorch.PyTorchTrial` by implementing
:meth:`~determined.pytorch.PyTorchTrial.build_callbacks`.

Advanced Usage
==============

Gradient Clipping
-----------------

Users need to pass a gradient clipping function to
:meth:`~determined.pytorch.PyTorchTrialContext.step_optimizer`.

.. _pytorch-custom-reducers:

Reducing Metrics
----------------

Determined supports proper reduction of arbitrary training and validation metrics, even during
distributed training, by allowing users to define custom reducers. Custom reducers can be either a
function or an implementation of the :class:`determined.pytorch.MetricReducer` interface. See
:meth:`determined.pytorch.PyTorchTrialContext.wrap_reducer` for more details.

.. _pytorch-reproducible-dataset:

Customize a Reproducible Dataset
--------------------------------

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
   the number of records to skip can be reliably calculated from the number of batches already
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

Profiling
---------

Determined provides support for the native PyTorch profiler, `torch-tb-profiler
<https://github.com/pytorch/kineto/tree/main/tb_plugin>`_. You can configure this by calling
:meth:`~determined.pytorch.PyTorchTrialContext.set_profiler` from within your Trial's ``__init__``.
``set_profiler`` accepts the same arguments as the PyTorch plugin's ``torch.profiler.profile``
method. However, Determined sets ``on_trace_ready`` to the appropriate TensorBoard path, and the
stepping of the profiler during training is automatically handled.

The following example profiles CPU and GPU activities on batches 3 and 4 (skipping batch 1, warming
up on batch 2), and repeats for 2 cycles:

.. code:: python

   class MyPyTorchTrial(det.pytorch.PyTorchTrial):
       def __init__(self, context):
           context.set_profiler(
               activities=[
                   torch.profiler.ProfilerActivity.CPU,
                   torch.profiler.ProfilerActivity.CUDA,
               ],
               schedule=torch.profiler.schedule(
                   wait=1,
                   warmup=1,
                   active=2,
                   repeat=2,
               ),
           )

See the `PyTorch tensorboard profiler tutorial
<https://pytorch.org/tutorials/intermediate/tensorboard_profiler_tutorial.html#use-profiler-to-record-execution-events>`_
for a complete list of accepted configurations parameters.

Porting Checklist
=================

If you port your code to Determined, you should walk through this checklist to ensure your code does
not conflict with the Determined library.

Remove Pinned GPUs
------------------

Determined handles scheduling jobs on available slots. However, you need to let the Determined
library handles choosing the GPUs.

Take `this script <https://github.com/pytorch/examples/blob/master/imagenet/main.py>`_ as an
example. It has the following code to configure the GPU:

.. code:: python

   if args.gpu is not None:
       print("Use GPU: {} for training".format(args.gpu))

Any use of ``args.gpu`` should be removed.

Remove Distributed Training Code
--------------------------------

To run distributed training outside Determined, you need to have code that handles the logic of
launching processes, moving models to pined GPUs, sharding data, and reducing metrics. You need to
remove this code to be not conflict with the Determined library.

Take `this script <https://github.com/pytorch/examples/blob/master/imagenet/main.py>`_ as an
example. It has the following code to initialize the process group:

.. code:: python

   if args.distributed:
       if args.dist_url == "env://" and args.rank == -1:
           args.rank = int(os.environ["RANK"])
       if args.multiprocessing_distributed:
           # For multiprocessing distributed training, rank needs to be the
           # global rank among all the processes
           args.rank = args.rank * ngpus_per_node + gpu
       dist.init_process_group(backend=args.dist_backend, init_method=args.dist_url,
                               world_size=args.world_size, rank=args.rank)

This example also has the following code to set up CUDA and converts the model to a distributed one.

.. code:: python

   if not torch.cuda.is_available():
       print('using CPU, this will be slow')
   elif args.distributed:
       # For multiprocessing distributed, DistributedDataParallel constructor
       # should always set the single device scope, otherwise,
       # DistributedDataParallel will use all available devices.
       if args.gpu is not None:
           torch.cuda.set_device(args.gpu)
           model.cuda(args.gpu)
           # When using a single GPU per process and per
           # DistributedDataParallel, we need to divide the batch size
           # ourselves based on the total number of GPUs we have
           args.batch_size = int(args.batch_size / ngpus_per_node)
           args.workers = int((args.workers + ngpus_per_node - 1) / ngpus_per_node)
           model = torch.nn.parallel.DistributedDataParallel(model, device_ids=[args.gpu])
       else:
           model.cuda()
           # DistributedDataParallel will divide and allocate batch_size to all
           # available GPUs if device_ids are not set
           model = torch.nn.parallel.DistributedDataParallel(model)
   elif args.gpu is not None:
       torch.cuda.set_device(args.gpu)
       model = model.cuda(args.gpu)
   else:
       # DataParallel will divide and allocate batch_size to all available GPUs
       if args.arch.startswith('alexnet') or args.arch.startswith('vgg'):
           model.features = torch.nn.DataParallel(model.features)
           model.cuda()
       else:
           model = torch.nn.DataParallel(model).cuda()

This code is unnecessary in the trial definition. When we create the model, we will wrap it with
``self.context.wrap_model(model)``, which will convert the model to distributed if needed. We will
also automatically set up horovod for you. If you would like to access the rank (typically used to
view per GPU training), you can get it by calling ``self.context.distributed.rank``.

To handle data loading in distributed training, this example has the code below:

.. code:: python

   traindir = os.path.join(args.data, 'train')
   valdir = os.path.join(args.data, 'val')
   normalize = transforms.Normalize(mean=[0.485, 0.456, 0.406],
                                   std=[0.229, 0.224, 0.225])

   train_dataset = datasets.ImageFolder(
       traindir,
       transforms.Compose([
           transforms.RandomResizedCrop(224),
           transforms.RandomHorizontalFlip(),
           transforms.ToTensor(),
           normalize,
       ]))

   # Handle distributed sampler for distributed training.
   if args.distributed:
       train_sampler = torch.utils.data.distributed.DistributedSampler(train_dataset)
   else:
       train_sampler = None

This should be removed since we will use distributed data loader if you following the instructions
of :meth:`~determined.pytorch.PyTorchTrial.build_training_data_loader` and
:meth:`~determined.pytorch.PyTorchTrial.build_validation_data_loader`.

Get Hyperparameters from PyTorchTrialContext
--------------------------------------------

Take the following code for example.

.. code:: python

   def __init__(self, context: PyTorchTrialContext):
       self.context = context
       if args.pretrained:
           print("=> using pre-trained model '{}'".format(args.arch))
           model = models.__dict__[args.arch](pretrained=True)
       else:
           print("=> creating model '{}'".format(args.arch))
           model = models.__dict__[args.arch]()

``args.arch`` is a hyperparameter. You should define the hyperparameter space in the
:ref:`experiment config <experiment-config-reference>`. By doing so, you get better tracking in the
WebUI, especially for experiments that use a searcher. Depending on how your trial is run, you can
access all the current hyperparameters from inside the trial by either calling
``self.context.get_hparams()`` if you submitted your trial with ``entrypoint: model_def:Trial`` or
passing in hyperparameters directly into the Trial ``__init__`` if using PyTorch Trainer API.

.. _pytorch_trainer_ug:

*****************
 PyTorch Trainer
*****************

With the PyTorch Trainer API, you can implement and iterate on model training code locally before
running on cluster. When you are satisfied with your model code, you configure and submit the code
on cluster.

The PyTorch Trainer API lets you do the following:

-  Work locally, iterating on your model code.
-  Debug models in your favorite debug environment (e.g., directly on your machine, IDE, or Jupyter
   notebook).
-  Run training scripts without needing to use an experiment configuration file.
-  Load previously saved checkpoints directly into your model.

Initializing the Trainer
========================

After defining the PyTorch Trial, initialize the trial and the trainer.
:meth:`~determined.pytorch.init` returns a :class:`~determined.pytorch.PyTorchTrialContext` for
instantiating :class:`~determined.pytorch.PyTorchTrial`. Initialize
:class:`~determined.pytorch.Trainer` with the trial and context.

.. code:: python

   from determined import pytorch
   def main():
       with det.pytorch.init() as train_context:
           trial = MyTrial(train_context)
           trainer = det.pytorch.Trainer(trial, train_context)

   if __name__ == "__main__":
       # Configure logging
       logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)
       main()

Training is configured with a call to :meth:`~determined.pytorch.Trainer.fit` with training loop
arguments, such as checkpointing periods, validation periods, and checkpointing policy.

.. code:: diff

   from determined import pytorch


   def main():
       with det.pytorch.init() as train_context:
           trial = MyTrial(train_context)
           trainer = det.pytorch.Trainer(trial, train_context)
   +       trainer.fit(
   +           checkpoint_period=pytorch.Batch(100),
   +           validation_period=pytorch.Batch(100),
   +           checkpoint_policy="all"
   +       )


   if __name__ == "__main__":
       # Configure logging
       logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)
       main()

Run Your Training Script Locally
================================

Run training scripts locally without submitting to a cluster or defining an experiment configuration
file. Be sure to specify ``max_length`` in the ``.fit()`` call, which is used in local training mode
to determine the maximum number of steps to train for.

.. code:: python

   from determined import pytorch


   def main():
       with det.pytorch.init() as train_context:
           trial = MyTrial(train_context)
           trainer = det.pytorch.Trainer(trial, train_context)
           trainer.fit(
               max_length=pytorch.Epoch(1),
               checkpoint_period=pytorch.Batch(100),
               validation_period=pytorch.Batch(100),
               checkpoint_policy="all",
           )


   if __name__ == "__main__":
       # Configure logging
       logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)
       main()

You can run this Python script directly (``python3 train.py``), or in a Jupyter notebook. This code
will train for one epoch, and checkpoint and validate every 100 batches.

Local Distributed Training
==========================

Local training can utilize multiple GPUs on a single node with a few modifications to the above
code. Both Horovod and PyTorch Distributed backends are supported.

.. code:: diff

    def main():
   +     # Initialize distributed backend before pytorch.init()
   +     dist.init_process_group(backend="gloo|nccl")
   +     # Set flag used by internal PyTorch training loop
   +     os.environ["USE_TORCH_DISTRIBUTED"] = "true"
   +     # Initialize DistributedContext
         with det.pytorch.init(
   +       distributed=core.DistributedContext.from_torch_distributed()
         ) as train_context:
             trial = MyTrial(train_context)
             trainer = det.pytorch.Trainer(trial, train_context)
             trainer.fit(
                 max_length=pytorch.Epoch(1),
                 checkpoint_period=pytorch.Batch(100),
                 validation_period=pytorch.Batch(100),
                 checkpoint_policy="all"
             )

This code can be directly invoked with your distributed backend's launcher: ``torchrun
--nproc_per_node=4 train.py``

Test Mode
=========

Trainer accepts a test_mode parameter which, if true, trains and validates your training code for
only one batch, checkpoints, then exits. This is helpful for debugging code or writing automated
tests around your model code.

.. code:: diff

    trainer.fit(
                 max_length=pytorch.Epoch(1),
                 checkpoint_period=pytorch.Batch(100),
                 validation_period=pytorch.Batch(100),
   +             test_mode=True
             )

Prepare Your Training Code for Deploying to a Determined Cluster
================================================================

Once you are satisfied with the results of training the model locally, you submit the code to a
cluster. This example allows for distributed training locally and on cluster without having to make
code changes.

Example workflow of frequent iterations between local debugging and cluster deployment:

.. code:: diff

    def main():
   +   local = det.get_cluster_info() is None
   +   if local:
   +       # Local: configure local distributed training.
   +       dist.init_process_group(backend="gloo|nccl")
   +       os.environ["USE_TORCH_DISTRIBUTED"] = "true"
   +       distributed_context = core.DistributedContext.from_torch_distributed()
   +       latest_checkpoint = None
   +   else:
   +       # On-cluster: Determined will automatically detect distributed context.
   +       distributed_context = None
   +       # On-cluster: configure the latest checkpoint for pause/resume training functionality.
   +       latest_checkpoint = det.get_cluster_info().latest_checkpoint

   +     with det.pytorch.init(
   +       distributed=distributed_context
         ) as train_context:
             trial = MNistTrial(train_context)
             trainer = det.pytorch.Trainer(trial, train_context)
             trainer.fit(
                 max_length=pytorch.Epoch(1),
                 checkpoint_period=pytorch.Batch(100),
                 validation_period=pytorch.Batch(100),
   +             latest_checkpoint=latest_checkpoint,
             )

To run Trainer API solely on-cluster, the code is much simpler:

.. code:: python

   def main():
       with det.pytorch.init() as train_context:
           trial_inst = model.MNistTrial(train_context)
           trainer = det.pytorch.Trainer(trial_inst, train_context)
           trainer.fit(
               checkpoint_period=pytorch.Batch(100),
               validation_period=pytorch.Batch(100),
               latest_checkpoint=det.get_cluster_info().latest_checkpoint,
           )

Submit Your Trial for Training on Cluster
=========================================

To run your experiment on cluster, you'll need to create an experiment configuration (YAML) file.
Your experiment configuration file must contain searcher configuration and entrypoint.

.. code:: python

   name: pytorch_trainer_trial
   searcher:
     name: single
     metric: validation_loss
     max_length:
       epochs: 1
   resources:
     slots_per_trial: 8
   entrypoint: python3 -m determined.launch.torch_distributed python3 train.py

Submit the trial to the cluster:

.. code:: bash

   det e create det.yaml .

If your training code needs to read some values from the experiment configuration,
``pytorch.init()`` accepts an ``exp_conf`` argument which allows calling
``context.get_experiment_config()`` from ``PyTorchTrialContext``.

Loading Checkpoints
===================

To load a checkpoint from a checkpoint saved using Trainer, you'll need to download the checkpoint
to a file directory and use :func:`determined.pytorch.load_trial_from_checkpoint_path`. If your
``Trial`` was instantiated with arguments, you can pass them via the ``trial_kwargs`` parameter of
``load_trial_from_checkpoint_path``.
