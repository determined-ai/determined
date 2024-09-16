.. _cluster-overview:

##########################
 Cluster Overview (WebUI)
##########################

The Cluster Overview page in the WebUI provides a comprehensive view of your Determined cluster's status, resource utilization, and configuration. This page is accessible to users with appropriate permissions and offers valuable insights into cluster performance and management.

********************
 Accessing the Page
********************

To access the Cluster Overview:

1. Sign in to the WebUI.
2. From the left navigation pane, select **Cluster**.
3. The overview will be the default view under the Cluster section.

********************
 Page Components
********************

The Cluster Overview page consists of several key components:

Resource Utilization
====================

This section displays real-time information about the cluster's resource usage:

- Total GPUs/CPUs available
- Currently active GPUs/CPUs
- Percentage of resource utilization

Resource Pools
==============

A list of configured resource pools, including:

- Pool names
- Number of GPUs/CPUs in each pool
- Current utilization of each pool

For more details on resource pools, visit :ref:`resource-pools`.

Cluster Topology
================

A visual representation of the cluster's node and GPU distribution:

- Each node is displayed with its unique identifier
- The number of available and in-use slots on each node
- GPU types (if applicable)

To view detailed topology information:

1. Navigate to Resource Pools from the Cluster section.
2. Select a specific Resource Pool.
3. Look for the **Topology** section in the resource pool details page.

Job Queue
=========

An overview of the current job queue, including:

- Number of queued jobs
- Job priorities
- Estimated start times

For more information on managing the job queue, see :ref:`job-queue`.

Cluster Configuration
=====================

Key configuration settings for the cluster, such as:

- Master node information
- Scheduler type
- Version information

*****************
 Actions
*****************

From the Cluster Overview page, administrators can perform several actions:

- Modify resource pool settings
- Adjust job queue priorities
- Access detailed logs and metrics

For specific instructions on these actions, refer to the respective documentation sections.

*****************
 Troubleshooting
*****************

If you encounter issues or need more information about cluster management, visit the :ref:`troubleshooting` guide or contact your system administrator.