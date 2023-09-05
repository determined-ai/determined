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

Step 2: Initialize the Dataset
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

Step 3: Pass the InferenceProcessor Class and Dataset
=====================================================

Finally, pass the InferenceProcessor class and the dataset to ``torch_batch_process``.

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
