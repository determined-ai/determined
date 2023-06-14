:orphan:

**Breaking Changes**

-  Users: The CLI command ``det user list`` will no longer show inactive users by default. To list
   both active and inactive users, you can use the ``det user list --include_inactive`` command.
   Similarly, the SDK method ``list_users`` will no longer display inactive users unless the
   ``include_inactive`` parameter is explicitly set to true.
