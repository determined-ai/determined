###########
 Tutorials
###########

Learning the basics of working with Determined and how to port your existing code to the Determined
environment.

*******************************
 Get Started with the Core API
*******************************

+---------------------------------+--------------------------------------------------------------+
| Title                           | Description                                                  |
+=================================+==============================================================+
| :doc:`core-api-tutorial`        | Learn how to take an existing training script and integrate  |
|                                 | it with Determined using the Core API.                       |
+---------------------------------+--------------------------------------------------------------+
| :doc:`core-api-mnist-tutorial`  | In five steps, learn how to integrate the PyTorch MNIST      |
|                                 | model into Determined using the Core API.                    |
+---------------------------------+--------------------------------------------------------------+

*********************************
 Get Started with the Trial APIs
*********************************

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

Looking for Examples?
=====================

Visit our `Examples <https://docs.determined.ai/latest/example-solutions/examples.html>`_ page then
open the ``examples/`` subdirectory of the `Determined GitHub repo
<https://github.com/determined-ai/determined/tree/master/examples>`__.

Go Further
==========

Visit the `Training API Guides
<https://docs.determined.ai/latest/training/apis-howto/overview.html>`_ for in-depth guides that
contain detailed information about the training APIs.

.. _pytorch mnist example: https://github.com/PyTorch/examples/blob/master/mnist/main.py

.. toctree::
   :hidden:

   PyTorch MNIST Tutorial <pytorch-mnist-tutorial>
   PyTorch Porting Tutorial <pytorch-porting-tutorial>
   TensorFlow Keras Fashion MNIST Tutorial <tf-mnist-tutorial>
   Integrate an Existing Training Script with the Determined Environment <core-api-tutorial>
   Run a Core API MNIST Trial on a Local Cluster <core-api-mnist-tutorial>
