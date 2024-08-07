:orphan:

**New Features**

-  RBAC: Add a pre-canned role called ``EditorProjectRestricted`` which supersedes the ``Viewer``
   role and precedes the ``Editor`` role.

   -  Like the ``Editor`` role, the ``EditorProjectRestricted`` role grants the permissions to
      create, edit, or delete experiments and NSC (Notebook, Shell or Command) type workloads within
      its designated scope. However, the ``EditorProjectRestricted`` role lacks the permissions to
      create or update projects.
