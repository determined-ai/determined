#################
 Deploy on Slurm
#################

+----------------------+
| Supported Versions   |
+======================+
| Determined >= 0.18.3 |
+----------------------+
| Slurm >= 19.05       |
+----------------------+
| Singularity >= 3.7   |
+----------------------+
| Launcher >= 3.0.14   |
+----------------------+
| Java >= 1.8          |
+----------------------+

.. note::

   Slurm deployment applies to the Enterprise Edition.

Determined Slurm integration delegates all job scheduling and prioritization to the Slurm workload
manager. This integration enables existing Slurm workloads and Determined workloads to coexist and
Determined workloads to access all of the advanced capabilities of the Slurm workload manager.

This section describes how to install Determined on a Slurm cluster.

*********
Reference
*********

-  :ref:`Determined Installation Requirements <system-requirements>`
-  `Slurm <https://slurm.schedmd.com/documentation.html>`__
-  `Singularity <https://docs.sylabs.io/guides/3.7/user-guide/introduction.html>`__

.. toctree::
   :maxdepth: 1
   :hidden:

   slurm-requirements
   install-on-slurm
   singularity
   slurm-known-issues
