### Steps for using a devcontainer (without Kubernetes)

0. Launch a cluster with the `det deploy` command.
1. Ensure that VSCode dev extensions are installed.
2. SSH to server instance and setup VSCode with remote SSH server.
3. Resize disk space on remote server if needed.
4. Setup github credentials and clone repository.
5. Stop the deployed server instance.
6. Open workspace file with VSCode connected to remote SSH server.
7. Use VSCode to build and launch devcontainer.
8. After remote server launches localhost:8080 should be available.
