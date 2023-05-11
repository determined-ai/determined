.. _model-hub-transformers-api:

##################
 Transformers API
##################

***************************
 ``model_hub.huggingface``
***************************

.. _basetransformertrial:

.. autoclass:: model_hub.huggingface.BaseTransformerTrial

The ``__init__`` method replicated below makes heavy use of the :ref:`helper functions
<transformers-functions>` in the next section.

.. literalinclude:: ../../../../model_hub/model_hub/huggingface/_trial.py
   :language: python
   :pyobject: BaseTransformerTrial.__init__

The ``evaluate_batch`` method replicated below should work for most models and tasks but can be
overwritten for more custom behavior in a subclass.

.. literalinclude:: ../../../../model_hub/model_hub/huggingface/_trial.py
   :language: python
   :pyobject: BaseTransformerTrial.train_batch

.. _transformers-functions:

Helper Functions
================

The ``BaseTransformerTrial`` calls many helper functions below that are also useful when subclassing
``BaseTransformerTrial`` or writing custom transformers trials for use with Determined.

.. automodule:: model_hub.huggingface
   :members: default_parse_config_tokenizer_model_kwargs, default_parse_optimizer_lr_scheduler_kwargs, build_using_auto, build_default_optimizer, build_default_lr_scheduler, default_load_dataset

Structured Dataclasses
======================

Structured dataclasses are used to ensure that Determined parses the experiment config correctly.
See the below classes for details on what fields can be used in the experiment config to configure
the dataset; transformers config, model, and tokenizer; as well as optimizer and learning rate
scheduler for use with the functions above.

.. autoclass:: model_hub.huggingface.DatasetKwargs

.. autoclass:: model_hub.huggingface.ConfigKwargs

.. autoclass:: model_hub.huggingface.ModelKwargs

.. autoclass:: model_hub.huggingface.OptimizerKwargs

.. autoclass:: model_hub.huggingface.LRSchedulerKwargs
