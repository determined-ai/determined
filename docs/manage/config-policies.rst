.. _config-policies:

#######################
 Config Policies Guide
#######################

Config Policies allow administrators to set limits on how users can define workloads (e.g.,
experiments and notebooks, tensorboards, shells, and commands). This feature enables administrators
to govern user behavior more closely at the workspace and cluster level. Administrators can include
any parameter referenced in the :ref:`experiment configuration reference
<experiment-config-reference>`.

.. include:: ../_shared/attn-enterprise-edition.txt

When implementing Config Policies, administrators should be aware of the following:

#. Separate Priority Limits: Administrators can now set different priority limits for experiments
   and NTSC (notebooks, tensorboards, shells, and commands) tasks. This allows for more granular
   control over task prioritization and resource allocation.

#. Priority Overrides: If a given scope has an invariant config policy set for a given task, and
   that invariant configuration policy specifies ``priority``, this priority can still be overridden
   using the ``SetJobPriority`` API endpoint. However, the new priority must not violate any
   constraints defined in the applicable policies for that scope.

These features provide administrators with greater flexibility in managing task priorities while
still maintaining policy-based control over resource usage.

**************
 Key Features
**************

-  Set policies via WebUI or CLI
-  Define limits for resource usage, environment settings, and more
-  Apply policies at different levels (cluster-wide or workspace)
-  Override capabilities for specific user roles or groups
-  Set different priority limits for experiments and NTSC (notebooks, TensorBoards, shells, and
   commands) tasks
-  Allow priority overrides within policy constraints

******************
 Setting Policies
******************

Administrators can set Config Policies using either the WebUI or the CLI.

WebUI
=====

Administrators can set Config Policies at either the cluster or workspace levels.

.. tabs::

   .. tab::

      Cluster Config Policy

      To set configuration policies at the cluster level, follow these steps:

      #. Sign in to the Determined WebUI as a cluster administrator.
      #. Navigate to **Config Policies**.
      #. Choose **Experiments** or **Tasks** to display the editable configuration file.
      #. Define the policies, then click **Apply**.
      #. Confirm you want to apply the policies.

   .. tab::

      Workspace Config Policy

      To set configuration policies at the workspace level, follow these steps:

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

Here are some example policies:

**Limit resources**

In the following example for an Agent resource manager (RM), the ``priority_limit`` is set to ``15``
and ``max_slots`` is set to ``1``. This means a user cannot set a priority value lower than 15 and
cannot set ``max_slots`` to greater than ``1``.

.. note::

   Tasks (NTSC - notebooks, Tensorboards, shells, and commands) and some resource managers (RMs)
   have a priority. Priority levels are between 1 and 99. The default cluster priority is usually
   42. The priority limit determines the order in which queued tasks run. In this example we are
   using limits because for the agent resource manager, a lower number means a higher priority. For
   a Kubernetes resource manager, a higher number means a higher priority.

.. code:: bash

   # Set priority limit to 15
   constraints:
      priority_limit: 15
      resources:
         max_slots: 1

If your priority limits are above the defaults, for example if your notebook priority is set to 10
and the limit is 15, then the request fails and the system displays a descriptive message.

**Limit maximum GPU usage per experiment**

-  Experiments

.. code:: yaml

   constraints:
     resources:
       max_slots: 4

-  Tasks

.. code:: yaml

   name: max-gpu-limit
   scope: cluster
   config:
     resources:
       max_gpus: 4

**Set different priority limits for experiments and NTSC tasks**

-  Experiments

.. code:: yaml

   constraints:
     resources:
       priority_limit: <experiment limit>

-  Tasks

.. code:: yaml

   constraints:
     resources:
       priority_limit: <task limit>

**Set default priority for all tasks**

To set a default priority for all tasks, provide the following code in both the Experiments and Task
tabs:

.. code:: yaml

   invariant_config:
     resources:
       priority: 5

*******************
 Priority Override
*******************

If a given scope has the priority field of a default invariant config set, this priority can still
be overridden using the SetJobPriority API endpoint. However, the new priority must not violate any
constraints placed on the scope.

****************
 Best Practices
****************

-  Start with cluster-wide policies for general governance.
-  Use workspace and project-level policies for more granular control.
-  Set different priority limits for experiments and NTSC tasks to better manage resource
   allocation.
-  Regularly review and update policies as organizational needs change.
-  Communicate policy changes to users to ensure smooth adoption.
-  Use the SetJobPriority API endpoint cautiously, ensuring that priority changes do not violate
   existing policies.
-  Use invariant configs to set default behaviors across workloads.
-  Be aware of the precedence order when setting policies at different levels.
-  Test your policies thoroughly to ensure they behave as expected, especially when combining global
   and workspace policies.
-  When updating policies, consider the impact on running workloads and communicate changes clearly
   to users.

Config policies follow a precedence order:

#. Global config policies
#. Workspace config policies
#. User-submitted configs

Global policies override workspace policies, and workspace policies override user configs. For
iterative fields (such as arrays and maps), data is merged rather than completely overridden.

.. note::

   Only users with the ClusterAdmin role can modify global config policies. ClusterAdmin and
   WorkspaceAdmin roles can modify workspace config policies.

Limit resources
===============

Administrators can set constraints on resource usage. The two main configurable constraints are:

-  ``resources.max_slots``: Limits the maximum number of slots (GPUs or CPUs) that can be used.
-  ``resources.priority_limit``: Sets the priority limit for tasks.

.. note::

   For Kubernetes resource managers, higher priority values indicate higher priority. For Agent
   resource managers, lower priority values indicate higher priority.

Here's an example of setting resource constraints:

.. code:: yaml

   constraints:
     resources:
       max_slots: 4
       priority_limit: 15

In this example, users cannot set ``max_slots`` to a value greater than 4, and they cannot set a
priority lower than 15 (for Agent RMs) or higher than 15 (for Kubernetes RMs).

Invariant Configs
=================

Invariant configs are settings that are applied to all workloads in a given scope (global or
workspace). These configs are merged with user-submitted configs according to the precedence rules
mentioned earlier.

**Invariant config example**

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

**Complete example with invariant_configs and constraints**

.. code:: yaml

   name: complete-policy-example
   scope: workspace
   workspace: research
   config:
     invariant_config:
       environment:
         image: "docker.io/determined/environments:cuda-11.3-pytorch-1.10-tf-2.8-gpu-0.19.4"
       resources:
         slots: 2
     constraints:
       resources:
         max_slots: 4
         priority_limit: 20

This example sets a default Docker image and number of slots for all workloads in the "research"
workspace, while also limiting the maximum number of slots and setting a priority limit.

.. note::

   When setting config policies, existing workloads will not be impacted. Additionally, when
   updating config policies, the entire set of invariant_configs and constraints will replace any
   existing policies.

.. warning::

   It is not recommended to set constraints and configs for the same field for the same workload
   type and scope (e.g., setting both invariant_config.resources.priority and
   constraints.resources.priority_limit).

.. important::

   When setting or updating config policies, keep in mind:

   -  Existing workloads will not be impacted by new or updated policies.
   -  When updating config policies, the entire set of invariant_configs and constraints will
      replace any existing policies.
   -  It's not recommended to set both constraints and configs for the same field, workload type,
      and scope.
