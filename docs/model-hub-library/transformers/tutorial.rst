.. _model-hub-transformers-tutorial:

##########
 Tutorial
##########

.. _huggingface datasets: https://huggingface.co/docs/datasets

.. _load_dataset: https://huggingface.co/docs/datasets/package_reference/loading_methods.html#datasets.load_dataset

.. _ner_trial.py: https://github.com/determined-ai/determined/tree/master/model_hub/examples/huggingface/token-classification/ner_trial.py

.. _qa_beam_search_trial.py: https://github.com/determined-ai/determined/tree/master/model_hub/examples/huggingface/question-answering/qa_beam_search_trial.py

.. _qa_trial.py: https://github.com/determined-ai/determined/tree/master/model_hub/examples/huggingface/question-answering/qa_trial.py

.. _question answering example: https://github.com/determined-ai/determined/tree/master/model_hub/examples/huggingface/question-answering

.. _squad.yaml: https://github.com/determined-ai/determined/tree/master/model_hub/examples/huggingface/question-answering/squad.yaml

.. _transformers trainer: https://huggingface.co/transformers/v4.4.2/main_classes/trainer.html

The easiest way to get started with **transformers** in Determined is to use one of the
:ref:`provided examples <model-hub-transformers-examples>`. In this tutorial, we will walk through
the `question answering example`_ to get a better understanding of how to use **model-hub** for
transformers.

The `question answering example`_ includes two implementations of
:doc:`/model-dev-guide/apis-howto/api-pytorch-ug`:

-  qa_trial.py_ uses the :py:class:`model_hub.huggingface.BaseTransformerTrial` parent ``__init__``
   function to build **transformers** config, tokenizer, and model objects; and optimizer and
   learning rate scheduler.

-  qa_beam_search_trial.py_ overrides the :py:class:`model_hub.huggingface.BaseTransformerTrial`
   parent ``__init__`` function to customize how the **transformers** config, tokenizer, and model
   objects are constructed.

To learn the basics, we'll walk through qa_trial.py_. We won't cover the model definition
line-by-line but will highlight the parts that make use of **model-hub**.

.. note::

   If you are new to Determined, we recommend going through the Quickstart for ML Developers
   document to get a better understanding of how to use PyTorch in Determined using
   :py:class:`determined.harness.pytorch.PyTorchTrial`.

After this tutorial, if you want to further customize a trial for your own use, you can look at
qa_beam_search_trial.py_ for an example.

************************
 Initialize the QATrial
************************

The ``__init__`` for ``QATrial`` is responsible for creating and processing the dataset; building
the **transformers** config, tokenizer, and model; and tokenizing the dataset. The specifications
for how we should perform these steps is passed from :class:`~determined.pytorch.PyTorchContext` via
the hyperparameters and data configuration fields. These fields are set to ``hparams`` and
``data_config`` class attributes in :py:meth:`model_hub.huggingface.BaseTransformerTrial.__init__`.
You can also get them by calling ``context.get_hparams()`` and ``context.get_data_config()``
respectively.

Note that ``context.get_hparams()`` and ``context.get_data_config()`` returns the
``hyperparameters`` and ``data`` section respectively of the :ref:`experiment configuration
<experiment-config-reference>` file squad.yaml_.

Build **transformers** config, tokenizer, and model
===================================================

First, we'll build the **transformer** config, tokenizer, and model objects by calling
:py:meth:`model_Hub.huggingface.BaseTransformerTrial.__init__`:

.. literalinclude:: ../../../model_hub/examples/huggingface/question-answering/qa_trial.py
   :language: python
   :lines: 37

This will parse the hyperparameters and fill the fields of
:py:class:`model_hub.huggingface.ConfigKwargs`, :py:class:`model_hub.huggingface.TokenizerKwargs`,
and :py:class:`model_hub.huggingface.ModelKwargs` if present in hyperparameters and then pass them
to :py:func:`model_hub.huggingface.build_using_auto` to build the config, tokenizer, and model using
`transformers autoclasses <https://huggingface.co/transformers/v4.4.2/model_doc/auto.html>`_. You
can look at the associated class definitions for the Kwargs objects to see the fields you can pass.

This step needs to be done before we can use the tokenizer to tokenize the dataset. In some cases,
you may need to first load the raw dataset and get certain metadata like the number of classes
before creating the **transformers** objects (see ner_trial.py_ for example).

.. note::

   You are not tied to using :py:func:`model.huggingface.build_using_auto` to build the config,
   tokenizer, and model objects. See qa_beam_search_trial.py_ for an example of a trial directly
   calling transformers methods.

Build the optimizer and LR scheduler
====================================

The :py:meth:`model_Hub.huggingface.BaseTransformerTrial.__init__` also parses the hyperparameters
into :py:func:`model_hub.huggingface.OptimizerKwargs` and
:py:func:`model_hub.huggingface.LRSchedulerKwargs` before passing them to
:py:func:`model_hub.huggingface.build_default_optimizer` and
:py:func:`model_hub.huggingface.build_default_lr_scheduler` respectively. These two build methods
have the same behavior and configuration options as the `transformers Trainer`_. Again, you can look
at the associated class definitions for the Kwargs objects to see the fields you can pass.

.. note::

   You are not tied to using these functions to build the optimizer and LR scheduler. You can very
   easily override the parent ``__init__`` methods to use whatever optimizer and LR scheduler you
   want.

Load the Dataset
================

.. literalinclude:: ../../../model_hub/examples/huggingface/question-answering/qa_trial.py
   :language: python
   :lines: 65

This example uses the helper function :py:meth:`model_hub.huggingface.default_load_dataset` to load
the SQuAD dataset. The function takes the ``data_config`` as input and parses the fields into those
expected by the :py:class:`model_hub.huggingface.DatasetKwargs` dataclass before passing it to the
load_dataset_ function from `Huggingface datasets`_.

Not all the fields of :py:class:`model_hub.huggingface.DatasetKwargs` are always applicable to an
example. For this example, we specify the following fields in squad.yaml_ for loading the dataset:

.. literalinclude:: ../../../model_hub/examples/huggingface/question-answering/squad.yaml
   :language: yaml
   :lines: 15-18

If the dataset you want to use is registered in `Huggingface datasets`_ then you can simply specify
the ``dataset_name``. Otherwise, you can set ``dataset_name: null`` and pass your own dataset in
using ``train_file`` and ``validation_file``. There is more guidance on how to use this example with
custom data files in qa_trial.py_.

.. note::

   You can also bypass :py:meth:`model_hub.huggingface.default_load_dataset` and call load_dataset_
   directly for more options.

Data processing
===============

Our text data needs to be converted to vectors before we can apply our models to them. This usually
involves some preprocessing before passing the result to the tokenizer for vectorization. This part
usually has task-specific preprocessing required as well to process the targets. **model-hub** has
no prescription for how you should process your data but all the provided examples implement a
``build_datasets`` function to create the tokenized dataset.

.. note::

   The Huggingface **transformers** and **datasets** library have optimized routiens for
   tokenization that caches results for reuse if possible. We have taken special care to make sure
   all our examples make use of this functionality. As you start implementing your own Trials, one
   pitfall to watch out for that prevents efficient caching is passing a function to ``Dataset.map``
   that contains unserializable objects.

Define metrics
==============

Next, we'll define the metrics that we wish to compute over the predictions generated for the
validation dataset.

.. literalinclude:: ../../../model_hub/examples/huggingface/question-answering/qa_trial.py
   :language: python
   :lines: 91-108

We use the metric function associated with the SQuAD dataset from `huggingface datasets`_ and apply
it after post-processing the predictions in the ``qa_utils.compute_metrics`` function.

Determined supports parallel evaluation via :ref:`custom reducers <pytorch-custom-reducers>`. The
``reducer`` we created above will aggregate predictions across all GPUs then apply the
``qa_utils.compute_metrics`` function to the result.

**********************************
 Fill in the Rest of PyTorchTrial
**********************************

The remaining class methods we must implement are
:py:meth:`determined.harness.pytorch.PyTorchTrial.build_training_data_loader`,
:py:meth:`determined.harness.pytorch.PyTorchTrial.build_validation_data_loader`, and
:py:meth:`determined.harness.pytorch.PyTorchTrial.evaluate_batch`.

Build the Dataloaders
=====================

The two functions below are responsible for building the dataloaders used for training and
validation.

.. literalinclude:: ../../../model_hub/examples/huggingface/question-answering/qa_trial.py
   :language: python
   :lines: 136-151

There are two things to note:

-  Batch size passed to the dataloader is ``context.get_per_slot_batch_size`` which is the effective
   per GPU batch size when performing distributed training.

-  The dataloader returned is a :py:class:`determined.harness.pytorch.DataLoader` which has the same
   signature as PyTorch dataloaders but automatically handles data sharding and resuming dataloader
   state when recovering from a fault.

Define the Training Routine
===========================

The ``train_batch`` method below for :py:class:`model_hub.huggingface.BaseTransformerTrial` is
sufficient for this example.

.. literalinclude:: ../../../model_hub/model_hub/huggingface/_trial.py
   :language: python
   :pyobject: BaseTransformerTrial.train_batch

Define the Evaluation Routine
=============================

Finally, we can define the evaluation routine for this example.

.. literalinclude:: ../../../model_hub/examples/huggingface/question-answering/qa_trial.py
   :language: python
   :lines: 153-166

After passing the batch through the model and doing some processing to get the predictions, we pass
the predictions to ``reducer.update`` to aggregate the predictions in each GPU. Once each GPU has
exhausted the batches in its dataloader, Determined automatically performs an all gather operation
to collect the predictions in the rank 0 GPU before passing them to the ``compute_metrics``
function.

*********************
 HF Library Versions
*********************

**model-hub** support for **transformers** is tied to specific versions of the source library to
ensure compatibility. Be sure to use the latest Docker image with all the necessary dependencies for
**transformers** with **model-hub**. All provided examples already have this Docker image specified:

.. literalinclude:: ../../../model_hub/examples/huggingface/question-answering/squad.yaml
   :language: yaml
   :lines: 39-40

We periodically bump these libraries up to more recent versions of ``transformers`` and ``datasets``
so you can access the latest upstream features. That said, once you create a trial definition using
a particular Docker image, you will not need to upgrade to a new Docker image for your code to
continue working with **model-hub**. Additionally, your code will continue to work with that image
even if you use it with a more recent version of the Determined cluster.

************
 Next Steps
************

-  Take a look at qa_beam_search_trial.py_ for an example of how you can further customize your
   trial.
-  Dive into :ref:`the api <model-hub-transformers-api>`.
