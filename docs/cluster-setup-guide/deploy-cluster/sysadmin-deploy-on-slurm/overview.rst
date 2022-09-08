#################
 Deploy on Slurm
#################

+----------------------+
| Supported Versions   |
+======================+
| Slurm >= 19.05       |
+----------------------+
| Singularity >= 3.7   |
| or PodMan >= 3.3.1   |
+----------------------+
| Launcher             |
| (`hpe-hpc-launcher`) |
| >= 3.0.19            |
+----------------------+
| Java >= 1.8          |
+----------------------+

.. note::

   Slurm deployment applies to the Enterprise Edition.

Determined Slurm integration delegates all job scheduling and prioritization to the Slurm workload
manager. This integration enables existing Slurm workloads and Determined workloads to coexist and
Determined workloads to access all of the advanced capabilities of the Slurm workload manager.

To install Determined on a Slurm cluster, ensure that the
:doc:`/cluster-setup-guide/deploy-cluster/sysadmin-deploy-on-slurm/slurm-requirements` are met, then
follow the steps in the
:doc:`/cluster-setup-guide/deploy-cluster/sysadmin-deploy-on-slurm/install-on-slurm` document.

***********
 Reference
***********

-  :ref:`Determined Installation Requirements <system-requirements>`
-  `Slurm <https://slurm.schedmd.com/documentation.html>`__
-  `Singularity <https://docs.sylabs.io/guides/3.7/user-guide/introduction.html>`__
-  `PodMan <https://docs.podman.io>`__

.. toctree::
   :maxdepth: 1
   :hidden:

   slurm-requirements
   install-on-slurm
   singularity
   slurm-known-issues
