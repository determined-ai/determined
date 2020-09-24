# TensorFlow (tf.keras) CIFAR-10 CNN Example

This folder contains the example code to run a basic CIFAR-10 trained CNN with Determined's TF Keras API. 
The file version can be found on this [Keras CNN example](https://github.com/fchollet/keras/blob/master/examples/cifar10_cnn.py)

### Files
* **model_def.py**: The core code for the model. This includes building and compiling the model.
* **data.py**: The data loading and preparation code for the model.
* **const.yaml**: A configuration file that trains the model with constant hyperparameter values. This is also where you can set the flags used in the original script.
* **distributed.yaml**: Same as const.yaml, but instead uses multiple GPUs.
* **adaptive.yaml**: Uses state-of-the-art ASHA hyperparameter tuning algorithm.

### Data:
The current implementation uses CIFAR-10 data downloaded from AWS S3.

### To Run:
Installation instructions can be found under `docs/install-admin.html` or at [Determined installation page](https://docs.determined.ai/latest/index.html). 
After configuring the settings in const.yaml, run the following command: `det -m <master host:port> experiment create -f const.yaml . `

### Results:
Upon completion of the experiment, model should achieve target accuracy of ~74%.
