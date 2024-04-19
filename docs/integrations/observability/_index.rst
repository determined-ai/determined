##########################
 Kubernetes Observability
##########################

This provides documentation and recommendations on how to set up monitoring for a Determined
installation on Kubernetes.

*********
 Prereqs
*********

-  Determined must be running in a Kubernetes cluster.

-  Determined must have the Helm value ``observability.enable_prometheus`` set to true. This is
   defaulted to true.

-  Determined CLI must be installed and configured to talk to the Determined instance.

-  Kubectl must be installed and configured for the Kubernetes cluster.

-  The Kubernetes namespace ``det-monitoring`` should be created.

   .. code:: bash

      kubectl create ns det-monitoring

-  Change directory to the determined repo ``tools/observability``. This can be done with

   .. code:: bash

      git clone https://github.com/determined-ai/determined.git && \
        cd determined/tools/observability

***************
 Token refresh
***************

Determined Prometheus export endpoint is secured by authentication. As a result a Determined
authentication token is needed for the Prometheus scraper. Determined tokens have an expiration of 1
week. So we are going to configure a token refresh cronjob to run on the Kubernetes cluster.

#. Create a Determined account that will be used in the job.

.. code:: bash

   det -u admin user create tokenrefresher

2. Change the password of the Determined account.

.. code:: bash

   det -u admin user change-password tokenrefresher

3. Store the username and password inside a credential.

.. code:: bash

   kubectl -n det-monitoring create secret generic token-refresh-username-pass \
     --from-literal="creds=tokenrefresher:testPassword1"

4. Create the job and cronjob.

.. attention::

   ``tokenRefresher.yaml`` won't work in every possible Kuberenetes set up. If you have a unique use
   case it might need to be modified to work. Hardcoding the Determined master ip can reduce a lot
   of assumptions the script makes.

.. code:: bash

   kubectl -n det-monitoring apply -f tokenRefresher.yaml

5. Wait for a few minutes then check the ``det-prom-token`` secret was created and ``det-token`` is
   more than 0 bytes.

.. code:: bash

   kubectl -n det-monitoring describe secret det-prom-token

***********************
 Install DCGM Exporter
***********************

The DCGM exporter is used to allow Prometheus to get GPU metrics. This can be installed in a variety
of different ways. If you are deploying in a cloud based environment you should follow their
documentation.

A setup method for GKE is included here for convenience. For other clouds or on prem deployments you
may have to install the DCGM exporter slightly differently and change the
``additionalScrapeConfigs`` accordingly in the ``grafana-prom-values.yaml`` in later steps.

If you are deploying on prem it is recommended to reference the `Nvidia docs on installing the DCGM
exporter
<https://docs.nvidia.com/datacenter/cloud-native/gpu-telemetry/latest/kube-prometheus.html#setting-up-dcgm>`__.

#. Create a namespace for the exporter

.. code:: bash

   kubectl create ns gmp-public

2. Copy and apply the file from the `GKE documentation
   <https://cloud.google.com/stackdriver/docs/managed-prometheus/exporters/nvidia-dcgm#install-exporter>`__
   in the ``gmp-public`` namespace

.. code:: bash

   kubectl apply -n gmp-public -f /tmp/file.yaml

3. Create a service for the DCGM exporter.

.. code:: bash

   kubectl apply -n gmp-public -f gkeDCGMExporterService.yaml

This differs from the GKE docs linked above because we are going to deploy a Prometheus installation
instead of using the managed service Google Cloud Offers. It is possible to use the managed offering
from Google Cloud but some features like GPU statistics by user will not work.

4. Verify DCGM works by port forwarding the service.

.. code:: bash

   kubectl -n gmp-public port-forward service/nvidia-dcgm-exporter 9400

5. In a new console tab check the service works.

.. code:: bash

   curl 127.0.0.1:9400/metrics

*******************************
 Install Kube Prometheus Stack
*******************************

Documentation on the `Kube Prometheus stack can be found here
<https://github.com/prometheus-community/helm-charts/tree/main/charts/kube-prometheus-stack>`__.

#. Add the Helm repo.

.. code:: bash

   helm repo add prometheus-community \
     https://prometheus-community.github.io/helm-charts && \
     helm repo update

2. Helm install the Kube Prometheus Stack. Change the password in the below command.

.. code:: bash

   helm -n det-monitoring install monitor prometheus-community/kube-prometheus-stack \
     --set grafana.adminPassword=testPassword \
     --values grafana-prom-values.yaml

3. Add API monitoring dashboard

.. code:: bash

   kubectl -n det-monitoring create configmap det-api-dash --from-file api-dash.json && \
     kubectl -n det-monitoring label configmap det-api-dash grafana_dashboard=1

4. TODO add other dashboards for monitoring
#. Check Prometheus is running properly. Port forward with this command

.. code:: bash

   kubectl -n det-monitoring port-forward service/monitor-kube-prometheus-st-prometheus 9090:9090

Verify that Prometheus is scraping DCGM and the Determined API server metrics. Go to `127.0.0.1:9090
<http://127.0.0.1:9090>`__ and check the query has 2 or more results with a 1 value.

.. code:: bash

   up{job=~"det-master-api-server|gpu-metrics"}

6. Access Grafana to view dashboards.

.. code:: bash

   kubectl -n det-monitoring port-forward svc/monitor-grafana 9000:80

Go to `127.0.0.1:9000 <http://127.0.0.1:9000>`__ and use the username ``admin`` and password the
password we set in the second step. The ``Determined API Server Monitoring`` dashboard should be
included.
