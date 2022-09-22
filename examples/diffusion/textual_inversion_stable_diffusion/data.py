import os
import PIL
import torch
from PIL import Image
from typing import Callable, List, Sequence

from torch.utils.data import Dataset
from torchvision import transforms

from training_templates import TEMPLATE_DICT

INTERPOLATION_DICT = {
    "nearest": transforms.InterpolationMode.NEAREST,
    "bilinear": transforms.InterpolationMode.BILINEAR,
    "bicubic": transforms.InterpolationMode.BICUBIC,
}


class TextualInversionDataset(Dataset):
    """Dataset for textual inversion, pairing tokenized captions with images.  Contains
    dictionaries of tokenized-caption, image pairs, corresponding to the 'input_ids',
    'pixel_values' keys, respectively.
    """

    def __init__(
        self,
        train_img_dirs: Sequence[str],
        tokenizer_fn: Callable,
        concept_tokens: Sequence[str],
        learnable_properties: Sequence[str],
        img_size: int = 512,
        interpolation: str = "bicubic",
        flip_p: float = 0.0,
        center_crop: bool = False,
    ):
        assert (
            len(train_img_dirs) == len(concept_tokens) == len(learnable_properties)
        ), "train_img_dirs, concept_tokens, and learnable_properties must have equal lens."

        assert (
            interpolation in INTERPOLATION_DICT
        ), f"interpolation must be in {list(INTERPOLATION_DICT.keys())}"
        self.interpolation = INTERPOLATION_DICT[interpolation]
        for prop in learnable_properties:
            assert (
                prop in TEMPLATE_DICT
            ), f"learnable_properties must be one of {list(TEMPLATE_DICT.keys())}, not {prop}."

        self.train_img_dirs = train_img_dirs
        self.tokenizer_fn = tokenizer_fn
        self.learnable_properties = learnable_properties
        self.img_size = img_size
        self.concept_tokens = concept_tokens
        self.center_crop = center_crop
        self.flip_p = flip_p

        self._base_img_trans = transforms.Compose(
            [
                transforms.Resize(size=self.img_size, interpolation=self.interpolation),
                transforms.ToTensor(),
                transforms.RandomHorizontalFlip(p=self.flip_p),
            ]
        )

        self.records = []
        for dir_path, token, prop in zip(
            self.train_img_dirs, concept_tokens, self.learnable_properties
        ):
            templates = TEMPLATE_DICT[prop]
            imgs = self._get_imgs_from_dir_path(dir_path)
            img_ts = self._convert_imgs_to_tensors(imgs)
            for img_t in img_ts:
                for text in templates:
                    text_with_token = text.format(token)
                    self.records.append(
                        {"input_ids": self._tokenize_text(text_with_token), "pixel_values": img_t}
                    )

    def _get_imgs_from_dir_path(self, dir_path: str) -> List[Image.Image]:
        """Gets all images from a directory and converts them to tensors."""
        imgs = []
        for file_path in os.listdir(dir_path):
            path = os.path.join(dir_path, file_path)
            try:
                img = Image.open(path)
                if not img.mode == "RGB":
                    img = img.convert("RGB")
                imgs.append(img)
            except PIL.UnidentifiedImageError:
                print(f"File at {path} raised UnidentifiedImageError and will be skipped.")
        return imgs

    def _convert_imgs_to_tensors(self, imgs: List[Image.Image]) -> List[torch.Tensor]:
        """Converts a list of PIL images into appropriately transformed tensors."""
        img_ts = []
        for img in imgs:
            if self.center_crop:
                img = transforms.CenterCrop(size=min(img.size))(img)
            img_t = self._base_img_trans(img)
            # Normalize the tensor to be in the range [-1, 1]
            img_t = (img_t - 0.5) * 2.0
            img_ts.append(img_t)
        return img_ts

    def _tokenize_text(self, text: str) -> torch.Tensor:
        """Tokenizes text and removes the batch dimension."""
        tokenized_text = self.tokenizer_fn(text)
        return tokenized_text

    def __len__(self):
        return len(self.records)

    def __getitem__(self, idx):
        return self.records[idx]
