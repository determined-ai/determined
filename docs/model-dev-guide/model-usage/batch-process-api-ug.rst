:orphan:

.. _torch_batch_processing_ug:

############################
 Torch Batch Processing API
############################

.. meta::
   :description: Learn how to use the Torch Batch Processing API.

In this guide, you'll learn about the :ref:`torch_batch_process_api_ref` and how to perform batch
inference (also known as offline inference).

+---------------------------------------------------------------------+
| Visit the API reference                                             |
+=====================================================================+
| :ref:`torch_batch_process_api_ref`                                  |
+---------------------------------------------------------------------+

.. caution::

   This is an experimental API and may change at any time.

**********
 Overview
**********

The Torch Batch Processing API takes in (1) a dataset and (2) a user-defined processor class and
runs distributed data processing.

With this API, you can perform the following tasks:

-  shard a dataset by number of workers available
-  apply user-defined logic to each batch of data
-  handle synchronization between workers
-  track job progress to enable preemption and resumption of trial

This is a flexible API that can be used for many different tasks, including batch (offline)
inference.

*******
 Usage
*******

The main arguments to torch_batch_process are processor class and dataset.

.. code:: python

   torch_batch_process(
       batch_processor_cls=MyProcessor
       dataset=dataset
   )

class or function e.g., ``torch_batch_process``
===============================================

Processor should be a subclass of TorchBatchProcessor. The two functions you must implement are the
__init__ and process_batch. The other lifecycle functions are optional.

class or function e.g., ``torch_batch_processor``
=================================================

During __init__ of TorchBatchProcessor, we pass in a TorchBatchProcessorContext object, which
contains useful methods that can be used within the TorchBatchProcessor class.

class or function e.g., ``torch_batch_processor``
=================================================

Add content here and a ref link to the class or function in the reference page.

******************************************
 How To Perform Batch (Offline) Inference
******************************************

In this section, we'll learn how to perform batch inference using the ``torch_batch_process`` API.
For more information about this use case or to obtain the tutorial files, visit this URL.

Step 1: Define a InferenceProcessor
===================================

The first step is to Define a InferenceProcessor. You should initialize your model in the __init__
function of InferenceProcessor.

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
           # During checkpoint, we persist prediction result
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
