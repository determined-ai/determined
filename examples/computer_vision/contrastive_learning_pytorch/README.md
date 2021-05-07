# Contrastive Learning
This example is adapted from https://github.com/HobbitLong/SupContrast to support contrastive loss for multi-node distributed training.  

The contrastive learning methods implemented include [SIMCLR](https://arxiv.org/pdf/2002.05709.pdf) and [supervised contrastive learning](https://arxiv.org/pdf/2004.11362.pdf).  The example runs with CIFAR-10 and CIFAR-100 but should be easily adaptable to custom datasets.

## Files
* [**losses.py**](losses.py): distributed implementation of contrastive loss for supervised and unsupervised learning.
* [**resnet_big.py**](resnet_big.py): architecture definitions (duplicated for original repo).
* [**train_eval_model_def.py**](train_eval_model_def.py): implements our PyTorchTrial interface for this example with a nested training loop to periodically train the classifier head over a dataloader of choice.
* [**model_def.py**](train_eval_model_def.py): alternate implementation of our PyTorchTrial interface that interleaves training the embedding and classifier heads in each train step.  

## Configurations
* [**supclr_train_eval.yaml**](supclr_train_eval.yaml): Train using supervised contrastive loss on CIFAR-10 using the trial definition in [**train_eval_model_def.py**](train_eval_model_def.py).
* [**supclr.yaml**](supclr.yaml): Train using supervised contrastive loss on CIFAR-10 using the trial definition in [**model_def.py**](model_def.py).
* [**simclr_train_eval.yaml**](simclr_train_eval.yaml): Train using unsupervised contrastive loss on CIFAR-10 using the trial definition in [**train_eval_model_def.py**](train_eval_model_def.py).
* [**simclr.yaml**](simclr.yaml): Train using unsupervised contrastive loss on CIFAR-10 using the trial definition in [**model_def.py**](model_def.py).

## How to Run
Installation instructions can be found under `docs/install-admin.html` or at [Determined installation page](https://docs.determined.ai/latest/index.html).

Once you have a Determined cluster installed, you can submit the experiment to the cluster by running the following command:

`det -m <master host:port> experiment create -f <experiment_config> .`

### Results
Running with [**supclr_train_eval.yaml**](supclr_train_eval.yaml) should yield an accuracy close to 96 on CIFAR-10.

Running with [**simclr_train_eval.yaml**](simclr_train_eval.yaml) should yield an accuracy close to 93 on CIFAR-10.

The experiment configurations **supclr.yaml** and **simclr.yaml** slightly underperforms the corresponding results above.  
