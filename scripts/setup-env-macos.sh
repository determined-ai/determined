#!/bin/bash
set -x

# Quick install script for mac

# Install homebrew
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/master/install.sh)"

# Brew install all packages
brew upgrade

brew install \
	go \
	libomp \
	node \
	git \
	pyenv \
	yarn

brew cask install \
	docker

# Install Python 3.6 and virtualenv
pyenv install 3.6.9
python3 -m pip install virtualenv

# Set up go mono repo
mkdir -p "$HOME/go"
mkdir -p "$HOME/go/bin"
mkdir -p "$HOME/go/pkg"
mkdir -p "$HOME/go/src"
