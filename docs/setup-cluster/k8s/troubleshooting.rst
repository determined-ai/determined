#################
 Troubleshooting
#################

If you cannot reach the Determined master after installing Determined on Kubernetes, follow these
debugging steps:

.. code:: bash

   # Get the name of the Helm deployment.
   helm list

   # Double check the IP address and port assigned to the Determined master by looking up the master service.
   kubectl get service determined-master-service-development-<helm deployment name>

   # Check the status of master deployment.
   kubectl describe deployment determined-master-deployment-<helm deployment name>

   # Check the logs of master pod.
   kubectl logs <determined-master-pod-name>
