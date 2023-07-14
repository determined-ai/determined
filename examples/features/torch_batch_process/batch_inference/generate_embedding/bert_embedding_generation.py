import logging
import os
import pathlib

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

        self.output_dir = "/tmp/data/embeddings"
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
            outputs = torch.mean(outputs["hidden_states"][-1], dim=1)

            self.output.append({"embeddings": outputs, "id": batch["_id"], "text": batch["text"]})
            self.last_index = batch_idx

    def on_checkpoint_start(self):
        if len(self.output) == 0:
            return
        file_name = f"bert_embedding_worker_{self.rank}_end_batch_{self.last_index}"
        file_path = pathlib.Path(self.output_dir, file_name)
        torch.save(self.output, file_path)
        self.output = []

    def on_finish(self):
        if self.rank == 0:
            chroma_dir = "/tmp/chroma"
            os.makedirs(chroma_dir, exist_ok=True)
            chroma_client = chromadb.Client(chromadb.config.Settings(persist_directory=chroma_dir))
            collection = chroma_client.create_collection(name="scidocs_embedding")

            embeddings = []
            documents = []
            ids = []

            for file in os.listdir(self.output_dir):
                file_path = pathlib.Path(self.output_dir, file)
                batches = torch.load(file_path)
                for batch in batches:
                    for embedding in batch["embeddings"]:
                        embedding = embedding.tolist()
                        embeddings.append(embedding)
                    ids += batch["id"]
                    documents += batch["text"]

            collection.add(embeddings=embeddings, documents=documents, ids=ids)
            logging.info(f"Embedding contains {collection.count()} entries")


if __name__ == "__main__":
    dataset = load_dataset("BeIR/scidocs", "corpus", split="corpus")
    experimental.torch_batch_process(
        EmbeddingProcessor,
        dataset,
        batch_size=64,
        checkpoint_interval=10,
    )
