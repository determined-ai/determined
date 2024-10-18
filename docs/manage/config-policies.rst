.. _config-policies:

#######################
 Config Policies Guide
#######################

Config Policies allow administrators to set limits on how users can define workloads (e.g.,
experiments, notebooks, tensorboards, shells, and commands). Administrators can include any
parameter referenced in the :ref:`experiment configuration reference <experiment-config-reference>`.

.. include:: ../_shared/attn-enterprise-edition.txt

Config Policies allow admins to define two types of configurations:

-  **Invariant Configs**: Settings applied to all workloads within a scope (global or workspace).
-  **Constraints**: Restrictions that prevent users from exceeding resource limits.

These policies are essential for managing task priorities, resource allocation, and setting specific
configurations across the organization. Global policies override workspace policies, and both
override user-submitted configurations.

**************
 Key Features
**************

-  Apply policies via WebUI or CLI
-  Define limits for resource usage, environment settings, and more
-  Apply policies at different levels (cluster-wide or workspace)
-  Override capabilities for specific user roles or groups
-  Set different priority limits for experiments and NTSC (notebooks, TensorBoards, shells, and
   commands) tasks
-  Allow priority overrides within policy constraints

******************
 Setting Policies
******************

Administrators can set Config Policies at either the cluster or workspace levels through the WebUI
or CLI.

WebUI
=====

Administrators can set Config Policies at both the cluster and workspace levels.

.. tabs::

   .. tab::

      Cluster Config Policy

      To set configuration policies at the cluster level:

      #. Sign in to the Determined WebUI as a cluster administrator.
      #. Navigate to **Config Policies**.
      #. Choose **Experiments** or **Tasks** to display the editable configuration file.
      #. Define the policies, then click **Apply**.
      #. Confirm you want to apply the policies.

   .. tab::

      Workspace Config Policy

      To set configuration policies at the workspace level:

      #. Sign in to the Determined WebUI as a cluster or workspace administrator.
      #. From your workspace, navigate to the **Config Policies** tab.
      #. Choose **Experiments** or **Tasks** to display the editable configuration file.
      #. Define the policies, then click **Apply**.
      #. Confirm you want to apply the policies.

CLI
===

Use the following commands to manage Task Config Policies via CLI:

.. code:: bash

   # List existing policies
   det policy list

   # Create a new policy
   det policy create <policy_name> --config <policy_config_file.yaml>

   # Update an existing policy
   det policy update <policy_name> --config <updated_policy_config_file.yaml>

   # Delete a policy
   det policy delete <policy_name>

*****************
 Policy Examples
*****************

Below are some examples of Config Policies to help guide administrators.

Limiting Resources
==================

Administrators can set constraints on resource usage, allowing them to manage how users allocate
resources for workloads such as experiments, notebooks, tensorboards, shells, and commands. The two
main configurable constraints are:

-  ``resources.max_slots``: Limits the maximum number of slots (GPUs or CPUs) that can be used.
-  ``resources.priority_limit``: Sets the priority limit for tasks.

For Kubernetes resource managers, higher priority values indicate higher priority. For Agent
resource managers, lower priority values indicate higher priority.

**Example 1: Limiting GPU/CPU Slots and Priority for Experiments**

The following example demonstrates how to set a maximum of 4 slots (GPUs or CPUs) and a priority
limit of 15. This configuration applies to both Kubernetes and Agent resource managers, though the
priority system behaves differently across the two:

-  In Kubernetes, higher priority values indicate higher priority.
-  In Agent resource managers, lower priority values indicate higher priority.

.. code:: yaml

   constraints:
     resources:
       max_slots: 4
       priority_limit: 15

In this example: - Users cannot set ``max_slots`` greater than 4. - Users cannot set a priority
lower than 15 for Agent RMs or higher than 15 for Kubernetes RMs.

**Example 2: Limiting Resources for Agent Resource Manager (RM)**

This example limits the number of slots and sets a priority limit specifically for an Agent resource
manager (RM). The ``priority_limit`` is set to ``15``, and ``max_slots`` is set to ``1``. This means
a user cannot set a priority value lower than 15 and cannot set ``max_slots`` greater than 1.

.. note::

   Tasks such as NTSC (notebooks, tensorboards, shells, and commands) and resource managers (RMs)
   have priority levels ranging from 1 to 99. The default cluster priority is typically 42. In Agent
   RMs, a lower priority number means a higher priority, while in Kubernetes RMs, a higher priority
   number means a higher priority.

.. code:: yaml

   constraints:
      priority_limit: 15
      resources:
         max_slots: 1

If a user tries to set a notebook priority to 10 when the limit is 15, the request will fail, and
the system will display an error message.

Limit Maximum GPU Usage per Experiment
======================================

The following configuration policy example limits GPU usage per experiment. This is configured in
the Config Policies > Experiments tab.

.. code:: yaml

   constraints:
     resources:
       max_slots: 4

Priority Limits
===============

You can set a different priority limit for experiments and tasks.

The following policy is configured in the Config Policies > Experiments tab.

.. code:: yaml

   constraints:
     resources:
       priority_limit: <experiment limit>

The following policy is configured in the Config Policies > Tasks tab.

-  Tasks

.. code:: yaml

   constraints:
     resources:
       priority_limit: <task limit>

Invariant Configs
=================

Invariant configs are applied to all workloads in a given scope (global or workspace) and merged
with user-submitted configurations.

**Set Default Priority for All Experiments**

The following configuration policy example sets a default priority for all experiments:

.. code:: yaml

   invariant_config:
     resources:
       priority: 5

**Example Invariant Config for Bind Mounts and Environment Variables**

This example shows how to apply default configurations across all workloads using bind mounts and
environment variables and is set in the Experiments tab:

.. code:: yaml

   invariant_config:
     bind_mounts:
       - host_path: "/etc/bindmounts"
         container_path: "/etc/bindmounts"
         read_only: true
     data:
       datapoint1: value1
     debug: true
     environment:
       environment_variables:
         cpu:
           - cpuval=originalcpu

When a user submits a workload, these settings will be applied in addition to (or overriding) the
user's settings.

.. warning::

   Do not set both constraints and configs for the same field within the same workload type and
   scope. It may lead to unpredictable behavior.

.. important::

   -  Existing workloads are not impacted by new or updated policies.
   -  Updating config policies replaces the entire set of invariant configs and constraints.

**********************
 Priority Enforcement
**********************

If a global or workspace-level policy sets a priority limit, any workload in the relevant scope that
exceeds this limit will be rejected. This ensures that all workloads adhere to the configured
priority constraints.

.. note::

   Higher values indicate higher priority in Kubernetes resource managers, while lower values
   indicate higher priority in Agent resource managers.

****************
 Best Practices
****************

-  Start with cluster-wide policies for broad governance.
-  Use workspace and project-level policies for more granular control.
-  Set different priority limits for experiments and NTSC tasks to better manage resource
   allocation.
-  Regularly review and update policies as organizational needs evolve.
-  Communicate policy changes clearly to users to ensure smooth adoption.
-  Use the SetJobPriority API cautiously, ensuring that priority changes do not violate existing
   policies.
-  Use invariant configs to set default behaviors across workloads.
-  Be mindful of the precedence order: global policies override workspace policies, which override
   user configs.
