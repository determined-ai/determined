# Batch inference with Core API
## Overview
This example illustrates how to run distributed batch inference with Core API. Determined's Core API is very flexible
and can be used to run almost anything, including batch inference. 

With Core API, we are able to write an example that
- is distributed across worker
- can be preempted and resumed
- can be monitored on the Determined UI

However, using Core API directly would require the user to directly handle 
- low-level parallel programming concepts such as gather, rank 
- Determined machinery such as creating and loading checkpoint, preemption and resumption
- initialization of appropriate distributed context

We include this example here as a comparison against `torch_batch_process` examples.

## Detailed on this example
We are running inference with a simple vision model on the MNIST dataset. We then store the prediction outcome to the
shared file system.

To run the example, simply run `det e create config.yaml .`
