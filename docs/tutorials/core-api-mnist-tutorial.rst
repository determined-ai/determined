.. _core_api_tutorial_part_2:

###########################################################
 Quickstart: Run a Core API MNIST Trial on a Local Cluster
###########################################################

.. meta::
   :description: In five steps, learn how to integrate the PyTorch MNIST model into Determined AI.
   :keywords: Core API,MNIST,model developer

In this tutorial, we’ll show you how to integrate a training example with the Determined
environment. We'll run our experiment on a local training environment requiring only a single CPU or
GPU. This tutorial is recommended as an introduction for model developers who are new to Determined
AI.

**Objective**

Our goal is to integrate the `PyTorch MNIST training example
<https://github.com/pytorch/examples/blob/main/mnist/main.py>`_ into Determined in five steps:

-  Start with a bare-bones script
-  Report metrics
-  Perform checkpointing
-  Perform hyperparameter search optimization
-  Run distributed training

**Prerequisites**

-  `System Requirements
   <https://docs.determined.ai/latest/cluster-setup-guide/deploy-cluster/sysadmin-deploy-on-prem/requirements.html#system-requirements>`_
-  `Docker
   <https://docs.determined.ai/latest/cluster-setup-guide/deploy-cluster/sysadmin-deploy-on-prem/requirements.html#install-docker>`_

**************************
 Get the Quickstart Files
**************************

To get started, add the `Core API MNIST tutorial files
<https://github.com/determined-ai/determined/tree/master/examples/tutorials/core_api_mnist>`_ to your
computer.

-  Create a new directory on your computer and name it something like ``core_api_mnist``.
-  Add the Core API MNIST Tutorial files.

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

Each step has a corresponding experiment configuration file (.yaml) and training script (.py). In
step one, you'll run a bare-bones script on your Determined cluster. To run the experiment starting
with step one, enter the following command:

.. code:: bash

   det -m experiment create -f const.yaml .

Continue running the other steps by specifying the experiment configuration file for the step. For
example, to run the metric reporting step, enter the following command:

.. code:: bash

   det -m experiment create -f metrics.yaml .

*********************
 View the Experiment
*********************

To view the experiment progress in your browser:

-  Enter the following URL: ``http://localhost:8080/``.

This is the cluster address for your local training environment.

-  Accept the default ``determined`` username, leave the password empty, and click **Sign In**.
