.. _k8s-dev-guide:

###################
 Development Guide
###################

***************
 Prerequisites
***************

Before setting up Determined Master, the user should set up a Kubernetes cluster with GPU enabled
nodes and Kubernetes version >= 1.19 and <= 1.21, though later versions may work. You can setup
Kubernetes manually, or you can use a managed Kubernetes service such as :ref:`GKE
<setup-gke-cluster>` or :ref:`EKS <setup-eks-cluster>`.

**********************************
 Set up a Development Environment
**********************************

To deploy a custom version of the Determined Master, we deploy a long-running pod in Kubernetes,
command it to sleep, then exec into it and build our Master. To add a sleep command, modify the
template ``helm/charts/determined/templates/master-deployment.yaml`` to include the command and
args:

.. code:: yaml

   spec:
     ...
     containers:
     - name: determined-master-{{ .Release.Name }}
       ...
       command: ["sleep"]
       args: ["99999m"]
       ...

Next apply the Determined Helm chart and exec into the pod containing Master.

.. code:: bash

   helm install <deployment-name> helm/charts/determined

   # List pods and find the Master pod
   kubectl get pods

   kubectl exec -it <master-pod-name> -- /bin/bash

*********************************
 Set up a Determined Environment
*********************************

Before installing Determined, you will need to install the dependencies specified in the
`contributing guide <https://github.com/determined-ai/determined/blob/master/CONTRIBUTING.md>`__.

You can use ``apt`` and ``pip`` to install most of the dependencies, but you will need to download
and manually install `golang <https://golang.org/dl/>`__, `node <https://deb.nodesource.com/>`__,
`protobuf <https://github.com/protocolbuffers/protobuf/releases>`__, and `helm
<https://helm.sh/docs/intro/install/>`__. Here is an example of installing all the necessary
dependencies:

.. code:: bash

   # Making man1 is necessary to prevent errors associated with installing jre
   mkdir /usr/share/man/man1

   apt-get update
   apt-get install -y --no-install-recommends software-properties-common git-all python3.7 python3-venv default-jre curl build-essential libkrb5-dev unzip

   # Download and install golang 1.15
   curl -L https://golang.org/dl/go1.15.7.linux-amd64.tar.gz | tar -xz
   chown -R root:root ./go/
   mv go /usr/local/

   # Download and install node and typescript
   curl -sL https://deb.nodesource.com/setup_12.x | bash -
   apt-get install -y nodejs
   npm install typescript -g

   # Download and install protobuf
   PB_REL="https://github.com/protocolbuffers/protobuf/releases"
   curl -LO $PB_REL/download/v3.12.1/protoc-3.12.1-linux-x86_64.zip
   unzip protoc-3.12.1-linux-x86_64.zip -d $HOME/.local

   # Download and install helm
   curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3
   chmod 700 get_helm.sh
   ./get_helm.sh

In addition to installing these packages, ``.bashrc`` needs to be updated with new paths.

.. code:: bash

   export GOPATH=$HOME/go
   export PATH=$PATH:/usr/local/go/bin:$GOPATH/bin

   export PATH="$PATH:$HOME/.local/bin"

After completing these steps, clone the Determined repository and create and activate a virtual
environment for Determined.

**********************************
 Prepare to run Determined Master
**********************************

After the dependencies have been installed, some changes need to be made within the Determined
repository. First, copy the Master configuration found at ``/etc/determined/master.yaml`` and place
it under a new name inside the Determined repo, for example at `determined/tools/k8s-master.yaml`
(there already exists a master.yaml in the tools dir).

.. code:: bash

   cp /etc/determined/master.yaml <path-to-determined>/tools/k8s-master.yaml

Next, modify the config file you copied and add one extra line at the end:

.. code:: bash

   root: build

After that, edit the file ``determined/tools/run-server.py``. Inside the main function's ``try``
clause, comment out everything except for four lines related to ``master``:

.. code::

   def main() -> None:
     ...
     try: # comment out all lines in here except for these four:
       master = run_master()
       ...
       master.start()
       wait_for_server(8080)
       ...
       master.join()

Do not modify the ``except`` or ``finally`` clauses.

Lastly, inside the ``run_master`` function of ``determined/tools/run-server.py``, change the config
file from ``master.yaml`` to the copied master config, i.e. ``k8s-master.yaml``.

.. code::

   def run_master() -> mp.Process:
     ...
       ["../master/build/determined-master", "--config-file", "k8s-master.yaml"],
     ...

We are now ready to build and run the Determined Master! From the Determined repo, run ``make all``
to build and ``make -C tools run`` to start the Master.

************
 Next Steps
************

-  :ref:`custom-pod-specs`
