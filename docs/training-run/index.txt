.. _experiments:

###################
 Run Training Code
###################

In this document, we will cover how to run your training code. You can submit your code to a cluster
and run them as experiments. We will go over the whole cycle of it.

**********************
 Create an Experiment
**********************

Currently, we only support use the CLI to create an experiment from scratch. You can alternatively
create an experiment from an existing experiment or trial on the WebUI. The CLI command to create an
experiment is as follows:

.. code::

   $ det experiment create <configuration file> <context directory>

The ``det experiment create`` command requires two arguments:

-  The :ref:`configuration file <experiment-configuration>` is a :ref:`YAML <topic-guides_yaml>`
   file that configures the experiment.
-  The context directory contains all code that are relevant to training and will be uploaded to the
   master.

We don't allow the total size of the files in the context to exceed 95 MiB. As a result, datasets
should typically not be included unless they are very small; instead, users should set up data
loaders to read data from an external source. Refer to :ref:`preparing data <prepare-data>` for more
suggestions on data loading.

Since project directories might include large artifacts that should not be packaged as part of the
model definition (e.g., data sets or compiled binaries), users can optionally include a
``.detignore`` file at the top level that specifies file paths to be omitted from the model
definition. The ``.detignore`` file uses the same syntax as `.gitignore
<https://git-scm.com/docs/gitignore>`__. Note that byte-compiled Python files (e.g., ``.pyc`` files
or ``__pycache__`` directories) are always ignored.

Local Test Mode
===============

The local test mode is to sanity-check your training code and run a compressed version of the full
experiment circle. It can help you debug the errors in your code without running the full experiment
circle that is expensive in the resources it takes and sometimes needs a long time to get ready. It
helps with quick iteration on the code. You can run the following command:

.. code::

   det experiment create --local --test-mode <configuration file> <context directory>

***********
 Get Ready
***********

The trials are created to train the model. The :ref:`searcher <hyperparameter-tuning>` described by
the experiment configuration defines a set of hyperparameter configurations, each of which
corresponds to one trial.

Once the context and configuration for an experiment have reached the master, the experiment waits
for the scheduler to assign slots to it. If there are no enough idle slots, the master will scale up
the resource pools automatically.

When a trial is ready to run, the master communicates with the appropriate agent (or agents, in the
case of :ref:`distributed training <multi-gpu-training>`), which creates containers with the
configured environment and the submitted training code. We supply a set of default container images
that are appropriate for many deep learning tasks, but users can also supply a :ref:`custom image
<custom-docker-images>` if desired. If the specified container images are not existent locally, the
trial container will fetch the images from the registry.

After starting the containers, each trial runs the ``startup-hook.sh`` that exists in the context
directory.

However, this whole process might take very long before each trial starts training.

****************
 Start Training
****************

To start training, we load and run the entrypoint, which is the user-defined Python class and
specified in the experiment configuration.

The user-provided class must be a subclass of a trial class included in Determined. Each trial class
is designed to support one deep learning application framework. While training or validating models,
the trial may need to load data from an external source therefore the training code needs to define
data loaders.

Our library automatically communicates with the master to get what needs to run and report back the
results. It might:

#. train the model for a few batches on the training dataset;
#. checkpoint the states of the model and other objects;
#. validate the model on the validation dataset.

********************
 Pause and Activate
********************

An important feature of Determined is the ability to have trials stop running and then start again
later without losing any training progress. The scheduler might choose to stop running a trial to
allow a trial from another experiment to run, but a user can also manually pause an experiment at
any time, which causes all of its trials to stop.

Checkpointing is essential to this ability. After a trial is set to be stopped, it takes a
checkpoint at the next available opportunity (i.e., once its current workload finishes running) and
then stops running, freeing up the slots it was using. When it resumes running, either because more
slots become available in the cluster or because a user activates the experiment, it loads the saved
checkpoint, allowing it to continue training from the same state it had before.
