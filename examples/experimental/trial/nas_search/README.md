*This folder contains the required files and the example code to use a simplified NAS model based on randomNAS_release script with Determined.*
## The repo can be found on [liamcli's example](https://github.com/liamcli/randomNAS_release/tree/6513a0a6a781ed1f0009ccd9bae622ae7f0a961d)

This script is based on randomNAS_release repo. For this implementation, we removed the system configuration since Determined will be managing this. We restricted changes to the original research code which prevents optimizations in certain areas to demonstrate Determined's capabilities with limited code changes.

This project can perform multiple functions:
  * Train a predefined architecture and perform evaluation on same arch
  * Train a random architecture and perform evaluation on same arch
  * Train one model and evaluate multiple architectures with the same weights,

### Folders and Files:
   **model_def.py**: Contains the core code for the model. This includes building and compiling the model.
   **data.py**: Contains the data loading and preparation code for the model.
   **const.yaml**: Contains the configuration for the experiment. This is also where you can set the flags used in the original script.  
   **random.yaml**: Contains the configuration for the experiment to use random search. This will randomly selected a starting seed to use.
   **randomNAS_files**: Contains the required files from the original repos. These files have been minimally changed to work with Determined.
   **randomNAS_files/genotypes.py**: Contains hard coded genotypes based on the original repo. If you would like to evaluate a particular architecture you can create a new genotype and save it here.  

### Data:
   The PTB data used for this script is automatically fetched based on [salesforce](https://github.com/salesforce/awd-lstm-lm/blob/32fcb42562aeb5c7e6c9dec3f2a3baaaf68a5cb5/getdata.sh).

   There are two options to access the data:
     
   1. Data can be downloaded by running the script provided by [salesforce](https://github.com/salesforce/awd-lstm-lm/blob/32fcb42562aeb5c7e6c9dec3f2a3baaaf68a5cb5/getdata.sh). The data needs to be available at the same path on all of the agents where Determined is running. The absolute file path will need to be uncommented and updated in the const.yaml and random.yaml under bind_mounts. Then when running, Determined will look for the absolute path to mount to the container_path assigned in the yaml files.  
     
   2. If the dataset does not exist during runtime, and the project will automatically download the data and save to the provided data_dir directory for future use.


### To Run:
   *Prerequisites*:  
      Installation instructions found under `docs/install-admin.html` or at the [Determined installation page](https://docs.determined.ai/latest/index.html).

   After configuring the settings in const.yaml. Run the following command:
     `det -m <master host:port> experiment create -f const.yaml . `

   To run and architecture search, set arch_to_use to None and eval_same_arch to True. An example can be found under arch_search.yaml.

   If you would like to train and evaluate a particular architecture, append the structure to randomNAS_files/genotypes.py. Then assign the variable name to arch_to_use in the yaml file and set eval_same_arch to True. An example can be found under train_one_arch.yaml.

   This project also allows for multiple architectures to be evaluate based on the same model. This is similar to weight sharing; however during training, only one architecture is used. This should only be used if you want to see what weight sharing might look like in Determined.

### Results
   Running an experiment with train_one_arch.yaml after 100 epochs should return about 71 perplexity while using the ASHA genotype. If you continue training over 300 epochs, perplexity should reach about 64.
