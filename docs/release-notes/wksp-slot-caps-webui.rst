:orphan:

**New Features**

-  WebUI: A new section called "Namespace Bindings" has been added to the Create Workspace and Edit
   Workspace modals. For OSS users, they can input a namespace that they want to bind their
   workspace to while using a Kubernetes RM. This will allow them to send all their workloads for
   that workspace to that particular namespace. If they do not specify a namespace, the workspace
   will be bound to the namespace specified in the ``resource_manager.default_namespace`` field in
   the master config YAML file. If this field is left blank, then Determined will use the
   ``default`` namespace instead. For EE users, an additional option of auto-creating a namespace,
   and setting the resource quota for that namespace has been added. Users can only set resource
   quotas in the WebUI for namespaces created by Determined. The Edit Workspace Modal will display
   the enforced resource quota for all namespaces bound to the workspace.

-  API: Added 2 API endpoints ``api/v1/namespace-bindings/workspace-ids-with-default-bindings`` and
   ``api/v1/namespace-bindings/bulk-auto-create`` that allow users to migrate to the new feature of
   workspace namespace bindings. The users can use the
   ``/api/v1/namespace-bindings/workspace-ids-with-default-bindings`` to fetch the workspace IDs of
   workspaces that have atleast one default binding, and pass those into
   ``/api/v1/namespace-bindings/bulk-auto-create`` to auto-create namespace bindings for all those
   workspaces. This endpoint will only auto-create bindings for those clusters that do not have an
   explicit binding. For example, if workspace ``W1`` has a default binding for cluster ``A`` and is
   bound to namespace ``N1`` for cluster ``B``, this endpoint will only auto-create a namespace and
   bind it for cluster ``A``.

-  CLI: Added optional arguments to create a workspace with bindings and resource quotas Ex. ``det w
   create <workspace-name> --cluster-name <cluster-name> --namespace <namespace-name>
   --resource-quota <resource-quota>``. Additional arguments such as ``--auto-create-namespace`` and
   ``--auto-create-namespace-all-clusters`` are also valid. Added a new command to set bindings for
   a workspace ``det w bindings set <workspace-id> --cluster-name <cluster-name> --namespace
   <namespace-name>``. Added a new command to set resource quota ``det w resource-quota set
   <workspace-id> <quota> --cluster-name <cluster-name>``. Added a new command to delete namespace
   bindings ``det w bindings delete <workspace-id> --cluster-name <cluster-name>``. Added a new
   command to list bindings for a given workspace ``det w bindings list <workspace-name>``. The
   field ``--cluster-name`` is only required when using MultiRM. The commands for auto-creating
   namespaces and setting resource quotas are only available for EE users.
