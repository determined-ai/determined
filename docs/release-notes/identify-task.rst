:orphan:

**Bug Fixes**

-  API: Fix a bug where a database query would add extra quotes around the task type.
      For tensorboard task types, this bug would always error and not allow authorized users to view
      the tensorboard or the experiments associated with it.
