
*This folder contains the example code to run a basic CIFAR10-trained CNN with Determined's PyTorch API.*
## The file version can be found on this [Keras CNN example](https://github.com/fchollet/keras/blob/master/examples/cifar10_cnn.py)

### Folders and Files
* **model_def.py**: Contains the core code for the model. This includes building and compiling the model.
* **const.yaml**: Contains the configuration for the experiment. This is also where you can set the flags used in the original script.
* **distributed.yaml**: Same as const.yaml, but instead uses multiple GPUs.
* **adaptive.yaml**: Uses state-of-the-art ASHA hyperparameter tuning algorithm.

### Data:
   The current implementation uses CIFAR10 data downloaded from AWS S3.

### To Run:
   *Prerequisites*:
      Installation instruction found under `docs/install-admin.html` or at [Determined installation page](https://docs.determined.ai/latest/index.html).

   After configuring the settings in const.yaml. Run the following command:
     `det -m <master host:port> experiment create -f const.yaml . `

### Results:
    Upon completion of the experiment, model should achieve target accuracy of ~74%.
