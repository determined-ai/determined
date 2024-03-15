:orphan:

**New Features**

*   RBAC: Add a pre-canned role called ``EditorRestricted`` which supersedes the ``Viewer`` role
    and precedes the ``Editor`` role.

    *   Like the ``Editor`` role, the ``EditorRestricted`` role grants the permissions to create,
    edit, or delete projects and experiments within its designated scope. However, the
    ``EditorRestricted`` role lacks the permissions to create or update NTSC type workloads.

    Therefore, a user with ``EditorRestricted`` privileges in a given scope are limited when using
    the webUI within that scope since the option to launch JupyterLab notebooks and kill running
    tasks will be unavailable. The user will also be unable to run CLI commands that create scoped
    notebooks, tensorboards, shells, and commands and will be unable to perform updates on these
    tasks (such as changing the task's priority or deleting it). ``EditorRestricted`` users can
    still, however, open and use scoped JupyterLab notebooks and perform all experiment-related
    jobs as those with the ``Editor`` role.

    *   The ``EditorRestricted`` role was created to allow workspace and cluster editors and admins
    to have more fine-grained control over GPU resources. Thus, users with this role lack the
    ability to launch or modify tasks that indefinitely consume slot-requesting resources within a
    given scope.
