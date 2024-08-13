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

   To get started with Pachyderm, visit the `Pachyderm documentation
   <https://docs.pachyderm.com/>`_.

****************************************
 Viewing Pachyderm Data from Determined
****************************************

You can view your Pachyderm repo when running a trial or when viewing checkpoints derived from your
Pachyderm data.

**Prerequisites**

-  Determined and Pachyderm installed and running.

   -  Follow a quick :ref:`tutorial <det-pach-cat-dog>` to learn how to set up Determined and
      Pachyderm after which you'll have created a batch inferencing pipeline with editable
      experiment configuration files.

-  The most recent commit ID from your Pachyderm repo (you can find this by visiting Repo Actions).
   For more information about Pachyderm repo input files, visit the `documentation
   <https://docs.pachyderm.com/products/mldm/latest/learn/console-guide/repo-actions/view-inputs//>`_.

**Add an Integrations Section to Your Experiment Configuration File**

To configure this basic data lineage, you must add an ``integrations`` section to your
:ref:`experiment configuration <experiment-config-reference>` file. This section must include your
Pachyderm repo ``dataset``, ``pachd``, and ``proxy``. For example:

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

After running the experiment, go to the Determined WebUI and then visit the experiment's
**Overview** tab. Select the data hyperlink to view your Pachyderm repo.

   .. image:: /assets/images/webui-data-link.png
      :alt: Determined trial run showing link to Pachyderm data repo

**Next Steps**

You can then reuse this ``integrations`` section in your other experiment configuration files such
as in your checkpointing configuration file.
