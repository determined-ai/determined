from typing import Any, Callable, Dict, TypeVar

import torch.nn as nn

A = TypeVar("A")
B = TypeVar("B")


def merge_dicts(d1: Dict[A, B], d2: Dict[A, B], f: Callable[[B, B], B]) -> Dict[A, B]:
    """
    Merges dictionaries with a custom merge function.
    E.g. if k in d1 and k in d2, result[k] == f(d1[k], d2[k]).
    Otherwise, if e.g. k is in only d1, result[k] == d1[k]
    """
    d1_keys = d1.keys()
    d2_keys = d2.keys()
    shared = d1_keys & d2_keys
    d1_exclusive = d1_keys - d2_keys
    d2_exclusive = d2_keys - d1_keys
    new_dict = {k: f(d1[k], d2[k]) for k in shared}
    new_dict.update({k: d1[k] for k in d1_exclusive})
    new_dict.update({k: d2[k] for k in d2_exclusive})
    return new_dict


class LambdaModule(nn.Module):
    """
    Wrap a lambda as an nn.Module.
    """

    def __init__(self, lam: Callable) -> None:
        super().__init__()
        self.lam = lam

    def forward(self, x: Any) -> Any:
        return self.lam(x)
