#################################
 Quickstart for Model Developers
#################################

This quickstart demonstrates the basic functionality of Determined using the MNIST dataset. It 
guides you through the process of installing Determined, running training jobs, and visualizing
experiment results in your web browser.

There are three examples included in this quickstart that showcase the scalability and enhanced 
functionality that can be achieved by making simple configuration changes:

-  Training on a local machine using a single CPU or GPU.
-  Running a distributed training job on multiple GPUs.
-  Utilizing hyperparameter tuning to optimize model performance.

In Determined, an *experiment* refers to a training job that can consist of one or more 
variations, or trials, on the same model. By using Determined API functions in your training 
loops, you will automatically receive metric frequency output, plots, and checkpointing for 
every experiment without the need to write extra code. 

You can utilize the WebUI to view various information about your model including configuration 
settings, output logs, and training metrics for all of your experiments.

In these quickstart example, the same model code and example datasets are used, differing only in
their configuration settings. For a complete list of all experiment configuration settings 
and more detailed information about setting, refer to :doc:`/reference/reference-training/experiment-config-reference`.

***************
 Prerequisites
***************

Software
========

-  Determined *agent* and *master* nodes configured with Ubuntu 16.04 or higher, CentOS 7,
   or macOS 10.13 or higher.

-  Agent nodes with Docker installed.

-  To run jobs with GPUs, install Nvidia drivers, version 384.81 or higher, on each agent. The
   drivers can be installed as part of a CUDA installation but the rest of the CUDA toolkit is not
   required.

Hardware
========

-  Master node:

   -  At least 4-CPU cores, Intel Broadwell or later. The master node does not require GPUs.
   -  8GB RAM
   -  200GB of free disk space.

-  Agent Node:

   -  At least 2-CPU cores, Intel Broadwell or later.
   -  If you are using GPUs, Nvidia GPUs with compute capability 3.7 or greater are required: K80,
      P100, V100, A100, GTX 1080, GTX 1080 Ti, TITAN, or TITAN XP.
   -  4GB RAM
   -  50GB of free disk space.

Docker
======

Install Docker to run containerized workloads. If you do not already have Docker installed, follow
the :doc:`Installing Docker
</cluster-setup-guide/deploy-cluster/sysadmin-deploy-on-prem/requirements>` instructions to install
and run Docker on Linux or macOS.

******************************
 Quickstart Training Examples
******************************

Download and extract the quickstart files to a local directory:

#. Download link: :download:`mnist_pytorch.tgz </examples/mnist_pytorch.tgz>`.

#. Extract the configuration and model files:

   .. code:: bash

      tar xzvf mnist_pytorch.tgz

You'll see the following files in the ``mnist_pytorch`` directory:

.. code::

   adaptive.yaml
   const.yaml
   data.py
   distributed.yaml
   layers.py
   model_def.py
   README.md

Configuration
=============

Each YAML-formatted configuration file corresponds to an example
experiment:

+------------------------+------------------------------------------------------+
| Configuration Filename | Example Experiment                                   |
+========================+======================================================+
| ``const.yaml``         | Train a single model on a single GPU/CPU, with       |
|                        | constant hyperparameter values.                      |
+------------------------+------------------------------------------------------+
| ``distributed.yaml``   | Train a single model using multiple, distributed     |
|                        | GPUs.                                                |
+------------------------+------------------------------------------------------+
| ``adaptive.yaml``      | Perform a hyperparameter search using the Determined |
|                        | adaptive hyperparameter tuning algorithm.            |
+------------------------+------------------------------------------------------+

Model and Pipeline Definition
=============================

Although the Python model and data pipeline definition files are not explained in this quickstart,
you might want to review them to see how to call the Determined API from your code:

+------------------+------------------------------------------------------------------------+
| Filename         | Experiment Type                                                        |
+==================+========================================================================+
| ``data.py``      | Model data loading and preparation code.                               |
+------------------+------------------------------------------------------------------------+
| ``layers.py``    | Convolutional layers used by the model.                                |
+------------------+------------------------------------------------------------------------+
| ``model_def.py`` | Model definition and training/validation loops.                        |
+------------------+------------------------------------------------------------------------+

Once you have gained a basic familiarity with Determined's tools and operations, 
you can start replacing the example files with your own model data and code. You can
then set the configuration parameters that are specific to the experiments you
want to run.

*****************************************
 Run a Local Single CPU/GPU Training Job
*****************************************

This exercise involves training a single model on a single *slot*, either a CPU or GPU, 
for a fixed number of batches using constant values for all hyperparameters. A slot is a 
CPU or GPU computing device, which the master schedules to run.

#. To begin, install the Determined library by running the command below. 
We recommend installing the package into a specific Python Environment rather than the Global Environment. Therefore, before running this command, you may need to activate the 
desired Environment.

   .. code:: bash

      pip install determined
      
#. Before starting Determined, ensure you have :doc:`Docker installed on your system and that it is open and accessible. You can refer to the :ref:`Determined Installation Requirements <system-requirements>` for instructions on how to install Docker on your system.

#. To run Determined and start a cluster locally, run the following command. If your local machine does not have a supported Nvidia GPU, include the ``no-gpu`` option in the command:

   .. code:: bash

      det deploy local cluster-up
      
   Non-Nvidia GPU option:
   
   .. code:: bash
   
      det deploy local cluster-up --no-gpu

#. In the ``mnist_pytorch`` directory, create an experiment specifying the ``const.yaml``
   configuration file:

   .. code:: bash

      det experiment create const.yaml .

   The last dot (.) argument uploads all of the files in the current directory as the *context
   directory* for your model. Determined copies the model context directory contents to the trial
   container working directory.

   You should receive confirmation that the experiment is created and scheduled to run:

   .. code:: console

      Preparing files (.../mnist_pytorch) to send to master... 8.6KB and 7 files
      Created experiment 1

   .. tip::

      To automatically stream log messages for the first trial in an experiment to ``stdout``,
      specifying the configuration file and context directory, enter:

      .. code:: bash

         det e create const.yaml . -f

      The ``-f`` option is the short form of ``--follow``.

#. To view the uncategorized experiments, enter your username and password. If you deployed locally,
   accept the default determined username with no password and click **Sign In**:

#. Enter the cluster address in the browser address bar to view experiment progress in the WebUI. If
   you installed locally using the ``det deploy local`` command, the URL is
   ``http://localhost:8080/``. Accept the default ``determined`` username and click **Sign In**. No
   password is required.

   .. image:: /assets/images/qs01c.png
      :width: 704px
      :align: center
      :alt: Dashboard

   The figure shows two experiments. Experiment **3** has **COMPLETED** and experiment **4** is
   still **ACTIVE**. Your experiment number and status can differ depending on how many times you
   run the examples.

#. While an experiment is in the ACTIVE, training state, click the experiment name to see the
   **Metrics** graph update for your currently defined metrics:

   .. image:: /assets/images/qs04.png
      :width: 704px
      :align: center
      :alt: Metrics graph detail

   In this example, the graph displays the loss.

#. After the experiment completes, click the experiment name to view the trial page:

   .. image:: /assets/images/qs03.png
      :width: 704px
      :align: center
      :alt: Trial page

With this fundamental understanding of Determined, you are ready to scale to distributed training in
the next example.

***************************************
 Run a Remote Distributed Training Job
***************************************

In the distributed training example, a Determined cluster consists of a master and one or more 
agents, with the master providing centralized management of the agent resources.

To run this example, you will need a Determined cluster with multiple GPUs. While this example
does not fully demonstrate the benefits of distributed training, it will show you how to 
work with added hardware resources.

The configuration file for this example, ``distributed.yaml``, is similar to the ``const.yaml`` file
used in the previous example, except for the addition of a ``resources.slots_per_trial`` field, 
which is defined and set to a value of ``8``:

.. code:: yaml

   resources:
     slots_per_trial: 8

This is the number of available GPU resources. The ``slots_per_trial`` value must be divisible by
the number of GPUs per machine. You can change the value to match your hardware configuration.

#. To connect to a Determined master running on a remote instance, set the remote IP address and
   port number in the ``DET_MASTER`` environment variable:

   .. code:: bash

      export DET_MASTER=<ipAddress>:8080

#. Create and run the experiment:

   .. code:: bash

      det experiment create distributed.yaml .

   You can also use the ``-m`` option to specify a remote master IP address:

   .. code:: bash

      det -m http://<ipAddress>:8080 experiment create distributed.yaml .

#. To view the WebUI dashboard, enter the cluster address in your browser address bar, accept the
   default ``determined`` username, and click **Sign In**. A password is not required.

#. Click the **Experiment** name to view the experiment’s trial display. The loss curve is similar
   to the single-GPU experiment in the previous exercise but the time to complete the trial is
   reduced by about half.

*********************************
 Run a Hyperparameter Tuning Job
*********************************

This example showcases hyperparameter search in Determined.

It uses the ``adaptive.yaml`` configuration file, which is similar to the ``const.yaml`` 
file used in the first example, but includes additional hyperparameter settings:

.. code:: yaml

   hyperparameters:
     global_batch_size: 64
     learning_rate:
       type: double
       minval: .0001
       maxval: 1.0
     n_filters1:
       type: int
       minval: 8
       maxval: 64
     n_filters2:
       type: int
       minval: 8
       maxval: 72
     dropout1:
       type: double
       minval: .2
       maxval: .8
     dropout2:
       type: double
       minval: .2
       maxval: .8

Hyperparameter searches involve multiple trials or model variations per experiment. The
configuration settings tell the search algorithm the ranges to explore for each hyperparameter.

The ``adaptive_asha`` search method and maximum number of trials, max_trials` are also specified:

.. code:: yaml

   searcher:
     name: adaptive_asha
     metric: validation_loss
     smaller_is_better: true
     max_trials: 16
     max_length:
       batches: 937

This example uses a fixed batch size and searches on dropout size, filters, and learning rate. The
``max_trials`` setting of ``16`` indicates how many model configurations to explore.

#. Create and run the experiment:

   .. code:: bash

      det experiment create adaptive.yaml .

#. To view the WebUI dashboard, enter your cluster address in the browser address bar, accept the
   default determined username, and click **Sign In**. No password is required.

#. The experiment can take some time to complete. You can monitor progress in the WebUI Dashboard by
   clicking the **Experiment** name. Notice that more trials have started:

   .. image:: /assets/images/qs05.png
      :width: 704px
      :align: center
      :alt: Trials graphic

   Determined runs the number of ``max_trials`` trials and automatically starts new trials as
   resources become available. For 16 trials, it should take about 10 minutes to train with at least
   one trial performing at about 98 percent validation accuracy. The hyperparameter search halts
   poorly performing trials.

************
 Learn More
************

For detailed information on administrator tasks and how to install Determined on different
platforms, see :doc:`/cluster-setup-guide/basic`.

In the :doc:`example-solutions/examples` documentation, you can find machine learning models that
have been ported to the Determined APIs. Each example includes a model definition and one or more
experiment configuration files, and instructions on how to run the example.

To learn more about the hyperparameter search algorithm, see the :doc:`Hyperparameter Tuning
</training/hyperparameter/overview>` section.

For faster, less structured ways to run a Determined cluster without writing a model, see:

-  :ref:`commands-and-shells`
-  :ref:`notebooks`
