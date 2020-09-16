
*This folder contains the required files to show how to build a multi prediction MNIST network using Determined's PyTorch API.*

### Folders and Files
* **model_def.py**: Contains the core code for the model. This includes building and compiling the model.
* **data.py**: Contains the data loading and preparation code for the model.
* **layers.py**: Defines the convolutional layers that the model uses. 
* **const.yaml**: Contains the basic configuration for the experiment. 
* **adaptive.yaml**: Uses state-of-the-art ASHA hyperparameter tuning algorithm. 

### To Run
   *Prerequisites*:
      Installation instruction found under `docs/install-admin.html` or at [Determined installation page](https://docs.determined.ai/latest/index.html).

   After configuring the settings in const.yaml. Run the following command:
     `det -m <master host:port> experiment create -f const.yaml . `

### Results
    Upon completion of the experiment, model should achieve target accuracy of ~97%. 
