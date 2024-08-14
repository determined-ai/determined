.. _pachyderm-integration:

###########
 Pachyderm
###########

`Pachyderm <https://www.pachyderm.com/>`_ provides data-driven pipelines with version control and
autoscaling that can be used alongside Determined. Pachyderm runs across all major cloud providers
and on-premise installations.

-  Use Pachyderm to store and version your data via repositories and pipelines.
-  Use Determined to train your models on Pachyderm data.

.. tip::

   To learn how to use Determined and Pachyderm together, follow this quick :ref:`tutorial
   <det-pach-cat-dog>`, after which you'll have created a batch inferencing pipeline with editable
   experiment configuration files.

.. tip::

   To learn more about how to get started with Pachyderm, visit the `Pachyderm documentation
   <https://docs.pachyderm.com/>`_.

****************************************
 Viewing Pachyderm Data from Determined
****************************************

Determined provides a basic data lineage to your Pachyderm repo. This allows you to view your
Pachyderm repo when running an experiment or when viewing checkpoints derived from your Pachyderm
data.

To enable basic data lineage, add an ``integrations`` section to your :ref:`experiment configuration
<experiment-config-reference>` file. This section should include your Pachyderm repo ``dataset``,
``pachd``, and ``proxy``. For example:

.. code:: yaml

   integrations:
      pachyderm:
         dataset:
            project: "test-project"
            repo: "test-repo"
            commit: "your commit id"
            branch: "master"
            token: "PACHD Token"
         pachd:
            host: "IP address for pachd"
            port: 30650
         proxy:
            scheme: "http"
            host: "12.345.67.89"
            port: 80

.. note::

   You can find the most recent commit ID from your Pachyderm repo by visiting Repo Actions or by
   running ``pachctl find commit``. For more information about Pachyderm repo input files, visit the
   `documentation
   <https://docs.pachyderm.com/products/mldm/latest/learn/console-guide/repo-actions/view-inputs//>`_.

Optionally, you can make the configuration more dynamic by setting the environment variables that
your training script will use to automatically generate or modify the ``config.yaml`` file.

For example:

.. code:: bash

   export PACH_PROJECT="your-project"
   export PACH_REPO="your-repo"
   export PACH_COMMIT="your-commit"
   export PACH_BRANCH="your-branch"
   export PACH_TOKEN="your-token"
   export PACHD_HOST="your-pachd-host"
   export PACHD_PORT=12345
   export PACH_PROXY_SCHEME="http"
   export PACH_PROXY_HOST="your-proxy-host"
   export PACH_PROXY_PORT=8080

After running your training script, go to the Determined WebUI and visit the experiment's
**Overview** tab. Select the data hyperlink to view your Pachyderm repo.

   .. image:: /assets/images/webui-data-link.png
      :alt: Determined trial run showing link to Pachyderm data repo

**Next Steps**

You can reuse this ``integrations`` section or the dynamic configuration method in your other
experiment configuration files, such as in your checkpointing configuration file.
