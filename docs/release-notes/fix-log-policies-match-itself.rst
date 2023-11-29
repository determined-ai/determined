:orphan:

**Bug Fixes**

-  Fix an issue where ``log_policies`` would be compared against the trial log printing experiment
   config which could often cause patterns like ``(.*) match (.*)`` to incorrectly always match.
