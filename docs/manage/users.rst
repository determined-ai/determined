.. _users:

###############
 User Accounts
###############

*******
 About
*******

Only the ``admin`` user can create users, change users' passwords, and activate or deactivate users.
Upon initial installation, the admin should set an admin password.

Default Accounts
================

Initially, there are two accounts:

-  ``admin`` (full privileges)
-  ``determined`` (for single-user installations)

Both have blank passwords by default.

Setting the Admin Password
==========================

Use the following CLI command to set the admin password:

.. code::

   det -u admin user change-password

Creating Individual User Accounts
=================================

You can add, edit, and manage users manually via the CLI or the WebUI.

To create users via the CLI, use the following command:

.. code::

   det -u admin user create <username>

To ensure that no one can access the Determined cluster as the ``determined`` user, deactivate it.
Deactivating the ``determined`` user does not remove any objects created by the user.

To deactivate the ``determined`` user, run the following command:

.. code::

   det -u admin user deactivate determined

Creating Remote Users
=====================

Admins can configure Determined to auto-provision users who have been added to your IdP. These users
are known as remote users.

To find out more, visit :ref:`remote-users`.

****************
 Authentication
****************

WebUI
=====

The WebUI will automatically redirect users to a sign-in page if there is no valid Determined
session established on that browser. After signing in, the user will be redirected to the URL they
initially attempted to access.

Users can end their Determined session by selecting their profile name in the upper left corner and
choosing **Sign Out**.

CLI
===

In the CLI, the ``user login`` subcommand can be used to authenticate a user:

.. code::

   det user login <username>

Logging in results in a persistent session, which lasts for 30 days. The session can be terminated
using:

.. code::

   det user logout

**************************
 Temporary Authentication
**************************

In some cases, it may be useful to execute a single command as a specific user without starting a
persistent session for that user (think of the ``sudo`` command on a Unix-like system). In
Determined, this can be achieved with the ``-u`` flag:

.. code::

   det -u <username> ...

This will execute the command as the given user without creating a permanent session for that user.
Although no persistent session is created, an authentication token is stored for that user so that
future attempts to execute commands as that user will not require re-authenticating. This token can
be discarded using the ``user logout`` subcommand:

.. code::

   det -u <username> user logout

******************
 Change Passwords
******************

Users have blank passwords by default. This might be sufficient for low-security or experimental
clusters, and it still provides the organizational benefits of associating each Determined object
with the user that created it. If desired, a user can change their own password using the ``user
change-password`` subcommand:

.. code::

   det user change-password

An admin can also change another user's password:

.. code::

   det -u admin user change-password <target-user>

.. warning::

   Although Determined supports password-based authentication, communication between the CLI, WebUI,
   and master does *not* take place over an encrypted channel by default. See :ref:`security` for
   information on configuring secure connections over HTTPS. Users should not be assigned "valuable"
   passwords, and passwords used with Determined should not be reused for other purposes.

*************
 List Assets
*************

WebUI
=====

Just as in the CLI, by default the WebUI will only show assets created by the current user. To see
assets belonging to all users, uncheck the "Show only mine" checkbox in the filter panel found in
the tab for each asset type.

.. _cli-1:

CLI
===

When using the CLI to list experiments, commands, etc., the default behavior is to only show assets
belonging to the current user. It is possible to show assets owned by all users by passing the
``-a`` flag to the respective commands:

.. code::

   det experiment list -a   # List all experiments.
   det command list -a      # List all commands.
   det notebook list -a     # List all notebooks.
   det tensorboard list -a  # List all TensorBoards.

.. _webui-1:

*******************************
 Activate and Deactivate Users
*******************************

When a user is created, they are designated as active by default. Only active users can interact
with Determined. The ``admin`` user can deactivate a user with the ``user deactivate`` subcommand:

.. code::

   det -u admin user deactivate <target-user>

All assets created by a deactivated user will remain available through both the WebUI and the CLI.

To reactivate a user, ``user activate`` can be used:

.. code::

   det -u admin user activate <target-user>

.. _run-as-user:

***********************************
 Run Tasks as Specific Agent Users
***********************************

For experiment, notebook, or command tasks using the ``bind_mount`` option in their
:ref:`experiment-config-reference`, setting the Unix user and group on the agent ensures file
permission consistency between the task and agent.

Configure this by linking a Determined user with the user and group configuration on an agent:

.. code::

   det user link-with-agent-user <target-user> --agent-uid <uid> --agent-user <username> --agent-gid <gid> --agent-group <group-name>

All arguments are required. This command can only be run by a system administrator.

Once set, any tasks created by the target user will be run as the specified user and group.

.. note::

   By default, if a user is not linked with a user and group on an agent, tasks created by that user
   will run as the root user on the agent. If deploying on a Slurm/PBS cluster, running as the root
   user is only permitted if the launcher ``user_name`` is also set to the root user, as described
   in :ref:`using_slurm`. This behavior may change in the future.

   If the task does not use ``bind_mount`` option, the effect of running as root will be limited to
   the task container and not intrude on the agent itself.

The default user and group that will be used when a Determined user is not explicitly linked to a
user and group on an agent can be configured in the ``master.yaml`` file located at
``/usr/local/determined/etc`` on the Determined master instance:

.. code:: yaml

   security:
     default_task:
       user: root
       uid: 0
       group: root
       gid: 0

.. note::

   A writable ``HOME`` directory is required by all Determined tasks. By default, all official
   Determined images contain a tool called ``libnss_determined`` that injects users into the
   container at runtime. If you are building custom images using a base image other than those
   provided by Determined, you may need to take one of the following steps:

      -  prebuild all users you might need into your custom image, or
      -  include ``libnss_determined`` in your custom image to ensure user injection works as
         expected, or
      -  find an alternate solution that serves the same purpose of injecting users into the
         container at runtime

.. _run-unprivileged-tasks:

***********************************
 Run Unprivileged Tasks by Default
***********************************

Some administrators of Determined may wish to run tasks as unprivileged users by default. In Linux,
unprivileged processes are sometimes run under the `nobody
<https://en.wikipedia.org/wiki/Nobody_(username)>`_ user, which has very few privileges. However,
the ``nobody`` user does not have a writable ``HOME`` directory, which is a requirement for tasks in
Determined, so the ``nobody`` user will typically not work in Determined.

For convenience, the default Determined environments contain an unprivileged user named
``det-nobody``, which does have a writable ``HOME`` directory. The ``det-nobody`` user is a suitable
default user when using the default Determined environment images and when running containers as
root is not desired. To use ``det-nobody`` by default, add the following configuration to
``master.yaml``:

.. code:: yaml

   security:
     default_task:
       user: det-nobody
       uid: 65533
       group: det-nobody
       gid: 65533

When combining the ``det-nobody`` user with custom Docker images, administrators should either build
the custom image as layers on top of the default Determined Environments as illustrated in
:ref:`custom-docker-images`, or they should create the ``det-nobody`` user themselves in their
custom images using ``groupadd`` and ``useradd``.
