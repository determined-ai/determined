import os
import pathlib
import uuid

import pinecone
import torch
from datasets import load_dataset
from torch.profiler import ProfilerActivity
from transformers import BertModel, BertTokenizer

from determined.experimental.inference import TorchBatchProcessor, torch_batch_process

PINCONE_ENV = "<PINECONE_ENV>"
API_KEY = "<API_KEY>"
pinecone.init(api_key=API_KEY, environment=PINCONE_ENV)


class MyProcessor(TorchBatchProcessor):
    def __init__(self, context):
        self.tokenizer = BertTokenizer.from_pretrained("bert-base-uncased")
        self.model = BertModel.from_pretrained("bert-base-uncased", output_hidden_states=True)

        self.device = context.get_device()

        self.model = context.prepare_model_for_inference(self.model)

        self.output = []

        self.context = context
        self.last_index = 0
        self.rank = self.context.get_distributed_context().get_rank()
        self.output_dir = f"/run/determined/workdir/shared_fs/bert_batch_out/worker_{self.rank}"
        self.index = pinecone.Index("swy-test")

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

            self.output.append({"embeddings": outputs, "ids": batch["_id"]})
            self.last_index = batch_idx

    def on_checkpoint_start(self):
        if len(self.output) == 0:
            return
        file_name = f"prediction_output_{self.last_index}"

        if not os.path.exists(self.output_dir):
            os.makedirs(self.output_dir, exist_ok=True)
        file_path = pathlib.PosixPath(self.output_dir, file_name)
        torch.save(self.output, file_path)
        self.output = []

    def on_finish(self):
        for filename in os.listdir(self.output_dir):
            file_path = pathlib.PosixPath(self.output_dir, filename)
            batches = torch.load(file_path)
            vector = []
            for batch in batches:
                for idx, record in enumerate(batch["embeddings"]):
                    id = batch["ids"][idx]
                    record = record.tolist()
                    vector.append(
                        (
                            id,  # Vector ID
                            record,  # Dense vector values
                            {"tag": "test"},  # Vector metadata
                        )
                    )

        upsert_response = self.index.upsert(vectors=vector, namespace="testing_1")


if __name__ == "__main__":
    dataset = load_dataset("BeIR/scidocs", "corpus", split="corpus")
    torch_batch_process(
        MyProcessor,
        dataset,
        batch_size=64,
        max_batches=1,  # Remove this line to iterate through the whole dataset
    )
