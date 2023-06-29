:orphan:

**New Features**

-  CLI: ``det deploy gcp up`` now uses a default Google Cloud Storage bucket
   ``$PROJECT-ID-determined-deploy`` to store the Terraform state unless a local Terraform state
   file is present or a different Cloud Storage bucket is specified.

-  CLI: A new list function ``det deploy gcp list --project-id <project_id>`` was added that lists
   all clusters under the default Cloud Storage bucket in the given project. Clusters from a
   particular Cloud Storage bucket can also be listed using ``det deploy gcp list --project-id
   <project_id> --tf-state-gcs-bucket-name <tf_state_gcs_bucket_name>``

-  CLI: A new delete subcommand ``det deploy gcp down --cluster-id <cluster_id> --project-id
   <project_id>`` was added that deletes a particular cluster from the project. ``det deploy gcp
   down`` can still be used to delete clusters with local Terraform state files.
