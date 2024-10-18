.. _config-policies:

#######################
 Config Policies Guide
#######################

Config Policies enable administrators to control how users define workloads, such as experiments,
notebooks, TensorBoards, shells, and commands. Administrators can include any parameter from the
:ref:`experiment configuration reference <experiment-config-reference>` to set up experiment
invariant configurations.

.. include:: ../_shared/attn-enterprise-edition.txt

Administrators can define two types of configuration policies:

-  **Constraints**: Restrictions that prevent users from exceeding resource limits.
-  **Invariant Configs**: Settings applied to all workloads within a scope (global or workspace).

These policies are essential for managing task priorities, resource allocation, and setting specific
configurations across the organization. Global policies override workspace policies, and both
override user-submitted configurations.

**************
 Key Features
**************

-  Apply policies using the WebUI or the CLI.
-  Define limits and non-overridable defaults for resource usage, environment settings, and more.
-  Enforce policies at both cluster and workspace levels.
-  Set different priority and slot request limits for experiments and NTSC (notebooks, TensorBoards,
   shells, and commands) tasks.
-  Enforce strict adherence to configured priority limits.

.. warning::

   To avoid unpredictable behavior, do not set both constraints and invariant configs for the same
   field within the same workload type and scope. Additionally, clearly specify whether the
   constraints apply to **experiments** or **NTSC** tasks.

.. important::

   -  Existing workloads are not impacted by new or updated policies.
   -  Updating config policies replaces the entire set of invariant configs and constraints for the
      affected scope.

******************
 Setting Policies
******************

Administrators can set Config Policies at either the cluster or workspace levels through the WebUI
or CLI.

.. important::

   Config policies are set as a complete unit. When updating config policies using the WebUI or CLI,
   both the invariant config and constraints for a given scope are updated together. It is not
   possible to update only one of these components independently.

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

Use the following command to manage workspace-level Config Policies via CLI:

.. code:: bash

   det config-policies

This command has several subcommands:

.. code:: bash

   # Show help for the config-policies command
   det config-policies help

   # Delete config policies
   det config-policies delete

   # Describe config policies
   det config-policies describe

   # Set config policies
   det config-policies set

For more detailed information on each subcommand, you can use the `-h` or `--help` flag. For
example:

.. code:: bash

   det config-policies set -h

This will display the specific options and arguments available for the `set` subcommand.

*****************
 Policy Examples
*****************

Below are some examples of Config Policies to help guide administrators.

Limiting Resources
==================

Administrators can set constraints on resource usage, allowing them to manage how users allocate
resources for workloads such as experiments and NTSC tasks. The
configurable constraints are:

-  ``constraints.resources.max_slots``: Limits the maximum number of slots (GPUs or CPUs) that can
   be used by submitted workloads. Specify whether this applies to experiment workloads or NTSC
   tasks.

-  ``constraints.priority_limit``: Sets the priority limit for tasks. This value also needs to
   specify whether it applies to experiments or NTSC tasks.

For Kubernetes resource managers, higher priority values indicate higher priority. For Agent
resource managers, lower priority values indicate higher priority.

**Example: Limiting GPU/CPU Slots and Priority for Experiments**

The following example demonstrates how to set a maximum of 4 slots (GPUs or CPUs) and a priority
limit of 15. This configuration applies to experiment workloads:

.. code:: yaml

   constraints:
     priority_limit: 15
     resources:
       max_slots: 4

In this example:

-  Users cannot set ``resources.max_slots`` greater than 4 for experiments.
-  Users cannot set a priority lower than 15 for Agent RMs or higher than 15 for Kubernetes RMs.
-  Any workload that exceeds these limits will be rejected, ensuring adherence to the configured
   constraints.

**Example: Limiting GPU/CPU Slots for NTSC Tasks**

To apply similar constraints to NTSC tasks:

.. code:: yaml

   constraints:
     priority_limit: 10
     resources:
       max_slots: 4

In this case:

-  Users cannot set ``resources.max_slots`` greater than 4 for NTSC tasks.
-  Users cannot set a priority higher than 10 for NTSC tasks in Kubernetes environments, and no
   lower than 10 for Agent environments.

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

You can set a different priority limit for experiments and NTSC tasks.

**Example: Setting Priority Limits for Experiments**

The following policy is configured in the Config Policies > Experiments tab.

.. code:: yaml

   constraints:
     priority_limit: <experiment limit>

**Example: Setting Priority Limits for NTSC Tasks**

The following policy is configured in the Config Policies > Tasks tab.

.. code:: yaml

   constraints:
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

When a user submits a workload, these settings will be applied in addition to (or override) the
user's settings.

****************
 Best Practices
****************

-  Start with cluster-wide policies for broad governance.
-  Use workspace-level policies for more granular control of the projects in a workspace.
-  Set different priority limits for experiments and NTSC tasks to better manage resource
   allocation.
-  Regularly review and update policies as organizational needs evolve.
-  Communicate configuration policies to users to ensure smooth adoption.
-  Use the SetJobPriority API cautiously, ensuring that priority changes do not violate existing
   policies.
-  Use invariant configs to set default behaviors across experiment workloads.
-  Be mindful of the precedence order: global policies override workspace policies, which override
   user configs.
