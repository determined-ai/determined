:orphan:

**Bug Fixes**

-  API: Fix a bug where a database query would add extra quotes around the task type.
      For tensorboard task types, this bug would allow users to view tensorboards, even if they did
      not have permission to view the all workspaces it included.
