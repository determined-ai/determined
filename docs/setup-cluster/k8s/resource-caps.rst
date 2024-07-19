.. _k8s-resource-caps:

####################################
 Manage Workspace Namespace Bindings
####################################

.. note::
    
    Managing resource quotas in Determined requires the Determined Enterprise Edition.

Determined makes it easier to manage resource quotas by handling them directly, so you 
don't need to manage them separately in Kubernetes. 
You can manage workspace namespace bindings and resource quotas using either the WebUI or CLI.

*******
 WebUI
*******

1. As an admin user, either create a new workspace or edit an existing workspace.

2. In the "Namespace Bindings" section, type a name for the namespace or leave it blank. If left blank, the workspace will be bound to the default namespace configured in the Master Configuration YAML file.

3. Toggle the "Auto Create Namespace" option on or off. When enabled, you can edit the resource quota directly in Determined.

4. Enter a resource quota (e.g., 5). This quota is editable only when "Auto Create Namespace" is on. If the quota is set in Kubernetes, it will appear here but won't be editable.

When completed, save your changes to apply the namespace bindings and resource quotas.

Configure Additional Resource Managers
======================================

For each additional resource manager, configure the resource quota as needed. Each workspace in a cluster automatically has a default binding to their resource manager's namespace.

*****
 CLI
*****

You can use the :ref:`command-line interface (CLI) <cli-ug>` to set one workspace namespace binding at a time. To get help for workspace bindings, run the following command:

.. code:: bash

 det w bindings -h


Set a Namespace Binding
=======================

To bind a workspace to a namespace for a particular cluster, use the following command:

.. code:: bash
    
  det w bindings set <workspace-id> --namespace <namespace-name>

Example:

.. code:: bash
    
  det w bindings set ws2 --namespace ws2-899f-3


Set Resource Quota
==================

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

To list all namespace bindings, use:

.. code:: bash

  det w bindings list








