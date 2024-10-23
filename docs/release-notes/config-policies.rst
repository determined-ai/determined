:orphan:

**New Features**

-  Cluster: In the enterprise edition of Determined, add :ref:`config policies <config-policies>` to
   enable administrators to set limits on how users can define workloads (e.g., experiments,
   notebooks, TensorBoards, shells, and commands). Administrators can define two types of
   configurations:

   -  **Invariant Configs for Experiments**: Settings applied to all experiments within a specific
      scope (global or workspace). Invariant configs for other tasks (e.g. notebooks, TensorBoards,
      shells, and commands) is not yet supported.

   -  **Constraints**: Restrictions that prevent users from exceeding resource limits within a
      scope. Constraints can be set independently for experiments and tasks.
