import functools
import logging
from typing import Dict

import attrdict
import ner_utils
import transformers

import determined.pytorch as det_torch
import model_hub.huggingface as hf
import model_hub.utils as utils


class NERTrial(hf.BaseTransformerTrial):
    def __init__(self, context: det_torch.PyTorchTrialContext) -> None:
        self.logger = logging.getLogger(__name__)
        self.context = context
        self.hparams = attrdict.AttrDict(context.get_hparams())
        self.data_config = attrdict.AttrDict(context.get_data_config())
        self.logger.info(self.context.get_experiment_config())

        # Load dataset and get metadata.
        # This needs to be done before we initialize the HF config, tokenizer, and model
        # because we need to know num_labels before doing so.
        self.raw_datasets = hf.default_load_dataset(self.data_config)
        datasets_metadata = ner_utils.get_dataset_metadata(self.raw_datasets, self.hparams)
        self.hparams["num_labels"] = datasets_metadata.num_labels

        super(NERTrial, self).__init__(context)

        # We need to create the tokenized dataset after init because we need to model and
        # tokenizer to be available.
        self.tokenized_datasets = ner_utils.build_tokenized_datasets(
            self.raw_datasets,
            self.model,
            self.data_config,
            self.tokenizer,
            datasets_metadata.text_column_name,
            datasets_metadata.label_column_name,
            datasets_metadata.label_to_id,
        )

        # Create metric reducer
        self.reducer = context.experimental.wrap_reducer(
            utils.PredLabelFnReducer(
                functools.partial(ner_utils.compute_metrics, datasets_metadata.label_list)
            ),
            for_training=False,
        )

    def build_training_data_loader(self) -> det_torch.DataLoader:
        return det_torch.DataLoader(
            self.tokenized_datasets["train"],
            batch_size=self.context.get_per_slot_batch_size(),
            collate_fn=transformers.DataCollatorForTokenClassification(self.tokenizer),
        )

    def build_validation_data_loader(self) -> det_torch.DataLoader:
        return det_torch.DataLoader(
            self.tokenized_datasets["validation"],
            batch_size=self.context.get_per_slot_batch_size(),
            collate_fn=transformers.DataCollatorForTokenClassification(self.tokenizer),
        )

    def evaluate_batch(self, batch: det_torch.TorchData, batch_idx: int) -> Dict:
        outputs = self.model(**batch)
        tmp_eval_loss, logits = outputs[:2]
        preds = logits.detach().cpu().numpy()
        out_label_ids = batch["labels"].detach().cpu().numpy()  # type: ignore
        self.reducer.update(preds, out_label_ids)  # type: ignore
        # Although we are returning the empty dictionary below, we will still get the metrics from
        # custom reducer that we passed to the context during initialization.
        return {}
