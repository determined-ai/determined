.. _k8s-resource-caps:

#####################################
 Manage Workspace Namespace Bindings
#####################################

.. note::

   Auto creating namespaces and managing resource quotas in Determined require the Determined
   Enterprise Edition.

Determined makes it easier to manage resource quotas by handling them directly, so you don't need to
manage them separately in Kubernetes. You can manage workspace namespace bindings and resource
quotas using either the WebUI or CLI.

*******
 WebUI
*******

#. As an admin user, either create a new workspace or edit an existing workspace.

#. In the "Namespace Bindings" section, type a name for the namespace or leave it blank. The
   behavior depends on the master configuration. For example, if the ``default_namespace`` is
   defined, leaving the namespace blank binds the workspace to the "default" namespace (even if it
   is not set) configured for that Kubernetes cluster. If a different namespace name is provided,
   the workspace is bound to that namespace.

#. Toggle the "Auto Create Namespace" option on or off. When enabled, the system automatically
   creates a namespace in the cluster, allowing you to edit the resource quota directly in
   Determined.

#. Enter a resource quota (e.g., 5). This quota is editable only when "Auto Create Namespace" is on.
   If the quota is set in Kubernetes, it will appear here but won't be editable.

When completed, save your changes to apply the namespace bindings and resource quotas. Any submitted
workloads that would cause workspace resource use to exceed the defined quota will stay pending
until resources are available.

Configure Additional Resource Managers
======================================

For each additional resource manager, configure the resource quota as needed. Each workspace in a
cluster automatically has a default binding to its resource manager's namespace.

*****
 CLI
*****

You can use the :ref:`command-line interface (CLI) <cli-ug>` to set one workspace namespace binding
at a time. To get help for workspace bindings, run the following command:

.. code:: bash

   det w bindings -h

Create a Workspace with Bindings and Quotas
===========================================

You can add bindings and set quotas during workspace creation:

.. code:: bash

   det w create <workspace-name> --cluster-name <cluster-name> --namespace <namespace-name> --resource-quota <resource-quota>

Additional arguments such as ``--auto-create-namespace`` and ``--auto-create-namespace-for-all`` are
also valid.

Auto Create Namespaces
======================

To auto create a namespace for a specific cluster:

.. code:: bash

   det w set <workspace-id> --cluster-name <cluster-name> --auto-create-namespace

To auto create namespaces for all clusters:

.. code:: bash

   det w set <workspace-id> --auto-create-namespace-all-clusters

Set a Namespace Binding
=======================

To bind a workspace to a namespace for a particular cluster, use the following command:

.. code:: bash

   det w bindings set <workspace-id> --cluster-name <cluster-name> --namespace <namespace-name>

For a cluster with a single resource manager, the ``cluster-name`` is optional.

Example:

.. code:: bash

   det w bindings set ws2 --namespace ws2-899f-3

Set a Resource Quota
====================

To set the resource quota for a workspace in a specific cluster, use:

.. code:: bash

   det w resource-quota set <workspace-id> <quota> --cluster-name <cluster-name>

Example:

.. code:: bash

   det w resource-quota set ws2 5 --cluster-name c1

If you set resource quotas in Kubernetes, they will display in Determined but will not be editable.

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
