.. _k8s-resource-caps:

#####################################
 Manage Workspace-Namespace Bindings
#####################################

.. note::

   Auto creating namespaces and managing resource quotas in Determined require the Determined
   Enterprise Edition.

Determined makes it easier to manage resource quotas for auto-created namespaces by handling them
directly, so you don't need to manage them separately in Kubernetes. You can set workspace-namespace
bindings and resource quotas using either the WebUI or the CLI.

********************
 WebUI Instructions
********************

#. As an admin user, either create a new workspace or edit an existing workspace.

#. In the "Namespace Bindings" section, enter a name for the namespace or leave it blank. If you
   specify a namespace in the namespace field, the workspace is bound to that namespace. If the
   field is left blank, the workspace is bound to the namespace specified in the
   ``resource_manager.default_namespace`` section of your master configuration (when set), and is
   otherwise bound to the default Kubernetes "default" namespace.

#. Toggle the "Auto Create Namespace" option on or off. When enabled, the system automatically
   creates a namespace in the cluster, allowing you to edit the resource quota directly in
   Determined.

#. Enter a resource quota (e.g., 5). This quota is editable only when "Auto Create Namespace" is on.
   If the quota for a namespace is created in Kubernetes, it will appear here but won't be editable.
   Determined displays the lowest quota on the namespace. If this lowest quota is a Determined-
   created quota, you can edit it in the WebUI. If the lowest quota was created in Kubernetes, it
   will not be editable in the WebUI.

When completed, save your changes to apply the namespace bindings and resource quotas. Any submitted
workloads that would cause workspace resource use to exceed the defined quota will stay pending
until resources are available or the quota is increased.

.. note::

   The resource quotas set and displayed in Determined specifically apply to Kubernetes GPU request
   limits. You can cap the GPU requests placed on a given workspace. Other Kubernetes resources like
   CPU or memory request limits are not managed by Determined.

.. important::

   If a job is pending due to lack of resources (not resource quota) and you change the resource
   quota, the job will be scheduled regardless of any new, more restrictive resource quota.

Configure Additional Resource Managers
======================================

For each additional resource manager, configure the resource quota as needed. Each workspace in a
cluster automatically has a default binding to its resource manager's configured or default
namespace.

******************
 CLI Instructions
******************

You can use the :ref:`command-line interface (CLI) <cli-ug>` to set one workspace namespace binding
at a time. To get help for setting workspace namespace bindings, run the following command:

.. code:: bash

   det w bindings -h

Create a Workspace with Bindings and Quotas
===========================================

You can add bindings and set quotas during workspace creation using the following command where
``cluster-name`` is the ``resource_manager.cluster_name`` specified in the master configuration:

.. code:: bash

   det w create <workspace-name> --cluster-name <cluster-name> --namespace <namespace-name> --resource-quota <resource-quota>

Additional arguments such as ``--auto-create-namespace`` and
``--auto-create-namespace-all-clusters`` are also valid.

Auto Create Namespaces
======================

To bind a workspace to an auto-created namespace for a specific cluster:

.. code:: bash

   det w bindings set <workspace-id> --cluster-name <cluster-name> --auto-create-namespace

To auto create namespaces for all clusters:

.. code:: bash

   det w bindings set <workspace-id> --auto-create-namespace-all-clusters

Set a Namespace Binding
=======================

To bind a workspace to an existing namespace for a particular cluster, use the following command:

.. code:: bash

   det w bindings set <workspace-id> --cluster-name <cluster-name> --namespace <namespace-name>

For a Determined cluster with a single resource manager, the ``cluster-name`` is optional.

Example:

.. code:: bash

   det w bindings set ws2 --namespace ws2-899f-3

Set a Resource Quota
====================

To set the resource quota on a workspace for a specific cluster, use:

.. code:: bash

   det w resource-quota set <workspace-id> <quota> --cluster-name <cluster-name>

Example:

.. code:: bash

   det w resource-quota set ws2 5 --cluster-name c1

Delete a Namespace Binding
==========================

To delete a workspace namespace binding, use:

.. code:: bash

   det w bindings delete <workspace-id> --cluster-name <cluster-name>

Note: An error will be thrown if you try to delete a default binding.

List Namespace Bindings
=======================

To list bindings for a particular workspace:

.. code:: bash

   det w bindings list <workspace-name>

***************
 API Endpoints
***************

The following API endpoints facilitate migrating to the workspace namespace bindings feature.

Fetch Workspace IDs with Default Bindings
=========================================

-  Endpoint: ``/api/v1/namespace-bindings/workspace-ids-with-default-bindings``
-  Description: Use this endpoint to fetch the workspace IDs of workspaces that have at least one
   default binding.
-  Usage: This can help identify which workspaces need namespace bindings to be auto-created.

Bulk Auto-Create Namespace Bindings
===================================

-  Endpoint: ``/api/v1/namespace-bindings/bulk-auto-create``
-  Description: Use this endpoint to auto-create namespace bindings for all specified workspaces.
-  Details: Pass the workspace IDs fetched from the previous endpoint into this endpoint. It will
   auto-create namespace bindings for clusters that do not have an explicit binding.
-  Example: If workspace W1 has a default binding for cluster A and is bound to namespace N1 for
   cluster B, this endpoint will only auto-create a namespace and bind it for cluster A.
