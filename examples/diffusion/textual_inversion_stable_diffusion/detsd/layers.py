import torch
import torch.nn as nn


class ExtendedEmbedding(nn.Module):
    """A class for extending an embedding layer with additional tokens while also leaving the old
    and new embeddings as separate nn.Module instances. This allows only the new embeddings to be
    trained, for instance."""

    def __init__(self, original_embedding: nn.Module, new_embedding_weights: torch.Tensor) -> None:
        super().__init__()
        self.original_embedding = original_embedding
        self.old_embedding_vocab_size, self.embedding_dim = self.original_embedding.weight.shape
        assert (
            new_embedding_weights.shape[-1] == self.embedding_dim
        ), "new_embedding_weights must have the same embedding_dim as the original_embedding"
        self.new_embedding = nn.Embedding.from_pretrained(
            embeddings=new_embedding_weights, freeze=False
        )

    def forward(self, input_ids: torch.Tensor) -> torch.Tensor:
        output = torch.zeros(*input_ids.shape, self.embedding_dim, device=input_ids.device)
        idxs_for_old_embedding = input_ids < self.old_embedding_vocab_size
        idxs_for_new_embedding = torch.logical_not(idxs_for_old_embedding)
        inputs_ids_for_old_embedding = input_ids[idxs_for_old_embedding]
        inputs_ids_for_new_embedding = (
            input_ids[idxs_for_new_embedding] - self.old_embedding_vocab_size
        )

        output[idxs_for_old_embedding] = self.original_embedding(inputs_ids_for_old_embedding)
        output[idxs_for_new_embedding] = self.new_embedding(inputs_ids_for_new_embedding)

        return output
