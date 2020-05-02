#!/bin/bash

set -e -x

PKGS=(docker-ce golang krb5-devel nodejs nvidia-container-toolkit python36 python36-devel)

# Read user choices.
read -r -p "Do you have an NVIDIA GPU and want to enable Docker GPU support? [y/n] "
if [[ $REPLY =~ ^[Yy].*$ ]]; then
    PKGS+=(nvidia-container-toolkit)
fi

# Download things.
sudo yum install -y yum-utils device-mapper-persistent-data lvm2
sudo yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo

distribution=$(
  . /etc/os-release
  echo "$ID$VERSION_ID"
)
curl -fsSL https://nvidia.github.io/nvidia-docker/"$distribution"/nvidia-docker.repo |
  sudo tee /etc/yum.repos.d/nvidia-docker.repo

sudo rpm --import https://mirror.go-repo.io/centos/RPM-GPG-KEY-GO-REPO
curl -fsSL https://mirror.go-repo.io/centos/go-repo.repo | sudo tee /etc/yum.repos.d/go-repo.repo

sudo yum install -y "${PKGS[@]}"

curl -fsSL https://bootstrap.pypa.io/get-pip.py | sudo python3.6

sudo pip3 install virtualenvwrapper

# Configure local things.
sudo usermod -aG docker "$USER"
sudo systemctl enable docker
sudo systemctl restart docker

echo -e "\x1b[1mConsider adding the following line to your shell startup script:\x1b[m"
echo
echo "    . /usr/bin/virtualenvwrapper.sh"

set +x
echo -e '
\x1b[32;1mInstallation complete!\x1b[m

You may want to run the following commands:

- To ensure that Go-related tools installed by Determined can be found:

    export PATH="$(go env GOPATH)"/bin:"$PATH"

  This command should also be added to your shell initialization script.

- To set up a Python virtualenv for use with Determined:

    mkvirtualenv ~/.virtualenvs/determined'
