.. _kubernetes-observability:

##########################
 Kubernetes Observability
##########################

This guide provides recommendations and helps you set up monitoring for a Determined installation on
Kubernetes.

***************
 Prerequisites
***************

-  Determined must be running in a Kubernetes cluster.
-  The Helm value ``observability.enable_prometheus`` must be set to ``true`` (this is the default
   setting).
-  :ref:`CLI <cli-ug>` must be installed.
-  ``Kubectl`` must be installed and configured.

******************
 How to Configure
******************

Create a Namespace
==================

-  Create a Kubernetes namespace, ``det-monitoring``:

   .. code:: bash

      kubectl create ns det-monitoring

Change Directory
================

-  Navigate to the ``tools/observability`` repository:

   .. code:: bash

      git clone https://github.com/determined-ai/determined.git && \
        cd determined/tools/observability

Token Refresh
=============

The Determined Prometheus export endpoint is secured by authentication. As a result, a Determined
authentication token, which expires after one week, is required for the Prometheus scraper. In this
section, we'll configure a token refresh cron job to run on the Kubernetes cluster.

#. Create a Determined account to use in the job.

.. code:: bash

   det -u admin user create tokenrefresher

2. Change the password of the Determined account.

.. code:: bash

   det -u admin user change-password tokenrefresher

3. Store the username and password inside a credential.

.. code:: bash

   kubectl -n det-monitoring create secret generic token-refresh-username-pass \
     --from-literal="creds=tokenrefresher:testPassword1"

4. Create the job and cron job.

.. attention::

   ``tokenRefresher.yaml`` may not work work in every Kuberenetes setup. If you have a unique use
   case, it might need modification. Hardcoding the Determined master IP can reduce assumptions made
   by the script.

.. code:: bash

   kubectl -n det-monitoring apply -f tokenRefresher.yaml

5. After a few minutes, check that the ``det-prom-token`` secret was created and ``det-token`` is
   more than 0 bytes.

.. code:: bash

   kubectl -n det-monitoring describe secret det-prom-token

Install DCGM Exporter
=====================

The DCGM exporter allows Prometheus to collect GPU metrics. The installation method varies depending
on your environment.

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

5. In a new console window, check verify the service.

.. code:: bash

   curl 127.0.0.1:9400/metrics

Install Kube Prometheus Stack
=============================

This section helps you install a Kube Prometheus Stack.

For more information, you can visit the `Kube Prometheus stack documentation
<https://github.com/prometheus-community/helm-charts/tree/main/charts/kube-prometheus-stack>`__.

#. Add the Helm repo and update.

.. code:: bash

   helm repo add prometheus-community \
     https://prometheus-community.github.io/helm-charts && \
     helm repo update

2. Install the Kube Prometheus Stack. Change the password in the below command.

.. code:: bash

   helm -n det-monitoring install monitor prometheus-community/kube-prometheus-stack \
     --set grafana.adminPassword=testPassword \
     --values grafana-prom-values.yaml

Monitoring Dashboard
====================

-  Add an API monitoring dashboard.

.. code:: bash

   kubectl -n det-monitoring create configmap det-api-dash --from-file api-dash.json && \
     kubectl -n det-monitoring label configmap det-api-dash grafana_dashboard=1

-  Add a resource utilization dashboard.

.. code:: bash

   kubectl -n det-monitoring create configmap det-resource-utilization-dash --from-file resource-utilization-dash.json && \
     kubectl -n det-monitoring label configmap det-resource-utilization-dash grafana_dashboard=1

-  Check that Prometheus is running correctly by port forwarding.

.. code:: bash

   kubectl -n det-monitoring port-forward service/monitor-kube-prometheus-st-prometheus 9090:9090

-  Verify that Prometheus is scraping DCGM and the Determined API server metrics.

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

.. image:: /assets/images/resource-util-dash-1.png
   :width: 704px
   :align: center
   :alt: Resource Utilization Dashboard Headlines

.. image:: /assets/images/resource-util-dash-2.png
   :width: 704px
   :align: center
   :alt: Resource Utilization Dashboard Cluster Overview

.. image:: /assets/images/resource-util-dash-3.png
   :width: 704px
   :align: center
   :alt: Resource Utilization Dashboard GPU Breakdown

.. image:: /assets/images/resource-util-dash-4.png
   :width: 704px
   :align: center
   :alt: Resource Utilization Dashboard Recent Tasks

Each panel in the dashboard is powered by one or more Prometheus queries.

*********
 Metrics
*********

Determined does not produce metrics, rather it uses existing tools to report information.

API Performance Metrics
=======================

Determined master reports API performance metrics using `grpc ecosystem
<https://github.com/grpc-ecosystem/go-grpc-prometheus?tab=readme-ov-file#metrics>`__.

Kubernetes and Container Metrics
================================

The kube-prometheus-stack enables kube-state-metrics and cAdvisor by default.

-  `kube-state-metrics
   <https://github.com/kubernetes/kube-state-metrics/tree/main/docs#exposed-metrics>`__ reports the
   state of kubernetes objects, including Determined and any resources it creates

-  `cAdvisor <https://github.com/google/cadvisor/blob/master/docs/storage/prometheus.md>`__ reports
   resource usage and performance of running containers, for example memory and cpu used

Nvidia DCGM Exporter
====================

Nvidia's Data Center GPU Manager collects data on Nvidia GPUs.

-  by default, only the most useful `subset
   <https://github.com/GoogleCloudPlatform/prometheus-engine/blob/8dd8a187486cccb5ede3132e5773ae786239dbc2/examples/nvidia-dcgm/exporter.yaml#L139-L169>`__
   of metrics are scraped by Prometheus

-  the full list of metrics generated by DCGM exporter can be found `here
   <https://github.com/NVIDIA/dcgm-exporter/blob/main/etc/dcp-metrics-included.csv>`__

Health Status
=============

Determined master reports a metric, ``determined_healthy``, with value ``1`` when major dependencies
are reachable and ``0`` otherwise. Visit :ref:_prometheus-grafana-alerts` for information on how to
set up alerts.

Viewing Metrics
===============

The Determined Master labels pods it creates with Determined state values. The following pod labels
can be accessed via ``kube_pod_labels`` metric from kube-state-metrics. Label names will be
formatted as ``label_determined_ai_<label_name>``, e.g. ``label_determined_ai_container_id``.

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

Kubernetes resource metrics and GPU metrics can be broken down by Determined resources by joining
data metrics with ``kube_pod_labels`` state metric. As an example, the following PromQL query
computes the average GPU Utilization by Determined experiment ID.

.. code:: bash

   avg by (label_determined_ai_experiment_id)(
      DCGM_FI_DEV_GPU_UTIL * on(pod) group_left(label_determined_ai_experiment_id)
      kube_pod_labels{label_determined_ai_experiment_id!=""}
   )

Some helpful resources (visit PromQL documentation for more details):

-  `joining metrics
   <https://github.com/kubernetes/kube-state-metrics/tree/main/docs#join-metrics>`__ from
   kube-state-metrics

-  `matching vectors
   <https://prometheus.io/docs/prometheus/latest/querying/operators/#vector-matching>`__ from
   Prometheus
