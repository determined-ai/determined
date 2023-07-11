import pinecone
import torch
from datasets import load_dataset
from transformers import BertModel, BertTokenizer

from determined.pytorch.experimental import TorchBatchProcessor, torch_batch_process

PINCONE_ENV = "<YOUR_PINECONE_ENV>"
API_KEY = "<YOUR_PINECONE_API_KEY>"


class BertEmbeddingProcessor(TorchBatchProcessor):
    def __init__(self, context):
        self.tokenizer = BertTokenizer.from_pretrained("bert-base-uncased")
        self.model = BertModel.from_pretrained("bert-base-uncased", output_hidden_states=True)

        self.device = context.device

        self.model = context.prepare_model_for_inference(self.model)

        self.output = []

        self.context = context
        self.rank = self.context.distributed.get_rank()
        self.index = pinecone.Index("bert-embedding-example")

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

            self.output.append({"embeddings": outputs, "id": batch["_id"]})

    def on_checkpoint_start(self):
        if len(self.output) == 0:
            return
        vector = []
        for batch in self.output:
            records = zip(batch["embeddings"], batch["id"])
            for record in records:
                embeddings = record[0].tolist()
                id = record[1]
                vector.append(
                    (
                        id,  # Vector ID
                        embeddings,  # Dense vector values
                    )
                )

        self.index.upsert(vectors=vector, namespace="scidocs")

        self.output = []


if __name__ == "__main__":
    pinecone.init(api_key=API_KEY, environment=PINCONE_ENV)
    dataset = load_dataset("BeIR/scidocs", "corpus", split="corpus")
    torch_batch_process(
        BertEmbeddingProcessor,
        dataset,
        batch_size=64,
        checkpoint_interval=10,
    )
