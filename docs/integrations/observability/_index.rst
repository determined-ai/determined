.. _kubernetes-observability:

##########################
 Kubernetes Observability
##########################

This guide provides recommendations and assists in setting up monitoring for a Determined
installation on Kubernetes.

***************
 Prerequisites
***************

-  Determined must be running within a Kubernetes cluster.
-  The Helm value ``observability.enable_prometheus`` must be set to ``true`` (this is the default
   setting).
-  :ref:`CLI <cli-ug>` must be installed.
-  ``Kubectl`` must be installed and configured appropriately.

*********************
 Configuration Steps
*********************

Create a Namespace
==================

-  Run the following command to create a namespace called ``det-monitoring``:

   .. code:: bash

      kubectl create ns det-monitoring

Change Directory
================

-  Clone the repository and navigate to the ``tools/observability`` directory:

   .. code:: bash

      git clone https://github.com/determined-ai/determined.git && \
        cd determined/tools/observability

Token Refresh
=============

The Determined Prometheus export endpoint is secured by authentication, requiring a refreshable
authentication token. Set up a cron job for token refresh as follows:

#. Create an account.

.. code:: bash

   det -u admin user create tokenrefresher

2. Change the account password.

.. code:: bash

   det -u admin user change-password tokenrefresher

3. Store the credentials.

.. code:: bash

   kubectl -n det-monitoring create secret generic token-refresh-username-pass \
     --from-literal="creds=tokenrefresher:testPassword1"

4. Deploy a cron job.

.. attention::

   ``tokenRefresher.yaml`` may not work work in every Kuberenetes setup. If you have a unique use
   case, it might need modification. Hardcoding the Determined master IP can reduce assumptions made
   by the script.

.. code:: bash

   kubectl -n det-monitoring apply -f tokenRefresher.yaml

5. Verify "secret" creation. After a few minutes, check that the ``det-prom-token`` secret was
   created and ``det-token`` is more than 0 bytes.

.. code:: bash

   kubectl -n det-monitoring describe secret det-prom-token

Install DCGM Exporter
=====================

Depending on your environment, follow these steps for installing the DCGM exporter:

Steps for General Cloud Environments
------------------------------------

In general, to install DCGM in a cloud-based environment, follow the documentation for that
environment.

If you are not following the steps described here for GKE as a reference, you may need to change the
``additionalScrapeConfigs`` in the ``grafana-prom-values.yaml``.

If you are deploying on-prem, visit `Nvidia docs on installing the DCGM exporter
<https://docs.nvidia.com/datacenter/cloud-native/gpu-telemetry/latest/kube-prometheus.html#setting-up-dcgm>`__.

Steps for GKE
-------------

#. Create a namespace for the exporter.

.. code:: bash

   kubectl create ns gmp-public

2. Apply the exporter from `the GKE docs
   <https://cloud.google.com/stackdriver/docs/managed-prometheus/exporters/nvidia-dcgm#install-exporter>`__.

.. code:: bash

   kubectl apply -n gmp-public -f https://raw.githubusercontent.com/GoogleCloudPlatform/prometheus-engine/main/examples/nvidia-dcgm/exporter.yaml

3. Create a service for the DCGM exporter.

.. code:: bash

   kubectl apply -n gmp-public -f gkeDCGMExporterService.yaml

This differs from the GKE documentation because we deploy a Prometheus installation instead of using
Google Cloud's managed service. While it is still possible to use Google Cloud's managed service,
some features, such as GPU statistics by user, will not be available.

4. Verify the DCGM exporter is functioning by port forwarding the service and checking metrics.

.. code:: bash

   kubectl -n gmp-public port-forward service/nvidia-dcgm-exporter 9400

5. In a new console window, verify the service.

.. code:: bash

   curl 127.0.0.1:9400/metrics

Install Kube Prometheus Stack
=============================

Follow these instructions for installing a Kube Prometheus Stack. For additional information, you
can visit the `Kube Prometheus stack documentation
<https://github.com/prometheus-community/helm-charts/tree/main/charts/kube-prometheus-stack>`__.

#. Add the Helm repo and update.

.. code:: bash

   helm repo add prometheus-community \
     https://prometheus-community.github.io/helm-charts && \
     helm repo update

2. Install the Kube Prometheus Stack. You'll need to change the password in the following command.

.. code:: bash

   helm -n det-monitoring install monitor prometheus-community/kube-prometheus-stack \
     --set grafana.adminPassword=testPassword \
     --values grafana-prom-values.yaml

Set Up a Monitoring Dashboard
=============================

This section guides you through the process of setting up monitoring dashboards for your Determined
installation on Kubernetes.

-  Add an API monitoring dashboard.

.. code:: bash

   kubectl -n det-monitoring create configmap det-api-dash --from-file api-dash.json && \
     kubectl -n det-monitoring label configmap det-api-dash grafana_dashboard=1

-  Add a resource utilization dashboard.

.. code:: bash

   kubectl -n det-monitoring create configmap det-resource-utilization-dash --from-file resource-utilization-dash.json && \
     kubectl -n det-monitoring label configmap det-resource-utilization-dash grafana_dashboard=1

-  Check Prometheus operation by port forwarding.

.. code:: bash

   kubectl -n det-monitoring port-forward service/monitor-kube-prometheus-st-prometheus 9090:9090

-  Verify metric scraping.

   -  Go to `127.0.0.1:9090 <http://127.0.0.1:9090>`__ and check that the query has two or more
      results with a ``1`` value.

   .. code:: bash

      up{job=~"det-master-api-server|gpu-metrics"}

-  Access Grafana to view the dashboards.

.. code:: bash

   kubectl -n det-monitoring port-forward svc/monitor-grafana 9000:80

-  Navigate to `127.0.0.1:9000 <http://127.0.0.1:9000>`__. Sign in with the username ``admin`` and
   the password you set above. You should see the ``Determined API Server Monitoring`` dashboard.

Dashboard Example
=================

After submitting experiments on the cluster, you should see populated panels in the imported Grafana
dashboard: **Grafana** -> **Dashboards**.

.. figure:: /assets/images/resource-util-dash-1.png
   :alt: Resource Utilization Dashboard Headlines

   Resource Utilization Dashboard Headlines

.. figure:: /assets/images/resource-util-dash-2.png
   :alt: Resource Utilization Dashboard Cluster Overview

   Resource Utilization Dashboard Cluster Overview

.. figure:: /assets/images/resource-util-dash-3.png
   :alt: Resource Utilization Dashboard GPU Breakdown

   Resource Utilization Dashboard GPU Breakdown

.. figure:: /assets/images/resource-util-dash-4.png
   :alt: Resource Utilization Dashboard Recent Tasks

   Resource Utilization Dashboard Recent Tasks

Each panel in the dashboard is powered by one or more Prometheus queries.

*********
 Metrics
*********

Determined does not generate its own metrics; instead, it utilizes existing tools to report
information.

API Performance Metrics
=======================

Determined master reports API performance metrics using `grpc ecosystem
<https://github.com/grpc-ecosystem/go-grpc-prometheus?tab=readme-ov-file#metrics>`__.

Kubernetes and Container Metrics
================================

The kube-prometheus-stack enables ``kube-state-metrics`` and ``cAdvisor`` by default.

-  `kube-state-metrics
   <https://github.com/kubernetes/kube-state-metrics/tree/main/docs#exposed-metrics>`__ reports the
   state of kubernetes objects, including those created by Determined.

-  `cAdvisor <https://github.com/google/cadvisor/blob/master/docs/storage/prometheus.md>`__ reports
   the resource usage and performance of running containers, including metrics such as memory and
   CPU usage.

Nvidia DCGM Exporter
====================

Nvidia's Data Center GPU Manager (DCGM) collects data on Nvidia GPUs.

-  By default, only the most useful `subset
   <https://github.com/GoogleCloudPlatform/prometheus-engine/blob/8dd8a187486cccb5ede3132e5773ae786239dbc2/examples/nvidia-dcgm/exporter.yaml#L139-L169>`__
   of metrics are scraped by Prometheus.

-  The full list of metrics generated by DCGM exporter can be found `here
   <https://github.com/NVIDIA/dcgm-exporter/blob/main/etc/dcp-metrics-included.csv>`__.

Health Status
=============

Determined master reports a metric, ``determined_healthy``, with value of ``1`` when major
dependencies are reachable, and ``0`` otherwise. Visit :ref:_prometheus-grafana-alerts for
information on how to set up alerts.

Viewing Metrics
===============

The Determined Master assigns specific state values to the pods it creates. These pod labels can be
accessed via the ``kube_pod_labels`` metric from "kube-state-metrics". Label names are formatted as
``label_determined_ai_<label_name>``, such as ``label_determined_ai_container_id``.

Kubernetes restricts pod labels to alphanumeric characters, underscores, hyphens, and dots. Any
other characters in Determined resource names will be converted underscores ``(_)`` before being
added as a pod label. Names longer than 63 characters will be truncated.

+-----------------------------+---------------------------------------------------+
| Label Key                   | Label Value                                       |
+=============================+===================================================+
| determined.ai/container_id  |                                                   |
+-----------------------------+---------------------------------------------------+
| determined.ai/experiment_id | ``task_type=TRIAL`` only                          |
+-----------------------------+---------------------------------------------------+
| determined.ai/resource_pool | name of the resource pool, including ``default``  |
+-----------------------------+---------------------------------------------------+
| determined.ai/task_id       |                                                   |
+-----------------------------+---------------------------------------------------+
| determined.ai/task_type     | Determined task type, e.g. ``TRIAL``,             |
|                             | ``NOTEBOOK``, ``TENSORBOARD``                     |
+-----------------------------+---------------------------------------------------+
| determined.ai/trial_id      | ``task_type=TRIAL`` only                          |
+-----------------------------+---------------------------------------------------+
| determined.ai/user          | Determined username that initiated the request    |
+-----------------------------+---------------------------------------------------+
| determined.ai/workspace     | name of the workspace, including                  |
|                             | ``Uncategorized``                                 |
+-----------------------------+---------------------------------------------------+

PromQL Example Query
====================

Kubernetes resource metrics and GPU metrics can be broken down by Determined resources by joining
data metrics with the ``kube_pod_labels`` state metric. As an example, the following PromQL query
computes the average GPU Utilization by Determined experiment ID.

.. code:: bash

   avg by (label_determined_ai_experiment_id)(
      DCGM_FI_DEV_GPU_UTIL * on(pod) group_left(label_determined_ai_experiment_id)
      kube_pod_labels{label_determined_ai_experiment_id!=""}
   )

Additional Resources
====================

For more details on metric operations:

-  Learn about `joining metrics
   <https://github.com/kubernetes/kube-state-metrics/tree/main/docs#join-metrics>`__ from
   kube-state-metrics.

-  Discover how to perform `vector matching
   <https://prometheus.io/docs/prometheus/latest/querying/operators/#vector-matching>`__ in
   Prometheus queries.
