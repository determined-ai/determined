# Training state-of-the-art architecture on ImageNet
In this example, we train a state-of-the-art architecture discovered by a recently published neural architecture search (NAS) algorithm called GAEA on the ImageNet dataset.  Please see the original paper by [Li et al.](https://arxiv.org/abs/2004.07802) for more information about the underlying NAS algorithm used to discover this architecture.

The training routine is based on [PC-DARTS](https://github.com/yuhuixu1993/PC-DARTS/blob/master/train_imagenet.py) with additional tricks for improved performance.  In particular, we use the swish activation and additional squeeze and excite modules in place of the auxiliary tower used in the PC-DARTS training routine.  

### Folders and Files
* **model_def.py**: Contains the model training code expected by Determined.
* **model.py**: Contains the model specification.
* **operations.py**: Contains the components used to build the model.
* **utils.py**: Contains special functions used during training.
* **data.py**: Contains the data loading code for the model.
* **distributed.yaml**: Experiment configuration for distributed training with fixed hyperparameter setting.

### Data
The data for ImageNet is the ILSVRC2012 version of the dataset, which is available [here](http://www.image-net.org/).  The code assumes the data is available in a Google Cloud Storage bucket; in the event a bucket is not provided, randomly generated data is used.

### To Run
   *Prerequisites*:  
      A Determined Google Cloud Compute cluster must be installed in order to run this example.  Please follow the directions [here](https://docs.determined.ai/latest/how-to/install-main.html) in order to install. 

   To run the example, please adjust the `distributed.yaml` configuration file to your liking.  In particular, you should at a minimum change the `bucket_name` field to reference your data bucket; you may also want to adjust the `slots_per_trial` parameter to increase the number of GPUs used for parallel training.  

   After configuring `distributed.yaml` you can submit an experiment to your Determined cluster by running the following command:
     `det -m <master host:port> experiment create distributed.yaml . `

### Expected Performance
After 300 steps (~24 Epochs), the top 5 validation accuracy should be close to 80% (see learning curve below).  At convergence, the top 1 validation accuracy should be close to the 76% reported in the original paper.  
![](./top5\_val.png)
