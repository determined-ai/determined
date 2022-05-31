.. toctree::
   :hidden:

   quickstart-mdldev
   quickstart-cluster

.. toctree::
   :caption: Introduction to Determined
   :hidden:

   Features <intro/features/list>
   Concepts <intro/concepts/overview>
   System Architecture <intro/architecture/overview>

.. toctree::
   :caption: Determined Interfaces
   :hidden:

   WebUI Interface <interfaces/webui-if>
   Commands and Shells <interfaces/commands-and-shells>
   Python API <interfaces/python-api>
   REST API <interfaces/rest-api>
   Jupyter Notebooks <interfaces/notebooks>
   TensorBoards <interfaces/tensorboard>
   Interactive Job Configuration <training/interactive-job-config>

.. toctree::
   :caption: Cluster Setup Guide
   :hidden:

   Getting Started <cluster-setup-guide/getting-started>
   Deploy on Prem <cluster-setup-guide/deploy-cluster/sysadmin-deploy-on-prem/overview>
   Deploy on AWS <cluster-setup-guide/deploy-cluster/sysadmin-deploy-on-aws/overview>
   Deploy on GCP <cluster-setup-guide/deploy-cluster/sysadmin-deploy-on-gcp/overview>
   Deploy on Kubernetes <cluster-setup-guide/deploy-cluster/sysadmin-deploy-on-k8s/overview>
   Security <cluster-setup-guide/security/overview>
   User Accounts <cluster-setup-guide/users>
   Logging and Elasticsearch <cluster-setup-guide/elasticsearch-logging-backend>
   Cluster Usage History <cluster-setup-guide/historical-cluster-usage-data>
   IDE Integration <cluster-setup-guide/ide-integration>
   Upgrade <cluster-setup-guide/upgrade>
   Troubleshooting <cluster-setup-guide/troubleshooting>

.. toctree::
   :caption: Training
   :hidden:

   Introduction to Distributed Training <training/dtrain-intro>
   Prepare Environment <training/setup-guide/overview>
   Basic Workflow <training/basic-workflow>
   Training API Guides <training/apis-howto/overview>
   Hyperparameter Tuning <training/hyperparameter/overview>
   Model Management <training/model-management/overview>
   Best Practices <training/best-practices/overview>

.. toctree::
   :caption: Reference
   :hidden:

   Training APIs <reference/training/overview>
   Configuration Files <reference/config/overview>
   Helm Chart Reference <reference/config/helm-config-reference>
   Model Hub APIs <reference/modelhub/overview>
   Python API <reference/determined/python-api-reference>
   REST API <reference/determined/rest-api-reference>
   Command Line Interface (CLI) <reference/determined/cli>

.. toctree::
   :caption: Model Hub Library
   :hidden:

   Huggingface Trainsformers <model-hub-library/transformers/overview>
   MMDetection <model-hub-library/mmdetection/overview>

.. toctree::
   :caption: Integrations
   :hidden:

   Works with Determined <integrations/ecosystem/ecosystem-integration>
   Prometheus and Grafana <integrations/prometheus/prometheus>

.. toctree::
   :caption: Tutorials
   :hidden:

   PyTorch MNIST Tutorial <tutorials/pytorch-mnist-tutorial>
   PyTorch Porting Tutorial <tutorials/pytorch-porting-tutorial>
   TensorFlow Keras Fashion MNIST Tutorial <tutorials/tf-mnist-tutorial>

.. toctree::
   :caption: Example Solutions
   :hidden:

   Examples <example-solutions/examples>

.. toctree::
   :hidden:

   attributions

#############################
 Determined AI Documentation
#############################

*****************************
 *Welcome to Determined AI!*
*****************************

.. raw:: html

   <div>
      <p class="landing-text">
         New features, upgrades, deprecation notices, known issues, and bug fixes:
         <a href=release-notes.html>Release Notes</a>
      </p>
   </div>

   <div class="landing">
      <div class="tiles-flex">
          <a class="tile" href="quickstart-mdldev.html">
              <h2 class="tile-title">Start here ...</h2>
              <p class="tile-description">Learn the basics steps needed to set up your Determined environment and train models.</p>
          </a>
          <a class="tile" href="intro/features/list.html">
              <h2 class="tile-title">Introducing Determined</h2>
              <p class="tile-description">Understand core concepts, key features, and the Determined architecture to get helpful context before diving into more detailed information.</p>
          </a>
          <a class="tile" href="cluster-setup-guide/getting-started.html">
              <h2 class="tile-title">Cluster Setup Guide</h2>
              <p class="tile-description">Get started with setting up an on-premise or cloud-based cluster, including AWS, GCP, and Azure.</p>
          </a>
          <a class="tile" href="training/dtrain-intro.html">
              <h2 class="tile-title">Training</h2>
              <p class="tile-description">Learn how to use Determined features and APIs in your workflow, and how to configure your distributed training experiments.</p>
          </a>
          <a class="tile" href="tutorials/pytorch-mnist-tutorial.html">
              <h2 class="tile-title">Tutorials</h2>
              <p class="tile-description">Step-by-step tutorials and deep dives give you practical, real-world experience using Determined.</p>
          </a>
          <a class="tile" href="reference/training/overview.html">
              <h2 class="tile-title">Reference</h2>
              <p class="tile-description">Explore the Determined and integrated training APIs along with configuration and CLI reference documentation.</p>
          </a>
          <a class="tile" href="example-solutions/examples.html">
              <h2 class="tile-title">Example Solutions</h2>
              <p class="tile-description">Explore example machine learning models that have been ported to the Determined APIs.</p>
          </a>
          <a class="tile" href="interfaces/python-api.html">
              <h2 class="tile-title">Python API</h2>
              <p class="tile-description">Use the Python API to interface with Determined to get many of the same capabilities available through the CLI.</p>
          </a>
      </div>
   </div>
