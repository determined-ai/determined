.. toctree::
   :caption: Get Started
   :hidden:

   quickstart-mdldev
   Tutorials <tutorials/index>
   Examples <example-solutions/examples>
   System Architecture <architecture/index>
   glossary

.. toctree::
   :caption: Administration Guide
   :hidden:

   Basic Setup <cluster-setup-guide/basic>
   Cluster Deployment <cluster-setup-guide/deploy-cluster/index>
   Security <cluster-setup-guide/security/overview>
   User Accounts <cluster-setup-guide/users>
   Logging and Elasticsearch <cluster-setup-guide/elasticsearch-logging-backend>
   Cluster Usage History <cluster-setup-guide/historical-cluster-usage-data>
   Upgrade <cluster-setup-guide/upgrade>
   Troubleshooting <cluster-setup-guide/troubleshooting>

.. toctree::
   :caption: Distributed Training Guide
   :hidden:

   Training <training/index>

   API Reference <reference-api/index>

   Configuration Reference <reference-config/index>

.. toctree::
   :caption: Interfaces
   :hidden:

   WebUI Interface <interfaces/webui-if>
   Commands and Shells <interfaces/commands-and-shells>
   IDE Integration <interfaces/ide-integration>
   Python API <interfaces/python-api>
   REST API <interfaces/rest-api>
   Jupyter Notebooks <interfaces/notebooks>
   TensorBoards <interfaces/tensorboard>

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

|

Determined is an open source deep learning training platform that makes building models fast and
easy.

With Determined you can:

-  Use state-of-the-art distributed training to train models faster without changing model code.
-  Automatically find high-quality models using advanced hyperparameter tuning.
-  Get more from your GPUs and reduce cloud GPU costs with preemptible instances and smart scheduling.
-  Leverage experiment tracking out-of-the-box to track and reproduce your work, tracking code
   versions, metrics, checkpoints, and hyperparameters.
-  Continue using popular deep learning libraries, such as TensorFlow, Keras, and PyTorch by simply
   integrating the Determined API with your existing model code.

Determined integrates these features into an easy-to-use, high-performance deep learning
environment so you can spend your time building models instead of managing infrastructure.

|

.. raw:: html

   <div class="landing">
      <div class="tiles-flex">
          <a class="tile" href="quickstart-mdldev.html">
              <h2 class="tile-title">Start here ...</h2>
              <p class="tile-description">Learn the basics steps needed to set up your Determined environment and train models.</p>
          </a>
          <a class="tile" href="introduction.html">
              <h2 class="tile-title">Introducing Determined</h2>
              <p class="tile-description">Learn about core concepts and key features to get helpful context before diving into more detailed information.</p>
          </a>
          <a class="tile" href="cluster-setup-guide/index.html">
              <h2 class="tile-title">Cluster Setup Guide</h2>
              <p class="tile-description">Set up an on-premise or cloud-based cluster, including AWS, GCP, and Azure.</p>
          </a>
          <a class="tile" href="training/index.html">
              <h2 class="tile-title">Training</h2>
              <p class="tile-description">Learn how to work with Training APIs, and how to configure your distributed training experiments.</p>
          </a>
          <a class="tile" href="reference-api/index.html">
              <h2 class="tile-title">API Reference</h2>
              <p class="tile-description">Explore the training and model hub API reference documentation.</p>
          </a>
          <a class="tile" href="reference-api/index.html">
              <h2 class="tile-title">Configuration Reference</h2>
              <p class="tile-description">Explore the Determined configuration file reference documentation.</p>
          </a>
          <a class="tile" href="tutorials/index.html">
              <h2 class="tile-title">Tutorials</h2>
              <p class="tile-description">Step-by-step tutorials and deep dives give you practical, real-world experience using Determined.</p>
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
