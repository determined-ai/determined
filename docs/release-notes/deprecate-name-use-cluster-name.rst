:orphan

**Breaking Changes**

-  Master: Add config field ``resource_manager.cluster_name`` and deprecate ``resource_anager.name``

      -  ``resource_manager.name`` has been deprecated and is replaced by
            ``resource_manager.cluster_name``. This change was prompted by the release of multiRM
            Determined as it provides a more intuitive way of referencing a given resource manager.

      -  In the case of remote (and local Kubernetes) clusters, Determined resource managers function
            relative to the speciifc cluster that provides the resources used by Determined tasks.
            Therefore, changing the underlying cluster referenced by a Determined resource manager
            will change the resource manager's response to (resource or cluster specific) requests.
            We change ``resource_manager.name`` to ``resource_manager.cluster_name`` to increase the
            emphasis on the correlation between a given cluster's resource availability and the
            Determined resource manager's behavior.

      -  Since ``cluster_name`` is used as a unique identifier tied to your Determined resource
            managers, this field must be unique for all resource managers when deploying multiRM
            Determined.

      -  When upgrading Determined, please replace ``name`` with ``cluster_name`` in the
            ``resource_manager`` section of your master configuration YAML.
