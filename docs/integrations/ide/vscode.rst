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

#. Copy the SSH command, then select ``Remote-SSH: Add new SSH Host...`` from the **Command
   Palette** in VS Code, and paste the copied SSH command when prompted. Finally, you'll be asked to
   pick a config file to use. The default option should work for most users.

#. The remote host will now be available in the VS Code **Remote Explorer**. For further detail,
   please refer to the `official documentation <https://code.visualstudio.com/docs/remote/ssh>`__.

############################
 Addendum for Windows Users
############################

#. On Windows, Visual Studio Code uses CMD to run its SSH command under the hood. Therefore, we need
   to make a few modifications to the config file. First, prepend the proxy command to use WSL. In a
   CMD shell, type `where wsl` and prepend the result to the ProxyCommand entry in your config file.
   This should lead to a ProxyCommand that looks like this:

   .. code::

      ProxyCommand C:\Windows\System32\wsl.exe /path/to/python -m determined.cli.tunnel <DET_MASTER_URL_HERE> %h

#. An additional quirk of using CMD for the SSH step is that the key file needs to be moved out of
   the WSL filesystem. Otherwise, there will be permission ambiguity that Windows OpenSSH won't
   like. To work around this, move or copy the file to the Windows filesystem. This key file can be
   found in WSL in the directory /home/<username>/.cache/determined/shell/<your_shell_id>/key.

   .. code::

      cp /home/<username>/.cache/determined/shell/<your_shell_id>/key /mnt/c/path/to/your/key

   Then follow this up by changing the IdentityFile field in your config with the new key path:

   .. code::

      IdentityFile C:\path\to\your\key

#. Verify that the new config has properly updated parameters.

   .. code::

      Host <YOUR SHELL HOSTNAME>
      HostName <YOUR SHELL HOSTNAME>
      ProxyCommand C:\Windows\System32\wsl.exe /path/to/python -m determined.cli.tunnel <DET_MASTER_URL_HERE> %h
      StrictHostKeyChecking no
      IdentitiesOnly yes
      IdentityFile C:\path\to\your\key
      User root
