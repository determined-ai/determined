#!/bin/sh

set -e -x

sudo apt-get install -y --no-install-recommends curl software-properties-common

curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -
sudo add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable"

curl -fsSL https://nvidia.github.io/nvidia-docker/gpgkey | sudo apt-key add -
curl -fsSL https://nvidia.github.io/nvidia-docker/ubuntu"$(lsb_release -rs)"/nvidia-docker.list | sudo tee /etc/apt/sources.list.d/nvidia-docker.list

curl -fsSL https://deb.nodesource.com/setup_11.x | sudo -E bash -

curl -fsSL https://dl.yarnpkg.com/debian/pubkey.gpg | sudo apt-key add -
echo "deb https://dl.yarnpkg.com/debian/ stable main" | sudo tee /etc/apt/sources.list.d/yarn.list

sudo add-apt-repository -y ppa:deadsnakes/ppa
sudo add-apt-repository -y ppa:longsleep/golang-backports

sudo apt-get update && sudo apt-get install -y --no-install-recommends \
    build-essential \
    docker-ce \
    golang-go \
    libkrb5-dev \
    nodejs \
    nvidia-docker2 \
    python3.6 \
    python3.6-dev \
    virtualenvwrapper \
    yarn
sudo systemctl reload docker

sudo usermod -aG docker $USER
echo "source /usr/share/virtualenvwrapper/virtualenvwrapper.sh" >> ~/.bashrc
