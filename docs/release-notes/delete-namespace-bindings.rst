:orphan:

**New Features**

-  CLI: Add a feature where Determined offers the users to delete a workspace namespace binding by
   using the command ``det w bindings delete <workspace-id> --cluster-name <cluster-name>``. An
   error will be thrown if the user tries to delete a default binding.

-  API: Add a feature where Determined offers the users to delete a workspace namespace binding by
   using the api endpoint ``api/v1/workspaces/1/namespace-bindings`` which takes in the workspace ID
   and string array of cluster names as parameters. An error will be thrown if the user tries to
   delete a default binding.
