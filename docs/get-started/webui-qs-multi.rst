.. _qs-webui-multi:

#############################
 Run a Hyperparameter Search
#############################

.. meta::
   :description: Learn how to run your first multi-trial experiment, or search, in Determined.
   :keywords: PyTorch API,MNIST,model developer,quickstart,search

Follow these steps to see how to run your first search in Determined.

A multi-trial search (or hyperparameter search) allows you to optimize your model by exploring
different configurations of hyperparameters automatically. This is more efficient than manually
tuning each parameter. In this guide, we'll show you how to modify the existing ``const.yaml``
configuration file used in the single-trial experiment to run a multi-trial search.

**Now that we have established a baseline performance by creating our single-trial experiment, we
can create a search (multi-trial experiment) and compare the outcome with our baseline. We hope to
see improvements gained through hyperparameter tuning and optimization.**

***************
 Prerequisites
***************

You must have a running Determined cluster with the CLI installed.

-  To set up a local cluster, visit :ref:`basic`.
-  To set up a remote cluster, visit the :ref:`Installation Guide <installation-guide>` where you'll
   find options for On Prem, AWS, GCP, Kubernetes, and Slurm.

.. note::

   Visit :ref:`qs-webui` to learn how to run your first single-trial experiment in Determined.

*********************************
 Prepare Your Configuration File
*********************************

In our single-trial experiment, our ``const.yaml`` file looks something like this:

.. code:: yaml

   name: mnist_pytorch_const
   hyperparameters:
      learning_rate: 1.0
      n_filters1: 32
      n_filters2: 64
      dropout1: 0.25
      dropout2: 0.5
   searcher:
      name: single
      metric: validation_loss
      max_length:
         batches: 1000  # approximately 1 epoch
      smaller_is_better: true
   entrypoint: python3 train.py

To convert this into a multi-trial search, we will need to modify the hyperparameters section and
the searcher configuration. We'll tell Determined to use Random Search which randomly selects values
from the specified ranges and set ``max_trials`` to 20.

Copy the following code and save the file as ``search.yaml`` in the same directory as your
``const.yaml`` file:

.. code:: yaml

   name: mnist_pytorch_search
   hyperparameters:
     learning_rate:
       type: log
       base: 10
       minval: 1e-4
       maxval: 1.0
     n_filters1:
       type: int
       minval: 16
       maxval: 64
     n_filters2:
       type: int
       minval: 32
       maxval: 128
     dropout1:
       type: double
       minval: 0.2
       maxval: 0.5
     dropout2:
       type: double
       minval: 0.3
       maxval: 0.6

   searcher:
     name: random
     metric: validation_loss
     max_trials: 20
     max_length:
       batches: 1000
     smaller_is_better: true

   entrypoint: python3 train.py

*******************
 Create the Search
*******************

Once you've created the new configuration file, you can create and run the search using the
following command:

.. code:: bash

   det experiment create search.yaml .

This will start the search, and Determined will run multiple trials, each with a different
combination of hyperparameters from the defined ranges.

********************
 Monitor the Search
********************

In the WebUI, navigate to the **Searches** tab to monitor the progress of your search. You’ll be
able to see the different trials running, their status, and their performance metrics. Determined
also offers built-in visualizations to help you understand the results.

   .. image:: /assets/images/qswebui-multi-trial-search.png
      :alt: Determined AI WebUI Dashboard showing a user's recent multi-trial search

*********************
 Analyze the Results
*********************

After the search is complete, you can review the best-performing trials and the hyperparameter
configurations that led to them. This will help you identify the optimal settings for your model.

Select **mnist_pytorch_search** to view all runs including single-trial experiments. Then choose
which runs you want to compare.

   .. image:: /assets/images/qswebui-mnist-pytorch-search.png
      :alt: Determined AI WebUI Dashboard with mnist pytorch search selected and ready to compare

************
 Go Further
************

Once you've mastered the basics, you can take your experiments to the next level by exploring more
advanced configurations. In this section, we'll cover how to run two additional configurations:
`dist_random.yaml` and `adaptive.yaml`. These examples introduce new concepts such as distributed
training and adaptive hyperparameter search methods.

Running `dist_random.yaml`
==========================

To run the distributed random search experiment, use the following command:

.. code:: bash

   det experiment create dist_random.yaml .

Running `adaptive.yaml`
=======================

To run the adaptive search experiment, use the following command:

.. code:: bash

   det experiment create adaptive.yaml .

These advanced configurations allow you to scale your experiments and optimize your model
performance more efficiently. As you become more comfortable with these concepts, you’ll be able to
leverage the full power of Determined for more complex machine learning workflows.

**************
 Key Concepts
**************

This section provides an overview of the key concepts you’ll need to understand when working with
Determined, particularly when running single-trial and multi-trial experiments.

Single-Trial Experiment (Run)
=============================

-  **Definition:** A single-trial experiment (or run) allows you to establish a baseline performance
   for your model.

-  **Purpose:** Running a single trial is useful for understanding how your model performs with a
   fixed set of hyperparameters. It serves as a benchmark against which you can compare results from
   more complex searches.

Multi-Trial Experiment (Search)
===============================

-  **Definition:** A multi-trial experiment (or search) allows you to optimize your model by
   exploring different configurations of hyperparameters automatically.
-  **Purpose:** A search systematically tests various hyperparameter combinations to find the
   best-performing configuration. This is more efficient than manually tuning each parameter.

Searcher
========

-  **Random Search:** Randomly samples hyperparameters from the specified ranges for each trial. It
   is straightforward and provides a simple way to explore a large search space.

-  **Adaptive ASHA:** Uses an adaptive algorithm to allocate resources dynamically to the most
   promising trials. It starts many trials but continues only those that show early success,
   optimizing resource usage.

Resource Allocation
===================

-  **Distributed Training:** Involves training your model across multiple GPUs (or CPUs) to speed up
   the process. This is particularly useful for large models or large datasets.
-  **Slots Per Trial:** Specifies the number of GPUs (or CPUs) each trial will use. For example,
   setting `slots_per_trial: 1` means each trial will use one GPU or CPU.

Metrics
=======

-  **Validation Loss:** A common metric used to evaluate the performance of a model during training.
   Lower validation loss usually indicates a better model.

-  **Accuracy:** Measures how often the model correctly predicts the target variable. It is
   typically used for classification tasks where you want to maximize the number of correct
   predictions.

Baseline Performance
====================

-  **Establishing a Baseline:** Before running a search, it's important to establish a baseline
   performance using a single-trial experiment. This gives you a reference point to compare the
   results of your multi-trial searches.

-  **Comparison in Run Tab:** Once you have established a baseline performance, you can create a
   search and compare all outcomes in the Run tab. This helps you determine the effectiveness of
   different hyperparameter configurations.
