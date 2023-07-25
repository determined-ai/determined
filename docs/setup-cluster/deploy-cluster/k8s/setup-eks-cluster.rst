.. _setup-eks-cluster:

###################################################
 Set Up and Manage an AWS Kubernetes (EKS) Cluster
###################################################

Determined can be installed on a cluster that is hosted on a managed Kubernetes service such as
`Amazon EKS <https://aws.amazon.com/eks/>`_. This document describes how to set up an EKS cluster
with GPU-enabled nodes. The recommended setup includes deploying a cluster with a single non-GPU
node that will host the Determined master and database, and an autoscaling group of GPU nodes. After
creating a suitable EKS cluster, you can then proceed with the standard :ref:`instructions for
installing Determined on Kubernetes <install-on-kubernetes>`.

Determined requires GPU-enabled nodes and the Kubernetes cluster to be running version >= 1.19 and
<= 1.21, though later versions may work.

***************
 Prerequisites
***************

Before setting up an EKS cluster, the user should have the latest versions of `AWS CLI
<https://docs.aws.amazon.com/cli/latest/userguide/install-cliv2.html>`_, `kubectl
<https://kubernetes.io/docs/tasks/tools/install-kubectl/>`_, and `eksctl
<https://eksctl.io/introduction/#installation>`_ installed on their local machine.

Additionally, make sure to be subscribed to the `EKS-optimized AMI with GPU support
<https://aws.amazon.com/marketplace/pp/B07GRHFXGM>`_. Continuing without subscribing will cause node
creation to fail.

*********************
 Create an S3 Bucket
*********************

One resource that ``eksctl`` does not automatically create is an S3 bucket, which is necessary for
Determined to store checkpoints. To quickly create an S3 bucket, use the command:

.. code:: bash

   aws s3 mb s3://<bucket-name>

The bucket name needs to be specified in both the ``eksctl`` cluster config as well as the
Determined Helm chart.

.. _cluster-creation:

********************
 Create the Cluster
********************

The quickest and easiest way to deploy an EKS cluster is with ``eksctl``. ``eksctl`` supports
cluster creation with either command-line arguments or a cluster config file. Below is a template
config that deploys a managed node group for Determined's master instance, as well as an autoscaling
GPU node group for workers. To fill in the template, insert the cluster name and S3 bucket name.

.. code:: yaml

   apiVersion: eksctl.io/v1alpha5
   kind: ClusterConfig

   metadata:
     name: <cluster-name> # Specify your cluster name here
     region: us-west-2 # The default region is us-west-2
     version: "1.19" # 1.20 and 1.21 are also supported

   # Cluster availability zones must be explicitly named in order for single availability zone node groups to work.
   availabilityZones:
     - "us-west-2b"
     - "us-west-2c"
     - "us-west-2d"

   iam:
     withOIDC: true # Enables IAM IODC provider
     serviceAccounts:
     - metadata:
         name: checkpoint-storage-s3-bucket
         # If no namespace is set, "default" will be used.
         # Namespace will be created if it does not already exist.
         namespace: default
         labels:
           aws-usage: "determined-checkpoint-storage"
       attachPolicy: # Inline policy can be defined along with `attachPolicyARNs`
         Version: "2012-10-17"
         Statement:
         - Effect: Allow
           Action:
           - "s3:ListBucket"
           Resource: 'arn:aws:s3:::<bucket-name>' # Name of the previously created bucket
         - Effect: Allow
           Action:
           - "s3:GetObject"
           - "s3:PutObject"
           - "s3:DeleteObject"
           Resource: 'arn:aws:s3:::<bucket-name>/*'
     - metadata:
         name: cluster-autoscaler
         namespace: kube-system
         labels:
           aws-usage: "determined-cluster-autoscaler"
       attachPolicy:
         Version: "2012-10-17"
         Statement:
         - Effect: Allow
           Action:
           - "autoscaling:DescribeAutoScalingGroups"
           - "autoscaling:DescribeAutoScalingInstances"
           - "autoscaling:DescribeLaunchConfigurations"
           - "autoscaling:DescribeTags"
           - "autoscaling:SetDesiredCapacity"
           - "autoscaling:TerminateInstanceInAutoScalingGroup"
           - "ec2:DescribeLaunchTemplateVersions"
           Resource: '*'

   managedNodeGroups:
     - name: managed-m5-2xlarge
       instanceType: m5.2xlarge
       availabilityZones:
         - us-west-2b
         - us-west-2c
         - us-west-2d
       minSize: 1
       maxSize: 2
       volumeSize: 200
       iam:
         withAddonPolicies:
           autoScaler: true
           cloudWatch: true
       ssh:
         allow: true # will use ~/.ssh/id_rsa.pub as the default ssh key
       labels:
         nodegroup-type: m5.2xlarge
         nodegroup-role: cpu-worker
       tags:
         k8s.io/cluster-autoscaler/enabled: "true"
         k8s.io/cluster-autoscaler/user-eks: "owned"
         k8s.io/cluster-autoscaler/node-template/label/nodegroup-type: m5.2xlarge
         k8s.io/cluster-autoscaler/node-template/label/nodegroup-role: cpu-worker

   nodeGroups:
     - name: g4dn-metal-us-west-2b
       instanceType: g4dn.metal # 8 GPUs per machine
       # Restrict to a single AZ to optimize data transfer between instances
       availabilityZones:
         - us-west-2b
       minSize: 0
       maxSize: 2
       volumeSize: 200
       volumeType: gp2
       iam:
         withAddonPolicies:
           autoScaler: true
           cloudWatch: true
       ssh:
         allow: true # This will use ~/.ssh/id_rsa.pub as the default ssh key.
       labels:
         nodegroup-type: g4dn.metal-us-west-2b
         nodegroup-role: gpu-worker
         # https://github.com/kubernetes/autoscaler/tree/master/cluster-autoscaler/cloudprovider/aws#special-note-on-gpu-instances
         k8s.amazonaws.com/accelerator: nvidia-tesla-t4
       tags:
         k8s.io/cluster-autoscaler/enabled: "true"
         k8s.io/cluster-autoscaler/user-eks: "owned"
         k8s.io/cluster-autoscaler/node-template/label/nodegroup-type: g4dn.metal-us-west-2b
         k8s.io/cluster-autoscaler/node-template/label/nodegroup-role: gpu-worker

The cluster specified above allows users to run experiments on an untainted g4dn.metal instances
with minor additions to their experiment configs. To create a cluster with tainted instances, see
the `Tainting Nodes` section below.

To launch the cluster with ``eksctl``, run:

.. code:: bash

   eksctl create cluster --config-file <cluster config yaml>

.. note::

   For an experiment to run, its config must be modified to specify a service account for S3 access
   . An example of this is provided in the Configuring Per-Task Pod Specs section of the
   :ref:`custom-pod-specs` guide.

*****************************
 Create a kubeconfig for EKS
*****************************

After creating the cluster, ``kubectl`` should be used to deploy apps. In order for ``kubectl`` to
be used with EKS, users need to create or update the cluster kubeconfig. This can be done with the
command:

.. code:: bash

   aws eks --region <region-code> update-kubeconfig --name <cluster_name>

********************
 Enable GPU support
********************

To use GPU instances, the NVIDIA Kubernetes device plugin needs to be installed. Use the following
command to install the plugin:

.. code:: bash

   # Deploy a DaemonSet that enables the GPUs.
   kubectl apply -f https://raw.githubusercontent.com/NVIDIA/k8s-device-plugin/master/nvidia-device-plugin.yml

*******************
 Enable Autoscaler
*******************

Lastly, EKS requires manual deployment of an autoscaler. Save the following configuration in a new
file such as ``determined-autoscaler.yaml``:

You will need to update the ``<cluster-autoscaler-image>`` to match the major and minor numbers of
your Kubernetes version. For example, if you are using Kubernetes 1.20, use the cluster-autoscaler
version 1.20 image found here: k8s.gcr.io/autoscaling/cluster-autoscaler:v1.20.0

For a full list of cluster-autoscaler releases see here:
https://github.com/kubernetes/autoscaler/releases

After finding the particular release you want, click on the release and scroll to the bottom to see
a list of image URLs. Example:
https://github.com/kubernetes/autoscaler/releases/tag/cluster-autoscaler-1.20.0

.. code:: yaml

   apiVersion: apps/v1
   kind: Deployment
   metadata:
     name: cluster-autoscaler
     namespace: kube-system
     labels:
       app: cluster-autoscaler
   spec:
     replicas: 1
     selector:
       matchLabels:
         app: cluster-autoscaler
     template:
       metadata:
         labels:
           app: cluster-autoscaler
         annotations:
           prometheus.io/scrape: 'true'
           prometheus.io/port: '8085'
       spec:
         serviceAccountName: cluster-autoscaler
         tolerations:
           - key: node-role.kubernetes.io/master
             operator: "Equal"
             value: "true"
             effect: NoSchedule
         containers:
           - image: <cluster-autoscaler-image>  # See, https://github.com/kubernetes/autoscaler/releases
             name: cluster-autoscaler
             resources:
               limits:
                 cpu: 100m
                 memory: 300Mi
               requests:
                 cpu: 100m
                 memory: 300Mi
             command:
               - ./cluster-autoscaler
               - --v=4
               - --stderrthreshold=info
               - --cloud-provider=aws
               - --skip-nodes-with-local-storage=false
               - --expander=least-waste
               - --scale-down-delay-after-add=5m
               - --node-group-auto-discovery=asg:tag=k8s.io/cluster-autoscaler/enabled,k8s.io/cluster-autoscaler/<cluster-name>
             volumeMounts:
               - name: ssl-certs
                 mountPath: /etc/ssl/certs/ca-certificates.crt
                 readOnly: true
             imagePullPolicy: "Always"
         volumes:
           - name: ssl-certs
             hostPath:
               path: "/etc/ssl/certs/ca-bundle.crt"

To deploy an autoscaler that works with Determined, apply the official autoscaler `configuration
<https://github.com/kubernetes/autoscaler/blob/master/cluster-autoscaler/cloudprovider/aws/examples/cluster-autoscaler-run-on-control-plane.yaml>`_
first, then apply the custom ``determined-autoscaler.yaml``.

.. code:: bash

   # Apply the official autoscaler configuration
   kubectl apply -f https://raw.githubusercontent.com/kubernetes/autoscaler/master/cluster-autoscaler/cloudprovider/aws/examples/cluster-autoscaler-run-on-control-plane.yaml

   # Apply the custom deployment
   kubectl apply -f <cluster-autoscaler yaml, e.g. `determined-autoscaler.yaml`>

.. _changes-to-experiment-config:

*************************************
 Change the Experiment Configuration
*************************************

To run an experiment with EKS, two additions must be made to the experiment config. A service
account must be specified in order to allow Determined to save checkpoints to S3 and tolerances, if
there are tainted nodes, must be listed for the experiment to be scheduled. An example of the
necessary changes is shown here:

.. code:: yaml

   environment:
     pod_spec:
       ...
       spec:
         ...
         serviceAccountName: checkpoint-storage-s3-bucket
         # Tolerations should only be included if nodes are tainted
         tolerations:
           - key: <tainted-group-key, e.g g4dn.metal-us-west-2b>
             operator: "Equal"
             value: "true"
             effect: "NoSchedule"

Details about pod configuration can be found in :ref:`per-task-pod-specs`.

****************************
 Make Changes to Determined
****************************

Following the deployment of EKS, make sure that the necessary changes to Determined have been
applied in order to successfully run experiments. These changes include adding the created S3 bucket
to Determined's Helm chart and specifying a service account in the default pod specs. When modifying
the Helm chart to include S3, no keys or endpoint urls are needed. Additionally, if running on
tainted nodes, be sure to add pod tolerations to the experiment spec to ensure they will get
scheduled.

.. _aws-lb:

*************************************
 Use an AWS Load Balancer (optional)
*************************************

It is possible to use `ALB <https://kubernetes-sigs.github.io/aws-load-balancer-controller/v2.4/>`_
with the Determined EKS cluster instead of :ref:`nginx <tls-on-kubernetes>`. Determined expects the
health check to be on ``/det/``, so the `config
<https://kubernetes-sigs.github.io/aws-load-balancer-controller/v2.4/guide/ingress/annotations/#health-check>`_
of ``alb.ingress.kubernetes.io/healthcheck-path`` must be set to ``/det/`` in the master ingress
yaml. An example of a master ingress yaml is shown here:

.. code:: yaml

   apiVersion: extensions/v1beta1
   kind: Ingress
   metadata:
     annotations:
       alb.ingress.kubernetes.io/inbound-cidrs: 0.0.0.0/0
       alb.ingress.kubernetes.io/listen-ports: '[{"HTTP": 80}]'
       alb.ingress.kubernetes.io/scheme: internal
       alb.ingress.kubernetes.io/healthcheck-path: "/det/"
       kubernetes.io/ingress.class: alb
     name: determined-master-ingress
   spec:
     rules:
      - host: yourhost.com
        http:
         paths:
         - backend:
             serviceName: determined-master-service-determined
             servicePort: 8080
           path: /*
           pathType: ImplementationSpecific

In order for this ingress to work as expected the Helm parameter of ``useNodePortForMaster`` must be
set to ``true`` and the AWS Load Balancer Controller must be `installed in the cluster
<https://docs.aws.amazon.com/eks/latest/userguide/aws-load-balancer-controller.html>`_.

***********************
 Manage an EKS Cluster
***********************

For general instructions on adding taints and tolerations to nodes, see the :ref:`Taints and
Tolerations <taints-on-kubernetes>` section in our :ref:`Guide to Kubernetes
<install-on-kubernetes>`. There, you can find an explanation of taints and tolerations, as well as
instructions for using ``kubectl`` to add them to existing clusters.

It is important to note that if you use EKS to create nodes with taints, you must also add
tolerations using ``kubectl``; otherwise, Kubernetes will be unable to schedule pods on the tainted
node.

To taint nodes, users will need to add a taint type and a tag to the node group specified in the
cluster config from :ref:`cluster-creation`. An example of the modifications is shown for a
g4dn.metal node group:

.. code:: yaml

   - name: g4dn-metal-us-west-2b
     ...
     taints:
       g4dn.metal-us-west-2b: "true:NoSchedule"
     ...
     tags:
       ...
       k8s.io/cluster-autoscaler/node-template/taint/g4dn.metal-us-west-2b: "true:NoSchedule"

Furthermore, tainting requires changes to be made to the GPU enabling DaemonSet and more additions
to the experiment config. First, to change the DaemonSet, save a copy of the `official version
<https://raw.githubusercontent.com/NVIDIA/k8s-device-plugin/master/nvidia-device-plugin.yml>`_ and
make the following additions to its tolerations:

.. code:: yaml

   spec:
     tolerations:
     ...
     - key: g4dn.metal-us-west-2b
       operator: Exists
       effect: NoSchedule

To modify the experiment config to run on tainted nodes, refer to the
:ref:`changes-to-experiment-config` section.

************
 Next Steps
************

-  :ref:`install-on-kubernetes`
-  :ref:`k8s-dev-guide`
