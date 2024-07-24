:orphan:

**New Features**

-  WebUI:
      -  A new section called "Namespace Bindings" has been added to the Create Workspace and Edit
         Workspace modals. Users can input a namespace to which they want to bind their workspace
         for a given Kubernetes cluster. If no namespace is specified, the workspace will be bound
         to the namespace specified in the ``resource_manager.default_namespace`` field in the
         master configuration YAML. If this field is left blank, then the workspace will be bound to
         the default Kubernetes ``default`` namespace instead.

      -  In the Enterprise Edition, users have the additional option of auto-creating a namespace to
         which the workspace will be bound. If this option is selected, users can also set a
         resource quota on that auto-created namespace. This will limit the GPU requests available
         to that workspace from a given Kubernetes cluster. The Edit Workspace Modal will display
         the enforced resource quota placed on the workspace, which is the lowest GPU limit resource
         quota that exists within the bound Kubernetes namespace.

      -  Once the workspace-namespace binding is saved, all workloads created in that workspace will
         be sent to the bound namespace. If a user decides to change their workspace-namespace
         binding, future workloads will get sent to the new namespace, but old workloads that are
         still in progress will remain running.

-  API:
      -  Added a new Post and Delete API endpoint
         ``/api/v1/workspaces/{workspace_id}/namespace-bindings`` that allows users to set and
         delete workspace namespace bindings.

      -  Added a Get API endpoint ``/api/v1/workspaces/{id}/list-namespace-bindings`` that allows
         users to list namespace bindings for a given workspace.

      -  Added a Post API endpoint ``/api/v1/workspaces/{id}/set-resource-quota`` that allows users
         to set the resource quotas on workspaces bound to auto-created namespaces.

      -  Added a Get API endpoint ``/api/v1/workspaces/{id}/get-k8-resource-quotas`` that gets the
         enforced Kubernetes resource quota for the namespaces bound to a given workspace.

      -  Added new parameters to the Get and Patch ``/api/v1/workspaces/{id}`` endpoint to allow
         creating namespace bindings and setting resource quotas.

      -  Added a Get API endpoint ``api/v1/namespace-bindings/workspace-ids-with-default-bindings``
         and a Post API endpoint ``api/v1/namespace-bindings/bulk-auto-create`` that allow users to
         migrate to the new workspace namespace bindings feature. The users can use the
         ``/api/v1/namespace-bindings/workspace-ids-with-default-bindings`` to fetch the workspace
         IDs of workspaces that have at least one default binding, and pass those into
         ``/api/v1/namespace-bindings/bulk-auto-create`` to auto-create namespace bindings for all
         those workspaces. This endpoint will only auto-create bindings for those clusters that do
         not have an explicit binding. For example, if workspace ``W1`` has a default binding for
         cluster ``A`` and is bound to namespace ``N1`` for cluster ``B``, this endpoint will only
         auto-create a namespace and bind it for cluster ``A``.

-  CLI:
      -  Users can create namespace bindings during workspace creation using the ``det w create
         <workspace-id> --namespace <namespace-name>`` command or can set it later on using the
         ``det w bindings set <workspace-id> --namespace <namespace-name>`` command.

      -  In the enterprise Edition, Users have additonal optional arguments
         ``--auto-create-namespace`` and ``--auto-create-namespace-all-clusters`` to bind a
         workspace to auto-created namespace(s). If a workspace is bound to an autocreated
         namespace, the users can also set the resource quota during workspace creation Ex. ``det w
         create <workspace-name> --cluster-name <cluster-name> --auto-create-namespace
         --resource-quota <resource-quota>``. Users can also the set resource quota using ``det w
         resource-quota set <workspace-id> <quota> --cluster-name <cluster-name>``. The field
         ``--cluster-name`` is only required when using MultiRM.

      -  Added a new command to delete namespace bindings ``det w bindings delete <workspace-id>
         --cluster-name <cluster-name>``. Added a new command to list bindings for a given workspace
         ``det w bindings list <workspace-name>``.
