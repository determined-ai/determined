import logging
import os

from train import MNistTrial

import determined as det
from determined import pytorch


class MNistFailable(MNistTrial):
    def train_batch(self, batch, epoch_idx, batch_idx):
        if "FAIL_AT_BATCH" in os.environ and int(os.environ["FAIL_AT_BATCH"]) == batch_idx:
            raise Exception(f"failed at this batch {batch_idx}")

        print("BATCH_IDX", batch_idx, "EPOCH IDX", epoch_idx)
        return super().train_batch(batch, epoch_idx, batch_idx)


if __name__ == "__main__":
    info = det.get_cluster_info()
    assert info, "This test is intended to run on cluster only."

    # Configure logging
    logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)

    with pytorch.init() as train_context:
        trial = MNistFailable(context=train_context, hparams=info.trial.hparams)
        trainer = pytorch.Trainer(trial, train_context)
        trainer.fit(
            checkpoint_policy="none",
            checkpoint_period=pytorch.Batch(3),
            validation_period=pytorch.Batch(1),
            latest_checkpoint=info.latest_checkpoint,
        )
