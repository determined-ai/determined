.. _job-queue:

########################
 Managing the Job Queue
########################

***********************
 Viewing the Job Queue
***********************

The Determined Queue Management system enhances scheduler functionality, providing better visibility
and control over job scheduling decisions. The Job Queue displays job ordering and allows dynamic
job modifications.

Queue Management is available for both the fair share scheduler and the priority scheduler. This
feature shows all submitted jobs and their states, and allows you to modify configurations like
priority, queue position, and resource pool.

To manage job queues, navigate to the WebUI ``Job Queue`` section or use the ``det job`` CLI
commands.

Jobs in the queue can be in either the ``Queued`` or ``Scheduled`` state:

-  ``Queued``: Job received but resources not allocated.
-  ``Scheduled``: Scheduled to run or running, with resources possibly allocated.

Completed or errored jobs are not considered active and are omitted from the list.

You can view the job queue using the CLI or WebUI.

-  In the WebUI, click the **Job Queue** tab.
-  In the CLI, use the following commands:

.. code::

   $ det job list
   $ det job ls

To view other resource pool queues, use the ``--resource-pool`` option:

.. code::

   $ det job list --resource-pool compute-pool

For more CLI options, visit the CLI documentation or run the ``det job list -h`` command.

Both the WebUI and CLI display a table of jobs, ordered by scheduling order. The table includes job
states and the number of slots allocated to each job. Note that scheduling order does not represent
job priority.

*************************
 Modifying the Job Queue
*************************

You can modify the job queue in the WebUI **Job Queue** section or using the ``det job update`` CLI
command. Changes can be made on a per-job basis by selecting a job and performing an operation.
Available operations include:

-  Changing priorities for resource pools (priority scheduler)
-  Changing weights for resource pools (fair share scheduler)
-  Changing resource pools

Constraints:

-  The priority and fair share fields are mutually exclusive. The priority field is active only for
   the priority scheduler, and the fair share field is active only for the fair share scheduler.
-  The change resource pool operation can only be performed on experiments. For other tasks, cancel
   and resubmit the task to change the resource pool.

Modify the Job Queue using the WebUI
====================================

#. Go to the **Job Queue** section.
#. Find the job to modify.
#. Click the three dots in the right-most column of the job.
#. Click the **Manage Job** option.
#. Make your changes on the pop-up page, and click **OK**.

.. _modify-job-queue-cli:

Modify the Job Queue using the CLI
==================================

To modify the job queue using the CLI, use the ``det job update`` command. Run ``det job update
--help`` for more information. Example operations:

.. code::

   $ det job update jobID --priority 10
   $ det job update jobID --resource-pool a100

To update multiple jobs in a batch, provide updates as shown:

.. code::

   $ det job update-batch job1.priority=1 job2.resource-pool="compute"

Example workflow:

.. code::

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
