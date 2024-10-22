.. _tutorials-index:

###########
 Tutorials
###########

.. meta::
   :description: Choose a tutorial to help you get started training machine learning models. You'll find beginner level and more advanced tutorials with links to user guides and examples.

************
 Quickstart
************

If you are new to Determined, find out how to :ref:`qs-webui`. For a deeper dive, visit the
:ref:`Quickstart for Model Developers <qs-mdldev>` where you'll learn how to perform the following
tasks:

-  Train on a local, single CPU or GPU.
-  Run a distributed training job on multiple GPUs.
-  Use hyperparameter tuning.

*******************************************************
 Get Started with a :ref:`Trial API <high-level-apis>`
*******************************************************

To get started with a :ref:`Trial API <high-level-apis>`, visit :ref:`pytorch-mnist-tutorial`. This
tutorial shows you how to port a simple image classification model for the MNIST dataset.

********************************
 Train Your Model in Determined
********************************

:ref:`Training API Guides <apis-howto-overview>` include the :ref:`api-core-ug` and walk you through
how to take your existing model code and train your model in Determined.

****************
 Try an Example
****************

Examples let you build off of an existing model that already runs on Determined. Visit our
:ref:`Examples <example-solutions>` to see if the model you'd like to train is already available.

.. _pytorch mnist example: https://github.com/PyTorch/examples/blob/main/mnist/main.py

.. toctree::
   :hidden:

   Quickstart for Model Developers <quickstart-mdldev>
   Logging Arbitrary Metadata <metadata-logging>
   Porting Your PyTorch Model to Determined <pytorch-mnist-tutorial>
   Get Started with Detached Mode <detached-mode/_index>
   Viewing Epoch-Based Metrics in the WebUI <viewing-epoch-based-metrics>
   Using Pachyderm to Create a Batch Inferencing Pipeline <pachyderm-cat-dog>
