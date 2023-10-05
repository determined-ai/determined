:orphan:

.. _some-anchor-name:

###########################################
 Get Started with Determined and Pachyderm
###########################################

.. meta::
   :description: Learn how to use Determined and Pachyderm together to classify images of cats and dogs.

:ref:`test2 <det-pach-cat-dog>`

In this user guide, we'll show you how to create a pipeline, add data, train models, download
checkpoints, create an inferencing pipeline and output a scatterplot. As an example, we'll be
working with datasets derived from a CNN dog/cat image classification dataset.

Problem statement

We are given a set of dog and cat images. The task is to build a model to predict the category of an
animal: dog or cat?

Dataset Description

xyz

etc

************
 Objectives
************

These step-by-step instructions walk you through SOMETHING for the purpose of performing the
following functions:

-  Installing Determined and Pachyderm
-  Ensuring Determined and Pachyderm are connected
-  Create a batch inferencing project
-  Create repos and pipelines for training data
-  Add data to the repos
-  Use the data to train our models
-  Download the best checkpoint and put it in a repo
-  Create a repo and pipeline for inferencing
-  Add files for batch inferencing
-  Add a results pipeline

Advanced: We will try adding notebook cells to do some of the following: - Report metrics - Report
checkpoints - Perform a hyperparameter search - Perform distributed training

After completing the steps in this user guide, you will be able to:

-  Understand how Determined and Pachyderm work together
-  Modify something
-  Understand how to do something
-  Use the Determined and Pachyderm to create a pipeline and train a model

***************
 Prerequisites
***************

**Required**

UNLESS THESE ARE PART OF STEP 0

-  Docker and Kubernetes

   -  To install Docker and enable Kubernetes, do this.
   -  Then start Docker and enable Kubernetes.

-  A running Determined cluster

   -  To install Determined, visit the Quick Install.
   -  Test your installation.

-  A running Pachyderm cluster

   -  To install Pachyderm, visit the Get Started.
   -  Test your installation.

**Recommended**

-  :ref:`qs-mdldev`
-  Pachyderm Beginner Tutorial https://docs.pachyderm.com/latest/get-started/beginner-tutorial/

**************
 Introduction
**************

This hands-on tutorial showcases the potential of Determined and Pachyderm together. In this
tutorial, we'll be working with a convolutional neural network (CNN) using PyTorch for image
classification.

Why is this relevant to you? Well, consider this scenario: You have vast datasets comprising images
of cats and dogs, and you need to set up an efficient, reproducible pipeline for training a model on
this data. This is where Pachyderm comes into play, offering you the ability to build a robust data
pipeline. You'll establish repositories for training and testing data, ensuring that your datasets
are versioned and easily manipulated if needed.

Once the data is in place, Determined steps in. Using a straightforward yaml configuration, you'll
instruct your Determined cluster to train the model based on the meticulously versioned data from
Pachyderm. Here, Determined simplifies tasks like downloading checkpoints, ensuring that your
workflow remains smooth and hitch-free.

But the journey doesn't end there. After training your model, you'll circle back to Pachyderm,
crafting a pipeline optimized for batch predictions. With the power to scale out on-demand by
adjusting the parallelism_spec value, you'll see firsthand how you can add an assortment of files to
the prediction repository at any given time. The highlight? Your pipelines will generate visually
appealing output images, providing a tangible result to all the hard work.

Embarking on this tutorial ensures not just familiarity with Determined and Pachyderm but a deep
understanding of how, when used together, they can streamline complex tasks. So, are you ready to
see a perfect blend of data versioning, model training, and scalable predictions come to life? Let's
get started!

*******************************************************
 Step x: Verify Determined and Pachyderm are Installed
*******************************************************

Before creating our project, we'll first verify that Determined and Pachyderm are installed and
connected.

CD into the directory and run this command:

.. code:: bash

   det version
   pachctl version

The system responds by letting you know the version you have installed.

This lets us know the installation is

********************************
 Step x: Get the Tutorial Files
********************************

To create a project in Pachyderm, you need, at minimum, xyz. To run an experiment in Determined, you
need, at minimum, a script and an experiment configuration (YAML) file.

Create a new directory.

Access the tutorial files directly from the `Github repository
<https://github.com/pachyderm/examples/tree/master/determined-pachyderm-batch-inferencing>`_.

The repository contains a ``setup.ipynb`` notebook where you can run the steps, or you can follow
the steps outline in this tutorial and copy/paste the commands into a terminal window. When running
the cells in the notebook, do not run the entire notebook. You'll need to run the steps individually
and then wait for the task to complete before running the next step.

Review the objectives, and what will happen next: - - - - - - -

*************************************************************
 Step x: Create a Project for Batch Inferencing in Pachyderm
*************************************************************

In this initial step, we'll set up a project in Pachyderm to group together all our training and
inferencing repositories and pipelines.

code

**********************************************
 Step x: Create Repos for Our Train/Test Data
**********************************************

In this step, we'll set up separate repositories for our 80:20 train/test split so we can maintain a
structured approach to our data pipeline.

code

*****************************************
 Step x: Create a Data Training Pipeline
*****************************************

In this step, we'll merge the data from our train/test repos and compress them into a tar file for
easy data access.

.. note::

   This would be a good place to perform data cleanup or data transformations as we prepare our data
   for model training.

code

*******************************************
 Step x: Add Files to the Train/Test Repos
*******************************************

With our compress pipeline in place, we can add files to our separate train/test repos. The compress
pipeline will compress the files and create a single tar file.

code

***********************************************************
 Step x: Create a Determined Experiment to Train Our Model
***********************************************************

In this step, we’ll create an experiment using the ``train.py`` script and its accompanying
``train.yaml`` experiment configuration file.

Determined needs to know where to download the data we want to use to train our model. To do this,
we'll need to provide the Pachyderm host, port, project, repo, and branch. To accomplish this, we'll
use the experiment configuration file, ``train.yaml``.

Get the Name of the Pachyderm Host and Port
===========================================

You'll first need to get the name of the Pachyderm host and port:

???

Edit the ``train.yaml`` File
============================

Edit the ``train.yaml`` file with your host and port name.

???

View the Experiment Configuration File
======================================

View the ``train.yaml`` file that contains the configuration settings to be sure it is accurate.

.. code:: bash

   cat ./determined/train.yaml

View the Experiment Configuration File
======================================

.. code:: bash

   det e create ./determined/train.yaml -f

   or

   det e create ./determined/train.yaml ./determined --config data.pachyderm -f

.. note::

   ``det e create const.yaml . -f`` instructs Determined to follow the logs of the first trial that
   is created as part of the experiment. The command will stay active and display the live output
   from the logs of the first trial as it progresses.

Open the Determined WebUI by navigating to the master URL. One way to do this is to navigate to
``http://localhost:8080/``, accept the default username of ``determined``, and click **Sign In**. A
password is not required.

include the shared note-local-dtrain-job text file link here

In the WebUI, select your experiment. You'll notice the tabs do not yet contain any information. In

the next section, we'll download checkpoints (for the purpose of SOMETHING).

**********************************************************************
 Step x: Add a Step Here About Model Validation and Model Performance
**********************************************************************

I don't know what we should add here. Maybe we just do the checkpointing and hyperparameter search
sections.

code

***********************
 Step 3: Checkpointing
***********************

Checkpointing periodically during training and reporting the checkpoints to the master gives us the
ability to stop and restart training. In this section, we’ll modify our script for the purpose of
checkpointing.

In this step, we’ll run our experiment using the ``model_def_checkpoints.py`` script and its
accompanying ``checkpoints.yaml`` experiment configuration file.

include the shared note-premade-tutorial-script

Step 3.1: Save Checkpoints
==========================

To save checkpoints, add the ``store_path`` function to your script:

literal include

Step 3.3: Run the Experiment
============================

Run the following command to run the experiment:

.. code:: bash

   det e create checkpoints.yaml . -f

In the Determined WebUI, nagivate to the **Checkpoints** tab.

Checkpoints are saved and deleted according to the default
:ref:`experiment-config-checkpoint-policy`. You can modify the checkpoint policy in the experiment
configuration file.

*******************************
 Step 4: Hyperparameter Search
*******************************

With the Core API you can run advanced hyperparameter searches with arbitrary training code. The
hyperparameter search logic is in the master, which coordinates many different Trials. Each trial
runs a train-validate-report loop:

.. table::

   +----------+--------------------------------------------------------------------------+
   | Train    | Train until a point chosen by the hyperparameter search algorithm and    |
   |          | obtained via the Core API.  The length of training is absolute, so you   |
   |          | have to keep track of how much you have already trained to know how much |
   |          | more to train.                                                           |
   +----------+--------------------------------------------------------------------------+
   | Validate | Validate your model to obtain the metric you configured in the           |
   |          | ``searcher.metric`` field of your experiment config.                     |
   +----------+--------------------------------------------------------------------------+
   | Report   | Use the Core API to report results to the master.                        |
   +----------+--------------------------------------------------------------------------+

To perform a hyperparameter search, we'll update our script to define the hyperparameter search
settings we want to use for our experiment. More specifically, we'll need to define the following
settings in our experiment configuration file:

-  ``name:`` ``adaptive_asha`` (name of our searcher. For all options, visit :ref:`search-methods`.

-  ``metric``: ``test_loss``

-  ``smaller_is_better``: ``True`` (This is equivalent to minimization vs. maximization of
   objective.)

-  ``max_trials``: 500 (This is the maximum number of trials the searcher should run.)

-  ``max_length``: 20 epochs (The max length of a trial. For more information, visit Adaptive ASHA
   in the :doc:`Experiment Configuration Reference
   </reference/training/experiment-config-reference>`.

In addition, we also need to define the hyperparameters themselves. Adaptive ASHA will pick values
between the ``minval`` and ``maxval`` for each hyperparameter for each trial.

.. note::

   To see early stopping in action, try setting ``max_trials`` to over 500 and playing around with
   the hyperparameter search values.

In this step, we’ll run our experiment using the ``model_def_adaptive.py`` script and its
accompanying ``adaptive.yaml`` experiment configuration file.

include the shared note note-premade-tutorial-script

Begin by accessing the hyperparameters in your code:

Step 4.1: Run the Experiment
============================

Run the following command to run the experiment:

.. code:: bash

   det e create adaptive.yaml .

In the Determined WebUI, navigate to the **Hyperparameters** tab.

You should see a graph in the WebUI that displays the various trials initiated by the Adaptive ASHA
hyperparameter search algorithm.

*************************************************************************************
 Step x: Download the Best Checkpoint from Determined and Add it to a Pachyderm Repo
*************************************************************************************

Now that we are happy with our model's performance

************
 Next Steps
************

In this user guide, you learned how to use the Core API to integrate a model into Determined. You
also saw how to modify a training script and use the appropriate configuration file to report
metrics and checkpointing, perform a hyperparameter search, and run distributed training.

include the shared note note-dtrain-learn-more

what's next?

You can adapt the steps in this tutorial to efficiently build and train large machine learning
models that require a high volume of complex data.
