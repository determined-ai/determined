##########################
 Kubernetes Observability
##########################

This provides documentation and recommendations on how to set up monitoring for a Determined
installation on Kubernetes.

*********
 Prereqs
*********

-  Determied must be running in a Kubernetes cluster.

-  Determined must have the Helm value ``observability.enable_prometheus`` set to true. (TODO add
   default true)

-  Determined CLI must be installed and configured to talk to the Determined instance.

-  Kubectl must be installed and configured for the Kubernetes cluster.

-  The Kubernetes namespace ``det-monitoring`` should be created and non empty.

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

#. Change the password of the Determined account.

.. code:: bash

   det -u admin user change-password tokenrefresher

#. Store the username and password inside a credential.

.. code:: bash

   kubectl -n det-monitoring create secret generic token-refresh-username-pass \
     --from-literal="creds=tokenrefresher:testPassword1"

#. Create the job and cronjob.

   TODO make this a header

   Warning ``tokenRefresher.yaml`` won't work in every Kuberenetes set up. If you have a unique use
   case it can be modified to work without much effort. Hard coding the Determined master ip can
   reduce a lot of assumptions the script makes.

.. code:: bash

   kubectl -n det-monitoring apply -f tokenRefresher.yaml

#. Wait for a few minutes then check the ``det-prom-token`` secret was created.

.. code:: bash

   kubectl -n det-monitoring describe secret det-prom-token

   Name:         det-prom-token
   Namespace:    default
   Labels:       <none>
   Annotations:  <none>

   Type:  Opaque

   Data
   ====
   det-token:  217 bytes``

***********************
 Install DCGM Exporter
***********************

DCGM is used to allow Prometheus to get GPU metrics. This can be installed in a variety of different
ways. If you are deploying in a cloud based environment you should follow their documentation.

Nvidia docs on installing the DCGM exporter. TODO link
https://docs.nvidia.com/datacenter/cloud-native/gpu-telemetry/latest/kube-prometheus.html#setting-up-dcgm

A setup method for GKE is included here for convenience. For other clouds or on prem deployments you
may have to install DCGM differently and change the ``additionalScrapeConfigs`` accordingly in the
``grafana-prom-values.yaml`` in later steps.

#. Create a namespace for the exporter

.. code:: bash

   kubectl create ns gmp-public

#. Copy and apply the file at this documentation in the ``gmp-public`` namespace

   TODO link
   https://cloud.google.com/stackdriver/docs/managed-prometheus/exporters/nvidia-dcgm#install-exporter

.. code:: bash

   kubectl apply -n gmp-public -f file.yaml

#. Create a service for the DCGM exporter.

.. code:: bash

   kubectl apply -n gmp-public -f gkeDCGMExporterService.yaml

Note this differs from the GKE docs linked above because we are going to deploy a Prometheus
instalation instead of using the managed service Google Cloud Offers. It is possible to use the
managed offering from Google Cloud but some features like GPU statistics by user will not work.

#. Verify DCGM works by port forwarding the service.

.. code:: bash

   kubectl -n gmp-public port-foward service/nvidia-dcgm-exporter 9400

#. In a new console tab check the service works.

   ``curl 127.0.0.1:9400/metrics`` TODO link

*******************************
 Install Kube Prometheus Stack
*******************************

#. Add the Helm repo.

.. code:: bash

   helm repo add prometheus-community \
     https://prometheus-community.github.io/helm-charts && \
     helm update

#. Helm install the Kube Prometheus Stack. Change the password in the below command.

.. code:: bash

   helm install monitor prometheus-community/kube-prometheus-stack \
     --set grafana.adminPassword=testPassword \
     --values grafanaPrometheus.yaml

#. Add API monitoring dashboard

.. code:: bash

   kubectl create configmap detapidash --from-file api-dash.json && \
     kubectl label configmap detapidash grafana_dashboard=1

#. TODO add any other dashboard
#. Check Prometheus is running properly. Port forward with this command

.. code:: bash

   kubectl port-forward service/monitor-kube-prometheus-st-prometheus 9090:9090

Go to 127.0.0.1:9090 and check the query ``up{twojobs}`` returns 1 for both results.

#. Access Grafana to view dashboards.

.. code:: bash

   kubectl port-forward svc/monitor-grafana 9000:80

Go to ``127.0.0.1:9000`` and use the username ``admin`` and password the password we set in the
second step.
