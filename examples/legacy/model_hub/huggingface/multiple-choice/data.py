"""
This DataCollatorForMultipleChoice class below is replicated from
   https://github.com/huggingface/transformers/blob/v4.3.3/examples/multiple-choice/run_swag.py
to work with Determined.  The license for the transformer's library is reproduced below.

==================================================================================================

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
import dataclasses
from typing import Optional, Union

import torch
import transformers


@dataclasses.dataclass
class DataCollatorForMultipleChoice:
    """
    Data collator that will dynamically pad the inputs for multiple choice received.

    Args:
        tokenizer: The tokenizer used for encoding the data.
        padding: Select a strategy to pad the returned sequences (according to the model's
            padding side and padding index) among:
            * `'longest'`: Pad to the longest sequence in the batch (or no padding if only a single
              sequence if provided).
            * `'max_length'`: Pad to a maximum length specified with the argument `max_length` or
              to the maximum acceptable input length for the model if that argument is not provided.
            * `'do_not_pad'` (default): No padding (i.e., can output a batch with sequences of
              different lengths).
        max_length: Maximum length of the returned list and optionally padding length (see above).
        pad_to_multiple_of: If set will pad the sequence to a multiple of the provided value.
            This is especially useful to enable the use of Tensor Cores on NVIDIA hardware wite
            compute capability >= 7.5 (Volta).
    """

    tokenizer: transformers.tokenization_utils_base.PreTrainedTokenizerBase
    padding: Union[bool, str, transformers.tokenization_utils_base.PaddingStrategy] = True
    max_length: Optional[int] = None
    pad_to_multiple_of: Optional[int] = None

    def __call__(self, features):
        label_name = "label" if "label" in features[0].keys() else "labels"
        labels = [feature.pop(label_name) for feature in features]
        batch_size = len(features)
        num_choices = len(features[0]["input_ids"])
        flattened_features = [
            [{k: v[i] for k, v in feature.items()} for i in range(num_choices)]
            for feature in features
        ]
        flattened_features = sum(flattened_features, [])

        batch = self.tokenizer.pad(
            flattened_features,
            padding=self.padding,
            max_length=self.max_length,
            pad_to_multiple_of=self.pad_to_multiple_of,
            return_tensors="pt",
        )

        # Un-flatten
        batch = {k: v.view(batch_size, num_choices, -1) for k, v in batch.items()}
        # Add back labels
        batch["labels"] = torch.tensor(labels, dtype=torch.int64)
        return batch
