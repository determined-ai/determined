# DeepSpeed CIFAR Example
This example is adapted from the 
[pipeline parallelism example in the DeepSpeedExamples](https://github.com/microsoft/DeepSpeedExamples/tree/master/pipeline_parallelism) 
repository. It is intended to demonstrate a simple usecase of DeepSpeed's PipelineEngine with Determined.

## Files
* **model_def.py**: The core code for the model. This includes building and compiling the model.
* **alexnet.py**: Specifies the AlexNet architecture.

### Configuration Files
* **ds_config.json**: The DeepSpeed config file.
* **distributed.yaml**: Determined config to train the model with 2-stage pipeline parallelism.

## Data
The CIFAR-10 dataset is downloaded from https://www.cs.toronto.edu/~kriz/cifar.html.

## To Run
If you have not yet installed Determined, installation instructions can be found
under `docs/install-admin.html` or at https://docs.determined.ai/latest/index.html

Run the following command: 
```
det experiment create distributed.yaml .
```

## Results
Training the model with the hyperparameter settings in `distributed.yaml` on 2 
NVidia Tesla V100s on a single node should yield a throughput of at least 800 samples/sec.  
