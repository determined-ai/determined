.. _task-config-policies-reference:

#################################
 Task Config Policies Reference
#################################

Add a description of the task config policies reference.

DRAFT CONTENT:

- Task Config Policies API Reference
- Task Config Policies CLI Reference
- Task Config Policies UI Reference
- Task Config Policies Best Practices
- Task Config Policies Examples
- Task Config Policies FAQ

This reference guide provides detailed information about Task Config Policies in Determined AI, including API endpoints, CLI commands, UI interactions, best practices, examples, and frequently asked questions.

Task Config Policies API Reference
==================================

The Task Config Policies API allows administrators to programmatically manage policies. Here are the main endpoints:

- ``GET /api/v1/task-policies``: List all task policies
- ``POST /api/v1/task-policies``: Create a new task policy
- ``GET /api/v1/task-policies/{policy_id}``: Get details of a specific policy
- ``PATCH /api/v1/task-policies/{policy_id}``: Update an existing policy
- ``DELETE /api/v1/task-policies/{policy_id}``: Delete a policy

For detailed request and response formats, refer to the API documentation.

Task Config Policies CLI Reference
==================================

Determined CLI provides commands to manage task policies:

.. code-block:: bash

   # List all policies
   det policy list

   # Create a new policy
   det policy create <policy_name> --config <policy_config_file.yaml>

   # Show details of a policy
   det policy show <policy_name>

   # Update an existing policy
   det policy update <policy_name> --config <updated_policy_config_file.yaml>

   # Delete a policy
   det policy delete <policy_name>

Task Config Policies UI Reference
=================================

The Determined WebUI provides a graphical interface for managing task policies:

1. Navigate to the "Admin" section
2. Select "Task Config Policies" from the menu
3. Use the interface to create, view, edit, or delete policies

Task Config Policies Best Practices
===================================

1. Start with cluster-wide policies for general governance
2. Use workspace and project-level policies for more granular control
3. Set different priority limits for experiments and NTSC tasks
4. Regularly review and update policies as organizational needs change
5. Communicate policy changes to users to ensure smooth adoption
6. Use the SetJobPriority API endpoint cautiously

Task Config Policies Examples
=============================

1. Limit maximum GPU usage per experiment:

   .. code-block:: yaml

      name: max-gpu-limit
      scope: cluster
      config:
        resources:
          max_gpus: 4

2. Set different priority limits for experiments and NTSC tasks:

   .. code-block:: yaml

      name: priority-limits
      scope: workspace
      workspace: research
      config:
        scheduling:
          max_priority:
            experiments: 80
            ntsc: 60

3. Set default priority for all tasks:

   .. code-block:: yaml

      name: default-priority
      scope: cluster
      config:
        scheduling:
          default_priority: 5

Task Config Policies FAQ
========================

Q: How are conflicting policies resolved?
A: Policies are applied in order of specificity: project-level, workspace-level, then cluster-wide. More specific policies override more general ones.

Q: Can users override policy-set priorities?
A: Users can override priorities using the SetJobPriority API, but the new priority must not violate any existing policy constraints.

Q: How often should policies be reviewed?
A: It's recommended to review policies quarterly or when there are significant changes in organizational needs or cluster usage patterns.

For more information on Task Config Policies, refer to the :ref:`Task Config Policies <task-config-policies>` documentation.
