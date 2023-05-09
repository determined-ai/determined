#!/usr/bin/env bash
set -xeuo pipefail

if [ -z "$(command -v helm)" ]; then
    curl -L https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
fi
if [ -z "$(command -v protoc)" ]; then
    curl --retry-connrefused --retry 10 -o /tmp/protoc.zip -L https://github.com/protocolbuffers/protobuf/releases/download/v3.20.3/protoc-3.20.3-linux-x86_64.zip
    sudo unzip -o /tmp/protoc.zip -d /usr/local
fi
if [ -z "$(command -v nodemon)" ]; then
    npm i -g nodemon
fi

make get-deps

make -C proto build
make -C harness build
make -C webui build
make -C docs build
make -C agent build

make -C tools prep-root
sudo mkdir -p /usr/share/determined
sudo ln -sfT ${PWD}/tools/build /usr/share/determined/master

sudo chmod a+w /var/cache
