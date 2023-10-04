###########
 Tutorials
###########

.. meta::
   :description: Choose a tutorial to help you get started training machine learning models. You'll find beginner level and more advanced tutorials with links to user guides and examples.

************
 Quickstart
************

If you are new to Determined, visit the :ref:`Quickstart for Model Developers <qs-mdldev>` where
you'll learn how to perform the following tasks:

-  Train on a local, single CPU or GPU.
-  Run a distributed training job on multiple GPUs.
-  Use hyperparameter tuning.

*******************************************************
 Get Started with a :ref:`Trial API <high-level-apis>`
*******************************************************

+---------------------------------+--------------------------------------------------------------+
| Title                           | Description                                                  |
+=================================+==============================================================+
| :doc:`pytorch-mnist-tutorial`   | Based on the `PyTorch MNIST example`_, this tutorial shows   |
|                                 | you how to port a simple image classification model for the  |
|                                 | MNIST dataset.                                               |
+---------------------------------+--------------------------------------------------------------+
| :doc:`tf-mnist-tutorial`        | The TensorFlow Keras Fashion MNIST tutorial describes how to |
|                                 | port a ``tf.keras`` model to Determined.                     |
+---------------------------------+--------------------------------------------------------------+

********************************
 Train Your Model in Determined
********************************

:ref:`Training API Guides <apis-howto-overview>` include the :ref:`api-core-ug` and walk you through
how to take your existing model code and train your model in Determined.

**********
 Examples
**********

Examples let you build off of an existing model that already runs on Determined. Visit our
:ref:`Examples <example-solutions>` to see if the model you'd like to train is already available.

.. _pytorch mnist example: https://github.com/PyTorch/examples/blob/master/mnist/main.py

.. toctree::
   :hidden:
   :glob:

   ./*
