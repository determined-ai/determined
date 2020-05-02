#!/bin/bash
set -x

# Install Homebrew.
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/master/install.sh)"

# Brew install all packages.
brew upgrade

brew install \
  go \
  node \
  git \
  pyenv \
  yarn

brew cask install \
  docker

pyenv install 3.6.10

# Set up Determined virtualenv.

set +x
echo -e '
\x1b[32;1mInstallation complete!\x1b[m

You may want to run the following commands:

- To ensure that Go-related tools installed by Determined can be found:

    export PATH="$(go env GOPATH)"/bin:"$PATH"

  This command should also be added to your shell initialization script.

- To set up a Python virtualenv for use with Determined:

    "$(pyenv root)"/versions/3.6.10/bin/python -m venv ~/.virtualenvs/determined'
