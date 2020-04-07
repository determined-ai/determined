#!/bin/sh

set -e -x

sudo yum install -y yum-utils device-mapper-persistent-data lvm2
sudo yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo

distribution=$(. /etc/os-release;echo $ID$VERSION_ID)
curl -fsSL https://nvidia.github.io/nvidia-docker/$distribution/nvidia-docker.repo | \
    sudo tee /etc/yum.repos.d/nvidia-docker.repo

curl -fsSL https://dl.yarnpkg.com/rpm/yarn.repo | sudo tee /etc/yum.repos.d/yarn.repo

sudo rpm --import https://mirror.go-repo.io/centos/RPM-GPG-KEY-GO-REPO
curl -fsSL https://mirror.go-repo.io/centos/go-repo.repo | sudo tee /etc/yum.repos.d/go-repo.repo

sudo yum install -y \
  docker-ce \
  golang \
  krb5-devel \
  nodejs \
  nvidia-docker2 \
  python36 \
  python36-devel \
  yarn
sudo systemctl start docker

curl -fsSL https://bootstrap.pypa.io/get-pip.py | sudo python3.6

sudo pip3 install virtualenvwrapper

sudo usermod -aG docker $USER
echo "source /usr/bin/virtualenvwrapper.sh" >> ~/.bashrc
