import os

from model_def import MNistTrial


class MNistFailable(MNistTrial):
    def train_batch(self, batch, epoch_idx, batch_idx):
        if "FAIL_AT_BATCH" in os.environ and int(os.environ["FAIL_AT_BATCH"]) == batch_idx:
            raise Exception(f"failed at this batch {batch_idx}")

        print("BATCH_IDX", batch_idx, "EPOCH IDX", epoch_idx)
        return super().train_batch(batch, epoch_idx, batch_idx)
