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

**Constraints and Limitations**

-  Priority Overrides: If a given scope has an invariant config policy set for a given task, and
   that invariant configuration policy specifies ``priority``, this priority can still be overridden
   using the ``SetJobPriority`` API endpoint. However, the new priority must not violate any
   constraints defined in the applicable policies for that scope.

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

Below are some examples of Config Policies to help guide administrators:

**Limit Resources**

This example demonstrates limiting resources for an Agent resource manager (RM). It sets the
``priority_limit`` to ``15`` and ``max_slots`` to ``1``. This means a user cannot set a priority
value lower than 15 and cannot set ``max_slots`` to greater than ``1``.

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

If a user attempts to set a notebook priority to 10 when the limit is 15, the request will fail, and
the system will display a descriptive error message.

**Limit Maximum GPU Usage per Experiment**

The following configuration policy example limits GPU usage per experiment. This is configured in the Config Policies > Experiments tab.

.. code:: yaml

   constraints:
     resources:
       max_slots: 4


**Set Different Priority Limits for Experiments and NTSC Tasks**

The following configuration policy example constrains the priority limit per experiment. This is configured in the Config Policies > Experiments tab.

-  Experiments

.. code:: yaml

   constraints:
     resources:
       priority_limit: <experiment limit>

The following configuration policy example constrains the priority limit per task. This is configured in the Config Policies > Tasks tab.

-  Tasks

.. code:: yaml

   constraints:
     resources:
       priority_limit: <task limit>

**Set Default Priority for All Experiments**

The following configuration policy example sets a default priority for all experiments:

.. code:: yaml

   invariant_config:
     resources:
       priority: 5

*******************
 Priority Override
*******************

If a global or workspace-level policy sets a default priority, it can be overridden using the
`SetJobPriority` API endpoint. However, the new priority must respect the constraints defined in the
applicable policies.

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

Limit resources
===============

Administrators can set constraints on resource usage. The main configurable constraints are:

-  ``resources.max_slots``: Limits the maximum number of slots (GPUs or CPUs) that can be used.
-  ``resources.priority_limit``: Sets the priority limit for tasks.

For Kubernetes resource managers, higher priority values indicate higher priority. For Agent
resource managers, lower priority values indicate higher priority.

Here’s an example of setting resource constraints:

.. code:: yaml

   constraints:
     resources:
       max_slots: 4
       priority_limit: 15

In this example, users cannot set ``max_slots`` to a value greater than 4, and they cannot set a
priority lower than 15 (for Agent RMs) or higher than 15 (for Kubernetes RMs).

Invariant Configs
=================

Invariant configs are applied to all workloads in a given scope (global or workspace) and merged
with user-submitted configurations.

**Example Invariant Config**

- Experiments

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

**Complete Example**

Here is a complete example that combines invariant configs and constraints for a research workspace:

- Experiments

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

.. warning::

   Do not set both constraints and configs for the same field within the same workload type and
   scope. It may lead to unpredictable behavior.

.. important::

   -  Existing workloads are not impacted by new or updated policies.
   -  Updating config policies replaces the entire set of invariant configs and constraints.
