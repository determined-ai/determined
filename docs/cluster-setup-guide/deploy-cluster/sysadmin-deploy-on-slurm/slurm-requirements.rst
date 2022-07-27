.. _slurm-requirements:

###########################
 Installation Requirements
###########################

******************
Basic Requirements
******************

Deploying Determined with Slurm has the following requirements.

-  The login node, admin node, and compute nodes must be configured with Ubuntu 20.04 or later,
   CentOS 7 or later, or SLES 15 or later.

-  Slurm 19.05 or greater.

-  Singularity 3.7 or greater.

-  A cluster-wide shared filesystem.

-  To run jobs with GPUs, the Nvidia drivers must be installed on each Determined agent. Determined
   requires a version greater than or equal to 450.80 of the Nvidia drivers. The Nvidia drivers can
   be installed as part of a CUDA installation but the rest of the CUDA toolkit is not required.

-  Determined supports the `active Python versions <https://endoflife.date/python>`__.

*********************
Launcher Requirements
*********************

The launcher has the following additional requirements on the installation node:

-  Support for an RPM or Debian-based package installer
-  Java 1.8 or greater
-  Access to the Slurm command line interface for the cluster
-  Access to a cluster-wide file system with a consistent path name across the cluster
