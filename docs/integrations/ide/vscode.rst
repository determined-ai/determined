.. _vscode:

####################
 Visual Studio Code
####################

#. Make sure `Visual Studio Code Remote - SSH
   <https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-ssh>`__ extension is
   installed.

#. Start a new shell and get its SSH command by running:

   .. code::

      det shell start --show-ssh-command

   You can also get the SSH command for an existing shell using:

   .. code::

      det shell show_ssh_command <SHELL UUID>

   If you are using Windows with WSL, do not to run these commands in your WSL shell. Use a Windows
   installation of `determined`.

#. Copy the SSH command, then select ``Remote-SSH: Add new SSH Host...`` from the **Command
   Palette** in VS Code, and paste the copied SSH command when prompted. Finally, you'll be asked to
   pick a config file to use. The default option should work for most users.

#. The remote host will now be available in the VS Code **Remote Explorer**. For further detail,
   please refer to the `official documentation <https://code.visualstudio.com/docs/remote/ssh>`__.
