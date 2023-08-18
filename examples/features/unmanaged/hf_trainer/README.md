## HuggingFace Trainer / Determined integration example

This example is based on the original HuggingFace example for [token classification](https://github.com/huggingface/transformers/tree/main/examples/pytorch/token-classification).

Only two lines are required to add the integration:

    from determined.experimental.hf_trainer import DetCallback

    # ...
    # Initialize our Trainer
    # trainer = Trainer(...

    trainer.add_callback(DetCallback())
