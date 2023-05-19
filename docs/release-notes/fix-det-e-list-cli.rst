:orphan:

**Bug Fixes**

-  CLI: ``det e list`` and ``det e list -a`` behaviors were erroneously switched.
      -  Earlier, ``det e list`` was showing both archived and unarchived experiments, and ``det e
         list -a`` was showing only unarchived experiments. This has now been fixed - ``det e list``
         will show only unarchived experiments and ``det e list -a`` will show both archived and
         unarchived experiments.
