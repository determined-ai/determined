.. _pytorch_mnist_quickstart:

#########################################
 Run Your First Experiment in Determined
#########################################

.. meta::
   :description: Learn how to run your first experiment in Determined by working with the PyTorch MNIST model. You'll need only a single CPU or GPU.
   :keywords: PyTorch API,MNIST,model developer,quickstart

In this tutorial, we’ll show you how to integrate a training example with the Determined
environment. We'll run our experiment on a local training environment requiring only a single CPU or
GPU.

.. note::

   This tutorial is recommended as an introduction for model developers who are new to Determined
   AI.

**Objective**

Our goal is to integrate the `PyTorch MNIST training example
<https://github.com/pytorch/examples/blob/main/mnist/main.py>`_ into Determined in four steps:

-  Download and extract the files
-  Set up our training environment
-  Run the experiment
-  View the experiment in our browser

**Prerequisites**

-  :doc:`Installation Requirements </set-up/on-prem/requirements>`

********************
 Download the Files
********************

To get started, we'll first download and extract the files we need and ``cd`` into the directory.

-  Download the :download:`mnist_pytorch.tgz </examples/mnist_pytorch.tgz>` file.
-  Open a terminal window, extract the files, and ``cd`` into the ``mnist_pytorch`` directory:

.. code::

   tar xzvf mnist_pytorch.tgz
   cd mnist_pytorch

**********************************
 Set Up Your Training Environment
**********************************

To start your experiment, you'll need a Determined cluster. If you are new to Determined AI
(Determined), you can install the Determined library and start a cluster locally:

.. code::

   pip install determined

   # If your machine has GPUs:
   det deploy local cluster-up

   # If your machine does not have GPUs:
   det deploy local cluster-up --no-gpu

.. include:: ../_shared/note-pip-install-determined.txt

********************
 Run the Experiment
********************

To run the experiment, enter the following command:

.. code::

   det experiment create const.yaml . -f

A notification displays letting you know the experiment has started.

.. code::

   Preparing files (.../mnist_pytorch) to send to master...
   Created experiment xxx

*********************
 View the Experiment
*********************

To view the experiment progress in your browser:

-  Enter the following URL: ``http://localhost:8080/``.

This is the cluster address for your local training environment.

-  Accept the default username of ``determined``, and click **Sign In**. A password is not required.

************
 Next Steps
************

In four simple steps, we've successfully configured our training environment in Determined to start
training the PyTorch MNIST example.

In this article, we learned how to run an experiment on a local, single CPU or GPU. If you want to
learn more details about the basic structure shown in the trial class, visit the
:ref:`pytorch-mnist-tutorial`.

To learn how to change your configuration settings, including how to run a distributed training job
on multiple GPUs, visit the :ref:`Quickstart for Model Developers <qs-mdldev>`.
