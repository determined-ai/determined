import logging
import os
import pathlib
import shutil

import chromadb
import torch
from datasets import load_dataset
from transformers import BertModel, BertTokenizer

from determined.pytorch import experimental


class EmbeddingProcessor(experimental.TorchBatchProcessor):
    def __init__(self, context):
        self.tokenizer = BertTokenizer.from_pretrained("bert-base-uncased")
        self.model = BertModel.from_pretrained("bert-base-uncased", output_hidden_states=True)

        self.device = context.device

        self.model = context.prepare_model_for_inference(self.model)

        self.output = []

        self.context = context
        self.last_index = 0
        self.rank = self.context.distributed.get_rank()

        self.output_dir = "/tmp/data/bert_scidocs_embeddings"
        os.makedirs(self.output_dir, exist_ok=True)

    def process_batch(self, batch, batch_idx) -> None:
        with torch.no_grad():
            tokenized_text = self.tokenizer.batch_encode_plus(
                batch["text"],
                truncation=True,
                padding="max_length",
                max_length=512,
                add_special_tokens=True,
            )
            inputs = torch.tensor(tokenized_text["input_ids"])
            inputs = inputs.to(self.device)
            masks = torch.tensor(tokenized_text["attention_mask"])
            masks = masks.to(self.device)

            outputs = self.model(inputs, masks)

            # To create an embedding vector for each document,
            # 1. we take the hidden states from the last layer (output["hidden_states"][-1]),
            #    which is a tensor of (#examples, #tokens, #hidden_states) size.
            # 2. we calculate the average across the token-dimension, resulting in a tensor of
            #    (#examples, #hidden_states) size.
            outputs = torch.mean(outputs["hidden_states"][-1], dim=1)

            self.output.append({"embeddings": outputs, "id": batch["_id"], "text": batch["text"]})
            self.last_index = batch_idx

    def on_checkpoint_start(self):
        """
        In this function, each worker persists the in-memory embeddings to the file system of the agent machine.
           - Note that our set-up is for demonstration purpose only. Production use cases should save to a
             shared file system directory bind-mounted to all agent machines and experiment containers.
        File names contain rank and batch index information to avoid duplication between:
        - files created by different workers
        - files created by the same worker for different batches of input data
        """
        if len(self.output) == 0:
            return
        file_name = f"bert_embedding_worker_{self.rank}_end_batch_{self.last_index}"
        file_path = pathlib.Path(self.output_dir, file_name)
        torch.save(self.output, file_path)
        self.output = []

    def on_finish(self):
        """
        In this function, the chief worker (rank 0):
        - initializes a Chroma client and creates a Chroma collection. The collection is persisted in the
          directory "/tmp/chroma" of the container. The "/tmp" directory in the container is a bind-mount of the
          "/tmp" directory on the agent machine (see distributed.yaml file).
          - Note that our set-up is for demonstration purpose only. Production use cases should use a
            shared file system directory bind-mounted to all agent machines and experiment containers.
        - reads in and insert embedding files generated from all workers to the Chroma collection
        """
        if self.rank == 0:
            chroma_dir = "/tmp/chroma"
            os.makedirs(chroma_dir, exist_ok=True)
            chroma_client = chromadb.PersistentClient(chroma_dir)
            collection = chroma_client.get_or_create_collection(name="scidocs_embedding")

            embeddings = []
            documents = []
            ids = []

            for file in os.listdir(self.output_dir):
                file_path = pathlib.Path(self.output_dir, file)
                batches = torch.load(file_path, map_location="cuda:0")
                for batch in batches:
                    for embedding in batch["embeddings"]:
                        embedding = embedding.tolist()
                        embeddings.append(embedding)
                    ids += batch["id"]
                    documents += batch["text"]

            collection.upsert(embeddings=embeddings, documents=documents, ids=ids)
            logging.info(f"Embedding contains {collection.count()} entries")

            # Clean-up temporary embedding files
            shutil.rmtree(self.output_dir)


if __name__ == "__main__":
    dataset = load_dataset("BeIR/scidocs", "corpus", split="corpus")
    # Persisting embeddings can take quite a while on Chroma
    # Adding a limit on dataset size to ensure the example finishes fast
    dataset = dataset.select(list(range(500)))
    experimental.torch_batch_process(
        EmbeddingProcessor,
        dataset,
        batch_size=64,
        checkpoint_interval=10,
    )
