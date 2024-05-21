:orphan:

**Breaking Changes**

-  master: on new deployments, the service will log an error and abort startup if no
   initialUserPassword is found in config.

To allow users to continue leaning on other reasonable default settings with CLI commands like `det
deploy local cluster-up`, an --initial-user-password flag is provided.
