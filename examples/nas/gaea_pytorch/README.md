# Geometry-Aware Exponential Algorithms for Neural Architecture Search (GAEA)
This example implements the NAS method introduced by [Li et al.](https://arxiv.org/abs/2004.07802) called GAEA for geometry-aware neural architecture search (check out the paper for more details about the NAS algorithm).  GAEA is state-of-the-art among NAS method on the [DARTS search space](https://arxiv.org/abs/1806.09055).  You can replicate the results in the paper and try GAEA for your own data using the code provided here.

## Architecture Search
The training routine is based on that used by [PC-DARTS](https://github.com/yuhuixu1993/PC-DARTS/blob/master/train_imagenet.py).  We also added options for additional tricks that are often used to improve performance; in particular, we support the swish activation, squeeze and excite modules, RandAugment, and exponential moving average of the weights for eval. 

### Data
The code is written to perform architecture search on the [CIFAR-10](https://www.cs.toronto.edu/~kriz/cifar.html) dataset but is easily modified for use with other datasets, including ImageNet.

### To Run
 To run the example, simply run the following command from the `search` directory:
` det -m <master host:port> experiment create const.yaml .`

After the architecture search stage is complete, you can evaluate the architecture by copying the resulting genotype from the log to `eval/model_def.py`.  

## Architecture Evaluation
This code as is evaluates the best architecture found for ImageNet from our paper.  The training routine is based on that used by [PC-DARTS](https://github.com/yuhuixu1993/PC-DARTS/blob/master/train_imagenet.py).  We also added options for additional tricks that are often used to improve performance; in particular, we support the swish activation, squeeze and excite modules, RandAugment, and exponential moving average of the weights for eval.  These tricks are not used in the evaluation since the goal is to replicate the result but feel free to try them for your own training runs.  

### Data
The data for ImageNet is the ILSVRC2012 version of the dataset, which is available [here](http://www.image-net.org/).  The code assumes the data is available in a Google Cloud Storage bucket; in the event a bucket is not provided, randomly generated data is used.

### To Run
   To run the example, please adjust the `eval/distributed.yaml` configuration file to your liking.  In particular, you should at a minimum change the `bucket_name` field to reference your data bucket; you may also want to adjust the `slots_per_trial` parameter to increase the number of GPUs used for distributed training.  

   After configuring `eval/distributed.yaml` you can submit an experiment to your Determined cluster by running the following command from the `eval` directory:
     `det -m <master host:port> experiment create distributed.yaml . `

### Expected Performance
After 24 Epochs, the top 5 validation accuracy should be close to 80% (see learning curve below).  At convergence, the top 1 validation accuracy should be close to the 76% reported in the original paper.  
![](./eval/top5\_val.png)
