.. _pytorch_mnist_quickstart:

##################################################
 Run a PyTorch API MNIST Trial on a Local Cluster
##################################################

.. meta::
   :description: Learn how to integrate the PyTorch MNIST model into Determined AI using only a single CPU or GPU.
   :keywords: PyTorch API,MNIST,model developer,quickstart


In this tutorial, we’ll show you how to integrate a training example with the Determined
environment. We'll run our experiment on a local training environment requiring only a single CPU or
GPU. 

.. note::

   This tutorial is recommended as an introduction for model developers who are new to Determined AI.

**Objective**

Our goal is to integrate the `PyTorch MNIST training example
<https://github.com/pytorch/examples/blob/main/mnist/main.py>`_ into Determined in four steps:

-  Download and extract the files
-  Set up our training environment
-  Run the experiment
-  View the experiment in our browser

**Prerequisites**

-  `System Requirements
   <https://docs.determined.ai/latest/cluster-setup-guide/deploy-cluster/sysadmin-deploy-on-prem/requirements.html#system-requirements>`_
-  `Docker
   <https://docs.determined.ai/latest/cluster-setup-guide/deploy-cluster/sysadmin-deploy-on-prem/requirements.html#install-docker>`_

*****************************
 Download the Files
*****************************

To get started, we'll first download and extract the files we need and ``cd`` into
the directory.

- Download the :download:`mnist_pytorch.tgz </examples/mnist_pytorch.tgz>` file.
- Open a terminal window, extract the file, and ``cd`` into the ``mnist_pytorch`` directory:

.. code::

   tar xzvf mnist_pytorch.tgz
   cd mnist_pytorch


**********************************
 Set Up Your Training Environment
**********************************

To start your experiment, you'll need a Determined cluster. If you are new to Determined AI
(Determined), you can install the Determined library and start a cluster locally:

.. code:: bash

   pip install determined
   det deploy local cluster-up

If your local machine does not have a supported Nvidia GPU, include the ``no-gpu`` option:

.. code:: bash

   pip install determined
   det deploy local cluster-up --no-gpu

.. note::

   If you want to see if Determined is already installed, you can type ``det --version``.

********************
 Run the Experiment
********************

To run the experiment, enter the following command:

.. code::

   det experiment create const.yaml .

A notification displays letting you know the experiment has started.

.. code::

   Preparing files (.../mnist_pytorch) to send to master... 2.5KB and 4 files
   Created experiment xxx

*********************
 View the Experiment
*********************

To view the experiment progress in your browser:

-  Enter the following URL: ``http://localhost:8080/``.

This is the cluster address for your local training environment.

-  Accept the default ``determined`` username, leave the password empty, and click **Sign In**.

************
 Next Steps
************

In this article, we learned how to run an experiment on a local, single CPU or GPU. To learn how to change your
configuration settings, including how to run a distributed training job on multiple GPUs, visit the
`Quickstart for Model Developers <https://docs.determined.ai/latest/quickstart-mdldev.html#>`_.
