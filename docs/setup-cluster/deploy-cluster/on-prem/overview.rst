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
services.

To install Determined on-premise, first :ref:`install Docker <install-docker>`. Then install
Determined by your preferred method.

.. toctree::
   :maxdepth: 1
   :hidden:

   requirements
   docker
   deploy
   linux-packages
   homebrew
   wsl
