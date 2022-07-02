########################
 PyTorch API
########################

+---------------------------------------------------------------------+
| API reference                                                       |
+=====================================================================+
| :doc:`/reference/reference-training/training/api-pytorch-reference` |
+---------------------------------------------------------------------+

This document guides you through training a PyTorch model in Determined. You need to implement a
trial class that inherits :class:`~determined.pytorch.PyTorchTrial` and specify it as the entrypoint
in the :doc:`experiment configuration </reference/reference-training/experiment-config-reference>`.

To implement :class:`~determined.pytorch.PyTorchTrial`, you need to override specific functions that
represent the components that are used in the training procedure. It is helpful to work off of a
skeleton to keep track of what is still required. A good starting template can be found below:

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

If you want to port training code that defines the training procedure and can already run outside
Determined, we suggest you read through the whole document to ensure you understand the API. Also,
we suggest you use a couple of PyTorch API features at one time and running the code will help
debug. You can also use fake data to test your training code with PyTorch API to get quicker
iteration. For more debugging tips, see :doc:`/training/best-practices/debug-models`.

To learn about this API, you can start by reading the trial definitions from the following examples:

-  :download:`cifar10_pytorch.tgz </examples/cifar10_pytorch.tgz>`
-  :download:`mnist_pytorch.tgz </examples/mnist_pytorch.tgz>`
-  :download:`fasterrcnn_coco_pytorch.tgz </examples/fasterrcnn_coco_pytorch.tgz>`

.. _pytorch-downloading-data:

******************
 Download Data
******************

.. note::

   Before loading data, read this document :doc:`/training/load-model-data` to understand how to work with
   different sources of data.

There are two ways to download your dataset in the PyTorch API:

#. Download the data in the :ref:`startup-hook.sh <startup-hooks>`.
#. Download the data in the constructor function :meth:`~determined.pytorch.PyTorchTrial.__init__`
   of :class:`~determined.pytorch.PyTorchTrial`.

If you run a distributed training experiment, we suggest you to use the second approach. During
distributed training, a trial needs running multiple processes on different containers. In order for
all the processes to have access to the data and prevent multiple download download processes (one
process per GPU) from conflicting with one another, the data should be downloaded to unique
directories on different ranks. See the following code example:

..
   code: python

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

**************
 Load Data
**************

.. note::

   Before loading data, read this document :doc:`/training/load-model-data` to understand how to work with
   different sources of data.

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
``slots_per_trial`` as defined in the :doc:`experiment configuration
</reference/reference-training/experiment-config-reference>`.

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
           train_dataset, batch_size=self.context.get_per_slot_batch_size(), shuffle=True,num_workers=self.context.get_hparam("workers", pin_memory=True))
       return train_loader

In the function :meth:`~determined.pytorch.PyTorchTrial.train_batch` returns a batch of data in one
of the following formats:

-  ``np.ndarray``

   .. code:: python

      np.array([[0, 0], [0, 0]])

-  ``torch.Tensor``

   .. code:: python

      torch.Tensor([[0, 0], [0, 0]])

-  tuple of ``np.ndarray``\ s or ``torch.Tensor``\ s

   .. code:: python

      (torch.Tensor([0, 0]), torch.Tensor([[0, 0], [0, 0]]))

-  list of ``np.ndarray``\ s or ``torch.Tensor``\ s

   .. code:: python

      [torch.Tensor([0, 0]), torch.Tensor([[0, 0], [0, 0]])]

-  dictionary mapping strings to ``np.ndarray``\ s or ``torch.Tensor``\ s

   .. code:: python

      {"data": torch.Tensor([[0, 0], [0, 0]]), "label": torch.Tensor([[1, 1], [1, 1]])}

-  combination of the above

   .. code:: python

      {
          "data": [
              {"sub_data1": torch.Tensor([[0, 0], [0, 0]])},
              {"sub_data2": torch.Tensor([0, 0])},
          ],
          "label": (torch.Tensor([0, 0]), torch.Tensor([[0, 0], [0, 0]])),
      }

************************
 Define a Training Loop
************************

Initializing Objects
====================

You need to initialize the objects that will be used in training in the constructor function
:meth:`~determined.pytorch.PyTorchTrial.__init__` of :class:`determined.pytorch.PyTorchTrial` using
the provided ``context``. See :meth:`~determined.pytorch.PyTorchTrial.__init__` for details.

.. warning::

   You might see significantly different metrics for trials which are paused and later continued
   than trials which are not paused if some of your models, optimizers, and learning rate schedulers
   are not wrapped. The reason is that the model's state might not be restored accurately or
   completely from the checkpoint, which is saved to a checkpoint and then later loaded into the
   trial during resuming training. When using PyTorch, this can sometimes happen if the PyTorch API
   is not used correctly.

Optimization Step
=================

In this step, you need to implement :meth:`~determined.pytorch.PyTorchTrial.train_batch` function.

Typically when training with the native PyTorch, you need to write a training loop, which goes
through the data loader to access and train your model one batch at a time. You can usually identify
this code by finding the common code snippet: ``for batch in dataloader``. In Determined,
:meth:`~determined.pytorch.PyTorchTrial.train_batch` also provides one batch at a time.

Take `this script implemented with the native PyTorch
<https://github.com/pytorch/examples/blob/master/imagenet/main.py>`_ as an example. It has the
following code for the training loop.

.. code:: python

   for i, (images, target) in enumerate(train_loader):
       # measure data loading time
       data_time.update(time.time() - end)

       if args.gpu is not None:
           images = images.cuda(args.gpu, non_blocking=True)
       if torch.cuda.is_available():
           target = target.cuda(args.gpu, non_blocking=True)

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
           progress.display(i)

As you noticed above, the loop manages the per-batch metrics. Determined automatically averages and
displays the metrics returned in :meth:`~determined.pytorch.PyTorchTrial.train_batch` allowing us to
remove print frequency code and the metric arrays.

Now, we will convert some PyTorch functions to now use Determined’s equivalent. We need to change
``loss.backward()``, ``optim.zero_grad()``, and ``optim.step()``. The ``self.context`` object will
be used to call ``loss.backwards`` and handle zeroing and stepping the optimizer. We update these
functions respectively:

.. code:: python

   self.context.backward(loss)
   self.context.step_optimizer(self.optimizer)

Note that ``self.optimizer`` is initialized with
:meth:`~determined.pytorch.PyTorchTrialContext.wrap_optimizer` in the
:meth:`~determined.pytorch.PyTorchTrial.__init__`.

The final :meth:`~determined.pytorch.PyTorchTrial.train_batch` will look like:

.. code:: python

   def train_batch(self, batch: TorchData, epoch_idx: int, batch_idx: int):
       images, target = batch
       output = self.model(images)
       loss = self.criterion(output, target)
       acc1, acc5 = self.accuracy(output, target, topk=(1, 5))

       self.context.backward(loss)
       self.context.step_optimizer(self.optimizer)

       return {"loss": loss.item(), 'top1': acc1[0], 'top5': acc5[0]}

Using Optimizer
===============

You need to call the :meth:`~determined.pytorch.PyTorchTrialContext.wrap_optimizer` method of the
:class:`~determined.pytorch.PyTorchTrialContext` to wrap your instantiated optimizers in the
:meth:`~determined.pytorch.PyTorchTrial.__init__` function. For example,

.. code:: python

   def __init__(self, context: PyTorchTrialContext):
       self.context = context

       optimizer = torch.optim.SGD(
            self.model.parameters(),
            self.context.get_hparam("lr"),
            momentum=self.context.get_hparam("momentum"),
            weight_decay=self.context.get_hparam("weight_decay"),
        )
       self.optimizer = self.context.wrap_optimizer(optimizer)

Then you need to step your optimizer in the :meth:`~determined.pytorch.PyTorchTrial.train_batch`
method of :class:`~determined.pytorch.PyTorchTrial`.

Using Learning Rate Scheduler
=============================

Determined has a few ways of managing the learning rate. Determined can automatically update every
batch or epoch, or you can manage it yourself.

You need to call the :meth:`~determined.pytorch.PyTorchTrialContext.wrap_lr_scheduler` method of the
:class:`~determined.pytorch.PyTorchTrialContext` to wrap your instantiated learning rate schedulers
in the :meth:`~determined.pytorch.PyTorchTrial.__init__` function. For example,

.. code:: python

   def __init__(self, context: PyTorchTrialContext):
       self.context = context

       ...
       lr_sch = torch.optim.lr_scheduler.StepLR(self.optimizer, gamma=.1, step_size=2)
       self.lr_sch = self.context.wrap_lr_scheduler(lr_sch, step_mode=LRScheduler.StepMode.STEP_EVERY_EPOCH)

If your learning rate scheduler uses manual step mode, you will need to step your learning rate
scheduler in the :meth:`~determined.pytorch.PyTorchTrial.train_batch` method of
:class:`~determined.pytorch.PyTorchTrial` by calling:

.. code:: python

   def train_batch(self, batch: pytorch.TorchData, epoch_idx: int, batch_idx: int)
       ...

       self.lr_sch.step()

       ...

Checkpointing
=============

A checkpoint includes the model definition (Python source code), experiment configuration file,
network architecture, and the values of the model's parameters (i.e., weights) and hyperparameters.
When using a stateful optimizer during training, checkpoints will also include the state of the
optimizer (i.e., learning rate). Users can also embed arbitrary metadata in checkpoints via a
:ref:`Python API <store-checkpoint-metadata>`.

PyTorch trials are checkpointed as a ``state-dict.pth`` file. This file is created in a similar
manner to the procedure described in the `PyTorch documentation
<https://pytorch.org/tutorials/beginner/saving_loading_models.html#saving-loading-a-general-checkpoint-for-inference-and-or-resuming-training>`__.
Instead of the fields in the documentation linked above, the dictionary will have four keys:
``models_state_dict``, ``optimizers_state_dict``, ``lr_schedulers_state_dict``, and ``callbacks``,
which are the ``state_dict`` of the models, optimizers, LR schedulers, and callbacks respectively.

****************************
 Define the Validation Loop
****************************

You need to implement :meth:`~determined.pytorch.PyTorchTrial.evaluate_batch` or
:meth:`~determined.pytorch.PyTorchTrial.evaluate_full_dataset`. To load data into the validation
loop define :meth:`~determined.pytorch.PyTorchTrial.build_validation_data_loader`. To define
reducing metrics, define :meth:`~determined.pytorch.PyTorchTrial.evaluation_reducer`.

***********
 Callbacks
***********

To execute arbitrary Python code during the lifecycle of a
:class:`~determined.pytorch.PyTorchTrial`, implement the
:class:`~determined.pytorch.PyTorchCallback` and supply them to the
:class:`~determined.pytorch.PyTorchTrial` by implementing
:meth:`~determined.pytorch.PyTorchTrial.build_callbacks`.

****************
 Advanced Usage
****************

Gradient Clipping
=================

Users need to pass a gradient clipping function to
:meth:`~determined.pytorch.PyTorchTrialContext.step_optimizer`.

.. _pytorch-custom-reducers:

Reducing Metrics
================

Determined supports proper reduction of arbitrary training and validation metrics, even during
distributed training, by allowing users to define custom reducers. Custom reducers can be either a
function or an implementation of the :class:`determined.pytorch.MetricReducer` interface. See
:meth:`determined.pytorch.PyTorchTrialContext.wrap_reducer` for more details.

.. _pytorch-reproducible-dataset:

Customize a Reproducible Dataset
==================================

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

*******************
 Porting Checklist
*******************

If you port your code to Determined, you should walk through this checklist to ensure your code does
not conflict with the Determined library.

Remove Pinned GPUs
=====================

Determined handles scheduling jobs on available slots. However, you need to let the Determined
library handles choosing the GPUs.

Take `this script <https://github.com/pytorch/examples/blob/master/imagenet/main.py>`_ as an
example. It has the following code to configure the GPU:

.. code:: python

   if args.gpu is not None:
       print("Use GPU: {} for training".format(args.gpu))

Any use of ``args.gpu`` should be removed.

Remove Distributed Training Code
==================================

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
================================================

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
:doc:`experiment configuration </reference/reference-training/experiment-config-reference>` and use
``self.context.get_hparams()``, which gives you access to all the hyperparameters for the current
trial. By doing so, you get better tracking in the WebUI, especially for experiments that use a
searcher.
