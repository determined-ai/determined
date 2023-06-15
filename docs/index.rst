.. toctree::
   :hidden:

   Welcome <self>

.. toctree::
   :caption: Get Started
   :maxdepth: 2
   :hidden:

   How It Works <architecture/index>
   Tutorials <tutorials/index>
   Quickstart for Model Developers <tutorials/quickstart-mdldev>
   Examples <example-solutions/examples>
   Model Hub Library <model-hub-library/index>

.. toctree::
   :caption: Set Up
   :hidden:

   Basic Setup <setup-cluster/basic>
   Setup Guides <setup-cluster/deploy-cluster/index>
   Security <setup-cluster/security/overview>
   User Accounts <setup-cluster/users>
   Workspaces and Projects <setup-cluster/workspaces>
   Logging and Elasticsearch <setup-cluster/elasticsearch-logging-backend>
   Cluster Usage History <setup-cluster/historical-cluster-usage-data>
   Monitor Experiment Through Webhooks  <integrations/notification/index>
   Upgrade <setup-cluster/upgrade>
   Troubleshooting <setup-cluster/troubleshooting>

.. toctree::
   :caption: Model Developer Guide
   :hidden:

   Overview <model-dev-guide/index>
   Distributed Training <model-dev-guide/dtrain/index>
   Prepare Container Environment <model-dev-guide/prepare-container/overview>
   Prepare Data <model-dev-guide/load-model-data>
   Training API Guides <model-dev-guide/apis-howto/overview>
   Hyperparameter Tuning <model-dev-guide/hyperparameter/overview>
   Submit Experiment <model-dev-guide/submit-experiment>
   How to Debug Models <model-dev-guide/debug-models>
   Model Management <model-dev-guide/model-management/overview>
   Best Practices <model-dev-guide/best-practices/overview>

.. toctree::
   :caption: Reference
   :hidden:

   Overview <reference/overview>
   Python SDK <reference/python-sdk>
   REST API <reference/rest-api>
   Training Reference <reference/training/index>
   Experiment Configuration Reference <reference/training/experiment-config-reference>
   Model Hub Reference <reference/model-hub/index>
   Deployment Reference <reference/deploy/index>
   Job Configuration Reference <reference/interface/job-config-reference>
   Custom Searcher Reference <reference/searcher/custom-searcher-reference>
   CLI Reference <reference/cli-reference>

.. toctree::
   :caption: Tools
   :hidden:

   Overview <interfaces/index>
   CLI User Guide <interfaces/cli-ug>
   Commands and Shells <interfaces/commands-and-shells>
   WebUI Interface <interfaces/webui-if>
   Jupyter Notebooks <interfaces/notebooks>
   TensorBoards <interfaces/tensorboard>
   Exposing Custom Ports <interfaces/proxy-ports>

.. toctree::
   :caption: Integrations
   :hidden:

   Works with Determined <integrations/ecosystem/ecosystem-integration>
   IDE Integration <interfaces/ide-integration>
   Prometheus and Grafana <integrations/prometheus/prometheus>
   attributions

##########################
 *Welcome to Determined!*
##########################

.. meta::
   :description: Visit the Determined AI documentation home page and get quick access to information, tutorials, quickstarts, user guides, and reference material.

You can quickly train almost any deep learning model using Determined.

.. raw:: html

   <div class="landing">
      <div class="tiles-flex">
         <div class="tile-container">
            <a class="tile" href="architecture/index.html">
               <img src="_static/images/tools.png" width="24" height="24" alt="tools icon">
               <h2 class="tile-title">How It Works</h2>
               <p class="tile-description">Learn about core concepts, key features, and system architecture.</p>
            </a>
         </div>
         <div class="tile-container">
            <a class="tile" href="tutorials/index.html">
               <img src="_static/images/getting-started.png" width="24" height="24" alt="getting started icon">
               <h2 class="tile-title">Tutorials</h2>
               <p class="tile-description">Try Determined and learn the basics including how to port your existing code to the Determined environment.</p>
            </a>
         </div>
         <div class="tile-container">
            <a class="tile" href="setup-cluster/deploy-cluster/index.html">
               <img src="_static/images/integrations.png" width="24" height="24" alt="integrations icon">
               <h2 class="tile-title">Set Up Determined</h2>
               <p class="tile-description">Set up an on-premise or cloud-based cluster, including AWS, GCP, and Azure.</p>
            </a>
         </div>
         <div class="tile-container">
            <a class="tile" href="tutorials/quickstart-mdldev.html">
               <img src="_static/images/setup.png" width="24" height="24" alt="setup icon">
               <h2 class="tile-title">Model Developer Quickstart</h2>
               <p class="tile-description">Learn the basic steps needed to set up a Determined environment and train models.</p>
            </a>
         </div>
         <div class="tile-container">
            <a class="tile" href="training/index.html">
               <img src="_static/images/developer-guide.png" width="24" height="24" alt="developer guide icon">
               <h2 class="tile-title">Model Developer Guide</h2>
               <p class="tile-description">Find user guides. Learn how to work with Training APIs and configure your distributed training experiments.</p>
            </a>
         </div>
         <div class="tile-container">
            <a class="tile" href="reference/overview.html">
               <img src="_static/images/reference.png" width="24" height="24" alt="reference icon">
               <h2 class="tile-title">Reference</h2>
               <p class="tile-description">Explore API libraries and configuration settings.</p>
            </a>
         </div>
      </div>
   </div>
