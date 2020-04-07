#!/bin/bash

set -e

## Specify the script/command that can install packages from both AUR and official Arch repos.
PKG_INSTALL_COMMAND=(yay -S --needed)

PKGS=(docker go yarn python36 python-virtualenvwrapper)

if which "${PKG_INSTALL_COMMAND[0]}" > /dev/null; then
  echo "Installing packages: ${PKGS[@]}..."
else
  >&2 echo "Problem finding the install script!"
  exit 1
fi

"${PKG_INSTALL_COMMAND[@]}" "${PKGS[@]}"
sudo usermod -aG docker "$USER"

## Install Nvidia support for Docker.
read -p "Do you have an Nvidia GPU and want to install Nvidia GPU support (y/n)? " -n 1 -r
if [[ $REPLY =~ ^[Yy]$ ]]; then
  echo "Installing nvidia-docker and setting the default configuration."
  "${PKG_INSTALL_COMMAND[@]}" nvidia-docker

  ## Set Docker's default runtime to the Nvidia one.
  if [ -f /etc/docker/daemon.json ]; then
    sudo tee -a /etc/docker/daemon.json.backup > /dev/null < /etc/docker/daemon.json 
  fi
  sudo sh -c 'echo <<EOT > /etc/docker/daemon.json
  {
    "default-runtime": "nvidia",
    "runtimes": {
      "nvidia": {
        "path": "/usr/bin/nvidia-container-runtime",
        "runtimeArgs": []
      }
    }
  }
  EOT'
fi

sudo systemctl enable --now docker

## Set up Determined virtualenv.
mkdir -p ~/.virtualenvs
python3.6 -m venv ~/.virtualenvs/det
