.. _model-hub-transformers:

##############
 Transformers
##############

`The Huggingface transformers library <https://github.com/huggingface/transformers>`_ is the de
facto library for natural language processing (NLP) models. It provides pretrained weights for
leading NLP models and lets you easily use these pretrained models for the most common NLP tasks,
such as language modeling, text classification, and question answering.

**model-hub** makes it easy to train transformer models in Determined while keeping the developer
experience as close as possible to working directly with **transformers**. The Determined library
serves as an alternative to the HuggingFace `Trainer Class
<https://huggingface.co/transformers/main_classes/trainer.html>`_ and provides access to the
benefits of using Determined, including:

-  Easy multi-node distributed training with no code modifications. Determined automatically sets up
   the distributed backend for you.
-  Experiment monitoring and tracking, artifact tracking, and :ref:`state-of-the-art hyperparameter
   search <hyperparameter-tuning>` without requiring third-party integrations.
-  :ref:`Automated cluster management, fault tolerance, and job rescheduling <features>` to free you
   from provisioning resources closely monitoring experiments.

.. include:: ../../_shared/note-dtrain-learn-more.txt

Model Hub Transformers is similar to the ``no_trainer`` version of **transformers** examples in that
you have more control over the training and evaluation routines if you want.

Given the above benefits, this library can be particularly useful if any of the following apply:

-  You are an Determined user that wants to get started quickly with **transformers**.
-  You are a **transformers** user that wants to easily run more advanced workflows like multi-node
   distributed training and advanced hyperparameter search.
-  You are a **transformers** user looking for a single platform to manage experiments, handle
   checkpoints with automated fault tolerance, and perform hyperparameter search/visualization.

*************
 Limitations
*************

The following HuggingFace **transformers** features are currently not supported:

-  TensorFlow version of transformers
-  Support for fairscale
-  Running on TPUs
