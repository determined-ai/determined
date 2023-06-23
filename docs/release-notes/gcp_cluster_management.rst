:orphan:

**New Features**

-  CLI: ``det deploy gcp up`` now uses a default gcs bucket ``$PROJECT-ID-determined-deploy`` to
   store the tf state unless a local tf state file is present or a different gcs bucket is
   specified.

-  CLI: A new list function ``det deploy gcp list --project-id <project_id>`` was added that lists
   all clusters under the default gcs bucket in the given project. Clusters from a particular gcs
   bucket can also be listed using ``det deploy gcp list --project-id <project_id>
   --tf-state-gcs-bucket-name <tf_state_gcs_bucket_name>``

-  CLI: A new delete subcommand ``det deploy gcp down --cluster-id <cluster_id> --project-id
   <project_id>`` was added that deletes a particular cluster from the project. ``det deploy gcp
   down`` can still be used to delete clusters with local tf state files.
