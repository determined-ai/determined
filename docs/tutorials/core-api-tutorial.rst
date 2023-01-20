.. _core_api_tutorial_part_1:

###################################################################################
 Quickstart: Integrate an Existing Training Script with the Determined Environment
###################################################################################

.. meta::
   :description: Learn how to take an existing training script and integrate it with Determined.
   :keywords: Core API,model developer

In this tutorial, we’ll show you how to take an existing training script
and integrate it with the Determined environment. We'll run our
experiment on a local training environment requiring only a single CPU
or GPU. This tutorial is recommended as an introduction for model
developers who are new to Determined AI.

**Objective**

Our goal is to increment an integer. This will give you an idea of how
you can modify your own training script to integrate it with the Core
API.

**Prerequisites**

-  `System Requirements
   <https://docs.determined.ai/latest/cluster-setup-guide/deploy-cluster/sysadmin-deploy-on-prem/requirements.html#system-requirements>`_
-  `Docker
   <https://docs.determined.ai/latest/cluster-setup-guide/deploy-cluster/sysadmin-deploy-on-prem/requirements.html#install-docker>`_

**************************
 Get the Quickstart Files
**************************

To get started, add the `quickstart files
<https://github.com/determined-ai/determined/tree/master/examples/tutorials/core_api>`_
to your computer.

-  Create a new directory on your computer and name it something like
   ``core_api``.
-  Add the Core API Tutorial files.

**********************************
 Set Up Your Training Environment
**********************************

To start your experiment, you'll need a Determined cluster. If you are
new to Determined AI (Determined), you can install the Determined
library and start a cluster locally:

.. code:: bash

   pip install determined
   det deploy local cluster-up

If your local machine does not have a supported Nvidia GPU, include the
``no-gpu`` option:

.. code:: bash

   pip install determined
   det deploy local cluster-up --no-gpu

.. note::

   If you want to see if Determined is already installed, you can type
   ``det --version``.

********************
 Run the Experiment
********************

Each stage has a corresponding experiment configuration file (.yaml) and
training script (.py). Stage 0 is intended to look like a completely
unmodified training script. Stage 1 creates the core_context and uses it
to start logging metrics. To run the experiment starting with stage 0,
enter the following command:

.. code:: bash

   det -m experiment create -f 0_start.yaml .

Continue running the other stages by specifying the experiment
configuration file for the stage. For example, to run stage 1, enter the
following command:

.. code:: bash

   det -m experiment create -f 1_metrics.yaml .

*********************
 View the Experiment
*********************

To view the experiment progress in your browser:

-  Enter the following URL: ``http://localhost:8080/``.

This is the cluster address for your local training environment.

-  Accept the default ``determined`` username, leave the password empty,
   and click **Sign In**.

************
 Next Steps
************

Go further with the Core API by visiting the tutorial, "Quickstart: Run
a Core API MNIST Trial on a Local Cluster", where can walk through
step-by-step instructions for integrating the PyTorch MNIST training
example into Determined.
