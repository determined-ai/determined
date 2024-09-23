.. _task-config-policies:

######################
 Task Config Policies
######################

Task Config Policies allow administrators to set limits on how users can define workloads (e.g., experiments and notebooks, tensorboards, shells, and commands). This feature enables enterprises to govern user behavior more closely at various levels within the Determined cluster.

**************
 Key Features
**************

- Set policies via WebUI or CLI
- Define limits for resource usage, environment settings, and more
- Apply policies at different levels (cluster-wide, workspace, or project)
- Override capabilities for specific user roles or groups
- Set different priority limits for experiments and NTSC (notebooks, tensorboards, shells, and commands) tasks

*******************
 Setting Policies
*******************

Administrators can set Task Config Policies using either the WebUI or the CLI.

WebUI
=====

1. Log in to the Determined WebUI as an administrator.
2. Navigate to the "Admin" section.
3. Select "Task Config Policies" from the menu.
4. Click "Create New Policy" or edit an existing policy.
5. Define the policy settings and scope (cluster-wide, workspace, or project).
6. Save the policy.

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

1. Limit maximum GPU usage per experiment:

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
 Policy Hierarchy
*******************

Policies are applied in the following order of precedence:

1. Project-level policies
2. Workspace-level policies
3. Cluster-wide policies

More specific policies (e.g., project-level) override more general policies (e.g., cluster-wide) when conflicts occur.

*******************
 Priority Override
*******************

If a given scope has the priority field of a default invariant config set, this priority can still be overridden using the SetJobPriority API endpoint. However, the new priority must not violate any constraints placed on the scope.

*****************
 Best Practices
*****************

- Start with cluster-wide policies for general governance.
- Use workspace and project-level policies for more granular control.
- Set different priority limits for experiments and NTSC tasks to better manage resource allocation.
- Regularly review and update policies as organizational needs change.
- Communicate policy changes to users to ensure smooth adoption.
- Use the SetJobPriority API endpoint cautiously, ensuring that priority changes do not violate existing policies.

For more detailed information on configuring and managing Task Config Policies, refer to the :ref:`Task Config Policies API Reference <task-config-policies-reference>`.