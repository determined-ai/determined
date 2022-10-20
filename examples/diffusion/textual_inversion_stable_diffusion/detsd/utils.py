from typing import List, Sequence, Tuple

import torch
import torch.nn as nn


def add_new_tokens_to_tokenizer(
    concept_token: str,
    initializer_tokens: Sequence[str],
    tokenizer: nn.Module,
) -> Tuple[List[int], List[int], str]:
    """Helper function for adding new tokens to the tokenizer and extending the corresponding
    embeddings appropriately, given a single concept token and its sequence of corresponding
    initializer tokens.  Returns the lists of ids for the initializer tokens and their dummy
    replacements, as well as the string representation of the dummies.
    """
    initializer_ids = tokenizer(
        initializer_tokens,
        padding="max_length",
        truncation=True,
        max_length=tokenizer.model_max_length,
        return_tensors="pt",
        add_special_tokens=False,
    ).input_ids

    try:
        special_token_ids = tokenizer.all_special_ids
    except AttributeError:
        special_token_ids = []

    non_special_initializer_locations = torch.isin(
        initializer_ids, torch.tensor(special_token_ids), invert=True
    )
    non_special_initializer_ids = initializer_ids[non_special_initializer_locations]
    if len(non_special_initializer_ids) == 0:
        raise ValueError(
            f'"{initializer_tokens}" maps to trivial tokens, please choose a different initializer.'
        )

    # Add a dummy placeholder token for every token in the initializer.
    dummy_placeholder_token_list = [
        f"{concept_token}_{n}" for n in range(len(non_special_initializer_ids))
    ]
    dummy_placeholder_tokens = " ".join(dummy_placeholder_token_list)
    num_added_tokens = tokenizer.add_tokens(dummy_placeholder_token_list)
    if num_added_tokens != len(dummy_placeholder_token_list):
        raise ValueError(
            f"Subset of {dummy_placeholder_token_list} tokens already exist in tokenizer."
        )

    dummy_placeholder_ids = tokenizer.convert_tokens_to_ids(
        dummy_placeholder_token_list
    )
    # Sanity check
    assert len(dummy_placeholder_ids) == len(
        non_special_initializer_ids
    ), 'Length of "dummy_placeholder_ids" and "non_special_initializer_ids" must match.'

    return non_special_initializer_ids, dummy_placeholder_ids, dummy_placeholder_tokens
