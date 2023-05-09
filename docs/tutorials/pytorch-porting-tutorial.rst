##########################
 PyTorch Porting Tutorial
##########################

.. meta::
   :description: By walking through this simple example, you'll learn how to organize your PyTorch code into Determined's PyTorch Trial API.

Determined provides a high-level framework APIs for PyTorch, Keras, and Estimators that let users
describe their model without boilerplate code. Determined reduces boilerplate by providing a
state-of-the-art training loop that provides distributed training, hyperparameter search, automatic
mixed precision, reproducibility, and many more features.

In this guide, we'll walk through an example and provide helpful hints to successfully organize
PyTorch code into Determined's PyTorchTrial API. Once your code is in the PyTorchTrial format, you
can easily take advantage of Determined Ai's open-source platform.

We suggest you follow along using the source code found on `Determined's GitHub repo
<https://github.com/determined-ai/determined/tree/master/examples/tutorials/imagenet_pytorch>`_.

While all codebases are different, code to perform deep learning training tends to follow a typical
pattern. Usually, there is a model, optimizer, data, and a learning rate scheduler.
:class:`determined.pytorch.PyTorchTrial` follows this pattern to reduce porting friction. To port,
we will copy the core machine learning code based on the traditional training pieces, while deleting
the unnecessary boilerplate code. We suggest isolating these components in the following order:

#. Model
#. Optimizer
#. Data
#. Train/validate batch
#. Learning Rate Scheduler
#. Other features such as automatic mixed precision (AMP), gradient clipping, and others

We will port each section following the same steps: First, we copy all the relevant code over. Then,
we will remove boilerplate code, and update all relevant objects to use the context object. Finally,
we will replace all configurations or hyperparameters.

*************
 Preparation
*************

Before we begin, we need to create our core files. Determined requires two files to be defined: the
model definition and the experiment configuration.

Model Definition
================

The model definition contains the trial class, which contains the model definition and training
loop. Your deep learning framework defines which ``Determined.Trial()`` class to inherit. In our
case, we will be working with PyTorch, which means we will be working with
:class:`determined.pytorch.PyTorchTrial`. Once we inherit this class, we will be required to
override specific functions that represent a training script's core components. When starting a new
port, it is helpful to work off of a skeleton to keep track of what is still required. A good
starting template can be found below:

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

Experiment Configuration
========================

We also need to create an experiment configuration file. This file defines specific experiment
information such as: number of samples, training length, and hyperparameters.

We recommend that all hyperparameters be added to this file. A good starting point is taking any
command-line arguments---you may see this as parser args---from the original script and immediately
adding them to your file.

In our case, since we're using hyperparameters that don't change, we've given this specific config
the name ``const.yaml``:

.. code:: yaml

   description: ImageNet_PyTorch_const
   hyperparameters:
       global_batch_size: 256
       dense1: 128
       data: /mnt/data
       arch: resnet18
       workers: 4
       start-epoch: 0
       lr: 0.1
       momentum: 0.9
       weight_decay: 1e-4
       pretrained: True
   records_per_epoch: 60000
   searcher:
       name: single
       metric: val_loss
       smaller_is_better: false
       max_length:
           epochs: 10
   entrypoint: model_def:ImageNetTrial
   max_restarts: 0

For now, we don't have to worry much about the other fields; however, we suggest setting
``max_restarts`` to zero so Determined will not retry running the experiment. For more information
on experiment configuration, see the :ref:`experiment configuration reference
<experiment-configuration>`.

*******
 Model
*******

Now that we've finished the prep work, we can begin porting by creating the model. Model code will
be placed in the Trial's ``__init__()`` function.

To refresh, as we work on the model, we want to follow this checklist:

-  Remove boilerplate code.
-  Copy all relevant code over.
-  Update all relevant objects to use the context object.
-  Replace all configurations or hyperparameters.

Remove Boilerplate Code and Copy All Relevant Code
==================================================

Based on the checklist, we want to first remove all boilerplate code, such as code related to
distributed training or device management. As we remove boilerplate code, we can immediately copy
relevant code, such as the model creation, to our PyTorchTrial. In the `original script
<https://github.com/pytorch/examples/blob/master/imagenet/main.py>`_ most of the model code is found
in lines 119-168, where it defines the model and sets up the GPU and script for distributed
training. Since Determined handles much of this logic, we can remove a lot of this as boilerplate.

Let's work through these lines:

.. code:: python

   if args.gpu is not None:
       print("Use GPU: {} for training".format(args.gpu))

In the experiment configuration file, we define the number of resources, usually GPUs; therefore, we
can omit this from the model definition.

.. code:: python

   if args.distributed:
       if args.dist_url == "env://" and args.rank == -1:
           args.rank = int(os.environ["RANK"])
       if args.multiprocessing_distributed:
           # For multiprocessing distributed training, rank needs to be the
           # global rank among all the processes.
           args.rank = args.rank * ngpus_per_node + gpu
       dist.init_process_group(
           backend=args.dist_backend,
           init_method=args.dist_url,
           world_size=args.world_size,
           rank=args.rank,
       )

Determined will automatically set up horovod for the user. If you would like to access the rank
(typically used to view per GPU training), you can get it by calling
``self.context.distributed.rank``.

.. code:: python

   if args.pretrained:
       print("=> using pre-trained model '{}'".format(args.arch))
       model = models.__dict__[args.arch](pretrained=True)
   else:
       print("=> creating model '{}'".format(args.arch))
       model = models.__dict__[args.arch]()

Here is where we actually define the model. We will copy and paste this code directly into our
``__init__`` function. For now, we can leave it as it was copied, but we will return to update this
code a bit later on.

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

This snippet sets up CUDA and converts it to a distributed model. Because Determined handles
distributed training automatically, this code is unnecessary. When we create the model, we will wrap
it with ``self.context.wrap_model(model)``, which will convert the model to distributed if needed.

Update Objects and Replace HP Configurations
============================================

We have copied all relevant code over and now need to clean it up. Our current ``__init__`` function
looks something like this:

.. code:: python

   def __init__(self, context: PyTorchTrialContext):
       self.context = context
       if args.pretrained:
           print("=> using pre-trained model '{}'".format(args.arch))
           model = models.__dict__[args.arch](pretrained=True)
       else:
           print("=> creating model '{}'".format(args.arch))
           model = models.__dict__[args.arch]()

First, we update all references to the parser arguments. Everywhere args are used will be changed to
``self.context.get_hparams()``. This function will give you access to all the hyperparameters in the
experiment configuration file. By converting the hyperparameters to be accessed in the experiment
configuration, it allows for better experiment tracking, and makes it easier to quickly run a
searcher experiment.

Finally, we need to wrap our model, so Determined can handle all the distributed training code we
previously removed:

.. code:: python

   self.model = self.context.wrap_model(model)

After all of these changes, we are left with the code below:

.. code:: python

   def __init__(self, context: PyTorchTrialContext):
       self.context = context

       arch = self.context.get_hparam("arch")
       if self.context.get_hparam("pretrained"):
           print("=> using pre-trained model '{}'".format(arch))
           model = models.__dict__[arch](pretrained=True)
       else:
           print("=> creating model '{}'".format(arch))
           model = models.__dict__[arch]()

       self.model = self.context.wrap_model(model)

Optimizer/Loss
==============

Next, we will port the optimizer and loss functions. The optimizer and loss will be placed in the
__init__() function.

Remove Boilerplate Code and Copy All Relevant Code
==================================================

Once again, we copy the relevant optimizer and loss definitions. In the original model, the
optimizer is defined with one line, which we copy over directly:

.. code:: python

   optimizer = torch.optim.SGD(model.parameters(), args.lr,
                               momentum=args.momentum,
                               weight_decay=args.weight_decay)

For the loss, this example uses ``CrossEntropyLoss()``. This can be added to PyTorchTrial with one
line.

.. code:: python

   self.criterion = nn.CrossEntropyLoss()

Update Objects and Replace HP Configurations
============================================

Now we update the arguments to reference the experiment configuration.

.. code:: python

   optimizer = torch.optim.SGD(self.model.parameters(), self.context.get_hparam("lr"), momentum=self.context.get_hparam("momentum"), weight_decay=self.context.get_hparam("weight_decay"))
   self.optimizer = self.context.wrap_optimizer(optimizer)

You may notice ``self.context.get_hparams()`` can become long. A simple trick is to set
``self.context.get_hparams`` to ``self.hparams``. Then you can use ``self.hparams[“variable”]``.

The init function should now look something like this.

.. code:: python

   def __init__(self, context: PyTorchTrialContext):
       self.context = context

       arch = self.context.get_hparam("arch")
       if self.context.get_hparam("pretrained"):
           print("=> using pre-trained model '{}'".format(arch))
           model = models.__dict__[arch](pretrained=True)
       else:
           print("=> creating model '{}'".format(arch))
           model = models.__dict__[arch]()

       self.model = self.context.wrap_model(model)

       optimizer = torch.optim.SGD(self.model.parameters(), self.context.get_hparam("lr"), momentum=self.context.get_hparam("momentum"), weight_decay=self.context.get_hparam("weight_decay"))
       self.optimizer = self.context.wrap_optimizer(optimizer)

       self.criterion = nn.CrossEntropyLoss()

We have been able to remove over 80 lines of code by porting to Determined!

******
 Data
******

Now, we can fill out ``build_train_data_loader()`` and ``build_validation_data_loader()``. Both of
these data loading functions return a ``determined.DataLoader``. A ``determined.DataLoader`` expects
the same parameters as a ``torch.DataLoader`` and will handle distributed training setup.

The original script handles the data in lines 202 - 233. For the data loaders, we follow the same
procedure for porting.

Remove Boilerplate Code and Copy All Relevant Code
==================================================

In the original code, the data is loaded based on the path, and prepared for distributed training as
seen below:

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
   if args.distributed:
       train_sampler = torch.utils.data.distributed.DistributedSampler(train_dataset)
   else:
       train_sampler = None

We will bring all the code over except the ``if args.distribued`` clause since Determined will
automatically do the right thing when running a distributed training job.

Update Objects and Replace HP Configurations
============================================

There are a few pieces that need to be changed. First, the data location should be set to a class
variable: self.download_directory. During distributed training, the data should be downloaded to
unique directories based on rank to prevent multiple download processes (one process per GPU) from
conflicting with one another. This root directory will be defined based on self.hparams and will
point to where the data is stored within the Docker container. If you want to learn more about how
to access data with Determined, check out our documentation.

We also update the ``torch.Dataloader`` to be a ``determined.pytorch.DataLoader``. The batch_size
will be set to ``self.context.get_per_slot_batch_size()``. We set ``batch_size`` to
``self.context.get_per_slot_batch_size()`` which automatically calculates the per-gpu batch size
based on ``global_batch_size`` and ``slots_per_trial`` as defined in the experiment configuration.
By using ``self.context.get_per_slot_batch_size()``, Determined will assign the appropriate per GPU
batch size.

The train function will look something like this:

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

In this example, we are using ImageNet as the dataset. If you do not have access to the dataset, the
CIFAR-10 dataset can be accessed with the code below:

.. code:: python

   def build_training_data_loader(self):
       transform = transforms.Compose(
           [transforms.ToTensor(), transforms.Normalize((0.5, 0.5, 0.5), (0.5, 0.5, 0.5))]
       )
       trainset = torchvision.datasets.CIFAR10(
           root=self.download_directory, train=True, download=True, transform=transform
       )
       return DataLoader(trainset, batch_size=self.context.get_per_slot_batch_size())

**************************
 Train / Validation Batch
**************************

It's time to set up the ``train_batch`` function. Typically in PyTorch, you loop through the
DataLoader to access and train your model one batch at a time. You can usually identify this code by
finding the common code snippet: ``for batch in dataloader``. In Determined, ``train_batch()`` also
provides one batch at a time, so we can copy the code directly into our function.

Remove Boilerplate Code and Copy All Relevant Code
==================================================

In the original implementation, we find the core training loop.

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
displays the metrics returned in ``train_batch`` allowing us to remove print frequency code and the
metric arrays.

Update Objects and Replace HP Configurations
============================================

Now, we will convert some PyTorch functions to now use Determined's equivalent. We need to change
``loss.backward()``, ``optim.zero_grad()``, and ``optim.step()``. The ``self.context`` object will
be used to call ``loss.backwards`` and handle zeroing and stepping the optimizer. We update these
functions respectively:

.. code:: python

   self.context.backward(loss)
   self.context.step_optimizer(self.optimizer)

The final ``train_batch`` will look like:

.. code:: python

   def train_batch(self, batch: TorchData, epoch_idx: int, batch_idx: int):
       images, target = batch
       output = self.model(images)
       loss = self.criterion(output, target)
       acc1, acc5 = self.accuracy(output, target, topk=(1, 5))

       self.context.backward(loss)
       self.context.step_optimizer(self.optimizer)

       return {"loss": loss.item(), 'top1': acc1[0], 'top5': acc5[0]}

******************
 Code Check Point
******************

At this point, you should be able to run your Determined model. Confirm that your model weights are
loaded correctly, it can functionally run a batch, and all your hyperparameters are correctly
accessing experiment configuration.

*************************
 Learning Rate Scheduler
*************************

Determined has a few ways of managing the learning rate. Determined can automatically update every
batch or epoch, or you can manage it yourself. In this case, we are doing the latter by using a
custom function to handle the learning rate adjustment. We define it in the ``__init__()`` function
and wrap it with ``self.context.wrap_lr_scheduler``.

Next, we call the function in ``train_batch()``. Since our model runs, we can also print the
learning rate per batch or epoch to confirm the accuracy. In this case, we will update the learning
rate to use ``torch.optim.StepLR()`` and wrap it with ``self.context.wrap_lr_scheduler``.

.. code:: python

   def __init__(self, context):
       ...
       lr_sch = torch.optim.lr_scheduler.StepLR(self.optimizer, gamma=0.1, step_size=2)
       self.lr_sch = self.context.wrap_lr_scheduler(
           lr_sch, step_mode=LRScheduler.StepMode.STEP_EVERY_EPOCH
       )

*********************
 Other Functionality
*********************

At this point, you can begin adding other features of your model. This may include using 16 FP
(automatic mixed precision) or gradient clipping. It's best to add one at a time to make it easier
to check that each component is properly working. Determined has a wide range of examples to
demonstrate several real-world use cases. Examples can be found on Determined's GitHub account.

***************
 Helpful Hints
***************

During porting, most of the time you can remove distributed training code.

If you are having trouble porting your model and would like to debug it prior to finishing the rest
of the code, you can use fake data in the data loader. This lets you run and test other parts of the
``model_def.py``.

Sometimes it's useful just getting an "ugly" version of the code. This is where you first directly
place the original code in the right function without updating any pieces.

Saving the extra model "features" until later helps you ensure the core functions are correct. This
makes it easier to debug other portions of the script.

For more debugging tips, check out the how-to guide on :ref:`model debugging <model-debug>`.
