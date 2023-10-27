#!/bin/bash

set -e
sudo apt-get update -y
sudo apt-get install -y binutils git
git clone https://github.com/aws/efs-utils
pushd efs-utils
./build-deb.sh
sudo chown _apt:root ./build/amazon-efs-utils*deb
sudo apt-get install -y ./build/amazon-efs-utils*deb
popd ..
rm -r efs-utils
