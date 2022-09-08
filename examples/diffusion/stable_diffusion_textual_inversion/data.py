import os
import PIL
import random
from PIL import Image

import numpy as np
import torch
from torch.utils.data import Dataset
from torchvision import transforms

from training_templates import IMAGEN_OBJECT_TEMPLATES_SMALL, IMAGEN_STYLE_TEMPLATES_SMALL

INTERPOLATION_DICT = {
    "nearest": transforms.InterpolationMode.NEAREST,
    "bilinear": transforms.InterpolationMode.BILINEAR,
    "bicubic": transforms.InterpolationMode.BICUBIC,
}


class TextualInversionDataset(Dataset):
    """Create a dataset of size num_images * repeats, with num_images the number of training
    images included in train_img_dir. The Dataset's __getitem__ method returns a dictionary with
    input_ids and pixel_values keys, where input_ids come from applying the tokenizer to a caption
    describing the image (randomly drawn from fixed templates) and pixel_values are the normalized
    tensor values of the image."""

    def __init__(
        self,
        train_img_dir,
        tokenizer,
        learnable_property="object",  # [object, style]
        size=512,
        repeats=100,
        interpolation="bicubic",
        flip_p=0.5,
        placeholder_token="*",
        center_crop=False,
    ):

        self.train_img_dir = train_img_dir
        self.tokenizer = tokenizer
        self.learnable_property = learnable_property
        self.size = size
        self.placeholder_token = placeholder_token
        self.center_crop = center_crop
        self.flip_p = flip_p

        self.image_paths = [
            os.path.join(self.train_img_dir, file_path)
            for file_path in os.listdir(self.train_img_dir)
        ]

        self.num_images = len(self.image_paths)
        self._length = self.num_images

        self._length = self.num_images * repeats
        assert (
            interpolation in INTERPOLATION_DICT
        ), f"interpolation must be in {list(INTERPOLATION_DICT.keys())}"
        self.interpolation = INTERPOLATION_DICT[interpolation]

        assert learnable_property in (
            "object",
            "style",
        ), f'learnable_property must be "object" or "style", not {learnable_property}'

        self.templates = (
            IMAGEN_STYLE_TEMPLATES_SMALL
            if learnable_property == "style"
            else IMAGEN_OBJECT_TEMPLATES_SMALL
        )
        self.flip_transform = transforms.RandomHorizontalFlip(p=self.flip_p)

    def __len__(self):
        return self._length

    def __getitem__(self, i):
        example = {}

        # Generate a random caption drawn from the templates and include in the example.
        placeholder_string = self.placeholder_token
        text = random.choice(self.templates).format(placeholder_string)

        example["input_ids"] = self.tokenizer(
            text,
            padding="max_length",
            truncation=True,
            max_length=self.tokenizer.model_max_length,
            return_tensors="pt",
        ).input_ids[0]

        # Add the corresponding normalized image tensor to the example.
        image = Image.open(self.image_paths[i % self.num_images])
        if not image.mode == "RGB":
            image = image.convert("RGB")
        image_t = transforms.ToTensor()(image)
        if self.center_crop:
            crop_size = min(image_t.shape[-1], image_t.shape[-2])
            image_t = transforms.CenterCrop(crop_size)(image_t)
        image_t = transforms.Resize(self.size, interpolation=self.interpolation)(image_t)
        # Normalize the tensor to be in the range [-1, 1]
        image_t = (image_t - 0.5) * 2.0
        example["pixel_values"] = image_t

        return example
