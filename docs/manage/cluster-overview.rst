.. _cluster-overview:

##########################
 Cluster Overview (WebUI)
##########################

The Cluster Overview page in the WebUI provides a comprehensive view of your Determined cluster's
status, resource utilization, and configuration. This page is accessible to users with appropriate
permissions and offers valuable insights into cluster performance and management.

********************
 Accessing the Page
********************

To access the Cluster Overview:

#. Sign in to the WebUI.
#. From the left navigation pane, select **Cluster**.
#. The overview will be the default view under the Cluster section.

*****************
 Page Components
*****************

The Cluster Overview page consists of several key components:

Resource Utilization
====================

This section displays real-time information about the cluster's resource usage:

-  Total slots available: The total number of compute resources (GPUs/CPUs) in the cluster.
-  Currently active slots: The number of slots currently in use.
-  Percentage of resource utilization: Visualized as a bar chart showing the proportion of used
   slots.

Slots Allocated Bar
-------------------

The slots allocated bar provides a visual representation of resource utilization across the cluster:

-  For the main overview: A large bar shows the overall cluster utilization.
-  For each resource pool: A smaller bar indicates the utilization within that specific pool.

The bar is divided into different colors representing various slot types:

-  Unspecified/Compute Slots (Gray): Used for resource pools with mixed slot types or when the slot
   type is not explicitly defined.
-  CUDA Slots (Green): Represents GPU resources, typically used for CUDA-enabled workloads.
-  CPU Slots (Blue): Indicates CPU resources for CPU-only tasks.

This color-coding helps quickly identify the distribution and usage of different resource types
across your cluster.

Resource Pools
==============

A list of configured resource pools, including:

-  Pool names
-  Number of GPUs/CPUs in each pool
-  Current utilization of each pool

For more details on resource pools, visit :ref:`resource-pools`.

Cluster Topology
================

A visual representation of the cluster's node and GPU distribution:

-  Each node is displayed with its unique identifier
-  The number of available and in-use slots on each node
-  GPU types (if applicable)

To view detailed topology information:

#. Navigate to Resource Pools from the Cluster section.
#. Select a specific Resource Pool.
#. Look for the **Topology** section in the resource pool details page.

Job Queue
=========

An overview of the current job queue, including:

-  Number of queued jobs
-  Job priorities
-  Estimated start times

For more information on managing the job queue, see :ref:`job-queue`.

Cluster Configuration
=====================

Key configuration settings for the cluster, such as:

-  Master node information
-  Scheduler type
-  Version information

*********
 Actions
*********

From the Cluster Overview page, administrators can perform several actions:

-  Modify resource pool settings
-  Adjust job queue priorities
-  Access detailed logs and metrics

For specific instructions on these actions, refer to the respective documentation sections.

*****************
 Troubleshooting
*****************

If you encounter issues or need more information about cluster management, visit the
:ref:`troubleshooting` guide or contact your system administrator.
