:orphan:

**New Features**

-  CLI: Add support for a ``--add-tag`` flag to AWS ``det deploy aws up``

   -  A new argument is added to ``det deploy aws up`` that allows users to specify tags that should
      be added to the underlying CloudFormation stack.

   -  New tags will not replace automatically-added tags such as ``deployment-type`` or
      ``managed-by``

   -  Any added tags that should persist across updates should be always be included when using
      ``det deploy aws up`` -- if the argument is missing, any previously added tags would be
      removed
