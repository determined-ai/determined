from dataclasses import dataclass
from typing import Callable

import torch.nn as nn
import torchvision.models as models


@dataclass
class BackboneMetadata:
    feature_size: int
    build_fn: Callable[[], nn.Module]


BACKBONE_METADATA_BY_NAME = {
    "resnet18": BackboneMetadata(
        feature_size=512, build_fn=lambda: models.resnet18(pretrained=True)
    ),
    "resnet34": BackboneMetadata(
        feature_size=512, build_fn=lambda: models.resnet34(pretrained=True)
    ),
    "resnet50": BackboneMetadata(
        feature_size=2048, build_fn=lambda: models.resnet50(pretrained=True)
    ),
    "resnet101": BackboneMetadata(
        feature_size=2048, build_fn=lambda: models.resnet101(pretrained=True)
    ),
}
