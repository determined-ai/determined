.. _env-setup-config:

#############################################
 General Environment Setup and Configuration
#############################################

*********************
 Context Directories
*********************

By using the ``-c <directory>`` option, files are transferred from a directory on the local machine
(the "context directory") to the container. The contents of the context directory are placed into
the working directory within the container before the command or shell starts running, so files in
the context can be easily accessed using relative paths.

.. code::

   $ mkdir context
   $ echo 'print("hello world")' > context/run.py
   $ det cmd run -c context python run.py

The total size of the files in the context directory must be less than 95 MB. Larger files, such as
datasets, must be mounted into the container (see next section), downloaded after the container
starts, or included in a :ref:`custom Docker image <custom-docker-images>`.

*********************
Environment Variables
*********************

-  ``DET_MASTER``: The network address of the master of the Determined installation. The value can
   be overridden using the ``-m`` flag.

************************
 Advanced Configuration
************************

:ref:`Additional configuration settings <command-notebook-configuration>` for both commands and
shells can be set using the ``--config`` and ``--config-file`` options. Commonly useful settings
include:

-  ``bind_mounts``: Specifies directories to be bind-mounted into the container from the host
   machine. (Due to the structured values required for this setting, it needs to be specified in a
   config file.)

-  ``resources.slots``: Specifies the number of slots the container will have access to.
   (Distributed commands and shells are not supported; all slots will be on one machine and
   attempting to use more slots than are available on one machine will prevent the container from
   being scheduled.)

-  ``environment.image``: Specifies a custom Docker image to use for the container.

-  ``description``: Specifies a description for the command or shell to distinguish it from others.

*****************
 IDE integration
*****************

Determined shells can be used in the popular IDEs similarly to a common remote SSH host.

Visual Studio Code
==================

#. Make sure `Visual Studio Code Remote - SSH
   <https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-ssh>`__ extension is
   installed.

#. Start a new shell and get its SSH command by running:

   .. code::

      det shell start --show-ssh-command

   You can also get the SSH command for an existing shell using:

   .. code::

      det shell show_ssh_command <SHELL UUID>

#. Copy the SSH command, then select ``Remote-SSH: Add new SSH Host...`` from the **Command
   Palette** in VS Code, and paste the copied SSH command when prompted. Finally, you'll be asked to
   pick a config file to use. The default option should work for most users.

#. The remote host will now be available in the VS Code **Remote Explorer**. For further detail,
   please refer to the `official documentation <https://code.visualstudio.com/docs/remote/ssh>`__.

PyCharm
=======

#. **PyCharm Professional** is required for `remote Python interpreters via SSH
   <https://www.jetbrains.com/help/pycharm/configuring-remote-interpreters-via-ssh.html>`__.

#. Start a new shell and get its SSH command by running:

   .. code::

      det shell start --show-ssh-command

   You can also get the SSH command for an existing shell using:

   .. code::

      det shell show_ssh_command <SHELL UUID>

#. As of this writing, PyCharm doesn't support providing custom options in the SSH commands via the
   UI, so you'll need to supply them via an entry in your ``ssh_config`` file, commonly located at
   ``~/.ssh/config`` on Linux and macOS systems. Determined SSH command line will have the following
   pattern:

   .. code::

      ssh -o "ProxyCommand=<YOUR PROXY COMMAND>" -o StrictHostKeyChecking=no -tt -o IdentitiesOnly=yes -i <YOUR KEY PATH> -p <YOUR PORT NUMBER> <YOUR USERNAME>@<YOUR SHELL HOSTNAME>

   You'll need to add the following to your SSH config:

   .. code::

      Host <YOUR SHELL HOSTNAME>
      HostName <YOUR SHELL HOSTNAME>
      ProxyCommand <YOUR PROXY COMMAND>
      StrictHostKeyChecking no
      IdentitiesOnly yes
      IdentityFile <YOUR KEY PATH>
      Port <YOUR PORT NUMBER>
      User <YOUR USERNAME>

#. In PyCharm, open **Settings/Preferences** > **Tools** > **SSH Configurations**. Click the plus
   icon to add a new configuration. Enter ``YOUR HOST NAME``, ``YOUR PORT NUMBER``, and ``YOUR
   USERNAME`` in the corresponding fields. Then switch ``Authentication type`` dropdown to ``OpenSSH
   config and authentication agent``. Save the new configuration by clicking "OK".

#. Use the new SSH configuration to setup a remote interpreter by following the official documentation.
