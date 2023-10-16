.. _integrations-index:

##############
 Integrations
##############

.. meta::
   :description: Discover how Determined integrates with other popular machine learning ecosystem tools.

Determined is designed to easily integrate with other popular ML ecosystem tools for tasks that are
related to model training, such as ETL, ML pipelines, and model serving. It is recommended to use
the :ref:`python-sdk` to interact with Determined.

-  :ref:`data-transformers`: Dive into how Determined integrates with data transformation tools such
   as :ref:`pachyderm-integration`.
-  :ref:`ides-index`: Determined shells can be used in the popular IDEs similarly to a common remote
   SSH host.
-  :ref:`notifications`: Make use of webhooks to integrate Determined into your existing workflows.
-  :ref:`prometheus-grafana`: Discover how to enable a Grafana dashboard to monitor Determined
   hardware and system metrics on a cloud cluster, such as AWS or Kubernetes.

Learn more:

Visit the `Works with Determined <https://github.com/determined-ai/works-with-determined>`__
repository to find examples of how to use Determined with a variety of ML ecosystem tools, including
Pachyderm, DVC, Delta Lake, Seldon, Spark, Argo, Airflow, and Kubeflow.

.. toctree::
   :hidden:
   :glob:

   ./*/_index
