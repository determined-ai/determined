*This folder contains the required files and the example code to use Tensorflow's Image Segmentation via UNet tutorial with Determined.*
## The file version can be found on [Tensorflow Image Segmentation with UNet](https://www.tensorflow.org/tutorials/images/segmentation)

For this implementation, we removed the system configuration since Determined will be managing this. The core functionality of the original script remains unchanged.

### Folders and Files
* **model_def.py**: Contains the core code for the model. This includes building and compiling the model.  
* **const.yaml**: Contains the configuration for the experiment. This is also where you can set the flags used in the original script.  

### Data
   The data used for this script was fetched via Tensorflow Datasets as done by the tutorial itself. The original Oxford-IIIT Pet dataset is linked [here](https://www.robots.ox.ac.uk/~vgg/data/pets/). 

### To Run
   *Prerequisites*:  
      Installation instruction found under at [Determined installation page](https://docs.determined.ai/latest/index.html).

   After configuring the settings in const.yaml. Run the following command:
     `det -m <master host:port> experiment create -f const.yaml . `

