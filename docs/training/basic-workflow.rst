.. _experiments:

###################
 Run Training Code
###################

In this document, we will cover how to run your training code. You can submit your code to a cluster
and run them as experiments. We will go over the whole cycle of it.

**********************
 Create an Experiment
**********************

Currently, we only support the use of the CLI to create an experiment from scratch. You can
alternatively create an experiment from an existing experiment or trial on the WebUI. The CLI
command to create an experiment is as follows:

.. code::

   $ det experiment create <configuration file> <context directory>

The ``det experiment create`` command requires two arguments:

-  The :ref:`configuration file <experiment-config-reference>` is a :ref:`YAML <topic-guides_yaml>`
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

***************
Local Test Mode
***************

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

.. _job-queue:

********************
 View the Job Queue
********************

The Determined Queue Management system extends scheduler functionality to offer better visibility
and control over scheduling decisions. It does this using the Job Queue, which provides better
information about job ordering, such as which jobs are queued, and permits dynamic job modification.

Queue Management is a new feature that is available to the fair share scheduler and the priority
scheduler. Queue Management, described in detail in the following sections, shows all submitted jobs
and their states, and lets you modify some configuration options, such as priority, position in the
queue, and resource pool.

To begin managing job queues, navigate to the WebUI ``Job Queue`` section or use the ``det job`` set
of CLI commands.

Queued jobs can be in the ``Queued`` or ``Scheduled`` state:

-  ``Queued``: Job received but resources not allocated
-  ``Scheduled``: Scheduled to run or running, and resources may have been allocated.

Completed or errored jobs are not counted as active and are omitted from this list.

You can view the job queue using the CLI or WebUI. In the WebUI, click the **Job Queue** tab. In the
CLI, use one of the following commands:

.. code::

   $ det job list
   $ det job ls

These commands show the default resource pool queue. To view other resource pool queues, use the
``--resource-pool`` option, specifying the pool:

.. code::

   $ det job list --resource-pool compute-pool

For more information about the CLI options, see the CLI documentation or use the ``det job list -h``
command.

The WebUI and the CLI display a table of results, ordered by scheduling order. The scheduling order
does not represent the job priority. In addition to job order, the table includes the job states and
number of slots allocated to each job.

**********************
 Modify the Job Queue
**********************

The job queue can be changed in the WebUI **Job Queue** section or by using the CLI ``det job
update`` command. You can make changes on a per-job basis by selecting a job and a job operation.
Available operations include:

-  changing priorities for resource pools using the priority scheduler
-  changing weights for resource pools using the fair share scheduler
-  changing the order of queued jobs
-  changing resource pools

There are a number of constraints associated with using the job queue to modify jobs:

-  The priority and fair share fields are mutually exclusive. The priority field is only active for
   the priority scheduler and the fair share field is only active for the fair share scheduler. It
   is not possible for both to be active simultaneously.

-  The ``ahead-of``, ``behind-of``, and WebUI **Move to Top** operations are only available for the
   priority scheduler and are not possible with the fair share scheduler. These operations are not
   yet fully supported for the Kubernetes priority scheduler.

-  The change resource pool operation can only be performed on experiments. To change the resource
   pool of other tasks, cancel the task and resubmit it.

Modify the Job Queue using the WebUI
====================================

To modify the job queue in the Webui,

#. Go to the **Job Queue** section.
#. Find the job to modify.
#. Click the three dots in the right-most column of the job.
#. Find and click the **Manage Job** option.
#. Make the change you want on the pop-up page, and click **OK**.

Modify the Job Queue using the CLI
====================================

To modify the job queue in the CLI, use the ``det job update`` command. Run ``det job update
--help`` for more information. Example operations:

.. code::

   $ det job update jobID --priority 10
   $ det job update jobID --resource-pool a100
   $ det job update jobID --ahead-of jobID-2

To update a job in batch, provide updates as shown:

.. code::

   $ det job update-batch job1.priority=1 job2.resource-pool="compute" job3.ahead-of=job1

Example workflow:

.. code::

   $ det job list
      # | ID       | Type            | Job Name   | Priority | Submitted            | Slots (acquired/needed) | Status          | User
   -----+--------------------------------------+-----------------+--------------------------+------------+---------------------------+---------
      0 | 0d714127 | TYPE_EXPERIMENT | first_job  |       42 | 2022-01-01 00:01:00  | 1/1                     | STATE_SCHEDULED | user1
      1 | 73853c5c | TYPE_EXPERIMENT | second_job |       42 | 2022-01-01 00:01:01  | 0/1                     | STATE_QUEUED    | user1

   $ det job update 73853c5c --ahead-of 0d714127

   $ det job list
      # | ID       | Type            | Job Name   | Priority | Submitted            | Slots (acquired/needed) | Status          | User
   -----+--------------------------------------+-----------------+--------------------------+------------+---------------------------+---------
      0 | 73853c5c | TYPE_EXPERIMENT | second_job |       42 | 2022-01-01 00:01:01  | 1/1                     | STATE_SCHEDULED | user1
      1 | 0d714127 | TYPE_EXPERIMENT | first_job  |       42 | 2022-01-01 00:01:00  | 0/1                     | STATE_QUEUED    | user1

   $ det job update-batch 73853c5c.priority=1 0d714127.priority=1

   $ det job list
      # | ID       | Type            | Job Name   | Priority | Submitted            | Slots (acquired/needed) | Status          | User
   -----+--------------------------------------+-----------------+--------------------------+------------+---------------------------+---------
      0 | 73853c5c | TYPE_EXPERIMENT | second_job |       1 | 2022-01-01 00:01:01  | 1/1                     | STATE_SCHEDULED | user1
      1 | 0d714127 | TYPE_EXPERIMENT | first_job  |       1 | 2022-01-01 00:01:00  | 0/1                     | STATE_QUEUED    | user1
