#!/bin/bash

set -e -x

# A command that can install both AUR and official packages; edit this if you use a different tool.
PKG_INSTALL_COMMAND=(yay -S --needed)

PKGS=(docker go npm python36)

if ! command -v "${PKG_INSTALL_COMMAND[0]}" >/dev/null; then
  echo "Problem finding the install command ${PKG_INSTALL_COMMAND[0]}!" >&2
  exit 1
fi

# Read user choices.
read -r -p "Do you have an NVIDIA GPU and want to enable Docker GPU support? [y/n] "
if [[ $REPLY =~ ^[Yy].*$ ]]; then
  PKGS+=(nvidia-container-toolkit)
fi

# Download things.
echo "Installing packages: ${PKGS[*]}..."
"${PKG_INSTALL_COMMAND[@]}" "${PKGS[@]}"

# Configure local things.
sudo usermod -aG docker "$USER"
sudo systemctl enable docker
sudo systemctl restart docker

set +x
echo -e '
\x1b[32;1mInstallation complete!\x1b[m

You may want to run the following commands:

- To ensure that Go-related tools installed by Determined can be found:

    export PATH="$(go env GOPATH)"/bin:"$PATH"

  This command should also be added to your shell initialization script.

- To set up a Python virtualenv for use with Determined:

    python3.6 -m venv ~/.virtualenvs/determined'
