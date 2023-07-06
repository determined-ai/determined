.. _saas-cloud-team-admin:

###################
 Controlling Access
###################

.. meta::
   :description: Controlling access to your managed cloud infrastructure involves managing roles.

Every member of an organization has a role within that organization, and
may have a role on each cluster. Cluster-level roles can be set on each
cluster, but users also have a default cluster role.

Users cannot modify their own access control roles, even to lower their
permissions (to ensure organizations cannot lock themselves out). For
this reason it is recommended to have 2 administrators for your
organization.

The organization members and their roles can be edited from the
``Members`` tab in the UI.

Organization Roles
==================

The ``admin`` role in an organization enables a user to create clusters
and perform all other administrative actions on them, and edit both
organization and cluster-level access for other users. This role
overrides any other cluster-level access and grants a user full access
to everything in the organization.

The ``user`` role in an organization enables a user to log in and see a
directory of clusters and team members. But they must also have access
to specific clusters to do anything more.

Users with the ``admin`` role also have the privileges of the ``user``
role.

Cluster Roles
=============

The ``admin`` role on a cluster enables a user to perform management
actions on that cluster such as pausing, resuming, reconfiguring,
deleting, etc. It also enables a user to modify the access control
settings for that particular cluster.

The ``user`` role on a cluster enables a user to interact with that
cluster through the web portal or the CLI. Users can browse experiment
history and results, submit workloads, etc.

Users with the ``admin`` role also have the privileges of the ``user``
role.

Editing User Roles
==================

The ``Members`` tab in an organization allows organization
administrators to add and remove users in the organization, modify their
role within the organization, and modify their default cluster role.

Access control settings for a specific cluster can be accessed by users
who have the ``admin`` role on that cluster (or the organization). The
``User Access`` option in the cluster context menu

Removing Pending Invites
========================

Organization administrators should also have access to the ``Invites``
tab from the ``Members`` page. From here administrators can view and
cancel pending invites to the organization.
