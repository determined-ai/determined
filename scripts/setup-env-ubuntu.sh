#!/bin/bash

set -e -x

PKGS=(build-essential docker-ce golang-go libkrb5-dev nodejs python3-venv)

# Read user choices.
read -r -p "Do you have an NVIDIA GPU and want to enable Docker GPU support? [y/n] "
if [[ $REPLY =~ ^[Yy].*$ ]]; then
    PKGS+=(nvidia-container-toolkit)
fi

# Download things.
sudo apt-get install -y --no-install-recommends curl software-properties-common

curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -
sudo add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable"

curl -fsSL https://nvidia.github.io/nvidia-docker/gpgkey | sudo apt-key add -
curl -fsSL https://nvidia.github.io/nvidia-docker/ubuntu"$(lsb_release -rs)"/nvidia-docker.list | sudo tee /etc/apt/sources.list.d/nvidia-docker.list

curl -fsSL https://deb.nodesource.com/setup_12.x | sudo -E bash -

sudo add-apt-repository -y ppa:longsleep/golang-backports

sudo apt-get update && sudo apt-get install -y --no-install-recommends "${PKGS[@]}"

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

    python3 -m venv ~/.virtualenvs/determined'
