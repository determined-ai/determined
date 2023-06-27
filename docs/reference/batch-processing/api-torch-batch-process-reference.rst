:orphan:

.. _torch_batch_process_api_ref:

###################################################
 ``name of det torch batch process`` API Reference
###################################################

.. meta::
   :description: Familiarize yourself with the Torch Batch Process API.

+--------------------------------------------+
| User Guide                                 |
+============================================+
| :ref:`torch_batch_processing_ug`           |
+--------------------------------------------+

.. caution::

   This is an experimental API and may change at any time.

The main arguments to torch_batch_process is processor class and dataset.

.. code:: python

   torch_batch_process(
       batch_processor_cls=MyProcessor
       dataset=dataset
   )

********************************************************
 Placeholder for torch_batch_process API docstring pull
********************************************************

Processor should be a subclass of TorchBatchProcessor. The two functions you must implement are the
__init__ and process_batch. The other lifecycle functions are optional.

********************************************************
 Placeholder for TorchBatchProcessor API docstring pull
********************************************************

During __init__ of TorchBatchProcessor, we pass in a TorchBatchProcessorContext object, which
contains useful methods that can be used within the TorchBatchProcessor class.

***************************************************************
 Placeholder for TorchBatchProcessorContext API docstring pull
***************************************************************

Add the Sphinx autoclass directive here.
