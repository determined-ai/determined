:orphan:

**Bug Fixes**

-  Checkpoints: Checkpoints that are deleted through experiment deletion will not attempt
      to patch checkpoints. As a consequence, this fixes the intended behavior of deleting
      tensorboard files through experiment deletion as well.
