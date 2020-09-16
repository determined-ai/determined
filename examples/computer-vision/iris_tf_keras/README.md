
*This folder contains the example code to run an Iris species categorization model with Determined's TF Keras API.*
## The file version can be found on this [Iris species categorization medium post](https://medium.com/@nickbortolotti/iris-species-categorization-using-tf-keras-tf-data-and-differences-between-eager-mode-on-and-off-9b4693e0b22).

### Folders and Files
* **model_def.py**: Contains the core code for the model. This includes building and compiling the model.
* **startup-hook.sh**: Contains extra dependencies that Determined is required to install.
* **const.yaml**: Contains the configuration for the experiment. This is also where you can set the flags used in the original script.
* **distributed.yaml**: Same as const.yaml, but instead uses multiple GPUs.
* **adaptive.yaml**: Uses state-of-the-art ASHA hyperparameter tuning algorithm.

### Data:
   The current implementation uses [UCI's Iris Data Set](https://archive.ics.uci.edu/ml/datasets/iris).

### To Run:
   *Prerequisites*:
      Installation instruction found under `docs/install-admin.html` or at [Determined installation page](https://docs.determined.ai/latest/index.html).

   After configuring the settings in const.yaml. Run the following command:
     `det -m <master host:port> experiment create -f const.yaml . `

### Results:
    Upon completion of the experiment, model should achieve a target accuracy of ~95%.
