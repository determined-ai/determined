:orphan:

**Security Fixes**

-  CLI: If no user accounts have already been created, and neither
   ``security.initial_user_password`` in master.yaml nor ``--initial-user-password`` is present when
   running ``det deploy local`` with the ``master-up`` or ``cluster-up`` commands, an initial
   password will be generated and displayed to the user.
