.. _install-gke:

####################
 Install Determined
####################

This user guide describes how to deploy a Determined cluster on Google Kubernetes Engine (GKE). The
``det deploy`` tool makes it easy to create and deploy these resources in GKE. cp` topic guide.

**************
 Requirements
**************

If you are installing GenAI Studio with your Determined cluster, additional requirements apply:

- **GKE Requirements:**
  - Before starting the installation, you’ll need a Google Cloud account and access to a service account with permissions needed to create instances and clusters. Identify a region with at least a100 GPUs available.
