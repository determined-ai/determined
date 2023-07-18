# Batch inference with Core API & Torch Batch Processing API

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

You will see that using the Torch Batch Processing API for the same task is a lot easier as it abstracted away all the 
low level details and provides useful helper functions.

## Detailed on this example

We are running inference with a simple vision model on the CIFAR10 dataset. We then store the prediction outcome to the
file system in the Core API example and to the same storage system used by Determined checkpoints in the 
`torch_batch_process` example. You can access the output through the underlying storage (e.g. s3 bucket, shared_fs).

To run the Core API example, simply run `det e create core_api_config.yaml .`
To run the Torch Batch Processing example, simply run `det e create torch_batch_process_config.yaml .`
