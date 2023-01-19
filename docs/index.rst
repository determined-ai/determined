.. toctree::
   :caption: Get Started
   :hidden:

   quickstart-mdldev
   Tutorials <tutorials/index>
   Examples <example-solutions/examples>
   Model Hub <model-hub-library/index>
   System Architecture <architecture/index>

.. toctree::
   :caption: Model Developer Guide
   :hidden:

   Introduction to Distributed Training <training/dtrain-introduction>
   Prepare Container Environment <training/setup-guide/overview>
   Prepare Data <training/load-model-data>
   Training API Guides <training/apis-howto/overview>
   Hyperparameter Tuning <training/hyperparameter/overview>
   Submit Experiment <training/submit-experiment>
   How to Debug Models <training/debug-models>
   Model Management <training/model-management/overview>
   Best Practices <training/best-practices/overview>

.. toctree::
   :caption: Administrator Guide
   :hidden:

   Basic Setup <cluster-setup-guide/basic>
   Cluster Deployment <cluster-setup-guide/deploy-cluster/index>
   Security <cluster-setup-guide/security/overview>
   User Accounts <cluster-setup-guide/users>
   Workspaces and Projects <cluster-setup-guide/workspaces>
   Logging and Elasticsearch <cluster-setup-guide/elasticsearch-logging-backend>
   Cluster Usage History <cluster-setup-guide/historical-cluster-usage-data>
   Monitor Experiment Through Webhooks  <integrations/notification/index>
   Upgrade <cluster-setup-guide/upgrade>
   Troubleshooting <cluster-setup-guide/troubleshooting>

.. toctree::
   :caption: Reference
   :hidden:

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

.. toctree::
   :caption: Integrations
   :hidden:

   Works with Determined <integrations/ecosystem/ecosystem-integration>
   IDE Integration <interfaces/ide-integration>
   Prometheus and Grafana <integrations/prometheus/prometheus>
   attributions

##########################
 Determined Documentation
##########################

**************************
 *Welcome to Determined!*
**************************

.. raw:: html

   <div>
      <p class="landing-text">
         New features, upgrades, deprecation notices, known issues, and bug fixes:
         <a href=release-notes.html>Release Notes</a>
      </p>
   </div>

|

Determined is an open source deep learning training platform that makes building models fast and
easy.

With Determined you can:

-  Use state-of-the-art distributed training to train models faster without changing model code.
-  Automatically find high-quality models using advanced hyperparameter tuning.
-  Get more from your GPUs and reduce cloud GPU costs with preemptible instances and smart
   scheduling.
-  Leverage experiment tracking out-of-the-box to track and reproduce your work, tracking code
   versions, metrics, checkpoints, and hyperparameters.
-  Continue using popular deep learning libraries, such as TensorFlow, Keras, and PyTorch by simply
   integrating the Determined API with your existing model code.

Determined integrates these features into an easy-to-use, high-performance deep learning environment
so you can spend your time building models instead of managing infrastructure.

|

.. raw:: html

   <div class="landing">
      <div class="tiles-flex">
         <div class="tile-container">
            <a class="tile" href="quickstart-mdldev.html">
                <h2 class="tile-title">Start here ...</h2>
                <p class="tile-description">Learn the basics steps needed to set up a Determined environment and train models.</p>
            </a>
         </div>
         <div class="tile-container">
             <a class="tile" href="introduction.html">
                 <h2 class="tile-title">Introducing Determined</h2>
                 <p class="tile-description">Learn about core concepts and key features before diving into more detailed information.</p>
             </a>
         </div>
         <div class="tile-container">
             <a class="tile" href="cluster-setup-guide/basic.html">
                 <h2 class="tile-title">Administrator Guide</h2>
                 <p class="tile-description">Set up an on-premise or cloud-based cluster, including AWS, GCP, and Azure.</p>
             </a>
         </div>
         <div class="tile-container">
             <a class="tile" href="training/dtrain-introduction.html">
                 <h2 class="tile-title">Model Developer Guide</h2>
                 <p class="tile-description">Learn how to work with Training APIs and configure your distributed training experiments.</p>
             </a>
         </div>
         <div class="tile-container">
             <a class="tile" href="tutorials/index.html">
                 <h2 class="tile-title">Tutorials</h2>
                 <p class="tile-description">Step-by-step tutorials and deep dives give you practical, real-world experience using Determined.</p>
             </a>
         </div>
         <div class="tile-container">
             <a class="tile" href="reference/python-sdk.html">
                 <h2 class="tile-title">Reference</h2>
                 <p class="tile-description">Explore API libraries and configuration settings.</p>
             </a>
         </div>
      </div>
   </div>
