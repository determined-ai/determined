.. _deepspeed-api:

#############
 Usage Guide
#############

.. meta::
   :description: This comprehensive guide explains the DeepSpeed API and provides step-by-step instructions for getting started. Learn how to use the DeepSpeed API by training a PyTorch model with the DeepSpeed engine.

This usage guide introduces DeepSpeed and guides you through how to train a PyTorch model with the
DeepSpeed engine. To implement :class:`~determined.pytorch.deepspeed.DeepSpeedTrial`, you need to
overwrite specific functions corresponding to common training aspects. It is helpful to work from a
skeleton trial to keep track of what is required, as the following example template shows:

.. code:: python

   from typing import Any, Dict, Iterator, Optional,  Union
   from attrdict import AttrDict

   import torch
   import deepspeed

   import determined.pytorch import DataLoader, TorchData
   from determined.pytorch.deepspeed import DeepSpeedTrial, DeepSpeedTrialContext

   class MyTrial(DeepSpeedTrial):
       def __init__(self, context: DeepSpeedTrialContext) -> None:
           self.context = context
           self.args = AttrDict(self.context.get_hparams())

       def build_training_data_loader(self) -> DataLoader:
           return DataLoader()

       def build_validation_data_loader(self) -> DataLoader:
           return DataLoader()

       def train_batch(
           self,
           dataloader_iter: Optional[Iterator[TorchData]],
           epoch_idx: int,
           batch_idx: int,
       ) -> Union[torch.Tensor, Dict[str, Any]]:
           return {}

       def evaluate_batch(
           self, dataloader_iter: Optional[Iterator[TorchData]], batch_idx: int
       ) -> Dict[str, Any]:
           return {}

The DeepSpeed API organizes training routines into common steps like creating the data loaders and
training and evaluating the model. The provided template shows the function signatures, including
the expected return types, for these methods.

Because DeepSpeed is built on top of PyTorch, there are many similarities between the API for
:class:`~determined.pytorch.PyTorchTrial` and :class:`~determined.pytorch.deepspeed.DeepSpeedTrial`.
The following steps show you how to implement each of the
:class:`~determined.pytorch.deepspeed.DeepSpeedTrial` methods beginning with training objects
initialization.

***************************************************
 Step 1- Configure and Initialize Training Objects
***************************************************

DeepSpeed training initialization consists of two steps:

#. Initialize the distributed backend.
#. Create the DeepSpeed model engine.

Refer to the `DeepSpeed Getting Started guide
<https://www.deepspeed.ai/getting-started/#writing-deepspeed-models/>`_ for more information.

Outside of Determined, this is typically done in the following way:

.. code:: python

   deepspeed.init_distributed(dist_backend=args.backend)
   net = ...
   model_engine, optimizer, lr_scheduler, _ = deepspeed.initialize(args=args, net=net, ...)

:class:`~determined.pytorch.deepspeed.DeepSpeedTrial` automatically initializes the distributed
training backend so all you need to do is initialize the model engine and other training objects in
the :class:`~determined.pytorch.deepspeed.DeepSpeedTrial`
:meth:`~determined.pytorch.deepspeed.DeepSpeedTrial.__init__` method.

Configuration
=============

DeepSpeed behavior during training is configured by passing arguments when initializing the model
engine. This can be done in two ways:

-  Using a configuration file specified as an argument with a field named ``deepspeed_config``.
-  Using a dictionary, which is passed in directly when initializing a model engine.

Both approaches can be used in combination with the Determined experiment configuration. See the
`DeepSpeed documentation <https://www.deepspeed.ai/docs/config-json/>`_ for more information on what
can be specified in the configuration.

If you want to use a DeepSpeed configuration file, the hyperparameters section can be used as
arguments to pass to ``deepspeed.initialize``. For example, if the DeepSpeed configuration file is
named ``ds_config.json``, the hyperparameter section of the Determined experiment configuration is:

.. code:: yaml

   hyperparameters:
     deepspeed_config: ds_config.json
     ...

If you want to overwrite some values in an existing DeepSpeed configuration file, use
:meth:`~determined.pytorch.deepspeed.overwrite_deepspeed_config` and an experiment configuration
similar to:

.. code:: yaml

   hyperparameters:
     deepspeed_config: ds_config.json
     overwrite_deepspeed_args:
         train_batch_size: 16
         optimizer:
           params:
             lr: 0.005
     ...

If you want to use a dictionary directly, specify a DeepSpeed configuration dictionary in the
hyperparameters section:

.. code:: yaml

   hyperparameters:
     optimizer:
       type: Adam
       params:
         betas:
           - 0.8
           - 0.999
         eps: 1.0e-08
         lr: 0.001
         weight_decay: 3.0e-07
     train_batch_size: 16
     zero_optimization:
       stage: 0
       allgather_bucket_size: 50000000
       allgather_partitions: true
       contiguous_gradients: true
       cpu_offload: false
       overlap_comm: true
       reduce_bucket_size: 50000000
       reduce_scatter: true

Initialization
==============

After configuration, you can initialize the model engine in the
:class:`~determined.pytorch.deepspeed.DeepSpeedTrial`. The following example corresponds to the
experiment configuration above, with a field in the ``hyperparameters`` section named
``overwrite_deepspeed_args``.

.. code:: python

   class MyTrial(DeepSpeedTrial):
       def __init__(self, context: DeepSpeedTrialContext) -> None:
           self.context = context
           self.args = AttrDict(self.context.get_hparams())

           model = Net(self.args)
           ds_config = overwrite_deepspeed_config(
               self.args.deepspeed_config, self.args.get("overwrite_deepspeed_args", {})
           )
           parameters = filter(lambda p: p.requires_grad, model.parameters())
           model_engine, __, __, __ = deepspeed.initialize(
               model=model, model_parameters=parameters, config=ds_config
           )
           self.model_engine = self.context.wrap_model_engine(model_engine)

After the model engine is initialized, you need to register it with Determined by calling
:meth:`~determined.pytorch.deepspeed.DeepSpeedTrialContext.wrap_model_engine`. Differing from
PyTorchTrial, you do not need to register the optimizer or learning rate scheduler with Determined
because both are attributes of the model engine.

If you want to use pipeline parallelism with a given model, pass layers of the model for
partitioning to the DeepSpeed PipelineModule before creating the pipeline model engine:

.. code:: python

   net = ...
   net = deepspeed.PipelineModule(
       layers=get_layers(net),
       loss_fn=torch.nn.CrossEntropyLoss(),
       num_stages=args.pipeline_parallel_size,
       ...,
   )

********************
 Step 2 - Load Data
********************

The next step is to build the data loader used for training and validation. The same process is used
to download the data :ref:`for PyTorchTrial <pytorch-downloading-data>`. :ref:`Building the data
loaders <pytorch-data-loading>` is also similar, except for the batch size specification for the
returned data loaders, which differs because the DeepSpeed model engines automatically handle
gradient aggregation.

Automatic gradient aggregation in DeepSpeed is specified in `configuration fields
<https://www.deepspeed.ai/docs/config-json/#batch-size-related-parameters/>`_:

-  ``train_batch_size``
-  ``train_micro_batch_size``
-  ``gradient_accumulation_steps``

which are related as follows:

.. code::

   train_batch_size = train_micro_batch_size * gradient_accumulation_steps * data_parallel_size,

where ``data_parallel_size`` is the number of model replicas across all GPUs used during training.
Therefore, a single train batch consists of multiple micro batches, specified by the
``gradient_accumulation_steps`` argument. Given a model parallelization scheme, you can specify two
fields and the third can be inferred.

The DeepSpeed model engines assume the model is processing micro batches and automatically handle
stepping the optimizer and learning rate scheduler every ``gradient_accumulation_steps`` micro
batches. This means that the ``build_training_data_loader`` should return batches of size
``train_micro_batch_size_per_gpu``. In most cases, ``build_validation_data_loader`` also returns
batches of size ``train_micro_batch_size_per_gpu``.

If you want exact epoch boundaries to be respected, the number of micro batches in the training data
loader should be divisible by ``gradient_accumulation_steps``.

If you are using pipeline parallelism, the validation data loader needs to have at least
``gradient_accumulation_steps`` worth of batches.

**********************************
 Step 3 - Training and Evaluation
**********************************

This step covers the training and evaluation routine for the standard data parallel model engine and
the pipeline parallel engine available in DeepSpeed.

After you create the DeepSpeed model engine and data loaders, define the training and evaluation
routines for the :class:`~determined.pytorch.deepspeed.DeepSpeedTrial`. Differing from
:class:`~determined.pytorch.PyTorchTrial`,
:meth:`~determined.pytorch.deepspeed.DeepSpeedTrial.train_batch` and
:meth:`~determined.pytorch.deepspeed.DeepSpeedTrial.evaluate_batch` take an iterator over the
corresponding data loader built from
:meth:`~determined.pytorch.deepspeed.DeepSpeedTrial.build_training_data_loader` and
:meth:`~determined.pytorch.deepspeed.DeepSpeedTrial.build_validation_dataloader` instead of a batch.

Data Parallel Training
======================

For data parallel training, only, the training and evaluation routines are:

.. code:: python

   def train_batch(
       self,
       dataloader_iter: Optional[Iterator[TorchData]],
       epoch_idx: int,
       batch_idx: int,
   ) -> Union[torch.Tensor, Dict[str, Any]]:
       inputs = self.context.to_device(next(dataloader_iter))
       loss = self.model_engine(inputs)
       self.model_engine.backward(loss)
       self.model_engine.step()
       return {"loss": loss}


   def evaluate_batch(
       self, dataloader_iter: Optional[Iterator[TorchData]], batch_idx: int
   ) -> Dict[str, Any]:
       inputs = self.context.to_device(next(dataloader_iter))
       loss = self.model_engine(inputs)
       return {"loss": loss}

You need to manually get a batch from the iterator and move it to the GPU using the provided
:meth:`~determined.pytorch.deepspeed.DeepSpeedTrialContext.to_device` helper function, which knows
the GPU assigned to a given distributed training process.

Pipeline Parallel Training
==========================

When using pipeline parallelism, the forward and backward steps during training are combined into a
single function call because DeepSpeed automatically interleaves multiple micro batches for
processing in a single training step. In this case, there is no need to manually get a batch from
the ``dataloader_iter`` iterator because the pipeline model engine requests it as needed while
interleaving micro batches:

.. code:: python

   def train_batch(
       self,
       dataloader_iter: Optional[Iterator[TorchData]],
       epoch_idx: int,
       batch_idx: int,
   ) -> Union[torch.Tensor, Dict[str, Any]]:
       loss = self.model_engine.train_batch()
       return {"loss": loss}


   def evaluate_batch(
       self, dataloader_iter: Optional[Iterator[TorchData]], batch_idx: int
   ) -> Dict[str, Any]:
       loss = self.model_engine.eval_batch()
       return {"loss": loss}

*****************************
 Known DeepSpeed Constraints
*****************************

Some DeepSpeed constraints are inherited concerning supported feature compatibility:

-  Pipeline parallelism can only be combined with Zero Redundancy Optimizer (ZeRO) stage 1.
-  Parameter offloading is only supported with ZeRO stage 3.
-  Optimizer offloading is only supported with ZeRO stage 2 and 3.
