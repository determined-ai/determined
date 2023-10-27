#!/bin/bash

# Exit on the first error
set -e

# Update the system package list and upgrade installed packages
sudo apt-get update -y

# Install necessary dependencies
sudo apt-get install -y binutils git

# Clone the efs-utils repository
git clone https://github.com/aws/efs-utils

# Change directory to efs-utils
cd efs-utils

# Build the Debian package
./build-deb.sh

# Change the ownership of the .deb file to _apt
sudo chown _apt:root ./build/amazon-efs-utils*deb

# Install the generated Debian package
sudo apt-get install -y ./build/amazon-efs-utils*deb

# Change back to the previous directory
cd ..

# Remove the efs-utils directory
rm -r efs-utils
