##########################
 Kubernetes Observability
##########################

This provides documentation and recommendations on how to set up monitoring for a Determined installation on Kubernetes.

*********
 Prereqs
*********

TODO bullet point

* Determied must be running in a Kubernetes cluster.
* Determined must have the Helm value ``observability.enable_prometheus`` set to true. (TODO add default true)
* Determined CLI must be installed and configured to talk to the Determined instance.
* Kubectl must be installed and configured for the Kubernetes cluster.
* The Kubernetes namespace ``det-monitoring`` should be created and non empty.
* Change directory to the determined repo ``tools/observability``.

  This can be done with ``git clone https://github.com/determined-ai/determined.git && cd determined/tools/observability``

***************
 Token refresh
***************

Determined Prometheus export endpoint is secured by authentication. As a result a Determined authentication token is needed for the Prometheus scraper. Determined tokens have an expiration of 1 week. So we are going to configure a token refresh cronjob to run on the Kubernetes cluster.

#. Create a Determined account that will be used in the job.

``det -u admin user create tokenrefresher``

#. Change the password of the Determined account.

``det -u admin user change-password tokenrefresher``

#. Store the username and password inside a credential.

``kubectl -n det-monitoring create secret generic token-refresh-username-pass --from-literal="creds=tokenrefresher:testPassword1"``

#. Create the job and cronjob.

   Warning ``tokenRefresher.yaml`` won't work in every Kuberenetes set up. If you have a unique use case it can be modified to work without much effort. Hard coding the Determined master ip can reduce a lot of assumptions the script makes.

``kubectl -n det-monitoring apply -f tokenRefresher.yaml``

#. Wait for a few minutes then check the ``det-prom-token`` was created.

``kubectl -n det-monitoring describe secret det-prom-token``

Should show
``
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

Nvidia GPU metrics 
