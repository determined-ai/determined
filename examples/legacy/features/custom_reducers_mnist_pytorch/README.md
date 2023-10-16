# PyTorch Custom Reducers (MNIST)
This tutorial shows how to use custom reducers with PyTorch.  In this example,
the custom reducer is a per-class F1 score.

This example is based on Determined's `mnist_pytorch` tutorial, with the custom
reducer as the only modification.

## Files
* **model_def.py**: Where the custom reducer is defined and used.
* All other files are identical to the `mnist_pytorch` tutorial code.

## To Run
If you have not yet installed Determined, installation instructions can be found
under `docs/install-admin.html` or at https://docs.determined.ai/latest/index.html

Run the following command: `det -m <master host:port> experiment create -f
const.yaml .`. The other configurations can be run by specifying the appropriate
configuration file in place of `const.yaml`.

## Results
You should see the per-class F1 scores in the Determined WebUI and while
viewing the tensorboard results for the experiment.  The remaining metrics
should match the behvaior of the `mnist_pytorch` tutorial.

The custom reducers should work whether you run a single-slot experiment or a
multi-slot experiment with distributed training.
