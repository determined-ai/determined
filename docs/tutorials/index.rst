###########
 Tutorials
###########

.. meta::
   :description: Choose a tutorial to help you get started training machine learning models. You'll find beginner level and more advanced tutorials with links to user guides and examples.

************
 Quickstart
************

To get started with your first experiment, visit the :ref:`Quickstart for Model Developers
<qs-mdldev>`.

******************************
 Get Started with a Trial API
******************************

+---------------------------------+--------------------------------------------------------------+
| Title                           | Description                                                  |
+=================================+==============================================================+
| :doc:`pytorch-mnist-tutorial`   | Based on the `PyTorch MNIST example`_, this tutorial shows   |
|                                 | you how to port a simple image classification model for the  |
|                                 | MNIST dataset.                                               |
+---------------------------------+--------------------------------------------------------------+
| :doc:`pytorch-porting-tutorial` | The PyTorch porting tutorial provides helpful hints to       |
|                                 | successfully integrate PyTorch code with the Determined      |
|                                 | PyTorchTrial API.                                            |
+---------------------------------+--------------------------------------------------------------+
| :doc:`tf-mnist-tutorial`        | The TensorFlow Keras Fashion MNIST tutorial describes how to |
|                                 | port a ``tf.keras`` model to Determined.                     |
+---------------------------------+--------------------------------------------------------------+

**********************************************
 Want to Learn About a Specific Training API?
**********************************************

:ref:`Training API Guides <apis-howto-overview>` describe how to take your existing model code and
train your model in Determined.

***********************
 Looking for Examples?
***********************

Examples let you build off of an existing model that already runs on Determined. Visit our
:ref:`Examples <example-solutions>` to see if the model you'd like to train is already available.

.. _pytorch mnist example: https://github.com/PyTorch/examples/blob/master/mnist/main.py

.. toctree::
   :hidden:

   Run Your First Experiment <pytorch-mnist-local-qs>
   PyTorch MNIST Tutorial <pytorch-mnist-tutorial>
   PyTorch Porting Tutorial <pytorch-porting-tutorial>
   TensorFlow Keras Fashion MNIST Tutorial <tf-mnist-tutorial>
