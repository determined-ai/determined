.. _pycharm:

#########
 PyCharm
#########

To use `remote Python interpreters via SSH
<https://www.jetbrains.com/help/pycharm/configuring-remote-interpreters-via-ssh.html>`__ in PyCharm,
**PyCharm Professional** is required.

**********************
 Starting a New Shell
**********************

Start a new shell and obtain its SSH command by running the following:

.. code::

   det shell start --show-ssh-command

For existing shells, use the following:

.. code::

   det shell show_ssh_command <SHELL UUID>

**************************
 Customizing SSH Commands
**************************

As of the current version, PyCharm lacks support for custom options in SSH commands via the UI.

Therefore, you must provide via an entry in your ``ssh_config`` file, typically located at
``~/.ssh/config`` on Linux systems. The Determined SSH command line follow this pattern:

.. code::

   ssh -o "ProxyCommand=<YOUR PROXY COMMAND>" -o StrictHostKeyChecking=no -tt -o IdentitiesOnly=yes -i <YOUR KEY PATH> -p <YOUR PORT NUMBER> <YOUR USERNAME>@<YOUR SHELL HOSTNAME>

Ensure the following configurations are added to your SSH config:

.. code::

   Host <YOUR SHELL HOSTNAME>
   HostName <YOUR SHELL HOSTNAME>
   ProxyCommand <YOUR PROXY COMMAND>
   StrictHostKeyChecking no
   IdentitiesOnly yes
   IdentityFile <YOUR KEY PATH>
   Port <YOUR PORT NUMBER>
   User <YOUR USERNAME>

*****************************************
 Setting Up SSH Configuration in PyCharm
*****************************************

#. In PyCharm, open **Settings/Preferences** > **Tools** > **SSH Configurations**.
#. Select the plus icon to add a new configuration.
#. Enter ``YOUR HOST NAME``, ``YOUR PORT NUMBER``, and ``YOUR USERNAME`` in the corresponding
   fields.
#. Switch the ``Authentication type`` dropdown to ``OpenSSH config and authentication agent``.
#. Save the new configuration by clicking **OK**.

Refer to the official PyCharm documentation for setting up a remote interpreter using the new SSH
configuration.
