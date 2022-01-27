# Bootstrap Your Own Latent

A Determined implementation of BYOL, based on `byol-pytorch`: https://github.com/lucidrains/byol-pytorch/tree/master/byol_pytorch

Original paper: https://arxiv.org/abs/2006.0

Some other BYOL implementations:
  - JAX, original authors: https://github.com/deepmind/deepmind-research/tree/master/byol
  - Pytorch: https://github.com/untitled-ai/self_supervised

This repo contains training configurations for the following datasets:
- ImageNet, with hyperparameters from the BYOL paper.
- STL-10, with hyperparameters inspired by https://generallyintelligent.ai/blog/2020-08-24-understanding-self-supervised-contrastive-learning/
- CIFAR-10, with hyperparameters derived from the BYOL paper + adjustments for image size.