.. _torch-batch-processing-ug:

############################
 Torch Batch Processing API
############################

.. meta::
   :description: Learn how to use the Torch Batch Processing API.

In this guide, you'll learn about the :ref:`Torch Batch Process API <torch-batch-process-api-ref>`
and how to perform batch inference (also known as offline inference).

+---------------------------------------------------------------------+
| Visit the API reference                                             |
+=====================================================================+
| :ref:`torch-batch-process-api-ref`                                  |
+---------------------------------------------------------------------+

.. caution::

   This is an experimental API and may change at any time.

**********
 Overview
**********

The Torch Batch Processing API takes in (1) a dataset and (2) a user-defined processor class and
runs distributed data processing.

This API automatically handles the following for you:

-  shards a dataset by number of workers available
-  applies user-defined logic to each batch of data
-  handles synchronization between workers
-  tracks job progress to enable preemption and resumption of trial

This is a flexible API that can be used for many different tasks, including batch (offline)
inference.

If you have some trained models in a :class:~determined.experimental.checkpoint.Checkpoint or a
:class:~determined.experimental.model.Model with more than one
:class:~determined.experimental.model.ModelVersion inside, you can associate the trial with the
:class:~determined.experimental.checkpoint.Checkpoint or
:class:~determined.experimental.model.ModelVersion used in a given inference run to aggregate custom
inference metrics.

You can then query those :class:~determined.experimental.checkpoint.Checkpoint or
:class:~determined.experimental.model.ModelVersion objects using the :ref:Python SDK <python-sdk> to
see all metrics associated with them.

*******
 Usage
*******

The main arguments to :meth:`~determined.pytorch.experimental.torch_batch_process` are processor
class and dataset.

.. code:: python

   torch_batch_process(
       batch_processor_cls=MyProcessor
       dataset=dataset
   )

In the experiment config file, use a distributed launcher as the API requires information such as
rank set by the launcher. Below is an example.

.. code:: yaml

   entrypoint: >-
       python3 -m determined.launch.torch_distributed
       python3 batch_processing.py
   resources:
     slots_per_trial: 4

``TorchBatchProcessor``
=======================

During :meth:`~determined.pytorch.experimental.TorchBatchProcessor.__init__` of
:class:`~determined.pytorch.experimental.TorchBatchProcessor`, we pass in a
:class:`~determined.pytorch.experimental.TorchBatchProcessorContext` object, which contains useful
methods that can be used within the :class:`~determined.pytorch.experimental.TorchBatchProcessor`
class.

:class:`~determined.pytorch.experimental.TorchBatchProcessor` is compatible with Determined's
:class:`~determined.pytorch.MetricReducer`. You can pass MetricReducer to
:class:`~determined.pytorch.experimental.TorchBatchProcessor` as follow:

``TorchBatchProcessorContext``
==============================

:class:`~determined.pytorch.experimental.TorchBatchProcessorContext` should be a subclass of
:class:`~determined.pytorch.experimental.TorchBatchProcessor`. The two functions you must implement
are the :meth:`~determined.pytorch.experimental.TorchBatchProcessor.__init__` and
:meth:`~determined.pytorch.experimental.TorchBatchProcessor.process_batch`. The other lifecycle
functions are optional.

.. code:: python

   class MyProcessor(TorchBatchProcessor):
       def __init__(self, context):
           self.reducer = context.wrap_reducer(reducer=AccuracyMetricReducer(), name="accuracy")

******************************************
 How To Perform Batch (Offline) Inference
******************************************

In this section, we'll learn how to perform batch inference using the Torch Batch Processing API.

Step 1: Define an InferenceProcessor
====================================

The first step is to define an InferenceProcessor. You should initialize your model in the
:meth:`~determined.pytorch.experimental.TorchBatchProcessor.__init__` function of the
InferenceProcessor. You should implement
:meth:`~determined.pytorch.experimental.TorchBatchProcessor.process_batch` function with inference
logic.

You can optionally implement
:meth:`~determined.pytorch.experimental.TorchBatchProcessor.on_checkpoint_start` and
:meth:`~determined.pytorch.experimental.TorchBatchProcessor.on_finish` to be run before every
checkpoint and after all the data has been processed, respectively.

.. code:: python

   """
   Define custom processor class
   """
   class InferenceProcessor(TorchBatchProcessor):
       def __init__(self, context):
           self.context = context
           self.model = context.prepare_model_for_inference(get_model())
           self.output = []
           self.last_index = 0

       def process_batch(self, batch, batch_idx) -> None:
           model_input = batch[0]
           model_input = self.context.to_device(model_input)

           with torch.no_grad():
               with self.profiler as p:
                   pred = self.model(model_input)
                   p.step()
                   output = {"predictions": pred, "input": batch}
                   self.output.append(output)

           self.last_index = batch_idx

       def on_checkpoint_start(self):
           """
           During checkpoint, we persist prediction result
           """
           if len(self.output) == 0:
               return
           file_name = f"prediction_output_{self.last_index}"
           with self.context.upload_path() as path:
               file_path = pathlib.Path(path, file_name)
               torch.save(self.output, file_path)

           self.output = []

Step 2: Link the Run to a Checkpoint or Model Version (Optional)
================================================================

You have the option to associate your batch inference run with the
:class:~determined.experimental.checkpoint.Checkpoint or
:class:~determined.experimental.model.ModelVersion employed during the run. This allows you to
compile custom metrics for that specific object, which can then be analyzed at a later stage.

The ``inference_example.py`` file in the `CIFAR10 Pytorch Example
<https://github.com/determined-ai/determined/tree/main/examples/computer_vision/cifar10_pytorch>`__
is an example.

Connect the :class:`~determined.experimental.checkpoint.Checkpoint` or
:class:`~determined.experimental.model.ModelVersion` to the inference run.

.. code:: python

   def __init__(self, context):
       self.context = context
       hparams = self.context.get_hparams()

       # Model Version
       model = client.get_model(hparams.get("model_name"))
       model_version = model.get_version(hparams.get("model_version"))
       self.context.report_task_using_model_version(model_version)

       # Or Checkpoint
       ckpt = client.get_checkpoint(hparams.get("checkpoint_uuid"))
       self.context.report_task_using_checkpoint(ckpt)

The :class:`~determined.experimental.checkpoint.Checkpoint` and
:class:`~determined.experimental.model.ModelVersion` used are now available to any query via
``.get_metrics()``.

Step 3: Initialize the Dataset
==============================

Initialize the dataset you want to process.

.. code:: python

   """
   Initialize dataset
   """
   transform = transforms.Compose(
       [transforms.ToTensor(), transforms.Normalize((0.5, 0.5, 0.5), (0.5, 0.5, 0.5))]
   )
   with filelock.FileLock(os.path.join("/tmp", "inference.lock")):
       inference_data = tv.datasets.CIFAR10(
           root="/data", train=False, download=True, transform=transform
       )

Step 4: Pass the InferenceProcessor Class and Dataset
=====================================================

Pass the ``InferenceProcessor`` class and the dataset to ``torch_batch_process``.

.. code:: python

   """
   Pass processor class and dataset to torch_batch_process
   """
   torch_batch_process(
        InferenceProcessor,
        dataset,
        batch_size=64,
        checkpoint_interval=10
    )

Step 5: Send and Query Custom Inference Metrics (Optional)
==========================================================

Report metrics anywhere in the trial to have them aggregated for the
:class:`~determined.experimental.checkpoint.Checkpoint` or
:class:`~determined.experimental.model.ModelVersion` in question.

For example, you could send metrics in
:meth:`~determined.pytorch.experimental.TorchBatchProcessor.on_finish`.

.. code:: python

   def on_finish(self):
       self.context.report_metrics(
           group="inference",
           steps_completed=self.rank,
           metrics={
               "my_metric": 1.0,
           },
       )

And check the metric afterwards from the SDK:

.. code:: python

   from determined.experimental import client

   # Checkpoint
   ckpt = client.get_checkpoint("<CHECKPOINT_UUID>")
   metrics = ckpt.get_metrics("inference")

   # Or Model Version
   model = client.get_model("<MODEL_NAME>")
   model_version = model.get_version(MODEL_VERSION_NUM)
   metrics = model_version.get_metrics("inference")
