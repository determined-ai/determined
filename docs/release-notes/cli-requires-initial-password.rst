:orphan:

Breaking Changes

-  Master: On new deployments, the service will log an error and abort startup if no
   ``initialUserPassword`` is found in the configuration.

To ensure users can still rely on reasonable default settings with CLI commands like ``det deploy
local cluster-up``, an ``--initial-user-password`` flag is now provided.
