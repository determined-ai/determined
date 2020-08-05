# source: https://raw.githubusercontent.com/google-research/uda/master/image/randaugment/augmentation_transforms.py
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
"""Transforms used in the Augmentation Policies.

Copied from AutoAugment: https://github.com/tensorflow/models/blob/master/research/autoaugment/
"""

import random
import numpy as np

# pylint:disable=g-multiple-import
from PIL import ImageOps, ImageEnhance, ImageFilter, Image

# pylint:enable=g-multiple-import


PARAMETER_MAX = 10  # What is the max 'level' a transform could be predicted


def _width_height_from_img_shape(img_shape):
    """`img_shape` in autoaugment is (height, width)."""
    return (img_shape[1], img_shape[0])


def create_cutout_mask(img_height, img_width, num_channels, size):
    """Creates a zero mask used for cutout of shape `img_height` x `img_width`.

  Args:
    img_height: Height of image cutout mask will be applied to.
    img_width: Width of image cutout mask will be applied to.
    num_channels: Number of channels in the image.
    size: Size of the zeros mask.

  Returns:
    A mask of shape `img_height` x `img_width` with all ones except for a
    square of zeros of shape `size` x `size`. This mask is meant to be
    elementwise multiplied with the original image. Additionally returns
    the `upper_coord` and `lower_coord` which specify where the cutout mask
    will be applied.
  """
    # Sample center where cutout mask will be applied
    height_loc = np.random.randint(low=0, high=img_height)
    width_loc = np.random.randint(low=0, high=img_width)

    # Determine upper right and lower left corners of patch
    upper_coord = (max(0, height_loc - size // 2), max(0, width_loc - size // 2))
    lower_coord = (
        min(img_height, height_loc + size // 2),
        min(img_width, width_loc + size // 2),
    )
    mask_height = lower_coord[0] - upper_coord[0]
    mask_width = lower_coord[1] - upper_coord[1]
    assert mask_height > 0
    assert mask_width > 0

    mask = np.ones((img_height, img_width, num_channels))
    zeros = np.zeros((mask_height, mask_width, num_channels))
    mask[upper_coord[0] : lower_coord[0], upper_coord[1] : lower_coord[1], :] = zeros
    return mask, upper_coord, lower_coord


def float_parameter(level, maxval):
    """Helper function to scale `val` between 0 and maxval .

  Args:
    level: Level of the operation that will be between [0, `PARAMETER_MAX`].
    maxval: Maximum value that the operation can have. This will be scaled
      to level/PARAMETER_MAX.

  Returns:
    A float that results from scaling `maxval` according to `level`.
  """
    return float(level) * maxval / PARAMETER_MAX


def int_parameter(level, maxval):
    """Helper function to scale `val` between 0 and maxval .

  Args:
    level: Level of the operation that will be between [0, `PARAMETER_MAX`].
    maxval: Maximum value that the operation can have. This will be scaled
      to level/PARAMETER_MAX.

  Returns:
    An int that results from scaling `maxval` according to `level`.
  """
    return int(level * maxval / PARAMETER_MAX)


def apply_policy(policy, img):
    """Apply the `policy` to the numpy `img`.

    Args:
      policy: A list of tuples with the form (name, probability, level) where
        `name` is the name of the augmentation operation to apply, `probability`
        is the probability of applying the operation and `level` is what strength
        the operation to apply.
      img: PIL image that will have `policy` applied to it.

    Returns:
      The result of applying `policy` to `img`.
    """
    width, height = img.size
    img_shape = [height, width]
    pil_img = img

    for xform in policy:
        assert len(xform) == 3
        name, probability, level = xform
        xform_fn = NAME_TO_TRANSFORM[name].pil_transformer(
            probability, level, img_shape
        )
        pil_img = xform_fn(pil_img)
    return pil_img


class TransformFunction(object):
    """Wraps the Transform function for pretty printing options."""

    def __init__(self, func, name):
        self.f = func
        self.name = name

    def __repr__(self):
        return "<" + self.name + ">"

    def __call__(self, pil_img):
        return self.f(pil_img)


class TransformT(object):
    """Each instance of this class represents a specific transform."""

    def __init__(self, name, xform_fn):
        self.name = name
        self.xform = xform_fn

    def pil_transformer(self, probability, level, img_shape):
        def return_function(im):
            if random.random() < probability:
                im = self.xform(im, level, img_shape)
            return im

        name = self.name + "({:.1f},{})".format(probability, level)
        return TransformFunction(return_function, name)


################## Transform Functions ##################
identity = TransformT("identity", lambda pil_img, level, _: pil_img)
flip_lr = TransformT(
    "FlipLR", lambda pil_img, level, _: pil_img.transpose(Image.FLIP_LEFT_RIGHT)
)
flip_ud = TransformT(
    "FlipUD", lambda pil_img, level, _: pil_img.transpose(Image.FLIP_TOP_BOTTOM)
)
# pylint:disable=g-long-lambda
auto_contrast = TransformT(
    "AutoContrast", lambda pil_img, level, _: ImageOps.autocontrast(pil_img)
)
equalize = TransformT("Equalize", lambda pil_img, level, _: ImageOps.equalize(pil_img))
invert = TransformT("Invert", lambda pil_img, level, _: ImageOps.invert(pil_img))
# pylint:enable=g-long-lambda
blur = TransformT("Blur", lambda pil_img, level, _: pil_img.filter(ImageFilter.BLUR))
smooth = TransformT(
    "Smooth", lambda pil_img, level, _: pil_img.filter(ImageFilter.SMOOTH)
)


def _rotate_impl(pil_img, level, _):
    """Rotates `pil_img` from -30 to 30 degrees depending on `level`."""
    degrees = int_parameter(level, 30)
    if random.random() > 0.5:
        degrees = -degrees
    return pil_img.rotate(degrees)


rotate = TransformT("Rotate", _rotate_impl)


def _posterize_impl(pil_img, level, _):
    """Applies PIL Posterize to `pil_img`."""
    level = int_parameter(level, 4)
    return ImageOps.posterize(pil_img, 4 - level)


posterize = TransformT("Posterize", _posterize_impl)


def _shear_x_impl(pil_img, level, img_shape):
    """Applies PIL ShearX to `pil_img`.

  The ShearX operation shears the image along the horizontal axis with `level`
  magnitude.

  Args:
    pil_img: Image in PIL object.
    level: Strength of the operation specified as an Integer from
      [0, `PARAMETER_MAX`].

  Returns:
    A PIL Image that has had ShearX applied to it.
  """
    level = float_parameter(level, 0.3)
    if random.random() > 0.5:
        level = -level
    return pil_img.transform(
        _width_height_from_img_shape(img_shape), Image.AFFINE, (1, level, 0, 0, 1, 0)
    )


shear_x = TransformT("ShearX", _shear_x_impl)


def _shear_y_impl(pil_img, level, img_shape):
    """Applies PIL ShearY to `pil_img`.

  The ShearY operation shears the image along the vertical axis with `level`
  magnitude.

  Args:
    pil_img: Image in PIL object.
    level: Strength of the operation specified as an Integer from
      [0, `PARAMETER_MAX`].

  Returns:
    A PIL Image that has had ShearX applied to it.
  """
    level = float_parameter(level, 0.3)
    if random.random() > 0.5:
        level = -level
    return pil_img.transform(
        _width_height_from_img_shape(img_shape), Image.AFFINE, (1, 0, 0, level, 1, 0)
    )


shear_y = TransformT("ShearY", _shear_y_impl)


def _translate_x_impl(pil_img, level, img_shape):
    """Applies PIL TranslateX to `pil_img`.

  Translate the image in the horizontal direction by `level`
  number of pixels.

  Args:
    pil_img: Image in PIL object.
    level: Strength of the operation specified as an Integer from
      [0, `PARAMETER_MAX`].

  Returns:
    A PIL Image that has had TranslateX applied to it.
  """
    level = int_parameter(level, 10)
    if random.random() > 0.5:
        level = -level
    return pil_img.transform(
        _width_height_from_img_shape(img_shape), Image.AFFINE, (1, 0, level, 0, 1, 0)
    )


translate_x = TransformT("TranslateX", _translate_x_impl)


def _translate_y_impl(pil_img, level, img_shape):
    """Applies PIL TranslateY to `pil_img`.

  Translate the image in the vertical direction by `level`
  number of pixels.

  Args:
    pil_img: Image in PIL object.
    level: Strength of the operation specified as an Integer from
      [0, `PARAMETER_MAX`].

  Returns:
    A PIL Image that has had TranslateY applied to it.
  """
    level = int_parameter(level, 10)
    if random.random() > 0.5:
        level = -level
    return pil_img.transform(
        _width_height_from_img_shape(img_shape), Image.AFFINE, (1, 0, 0, 0, 1, level)
    )


translate_y = TransformT("TranslateY", _translate_y_impl)


def _crop_impl(pil_img, level, img_shape, interpolation=Image.BILINEAR):
    """Applies a crop to `pil_img` with the size depending on the `level`."""
    height, width = img_shape
    cropped = pil_img.crop((level, level, width - level, height - level))
    resized = cropped.resize((width, height), interpolation)
    return resized


crop_bilinear = TransformT("CropBilinear", _crop_impl)


def _solarize_impl(pil_img, level, _):
    """Applies PIL Solarize to `pil_img`.

  Translate the image in the vertical direction by `level`
  number of pixels.

  Args:
    pil_img: Image in PIL object.
    level: Strength of the operation specified as an Integer from
      [0, `PARAMETER_MAX`].

  Returns:
    A PIL Image that has had Solarize applied to it.
  """
    level = int_parameter(level, 256)
    return ImageOps.solarize(pil_img, 256 - level)


solarize = TransformT("Solarize", _solarize_impl)


def _cutout_pil_impl(pil_img, level, img_shape):
    """Apply cutout to pil_img at the specified level."""
    size = int_parameter(level, 20)
    if size <= 0:
        return pil_img
    img_height, img_width, num_channels = (img_shape[0], img_shape[1], 3)
    _, upper_coord, lower_coord = create_cutout_mask(
        img_height, img_width, num_channels, size
    )
    pixels = pil_img.load()  # create the pixel map

    for i in range(upper_coord[1], lower_coord[1]):  # for every col:
        for j in range(upper_coord[0], lower_coord[0]):  # For every row
            pixels[i, j] = (125, 115, 104)  # set the colour accordingly
    return pil_img


cutout = TransformT("Cutout", _cutout_pil_impl)


def _enhancer_impl(enhancer):
    """Sets level to be between 0.1 and 1.8 for ImageEnhance transforms of PIL."""

    def impl(pil_img, level, _):
        v = float_parameter(level, 1.8) + 0.1  # going to 0 just destroys it
        return enhancer(pil_img).enhance(v)

    return impl


color = TransformT("Color", _enhancer_impl(ImageEnhance.Color))
contrast = TransformT("Contrast", _enhancer_impl(ImageEnhance.Contrast))
brightness = TransformT("Brightness", _enhancer_impl(ImageEnhance.Brightness))
sharpness = TransformT("Sharpness", _enhancer_impl(ImageEnhance.Sharpness))

ALL_TRANSFORMS = [
    flip_lr,
    flip_ud,
    auto_contrast,
    equalize,
    invert,
    rotate,
    posterize,
    crop_bilinear,
    solarize,
    color,
    contrast,
    brightness,
    sharpness,
    shear_x,
    shear_y,
    translate_x,
    translate_y,
    cutout,
    blur,
    smooth,
]

NAME_TO_TRANSFORM = {t.name: t for t in ALL_TRANSFORMS}
TRANSFORM_NAMES = NAME_TO_TRANSFORM.keys()
