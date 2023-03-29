import logging

import model_def

import determined as det
from determined import pytorch

if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)

    with pytorch.init() as context:
        trial = model_def.OneVarPytorchTrial(context, lr=0.001)
        trainer = pytorch.Trainer(trial, context)
        trainer.fit()
