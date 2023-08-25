# PyTorch Geometric - proteins_topk_pool example

This example demonstrates how to use the [PyTorch
Geometric](https://pytorch-geometric.readthedocs.io/en/latest/) library with
Determined. It was adapted from the [protein_topk_pool
example](https://github.com/rusty1s/pytorch_geometric/blob/master/examples/proteins_topk_pool.py).

The key parts of this example are contained in the following functions in `model_def.py`:
- `build_training_data_loader`, `build_validation_data_loader`:
  use `determined.pytorch.DataLoader` in conjunction with `torch_geometric.data.dataloader.Collater`
  to make use of graph data mini-batching.
- `get_batch_length`: `pytorch_geometric` provides its own class for batches,
  `torch_geometric.data.batch.Batch`. Since it has a custom way of exposing the batch dimension size,
  `batch.num_graphs`, the users must override this method, so the trial could properly calculate
  the batch sizes at runtime.

Also, this example has a few extra dependencies installed via `startup-hook.sh`,
specifically `torch_sparse` and `torch_scatter`.
Building these packages with CUDA support (i.e. in GPU environments) may take
a significant amount of time (30-40 minutes), so our code pins them to specific
PyTorch & CUDA version instead, and uses prebuilt upstream Python wheels.
Make sure to change the pinned version appropriately if you're planning to use
a different version of PyTorch or CUDA.

## Files
* **model_def.py**: Model and trial definition.
* **startup-hook.sh**: Install extra dependencies.

### Configuration Files
* **const.yaml**: Train the model on a single GPU with constant hyperparameter values.
* **distributed.yaml**: Distributed training on 4 GPUs.
* **adaptive.yaml**: Hyperparameter search enabled.

## Data
The example uses the `PROTEINS` dataset which is provided as part of pytorch_geometric library.

## To Run
If you have not yet installed Determined, installation instructions can be found
under `docs/install-admin.html` or at https://docs.determined.ai/latest/index.html

This example requires Determined version 0.16.2 or newer.

Run the following command: `det -m <master host:port> experiment create -f
const.yaml .`. The other configurations can be run by specifying the appropriate
configuration file in place of `const.yaml`.

## Results
The trial will achieve ~75% accuracy after training for 10 epochs, and will approach ~80% accuracy
by epoch 200 in a few minutes of runtime on an NVIDIA K80.
