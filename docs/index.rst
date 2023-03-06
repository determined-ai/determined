.. toctree::
   :hidden:

   Welcome <self>

.. toctree::
   :caption: Getting Started
   :maxdepth: 2
   :hidden:

   Try Determined <tutorials/index>
   How It Works <architecture/index>
   quickstart-mdldev
   Examples <example-solutions/examples>
   Model Hub Library <model-hub-library/index>

.. toctree::
   :caption: Set Up Determined
   :hidden:

   Set Up Guide <cluster-setup-guide/deploy-cluster/index>
   Basic Setup <cluster-setup-guide/basic>
   Security <cluster-setup-guide/security/overview>
   User Accounts <cluster-setup-guide/users>
   Workspaces and Projects <cluster-setup-guide/workspaces>
   Logging and Elasticsearch <cluster-setup-guide/elasticsearch-logging-backend>
   Cluster Usage History <cluster-setup-guide/historical-cluster-usage-data>
   Monitor Experiment Through Webhooks  <integrations/notification/index>
   Upgrade <cluster-setup-guide/upgrade>
   Troubleshooting <cluster-setup-guide/troubleshooting>

.. toctree::
   :caption: Model Developer Guide
   :hidden:

   Overview <training/index>
   Distributed Training <training/dtrain-introduction>
   Prepare Container Environment <training/setup-guide/overview>
   Prepare Data <training/load-model-data>
   Training API Guides <training/apis-howto/overview>
   Hyperparameter Tuning <training/hyperparameter/overview>
   Submit Experiment <training/submit-experiment>
   How to Debug Models <training/debug-models>
   Model Management <training/model-management/overview>
   Best Practices <training/best-practices/overview>

.. toctree::
   :caption: Reference
   :hidden:

   Overview <reference/overview>
   Python SDK <reference/python-sdk>
   REST API <reference/rest-api>
   Training Reference <reference/reference-training/index>
   Model Hub Reference <reference/reference-model-hub/index>
   Deployment Reference <reference/reference-deploy/index>
   Job Configuration Reference <reference/reference-interface/job-config-reference>
   Custom Searcher Reference <reference/reference-searcher/custom-searcher-reference>

.. toctree::
   :caption: Tools
   :hidden:

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

.. raw:: html

    <table border-style=hidden>
   <tr>
     <th><a href="cluster-setup-guide/deploy-cluster/index.html"><h4>Set Up</h4></a></th>
     <th><h4>|</h4></th>
     <th><a href="reference/overview.html"><h4>Reference</h4></a></th>
   </tr>
   </table>

##########################
 *Welcome to Determined!*
##########################

You can quickly train almost any deep learning model using Determined.

.. raw:: html

   <div class="landing">
      <div class="tiles-flex">
         <div class="tile-container">
             <a class="tile" href="tutorials/index.html">
                 <h2 class="tile-title">Try It</h2>
                 <p class="tile-description">Learn the basics of working with Determined and how to port your existing code to the Determined environment.</p>
             </a>
         </div>
         <div class="tile-container">
             <a class="tile" href="architecture/index.html">
                 <h2 class="tile-title">How It Works</h2>
                 <p class="tile-description">Learn about core concepts, key features, and system architecture.</p>
             </a>
         </div>
         <div class="tile-container">
             <a class="tile" href="cluster-setup-guide/deploy-cluster/index.html">
                 <h2 class="tile-title">Set Up Determined</h2>
                 <p class="tile-description">Set up an on-premise or cloud-based cluster, including AWS, GCP, and Azure.</p>
             </a>
         </div>
         <div class="tile-container">
            <a class="tile" href="quickstart-mdldev.html">
                <h2 class="tile-title">Model Developer Quickstart</h2>
                <p class="tile-description">Learn the basic steps needed to set up a Determined environment and train models.</p>
            </a>
         </div>
         <div class="tile-container">
             <a class="tile" href="training/index.html">
                 <h2 class="tile-title">Model Developer Guide</h2>
                 <p class="tile-description">Learn how to work with Training APIs and configure your distributed training experiments.</p>
             </a>
         </div>
         <div class="tile-container">
             <a class="tile" href="reference/overview.html">
                 <h2 class="tile-title">Reference</h2>
                 <p class="tile-description">Explore API libraries and configuration settings.</p>
             </a>
         </div>
      </div>
   </div>

|

.. raw:: html

   <div>
      <p class="landing-text">
         New features, upgrades, deprecation notices, known issues, and bug fixes:
         <a href=release-notes.html>Release Notes</a>
      </p>
   </div>
