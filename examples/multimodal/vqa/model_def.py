from typing import Any, Dict, Sequence, Union

import numpy as np
import torch
import torch.nn as nn

import data
from interpret import vqa_resnet_interpret

from determined.pytorch import DataLoader, PyTorchTrial, LRScheduler
from determined.tensorboard.metric_writers.pytorch import TorchWriter
import determined as det
from utils import FullVQANet

TorchData = Union[Dict[str, torch.Tensor], Sequence[torch.Tensor], torch.Tensor]


class VQATrial(PyTorchTrial):
    def __init__(self, context: det.TrialContext) -> None:
        self.context = context
        self.hparams = context.get_hparams()
        self.logger = TorchWriter()

        self.model = self.context.wrap_model(
            FullVQANet(
                self.hparams["output_features"],
                self.hparams["max_answers"],
                self.hparams["num_tokens"],
            )
        )

        ckpt = torch.load("2017-08-04_00:55:19.pth")
        ckpt = ckpt["weights"]
        weights = {k[len("module.") :]: ckpt[k] for k in ckpt}
        state = self.model.state_dict()
        for w in weights:
            assert w in state
        state.update(weights)

        self.optimizer = self.context.wrap_optimizer(
            torch.optim.Adam(
                [p for p in self.model.parameters() if p.requires_grad],
                lr=self.hparams["lr"],
            )
        )

        self.scheduler = self.context.wrap_lr_scheduler(
            torch.optim.lr_scheduler.LambdaLR(
                self.optimizer,
                lambda iteration: 0.5
                ** (float(iteration) / self.hparams["lr_halflife"]),
            ),
            step_mode=LRScheduler.StepMode.STEP_EVERY_BATCH,
        )

    def train_batch(
        self, batch: TorchData, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        v, q, a, idx, q_len = batch

        self.model.resnet_layer4.requires_grad_ = False

        out = self.model(v, q, q_len)
        nll = -nn.functional.log_softmax(out)
        loss = (nll * a / 10).sum(dim=1).mean()

        self.context.backward(loss)
        self.context.step_optimizer(self.optimizer)
        self.scheduler.step()

        _, predicted_index = out.max(dim=1, keepdim=True)
        agreeing = a.gather(dim=1, index=predicted_index)
        acc = (agreeing * 0.3).clamp(max=1)

        return {"loss": loss, "acc": acc}

    def evaluate_batch(
        self,
        batch: TorchData,
    ) -> Dict[str, torch.Tensor]:
        v, q, a, idx, q_len = batch

        out = self.model(v, q, q_len)
        nll = -nn.functional.log_softmax(out)
        loss = (nll * a / 10).sum(dim=1).mean()

        _, predicted_index = out.max(dim=1, keepdim=True)
        agreeing = a.gather(dim=1, index=predicted_index)
        acc = (agreeing * 0.3).clamp(max=1)

        # Interpretability
        n_images = v.size()[0]
        sample = np.random.choice(n_images)
        text_attr, image_attr = vqa_resnet_interpret(
            self.model,
            self.val_dataset,
            self.context.device,
            v[sample],
            self.val_dataset.raw_questions[idx[sample]],
            self.val_dataset.raw_answers[idx[sample]],
            q=q[sample],
            q_len=q_len[sample],
        )
        self.logger.writer.add_image("TextAttribution", text_attr, idx[sample])
        self.logger.writer.add_image("ImageAttribution", image_attr, idx[sample])
        print("done interpreting an image")

        return {"loss": loss}

    def build_training_data_loader(self) -> DataLoader:
        self.train_dataset = data.get_dataset(
            self.hparams["bucket_name"],
            self.hparams["image_size"],
            self.hparams["train_central_fraction"],
            val=True,
        )
        loader = DataLoader(
            self.train_dataset,
            batch_size=self.context.get_per_slot_batch_size(),
            shuffle=True,  # only shuffle the data in training
            num_workers=self.hparams["data_workers"],
            collate_fn=data.collate_fn,
        )
        return loader

    def build_validation_data_loader(self) -> DataLoader:
        self.val_dataset = data.get_dataset(
            self.hparams["bucket_name"],
            self.hparams["image_size"],
            self.hparams["train_central_fraction"],
            val=True,
        )
        self.val_dataset.length = 1000
        loader = DataLoader(
            self.val_dataset,
            batch_size=self.context.get_per_slot_batch_size(),
            shuffle=False,  # only shuffle the data in training
            num_workers=self.hparams["data_workers"],
            collate_fn=data.collate_fn,
        )
        return loader
