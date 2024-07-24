:orphan:

**New Features**

-  WebUI: Add a "Namespace Bindings" section to the Create and Edit Workspace modals.

   -  Users can input a namespace for a Kubernetes cluster. If no namespace is specified, the
      workspace will use the ``resource_manager.default_namespace`` from the master configuration
      YAML or default to the Kubernetes "default" namespace.

   -  In the enterprise edition, users can auto-create namespaces and set resource quotas, limiting
      GPU requests for that workspace. The Edit Workspace modal displays the lowest GPU limit
      resource quota within the bound namespace.

   -  Once saved, all workloads in the workspace will be sent to the bound namespace. Changing the
      binding will affect future workloads, while in-progress workloads remain in their original
      namespace.

-  CLI: Add new commands for creating and managing workspace namespace bindings.

-  Allow creating namespace bindings during workspace creation with ``det w create <workspace-id>
   --namespace <namespace-name>`` or later with ``det w bindings set <workspace-id> --namespace
   <namespace-name>``.

-  In the enterprise edition, users can use additional arguments ``--auto-create-namespace`` and
   ``--auto-create-namespace-all-clusters`` for auto-created namespaces. Set resource quotas during
   workspace creation with ``det w create <workspace-name> --cluster-name <cluster-name>
   --auto-create-namespace --resource-quota <resource-quota>``, or later with ``det w resource-quota
   set <workspace-id> <quota> --cluster-name <cluster-name>``.

-  Add command to delete namespace bindings with ``det w bindings delete <workspace-id>
   --cluster-name <cluster-name>``.

-  Add command to list bindings for a workspace with ``det w bindings list <workspace-name>``. The
   ``--cluster-name`` field is required only for MultiRM setups.

**API:**

-  API: Add new endpoints for creating and managing workspace namespace bindings.
-  Add POST and DELETE endpoints to ``/api/v1/workspaces/{workspace_id}/namespace-bindings`` for
   setting and deleting workspace namespace bindings.
-  Add a GET endpoint ``/api/v1/workspaces/{id}/list-namespace-bindings`` to list namespace bindings
   for a workspace.
-  Add a POST endpoint ``/api/v1/workspaces/{id}/set-resource-quota`` to set resource quotas on
   workspaces bound to auto-created namespaces.
-  Add a GET endpoint ``/api/v1/workspaces/{id}/get-k8-resource-quotas`` to retrieve enforced
   Kubernetes resource quotas for bound namespaces
