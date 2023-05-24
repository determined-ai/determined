:orphan:

**New Features**

-  API: Earlier GetExperiments(archived = False) used to list experiments from both archived and
   unarchived projects and workspaces. Now when GetExperiments(archived = False) is called, it will
   only list unarchived experiments from unarchived projects and workspaces. This will also affect
   the CLI command ``det e list`` which also used to list unarchived experiments from both archived
   and unarchived projects and workspaces. Now, it will only list unarchived experiments from
   unarchived projects and workspaces.
