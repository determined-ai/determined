
*This folder contains the required files and the example code to use TF Keras's Basic Image Classification tutorial with Determined.*
## The file version can be found on the [Tensorflow tutorials page](https://www.tensorflow.org/tutorials/keras/classification)

### Folders and Files
* **model_def.py**: Contains the core code for the model. This includes building and compiling the model.
* **data.py**: Contains the data loading and preparation code for the model.
* **const.yaml**: Contains the basic configuration for the experiment. 
* **distributed.yaml**: Same as const.yaml, but instead uses multiple GPUs.
* **adaptive.yaml**: Uses state-of-the-art ASHA hyperparameter tuning algorithm. 

### To Run
   *Prerequisites*:
      Installation instruction found under `docs/install-admin.html` or at [Determined installation page](https://docs.determined.ai/latest/index.html).

   After configuring the settings in const.yaml. Run the following command:
     `det -m <master host:port> experiment create -f const.yaml . `

### Results
    Upon completion of the experiment, model should achieve target accuracy of ~85%. 
