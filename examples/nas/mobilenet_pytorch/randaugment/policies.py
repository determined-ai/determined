# Source: https://raw.githubusercontent.com/google-research/uda/master/image/randaugment/policies.py
# coding=utf-8
# Copyright 2019 The Google UDA Team Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Augmentation policies found by AutoAugment."""


def imagenet_policies():
    """AutoAugment policies found on ImageNet.

    This policy also transfers to five FGVC datasets with image size similar to
    ImageNet including Oxford 102 Flowers, Caltech-101, Oxford-IIIT Pets,
    FGVC Aircraft and Stanford Cars.
    """
    policies = [
        [("Posterize", 0.4, 8), ("Rotate", 0.6, 9)],
        [("Solarize", 0.6, 5), ("AutoContrast", 0.6, 5)],
        [("Equalize", 0.8, 8), ("Equalize", 0.6, 3)],
        [("Posterize", 0.6, 7), ("Posterize", 0.6, 6)],
        [("Equalize", 0.4, 7), ("Solarize", 0.2, 4)],
        [("Equalize", 0.4, 4), ("Rotate", 0.8, 8)],
        [("Solarize", 0.6, 3), ("Equalize", 0.6, 7)],
        [("Posterize", 0.8, 5), ("Equalize", 1.0, 2)],
        [("Rotate", 0.2, 3), ("Solarize", 0.6, 8)],
        [("Equalize", 0.6, 8), ("Posterize", 0.4, 6)],
        [("Rotate", 0.8, 8), ("Color", 0.4, 0)],
        [("Rotate", 0.4, 9), ("Equalize", 0.6, 2)],
        [("Equalize", 0.0, 7), ("Equalize", 0.8, 8)],
        [("Invert", 0.6, 4), ("Equalize", 1.0, 8)],
        [("Color", 0.6, 4), ("Contrast", 1.0, 8)],
        [("Rotate", 0.8, 8), ("Color", 1.0, 2)],
        [("Color", 0.8, 8), ("Solarize", 0.8, 7)],
        [("Sharpness", 0.4, 7), ("Invert", 0.6, 8)],
        [("ShearX", 0.6, 5), ("Equalize", 1.0, 9)],
        [("Color", 0.4, 0), ("Equalize", 0.6, 3)],
    ]
    return policies


def get_trans_list():
    trans_list = [
        "Invert",
        "Sharpness",
        "AutoContrast",
        "Posterize",
        "ShearX",
        "TranslateX",
        "TranslateY",
        "ShearY",
        "Cutout",
        "Rotate",
        "Equalize",
        "Contrast",
        "Color",
        "Solarize",
        "Brightness",
    ]
    return trans_list


def randaug_policies():
    trans_list = get_trans_list()
    op_list = []
    for trans in trans_list:
        for magnitude in range(1, 10):
            op_list += [(trans, 0.5, magnitude)]
    policies = []
    for op_1 in op_list:
        for op_2 in op_list:
            policies += [[op_1, op_2]]
    return policies
