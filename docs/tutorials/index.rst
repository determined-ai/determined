###########
 Tutorials
###########

Learn the basics of working with Determined and how to port your existing code to the Determined
environment.

**************************
 Get Started in 5 Minutes
**************************

+---------------------------------+--------------------------------------------------------------+
| Title                           | Description                                                  |
+=================================+==============================================================+
| :doc:`pytorch-mnist-local-qs`   | In a few steps, learn how to run your first experiment in    |
|                                 | Determined using only a single CPU or GPU.                   |
+---------------------------------+--------------------------------------------------------------+

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

Go Further
==========

Visit the `Training API Guides
<https://docs.determined.ai/latest/training/apis-howto/overview.html>`_ for in-depth guides that
describe how to take your existing model code and train your model in Determined.

Looking for Examples?
=====================

Examples let you build off of an existing model that already runs on Determined. Visit our `examples
<https://docs.determined.ai/latest/example-solutions/examples.html>`_ to see if the model you'd like
to train is already available.

.. _pytorch mnist example: https://github.com/PyTorch/examples/blob/master/mnist/main.py

.. toctree::
   :hidden:

   Run Your First Experiment <pytorch-mnist-local-qs>
   PyTorch MNIST Tutorial <pytorch-mnist-tutorial>
   PyTorch Porting Tutorial <pytorch-porting-tutorial>
   TensorFlow Keras Fashion MNIST Tutorial <tf-mnist-tutorial>
