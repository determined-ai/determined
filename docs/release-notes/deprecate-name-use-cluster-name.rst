:orphan:

**Breaking Changes**

-  Master Configuration: ``resource_manager.name`` field is replaced by
      ``resource_manager.cluster_name`` for better clarity and to support multiple resource
      managers.

      -  Resource managers operate relative to the specific cluster providing resources for
            Determined tasks, so changing the cluster will affect the resource manager's responses.
      -  The ``cluster_name`` must be unique for all resource managers when deploying multiple
            resource managers in Determined.
      -  During upgrade, replace ``name`` with ``cluster_name`` in the ``resource_manager`` section
            of your master configuration YAML.
