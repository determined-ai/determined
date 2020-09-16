
*This folder contains the example code to train a simple DNN on the MNIST dataset using TensorFlow's Estimator API.*

### Folders and Files
* **model_def.py**: Contains the core code for the model. This includes building and compiling the model.
* **const.yaml**: Contains the configuration for the experiment. This is also where you can set the flags used in the original script.
* **distributed.yaml**: Same as const.yaml, but instead uses multiple GPUs.
* **adaptive.yaml**: Uses state-of-the-art ASHA hyperparameter tuning algorithm.

### To Run:
   *Prerequisites*:
      Installation instruction found under `docs/install-admin.html` or at [Determined installation page](https://docs.determined.ai/latest/index.html).

   After configuring the settings in const.yaml. Run the following command:
     `det -m <master host:port> experiment create -f const.yaml . `

### Results:
    Upon completion of the experiment, model should achieve a target accuracy of ~95%.
