#############
 PyTorch API
#############

**********
 Overview
**********

This document guides you through training a PyTorch model in Determined. You need to implement a
trial class that inherits :class:`~determined.pytorch.PyTorchTrial` and specify it as the entrypoint
in the :doc:`experiment configuration </training-apis/experiment-config>`.

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
iteration. For more debugging tips, check out the document :doc:`/training-debug/index`.

To learn about this API, you can start by reading the trial definitions from the following examples:

-  :download:`cifar10_pytorch.tgz </examples/cifar10_pytorch.tgz>`
-  :download:`mnist_pytorch.tgz </examples/mnist_pytorch.tgz>`
-  :download:`fasterrcnn_coco_pytorch.tgz </examples/fasterrcnn_coco_pytorch.tgz>`

.. _pytorch-downloading-data:

******************
 Downloading Data
******************

.. note::

   Before loading data, read this document :doc:`/prepare-data/index` to understand how to work with
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
 Loading Data
**************

.. note::

   Before loading data, read this document :doc:`/prepare-data/index` to understand how to work with
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
</training-apis/experiment-config>`.

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
 Defining Training Loop
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

**************************
 Defining Validation Loop
**************************

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

.. toctree::
   :maxdepth: 1
   :hidden:

   api-pytorch-advanced
   api-pytorch-porting
   api-pytorch-reference
