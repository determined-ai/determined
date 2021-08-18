import inspect

import datasets as hf_datasets
import torch.nn as nn


"""
The removed_unused_columns function below is largely derived from
transformer's trainer._removed_unused_columns method.

The license for the transformer's library is reproduced below.

============================================================================

Copyright 2020 The HuggingFace Team. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
"""


def remove_unused_columns(model: nn.Module, dataset: hf_datasets.Dataset) -> None:
    # This method is implemented in transformer's Trainer.
    # Inspect model forward signature to keep only the arguments it accepts.
    signature = inspect.signature(model.forward)
    signature_columns = list(signature.parameters.keys())
    # Labels may be named label or label_ids, the default data collator handles that.
    signature_columns += ["label", "label_ids"]
    columns = [k for k in signature_columns if k in dataset.column_names]
    dataset.set_format(type=dataset.format["type"], columns=columns)
