:orphan:

**New Features**

*   RBAC: Add a pre-canned role called ``EditorRestricted`` which supersedes the ``Viewer`` role
    and precedes the ``Editor`` role.

    *   Like the ``Editor`` role, the ``EditorRestricted`` role grants the permissions to create,
    edit, or delete projects and experiments within its designated scope. However, the
    ``EditorRestricted`` role lacks the permissions to create or update NTSC type workloads.

    Therefore, a user with ``EditorRestricted`` privileges in a given scope are limited using the
    webUI within that scope as the option to lauch JupyterLab notebooks and kill tasks will be
    unavailable. The user will also be unable to use CLI commands that create or update scoped
    nootebooks, tensorboards, shells, and commands (such as changing a task's priority or deleting
    it). ``EditorRestricted`` users can still, however, open and edit the code in scoped JupyterLab
    notebooks and perform all experiment-related jobs as those with the ``Editor`` role.

    *   The ``EditorRestricted`` role was created to allow workspace and cluster editors and admins
    to have more fine-grained control over GPU resources. Thus, users with this role lack the
    ability to indefinitely consume slot-requesting resources by launching long-running tasks
    within a given scope.
