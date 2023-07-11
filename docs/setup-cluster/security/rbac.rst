.. _rbac:

######
 RBAC
######

.. attention::

   RBAC is only available on Determined Enterprise Edition. Please see RBAC's current limitations
   :ref:`here <rbac-limitations>`.

*****************
 Getting Started
*****************

.. warning::

   If you are looking to enable RBAC on an existing Determined installation, please see the
   :ref:`migration guide below <rbac-migrate-existing>`.

By default, a Determined installation comes with role-based access control disabled. To enable RBAC,
set the following option in the master config:

.. code:: yaml

   security:
     authz:
       type: rbac

Brand new Determined installations include two user accounts:

-  The ``admin`` user has full cluster access by default through the pre-canned ``ClusterAdmin``
   role.
-  The ``determined`` user has no permissions.

Both accounts have empty passwords. You are encouraged to set strong passwords or deactivate these
accounts for security reasons.

Example Setup (CLI)
===================

In this section, we will configure a Determined instance to support a cluster administrator account,
and a few engineers with varying level of access.

First, create a new user ``alice``, set a password, make it an admin, and deactivate the default
accounts:

.. code:: bash

   det -u admin user create alice
   det user change-password alice
   det rbac assign-role -u alice ClusterAdmin
   det user login alice
   det user deactivate admin
   det user deactivate determined

We want to enable two teams, one working on a traffic light detection model, and another one working
on a stop signs detection model. We'd like these two teams to have their separate, compartmentalized
workspaces. Start with creating non-priviledged users and workspaces:

.. code:: bash

   det user create mle-traffic-00
   det user create mle-traffic-01
   det user create mle-traffic-02
   det user create mle-stop-00
   det workspace create "Traffic Lights"
   det workspace create "Stop Signs"

We have several users on the traffic lights team, so we will create a user group to simplify the
permission management:

.. code:: bash

   det user-group create "Traffic Lights Team"
   det user-group add-user "Traffic Lights Team" mle-traffic-00,mle-traffic-01,mle-traffic-02

Give ``Editor`` role on traffic lights workspace to entire traffic lights team, and
``WorkspaceAdmin`` role to the ``mle-traffic-00`` and ``mle-stop-00`` users on their respective
workspace.

.. code:: bash

   det rbac assign-role -g "Traffic Lights Team" -w "Traffic Lights" Editor
   det rbac assign-role -u mle-traffic-00 -w "Traffic Lights" WorkspaceAdmin
   det rbac assign-role -u mle-stop-00 -w "Stop Signs" WorkspaceAdmin

We now have two independent workspaces, one for each team.

.. image:: /assets/images/_rbac-example-setup-dark.png
   :class: only-dark
   :alt: Determined AI system architecture diagram dark mode

.. image:: /assets/images/_rbac-example-setup-light.png
   :class: only-light
   :alt: Determined AI system architecture diagram light mode

.. note::

   You can learn more about pre-canned roles such as ``Editor``, ``WorkspaceAdmin`` :ref:`in the
   Pre-Canned Roles section <rbac-precanned>`.

Determined requires the projects to exist within the workspaces to run and organize the experiments.
Project creation requires ``PERMISSION_TYPE_CREATE_PROJECT`` permission which is included in
``Editor``, ``WorkspaceAdmin``, and ``ClusterAdmin`` roles. Since ``mle-traffic-01`` user is a
member of ``Traffic Lights Team`` which has the ``Editor`` role on the ``Traffic Lights`` workspace,
they can create the project themselves, no cluster admin access or participation is necessary:

.. code:: bash

   det -u mle-traffic-01 project create "Traffic Lights" "Green"

.. tip::

   We use ``det -u USER_NAME`` to dynamically change the username for the current command here.

   Similarly, you can run ``det user login USER_NAME`` to switch the current CLI user permanently.

As a non-privileged user, we can now :ref:`submit an experiment <quickstart-submit-experiment>` into
this project:

.. code:: bash

   # Per quickstart, `cd` into the example directory before the next command.
   det -u mle-traffic-01 experiment create const.yaml . --config workspace="Traffic Lights" --config project="Green" --config name="green light"

Repeat the process to submit a second test experiment as a Stop Signs engineer:

.. code:: bash

   det -u mle-stop-00 project create "Stop Signs" "Euro"
   det -u mle-stop-00 experiment create const.yaml . --config workspace="Stop Signs" --config project="Euro" --config name="euro stop"

Since the admin and non-admin users on various teams have different access, listing all experiments
will produce different results. The admin will see both experiments. Traffic lights and stop signs
engineers will only have access to the experiments in their respective workspaces. Compare:

.. code:: bash

   det -u alice experiment list --all
   det -u mle-stop-00 experiment list --all
   det -u mle-traffic-00 experiment list --all

*************
 Manage RBAC
*************

Inspecting the Setup
====================

CLI
---

To get the current user's permission list:

.. code:: bash

   det rbac my-permissions

To list all permissions on the role as well as all users and groups who bear it, whether globally or
at a workspace level:

.. code:: bash

   det rbac describe-role ROLE

To list all existing roles and their permissions:

.. code:: bash

   det rbac list-roles

To list existing users, group and their membership:

.. code:: bash

   det user list
   det user-group list
   det user-group describe GROUP_NAME

To list the role assignments for a user or a group:

.. code:: bash

   det rbac list-groups-roles GROUP_NAME
   det rbac list-users-roles USER_NAME

WebUI
-----

To see the user and group management UI,

#. Click on your username in the upper left corner.
#. Click "Settings"
#. See "User Management" and "Group Management" tabs at the top.

To create new users, use "New User" button at the "User Management" screen.

To see or modify user roles assigned at the global scope,

#. Click triple-dot icon on the right of the user entry.
#. Select "Edit" option in the dropdown.

To see or modify group roles assigned at the global scope,

#. Click triple-dot icon on the right of the group entry.
#. Select "Add Roles" option in the dropdown.

To see group membership, click on the plus icon on the left of the group entry.

To remove member users, open group membership list and "Remove" button next to the user entry.

To add member users,

#. Click triple-dot icon on the right of the group entry.
#. Select "Edit/Add Users" option in the dropdown.

Managing User Groups
====================

CLI
---

To create a group ``GROUP_NAME``, add and remove users ``USER_NAME1``, ``USER_NAME2``, and
``USER_NAME3``:

.. code:: bash

   det user-group create GROUP_NAME
   det user-group add-user GROUP_NAME USER_NAME1,USER_NAME2,USER_NAME3
   det user-group remove-user GROUP_NAME USER_NAME1,USER_NAME2,USER_NAME3

To rename a group:

.. code:: bash

   det user-group change-name GROUP_NAME

To delete a group:

.. code:: bash

   det user-group delete GROUP_NAME

To list existing groups, or a particular group membership:

.. code:: bash

   det user-group list
   det user-group describe GROUP_NAME

WebUI
-----

.. attention::

   Only users with the ClusterAdmin role can add/remove users and groups.

To see user group management UI,

#. Click on your username in the upper left corner.
#. Click "Admin".
#. Click the "Groups" tab at the top.

To create new groups, use "New Group" button at the "Groups" screen.

To delete a group,

#. Click triple-dot icon on the right of the group entry.
#. Select "Delete" option in the dropdown.

Managing Role Assignments
=========================

CLI
---

To assign or unassign a role for a user or a group globally:

.. code:: bash

   det rbac assign-role -u USER_NAME ROLE_NAME
   det rbac unassign-role -u USER_NAME ROLE_NAME
   det rbac assign-role -g GROUP_NAME ROLE_NAME
   det rbac unassign-role -g GROUP_NAME ROLE_NAME

To assign or unassign a role for a user or a group on a particular workspace, use ``-w
WORKSPACE_NAME`` switch:

.. code:: bash

   det rbac assign-role -u USER_NAME ROLE_NAME -w WORKSPACE_NAME
   det rbac unassign-role -u USER_NAME ROLE_NAME -w WORKSPACE_NAME
   det rbac assign-role -g GROUP_NAME ROLE_NAME -w WORKSPACE_NAME
   det rbac unassign-role -g GROUP_NAME ROLE_NAME -w WORKSPACE_NAME

WebUI
-----

To assign or unassign a role for a user or a group globally, first go to user or group management
UI:

.. attention::

   Only users with the ClusterAdmin role can add/edit global role assignments for users and groups
   from the "Admin" menu as described immediately below.

#. Click on your username in the upper left corner.
#. Click "Admin".
#. See "Users" and "Groups" tabs at the top.

Then, for users:

#. Click triple-dot icon on the right of the user entry.
#. Select "Edit" option in the dropdown.

For groups:

#. Click triple-dot icon on the right of the group entry.
#. Select "Add Roles" option in the dropdown.

To assign or unassign a role for a user or a group on a particular workspace,

.. attention::

   Only users with the WorkspaceAdmin role can add/edit workspace-scoped role assignments for users
   and groups.

#. Go to the workspaces page, select the target workspace.
#. Click "Members" tab at the top.
#. To create new role assignments, click the "Add Members" button at the top right.
#. To remove existing role assignments, click the triple-dot menu for a user/group and select
   "Remove".
#. To edit the role, click on the dropdown in the role column for a user/group and make your
   selection.

.. _rbac-concepts:

*********************
 About RBAC Concepts
*********************

User Groups
===========

User groups in Determined are organizational units containing one or more Determined users. User
groups currently have no inherent functionality and are not directly useful unless paired with the
RBAC feature. Users with cluster admin permissions may create groups and add as many users to them
as needed.

RBAC Permissions and Scopes
===========================

RBAC allows granting users or user groups a permission to do certain actions, such as various API
calls, on certain resources, such as experiments. This is achieved using role assignments, which are
comprised of security principal, role, and scope. Roles, in turn, are comprised of permissions.

Security Principal
------------------

A security principal is an entity that is performing an action on a resource. Determined supports
individual users or user groups as security principals.

Scope
-----

A scope in Determined refers to where a user may exercise their permitted actions and currently has
two possible values: global and workspace-specific. A global-level permission is valid anywhere in
Determined, allowing the user to perform the action on any workspace. A workspace-level permission
restricts actions so that they are only permissible on the specified workspaces. When using
workspace-level permissions, the admin must specify which workspace(s) the permission is valid for.

Role
----

A role is a collection of permissions. It allows combining commonly used permissions, for example
when several permissions are used by the same persona, like an ML engineer. Determined currently
supports several :ref:`built-in roles <rbac-precanned>`.

Permission
----------

A permission is a description of a type of access to a resource or set of resources. Permissions
typically map to an action on an entity type, for example:

-  ``PERMISSION_TYPE_VIEW_EXPERIMENT_METADATA``: view high-level experiment properties.
-  ``PERMISSION_TYPE_VIEW_EXPERIMENT_ARTIFACTS``: view experiment code and checkpoints.
-  ``PERMISSION_TYPE_ADMINISTRATE_USER``: manage user accounts. This is only available on the global
   scope.
-  ``PERMISSION_TYPE_ASSIGN_ROLES``: assign roles.

Current Limitations
===================

.. _rbac-limitations:

As RBAC is progressively rolled out there are various parts of Determined that are not yet gated
behind RBAC. These currently include:

-  The job queue APIs: Any logged-in user can see high-level metadata about all active jobs in the
   queue, though RBAC is enforced when clicking on a task to access its details and artifacts.
-  RBAC is not currently applied to master logs, which can contain information about jobs that the
   user is not authorized to see.
-  Historical usage shows cluster-wide usage related to other users and workspaces.

*****************
 Usage Reference
*****************

CLI
===

The :ref:`Determined CLI <cli-ug>` has built-in help. Please see help for the top-level commands, as
well as their subcommands:

.. code::

   det user -h
   det user-group -h
   det rbac -h
   det rbac assign-role -h

.. _rbac-precanned:

******************
 Pre-Canned Roles
******************

Determined ships with several pre-canned roles. We are looking to add an ability to edit or create
custom roles in a future release.

To list all existing cluster roles and the concrete permissions they include:

.. code:: bash

   det rbac list-roles

``Viewer``
==========

The ``Viewer`` role allows a user to see workspaces, projects, notebooks, TensorBoards, shells,
commands (NTSC), and experiments, as well as experiment metadata and artifacts within its scope.

``Editor``
==========

The ``Editor`` role supersedes the ``Viewer`` role, and includes permissions to create, edit, or
delete projects, NTSC, and experiments within its scope.

``WorkspaceAdmin``
==================

The ``WorkspaceAdmin`` role supersedes the ``Editor`` role, and includes permissions to edit or
delete workspaces, and modify role assignments within its scope.

Users who take this role on a particular workspace can assign roles to other users on this
workspace, that is, add other members (viewers, editors, or workspace admins) to the workspace.

``WorkspaceCreator``
====================

The ``WorkspaceCreator`` role grants the single permission to create new workspaces. It can only be
assigned globally.

.. note::

   By default, when a user creates a workspace, they automatically get assigned the
   ``WorkspaceAdmin`` role.

   This behavior can be configured using master config:

   .. code:: yaml

      security:
        authz:
          workspace_creator_assign_role:
            enabled: true
            role_id: ROLE_ID

   where ``ROLE_ID`` is the integer role identifier, as listed in ``det rbac list-roles``. To
   disable the assignment of any roles to the newly created workspace, set ``enabled: false``.

``ClusterAdmin``
================

``ClusterAdmin`` is the highest role intended for cluster administrators or superusers. It includes
all permissions, and can only be assigned globally.

.. _rbac-migrate-existing:

*****************************************
 Migrating Existing Installation to RBAC
*****************************************

#. Upgrade Determined to the latest RBAC-enabled version.

#. Enable RBAC UI in the master config:

   .. code:: yaml

      security:
        authz:
          rbac_ui_enabled: true

#. Restart Determined for the config change to take effect. This config option will enable RBAC APIs
   and UI, but the RBAC rules will not be enforced, allowing administrators to set it up first.

#. For all cluster administrators or superusers, grant the ``ClusterAdmin`` role. This will ensure
   the admins are not "locked out" once strict RBAC enforcement is enabled.

   .. code:: bash

      det rbac assign-role -u ADMIN_USER_NAME ClusterAdmin

#. Enable RBAC enforcement in the master config:

   .. code:: yaml

      security:
        authz:
          type: rbac

#. Restart master for the change to take effect.

#. Proceed to configure RBAC as desired.

.. note::

   Workspace creators for workspaces created after upgrading to 0.19.6+ will have ``WorkspaceAdmin``
   role assigned for their workspaces.

   Users will have no default access otherwise.
