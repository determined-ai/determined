.. _deepspeed-advanced:

################
 Advanced Usage
################

.. meta::
   :description: This article covers advanced usage of the DeepSpeed API including model parallelism and pipeline parallelism, gradient accumulation, zero-offloading, and more.

*********************************
 Training Multiple Model Engines
*********************************

If the model engines use the same :class:`~determined.pytorch.deepspeed.ModelParallelUnit`, you can
train multiple model engines in a single :class:`~determined.pytorch.deepspeed.DeepSpeedTrial` by
calling :meth:`~determined.pytorch.deepspeed.DeepSpeedTrialContext.wrap_model_engine` on additional
model engines you want to use, and by modifying
:meth:`~determined.pytorch.deepspeed.DeepSpeedTrial.train_batch` and
:meth:`~determined.pytorch.deepspeed.DeepSpeedTrial.evaluate_batch` accordingly.

The accounting for number of samples is with respect to the ``train_batch_size`` for the first model
engine passed to :meth:`~determined.pytorch.deepspeed.DeepSpeedTrialContext.wrap_model_engine`.

For more advanced cases where model engines have different model parallel topologies, contact
support on the Determined `community Slack
<https://join.slack.com/t/determined-community/shared_invite/zt-cnj7802v-KcVbaUrIzQOwmkmY7gP0Ew/>`_.

*****************
 Custom Reducers
*****************

Determined supports arbitrary training and validation metrics reduction, including during
distributed training, by letting you define custom reducers. Custom reducers can be a function or an
implementation of the :class:`determined.pytorch.MetricReducer` interface. See
:meth:`determined.pytorch.PyTorchTrialContext.wrap_reducer` for more information.

*******************************************
 Manual Distributed Backend Initialization
*******************************************

By default, :class:`~determined.pytorch.deepspeed.DeepSpeedTrial` initializes the distributed
backend by calling ``deepspeed.init_distributed`` before a trial is created. This initializes the
``torch.distributed`` backend to use the NVIDIA Collective Communications Library (NCCL). If you
want to customize the distributed backend initialization, set the ``DET_MANUAL_INIT_DISTRIBUTED``
environment variable in your experiment configuration:

.. code:: yaml

   environment:
     environment_variables:
       - DET_MANUAL_INIT_DISTRIBUTED=1

*****************************
 Manual Gradient Aggregation
*****************************

:class:`~determined.pytorch.deepspeed.DeepSpeedTrial` automatically ensures a total of
``train_batch_size`` samples are processed in each training iteration. With the assumption that
:meth:`~determined.pytorch.deepspeed.DeepSpeedTrial.train_batch` calls the model engine's forward,
backward, and optimizer step methods once, :class:`~determined.pytorch.deepspeed.DeepSpeedTrial`
calls :meth:`~determined.pytorch.deepspeed.DeepSpeedTrial.train_batch`:

-  ``gradient_accumulation_steps`` times when not using pipeline parallelism
-  once when using pipeline parallelism

to reach ``model_engine.train_batch_size()`` for the first wrapped model engine.

To disable this behavior, call
:meth:`~determined.pytorch.deepspeed.DeepSpeedTrialContext.disable_auto_grad_accumulation` in the
:meth:`~determined.pytorch.deepspeed.DeepSpeedTrial.__init__` method of
:class:`~determined.pytorch.deepspeed.DeepSpeedTrial`. In this case, make sure the first model
engine processes ``train_batch_size`` samples in each call to
:meth:`~determined.pytorch.deepspeed.DeepSpeedTrial.train_batch`.

*********************
 Custom Data Loaders
*********************

By default, :meth:`~determined.pytorch.deepspeed.DeepSpeedTrial.build_training_data_loader` and
:meth:`~determined.pytorch.deepspeed.DeepSpeedTrial.build_validation_data_loader` are expected to
return a :class:`determined.pytorch.DataLoader`, which is a thin wrapper around
``torch.utils.data.DataLoader`` that supports reproducibility and data sharding for distributed
training.

Override this requirement and return a ``torch.utils.data.DataLoader`` by setting
:meth:`~determined.pytorch.deepspeed.DeepSpeedTrialContext.disable_dataset_reproducibility_checks`.
Review :ref:`customizing a reproducible dataset <pytorch-reproducible-dataset>` for recommended best
practices when using a custom data loader.

A common use case for a custom data loader is if you created the data loader when building the model
engine as show in this example:

.. code:: python

   class MyTrial(DeepSpeedTrial):
       def __init__(self, context: DeepSpeedTrialContext) -> None:
           self.context = context
           self.args = AttrDict(self.context.get_hparams())

           training_data = ...
           model = Net(self.args)
           parameters = filter(lambda p: p.requires_grad, model.parameters())

           model_engine, __, __, self.train_dataloader = deepspeed.initialize(
               args=self.args,
               model=model,
               model_parameters=parameters,
               training_data=training_data,
           )
           self.model_engine = self.context.wrap_model_engine(model_engine)

       def build_training_data_loader(self) -> torch.utils.data.DataLoader:
           return self.train_dataloader

**************************
 Custom Model Parallelism
**************************

:class:`~determined.pytorch.deepspeed.DeepSpeedTrial` relies on a
:class:`~determined.pytorch.deepspeed.ModelParallelUnit` to provide data parallel world size and to
determine whether a GPU slot should build the data loaders and report metrics. For data parallel
training with DeepSpeed, the data parallel world size is equal to the number of GPU slots and all
GPU slots build the data loaders and report metrics. If the model engine passed to
:meth:`~determined.pytorch.deepspeed.DeepSpeedTrialContext.wrap_model_engine` is a
``PipelineEngine``, the :class:`~determined.pytorch.deepspeed.ModelParallelUnit` is built using the
MPU associated with the model engine. To change this behavior to support custom model parallelism,
pass a custom :class:`~determined.pytorch.deepspeed.ModelParallelUnit`to
:meth:`~determined.pytorch.deepspeed.DeepSpeedTrialContext.set_mpu` as shown in the following
example:

.. code:: python

   context.set_mpu(
       ModelParallelUnit(
           data_parallel_rank=[fill in],
           data_parallel_world_size=[fill in],
           should_report_metrics=[fill in],
           should_build_dataloader=[fill in]
       )
   )
