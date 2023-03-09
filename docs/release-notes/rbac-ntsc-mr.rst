:orphan:

**New Features**

-  RBAC: Following on the initial RBAC support added in 0.19.7 the enterprise edition of Determined
   (`HPE Machine Learning Development Environment
   <https://www.hpe.com/us/en/solutions/artificial-intelligence/machine-learning-development-environment.html>`_)
   has added support for Role-Based Access Control over new entities:

   -  JupyterLab Notebooks, Tensorboards, Shells, and Commands are now housed under workspaces.
      Access to these tasks can now be restricted by role. Launching Tensorboards from WebUI
      launches at the experiment worspace by default.

   -  Model Registry: models are now associated with workspaces. Models can be moved between
      workspaces and access to them can be restricted by role.

   These changes allow for more granular control over who can access what resources. See :ref:`rbac`
   for more information.
