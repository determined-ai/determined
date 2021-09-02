# Using model-hub mmdetection with [Hydra](https://hydra.cc/)
Hydra is a framework for configuring applications that works very well with machine learning experiments.  
You can use Determined's python API with Hydra to:
* Easily submit experiments with different configurations
* Perform parameter sweeps
* Compose configurations

## Setup
You need to install Determined and Hydra in order to try this out.
```
pip install hydra-core>=1.1
pip install determined
```

## Submitting experiments
Make sure the `DET_MASTER` environment variable is set.  Then you can create experiments by running
```
python mmdet_experiment.py hyperparameters.config_file=mask_rcnn/mask_rcnn_r50_fpn_1x_coco.py
```

Hydra makes it easy to modify the configuration from the CLI:
```
python mmdet_experiment.py hyperparameters.config_file=faster_rcnn/faster_rcnn_r50_fpn_1x_coco.py
```

Or try multiple values:
```
python mmdet_experiment.py --multirun \
    hyperparameters.config_file=faster_rcnn/faster_rcnn_r50_fpn_1x_coco.py,detr/detr_r50_8x2_150e_coco.py
```

Configuration with Hydra is also highly flexible and extensible.  
For example, you can run hyperparameter search on the optimizer learning rate by
```
python mmdet_experiment.py searcher=adaptive +hyperparameters=tune_optimizer hyperparameters.config_file=mask_rcnn/mask_rcnn_r50_fpn_1x_coco.py
```
You can look the [config directory](configs) to see how we use some of this functionality.  Feel free to add your own configs as needed to further customize the behavior.



