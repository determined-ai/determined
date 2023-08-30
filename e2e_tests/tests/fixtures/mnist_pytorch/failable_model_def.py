import os
from typing import Tuple, cast

import torch
from model_def import MNistTrial


class MNistFailable(MNistTrial):
    def train_batch(self, batch, epoch_idx, batch_idx):
        if "FAIL_AT_BATCH" in os.environ and int(os.environ["FAIL_AT_BATCH"]) == batch_idx:
            raise Exception(f"failed at this batch {batch_idx}")

        print("BATCH_IDX", batch_idx, "EPOCH IDX", epoch_idx)
        batch = cast(Tuple[torch.Tensor, torch.Tensor], batch)
        data, labels = batch

        output = self.model(data)
        loss = torch.nn.functional.nll_loss(output, labels)

        self.context.backward(loss)
        self.context.step_optimizer(self.optimizer)

        return {"loss": loss}
