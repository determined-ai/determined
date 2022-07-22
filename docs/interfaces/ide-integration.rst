#################
 IDE Integration
#################

Determined shells can be used in the popular IDEs similarly to a common remote SSH host.

********************
 Visual Studio Code
********************

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

*********
 PyCharm
*********

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

#. Use the new SSH configuration to setup a remote interpreter by following the official
   documentation.
