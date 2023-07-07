#!/bin/bash

pip install 'pydantic<2.0.0' # resolves 'pydantic.dataclasses.dataclass only supports init=False'
pip install -U "ray[air]"
