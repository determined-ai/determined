# Distributed Batch Inference Metrics Example

This example shows how to create and organize metrics for batch inference in a distributed job
utilizing ``TorchBatchProcessor``. We can now link an inference trial to a ``ModelVersion`` or ``Checkpoint``
and later analyze the metrics that were generated for a given ``ModelVersion``.

This example is largely meant to be a simple toy example to demonstrate the new metrics functionality to group by
saved ``ModelVersion``. 

## Prerequisites

You will need to generate a ``Model`` with a set of ``ModelVersion`` for the MNIST dataset ahead of time
to run this. 

1. Create a new ``Model`` through the frontend UI or SDK.
2. Run the example ``determined/examples/tutorials/mnist_pytorch`` and save the ``Checkpoint`` to the ``Model`` from (1) as a ``ModelVersion``. 
3. Edit the ``distributed_inference.yaml`` to point to your ``Model`` from (1) and associated ``ModelVersion`` from (2).

More information: [Model Registry Documentation](https://docs.determined.ai/latest/model-dev-guide/model-management/model-registry-org.html)

## How to get Metrics
The results can be fetched later by SDK like so:

```python3
from determined.experimental import client 
model = client.get_model("<YOUR_MODEL_NAME_HERE>") 
model_version = model.get_version(1) 
metrics = model_version.get_metrics()  # Generator of all associated metrics 
```
