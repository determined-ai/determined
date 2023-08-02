.. _k8s-dev-guide:

###################
 Development Guide
###################

***************
 Prerequisites
***************

Before setting up the Determined master, set up a Kubernetes cluster with GPU-enabled nodes and
Kubernetes version >= 1.19 and <= 1.21. Later versions of Kubernetes may also work. You can set up
Kubernetes manually, or you can use a managed Kubernetes service such as :ref:`GKE
<setup-gke-cluster>` or :ref:`EKS <setup-eks-cluster>`.

**********************************
 Set up a Development Environment
**********************************

To deploy a custom version of the Determined master, we deploy a long-running pod in Kubernetes,
command it to sleep, then exec into it and build our master. To add a sleep command, modify the
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

Next apply the Determined Helm chart and exec into the pod containing the master.

.. code:: bash

   helm install <deployment-name> helm/charts/determined

   # List pods and find the master pod.
   kubectl get pods

   kubectl exec -it <master-pod-name> -- /bin/bash

*********************************
 Set up a Determined Environment
*********************************

Before installing Determined, install the dependencies specified in the `contributing guide
<https://github.com/determined-ai/determined/blob/master/CONTRIBUTING.md>`__.

You can use ``apt`` and ``pip`` to install most of the dependencies, but you will need to download
and manually install `Go <https://golang.org/dl/>`__, `Node <https://deb.nodesource.com/>`__, `the
protobuf compiler <https://github.com/protocolbuffers/protobuf/releases>`__, and `Helm
<https://helm.sh/docs/intro/install/>`__. Here is an example of installing all the necessary
dependencies:

.. code:: bash

   apt-get update
   DEBIAN_FRONTEND=noninteractive apt-get install -y software-properties-common
   DEBIAN_FRONTEND=noninteractive add-apt-repository -y ppa:deadsnakes/ppa
   DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends git-all python3.8-dev python3.8-venv default-jre curl build-essential libkrb5-dev unzip jq

   # Download and install Go 1.20.
   curl -L https://go.dev/dl/go1.20.linux-amd64.tar.gz | tar -xz
   chown -R root:root go
   mv go /usr/local/

   # Download and install Node and TypeScript.
   curl -sL https://deb.nodesource.com/setup_16.x | bash -
   apt-get install -y nodejs
   npm install typescript -g

   # Download and install the protobuf compiler.
   PB_REL="https://github.com/protocolbuffers/protobuf/releases"
   curl -LO $PB_REL/download/v3.19.0/protoc-3.19.0-linux-x86_64.zip
   unzip protoc-3.19.0-linux-x86_64.zip -d $HOME/.local

   # Download and install Helm.
   curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3
   chmod 700 get_helm.sh
   ./get_helm.sh

In addition to installing these packages, update ``.bashrc`` with the new paths.

.. code:: bash

   export GOPATH=$HOME/go
   export PATH=$PATH:/usr/local/go/bin:$GOPATH/bin

   export PATH="$PATH:$HOME/.local/bin"

After completing these steps, clone the Determined repository and create and activate a virtual
environment for Determined. To create a virtual environment, you may use conda or python3-venv. Here
is an example for cloning the repository, then creating and activating an environment with
python3-venv:

.. code:: bash

   git clone https://github.com/determined-ai/determined.git

   mkdir ~/.virtualenvs
   python3.8 -m venv ~/.virtualenvs/determined

   . ~/.virtualenvs/determined/bin/activate

**************************************
 Prepare to run the Determined Master
**************************************

Once the dependencies are installed, prepare the repository to run ``devcluster``, a tool for
running Determined. First, enter the Determined repository and run:

.. code:: bash

   ``make all``

Once that has finished, create a new file at ``~/.devcluster.yaml`` and populate it with the
following fields:

.. code:: bash

   startup_input: "p"

   cwd: /root/determined

   commands:
   p: make -C harness build  # rebuild Python
   w: make -C webui build    # rebuild WebUI
   c: make -C docs build     # rebuild docs

   stages:
   - master:
         pre:
         - sh: make -C proto build
         - sh: make -C master build
         - sh: make -C tools prep-root

         config_file:
         checkpoint_storage:
            type: "gcs"
            bucket: <name of your bucket>
            save_experiment_best: 0
            save_trial_best: 1
            save_trial_latest: 1

         db:
            user: "postgres"
            password: "postgres"
            host: <name of determined db service from `kubectl get services`>
            port: 5432
            name: "determined"
         port: 8081

         resource_manager:
            type: "kubernetes"
            namespace: default
            max_slots_per_pod: 1
            master_service_name: <name of determined master service from `kubectl get services`>

         log:
            level: debug
         root: tools/build

You are now ready to build and run the Determined master! From the Determined repo, run ``devcluster
--no-guess-host`` to build and run the master.

************
 Next Steps
************

-  :ref:`custom-pod-specs`
