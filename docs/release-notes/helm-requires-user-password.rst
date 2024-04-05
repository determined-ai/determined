:orphan:

**Security Fixes**

-  Helm: When deploying a new cluster with Helm, configuring an initial password for the "admin" and
   "determined" users is required and is no longer a separate step. To specify an initial password
   for these users, visit the helm/charts/determined/values.yaml file and use either
   initialUserPassword (preferred) or defaultPassword (deprecated). For reference, visit
   :ref:helm-config-reference.
