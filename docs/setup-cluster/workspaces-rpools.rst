.. _resource-pool-binding:

######################################
 Binding Resource Pools to Workspaces
######################################

.. meta::
   :description: Discover how to associate resource pools to specific workspaces in the same way you associate certain artifacts, like experiments, to workspaces.

You can associate :ref:`resource pools <resource-pools>` to specific workspaces similar to how you
associate certain artifacts, like experiments, to workspaces.

.. attention::

   Modifying resource pool bindings requires either an Admin or a Cluster Admin role.

**********
 Overview
**********

Binding and unbinding resource pools allows administrators to control resource pool availability
within the cluster.

Resource pools can be either unbound, meaning they are shared across the entire cluster, or bound to
specific workspaces. Experiments, notebooks, TensorBoards, shells, or commands associated with a
particular workspace can only use resource pools that are either unbound or bound to a particular
workspace.

In addition, you can set a bound resource pool as the default compute or auxiliary pool for the
workspace. If a user leaves the resource pool configuration option blank for their task, workloads
will be sent to the default compute or auxiliary pool.

When combined with :ref:`Role-Based Access Control (RBAC) <rbac>`, administrators can restrict
compute resources to specific users and groups, enabling resource multi-tenancy for experiments and
related artifacts.

**************************************
 Binding or Unbinding a Resource Pool
**************************************

You can bind or unbind a resource pool to a workspace. By default, all resource pools are unbound,
making them globally available to all workspaces in the cluster.

.. note::

   Default resource pools cannot be bound to a workspace.

WebUI
=====

An administrator who is an Admin in Determined or a user with the Cluster Admin role (requires
Determined Enterprise Edition) can change resource pool bindings in the WebUI by following these
steps:

-  Go to the Cluster option in the left menu pane.
-  Click the options "..." menu of the resource pool you want to bind.
-  Select **Manage bindings** from the menu.
-  On the right side, choose the workspace you want to bind by clicking on it in the available
   workspaces list.
-  You can use the search bar to narrow down the list.
-  The selected workspace will be added to the list of bound workspaces on the left.
-  To remove a bound workspace, click on it in the list of bound workspaces on the left.
-  If you want to remove all workspace bindings, click **Remove all** below the list of bound
   workspaces.
-  Once you are satisfied with the list of bound workspaces, click **Apply**.

CLI
===

You can use the resource pool command, ``det rp bindings``, to add, list, replace, bind, or unbind a
resource pool using the CLI. To get help, run the following command:

.. code:: bash

   det rp bindings -h

For example, to bind a resource pool, run the following command:

.. code:: bash

   det rp bindings add "<resource_pool_name>" "<workspace_name>"

To bind multiple workspaces to a resource pool, run the following command:

.. code:: bash

   det rp bindings add "<resource_pool_name>" "<workspace1_name>" "<workspace2_name>" "<workspace3_name>"

To unbind a resource pool, run the following command:

.. code:: bash

   det rp bindings remove "<resource_pool_name>" "<workspace_name>"

In the Cluster view, a bound resource pool is indicated by a lock symbol and the number of
workspaces it is bound to. Moreover, clicking on a resource pool card from the Cluster view displays
all the workspaces that are bound to that resource pool.

The resource pool binding to workspaces follows a many-to-many relationship. This gives
administrators the flexibility to bind multiple resource pools to the same workspace or the same
resource pool to multiple workspaces.

.. attention::

   If a resource pool has tasks running from a particular workspace and that resource pool is
   unbound from that workspace, the existing tasks will continue running. However, new tasks from
   the unbound workspace will not be scheduled on that resource pool.
