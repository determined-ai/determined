.. _deploy-on-prem-overview:

################
 Deploy on Prem
################

On-premise deployments are useful if you already have access to the machines that you would like to
install Determined on, whether that means a single laptop for local development or a fleet of
multi-GPU servers.

``det deploy`` is the most convenient on-premise option; once installed, it will allow you to start
a cluster by running a single command on each machine. If you would like more control over the
process, you can instead manually manage the Docker images that ``det deploy`` uses internally. If
you are using Ubuntu, you also have the option of installing most components of Determined using
Debian packages and running them as `systemd <https://freedesktop.org/wiki/Software/systemd/>`__
services. If you are using enterprise Linux, such as Red Hat Enterprise Linux or AlmaLinux, you also
have the option of installing Determined using RPM packages.

To install Determined on-premise, first :ref:`install Docker <install-docker>`. Then install
Determined by your :ref:`preferred method <deploy-on-prem-options-index>`.

.. include:: ../../_shared/tip-keep-install-instructions.txt

.. toctree::
   :maxdepth: 1
   :hidden:

   requirements
   options/index
