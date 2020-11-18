import inspect
from typing import Any, List, Tuple, Union

import torch.nn as nn
from datasets import Dataset


def get_label_list(labels: Union[List[Any], Tuple[Any]]) -> List[Any]:
    # Get a list of unique labels sorted alphabetically.
    unique_labels = set()
    for label in labels:
        unique_labels.add(label)
    label_list = list(unique_labels)
    label_list.sort()
    return label_list


def remove_unused_columns(model: nn.Module, dataset: Dataset) -> None:
    # This method is implemented in transformer's Trainer.
    # Inspect model forward signature to keep only the arguments it accepts.
    signature = inspect.signature(model.forward)
    signature_columns = list(signature.parameters.keys())
    # Labels may be named label or label_ids, the default data collator handles that.
    signature_columns += ["label", "label_ids"]
    columns = [k for k in signature_columns if k in dataset.column_names]
    dataset.set_format(type=dataset.format["type"], columns=columns)
