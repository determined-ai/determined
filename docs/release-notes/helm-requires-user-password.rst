:orphan:

**Security Fixes**

-  Helm: When deploying a new cluster with Helm, an initial password for the ``admin`` and
      ``determined`` users should be specified using either ``initialUserPassword`` or
      ``defaultPassword`` (see helm/charts/determined/values.yaml). Configuring a password is no
      longer done as a separate step.
