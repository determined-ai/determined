#!/bin/bash

pip uninstall -y pynvml
pip install -U "ray[air]" nvidia-ml-py
