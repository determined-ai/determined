:orphan:

**Security**

-  Add a configuration setting, ``initial_user_password``, to the master configuration file forcing
   the setup of an initial user password for the built-in ``determined`` and ``admin`` users during
   the first launch, specifically when a cluster's database is bootstrapped.

.. important::

   For any publicly-accessible cluster, you should ensure all users have a password set.
