.. _config-policies:

#######################
 Config Policies Guide
#######################

Config Policies allow administrators to set limits on how users can define workloads (e.g.,
experiments and notebooks, tensorboards, shells, and commands). This feature enables administrators
to govern user behavior more closely at the workspace and cluster level. Administrators can include
any parameter referenced in the :ref:`experiment configuration <experiment-config-reference>`.

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

#. Limit resources.

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

#. Limit maximum GPU usage per experiment:

.. code:: yaml

   name: max-gpu-limit
   scope: cluster
   config:
     resources:
       max_gpus: 4

2. Set different priority limits for experiments and NTSC tasks:

.. code:: yaml

   name: priority-limits
   scope: workspace
   workspace: research
   config:
     scheduling:
       max_priority:
         experiments: 80
         ntsc: 60

3. Set default priority for all tasks:

.. code:: yaml

   name: default-priority
   scope: cluster
   config:
     scheduling:
       default_priority: 5

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
