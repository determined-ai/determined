from typing import List, Tuple

import torch.nn as nn


def add_new_tokens_to_tokenizer(
    concept_str: str,
    initializer_strs: str,
    tokenizer: nn.Module,
) -> Tuple[List[int], List[int], str]:
    """Helper function for adding new tokens to the tokenizer and extending the corresponding
    embeddings appropriately, given a single concept token and its sequence of corresponding
    initializer tokens.  Returns the lists of ids for the initializer tokens and their dummy
    replacements, as well as the string representation of the dummies.
    """
    initializer_ids = tokenizer(
        initializer_strs,
        return_tensors="pt",
        add_special_tokens=False,
    ).input_ids

    # Add a dummy placeholder token for every token in the initializer.
    dummy_placeholder_str_list = [f"<{concept_str}>_{n}" for n in range(len(initializer_ids))]
    dummy_placeholder_strs = " ".join(dummy_placeholder_str_list)

    tokenizer.add_tokens(dummy_placeholder_str_list)
    dummy_placeholder_ids = tokenizer.convert_tokens_to_ids(dummy_placeholder_str_list)
    # Sanity check that the dummies correspond to the correct number of ids.
    assert len(dummy_placeholder_ids) == len(
        initializer_ids
    ), 'Length of "dummy_placeholder_ids" and "initializer_ids" must match.'

    return initializer_ids, dummy_placeholder_ids, dummy_placeholder_strs
