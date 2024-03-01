#!/bin/bash

pip install 'pydantic<2.0.0' # resolves 'pydantic.dataclasses.dataclass only supports init=False'
pip install -U "ray[air]==2.9.3"
pip install 'pandas==1.5.3'
pip show pandas
pip show pydantic
pip show ray
