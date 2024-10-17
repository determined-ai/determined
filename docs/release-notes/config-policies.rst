:orphan:

**New Features**

-  Cluster: Using config policies, administrators can now set limits on how users can define workloads (e.g.,
experiments, notebooks, tensorboards, shells, and commands). Admins may define two types of configurations:

  -  **Invariant Configs for Experiments**: Settings applied to all experiments within a scope (global or workspace).
  -  **Constraints**: Restrictions that prevent users from exceeding resource limits within a scope. Can be set independently for experiments and tasks.

  Visit :ref:`Config Policies <config-policies>` for more details.
