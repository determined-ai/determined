# Vision Transformer with HuggingFace Trainer API and Determined
This example is adapted from the 
[VisionTransformer example in the Hugging Face examples](https://github.com/huggingface/transformers/tree/main/examples/pytorch/image-classification). 
It is intended to demonstrate how to use Determined callback with Hugging Face Trainer API to
enable Determined's distributed training, fault tolerance, checkpointing and metrics reporting.

## Files
* **image_classification.py**: The code from Hugging Face that (1) loads Vision Transformer from Model Hub; (2)
configure Trainer; (3) uses Determined callback.

The key portion of the code is providing Determined callback to the Trainer in line 413:
```
    det_callback = DetCallback(training_args, filter_metrics=["loss", "accuracy"], tokenizer=feature_extractor)
    trainer.add_callback(det_callback)
```

### Configuration Files
* **const.yaml**: Train the model with constant hyperparameter values.
* **distributed.yaml**: Same as `const.yaml`, but trains the model with multiple GPUs (distributed training).
* **adaptive.yaml**: Perform a hyperparameter search using Determined's state-of-the-art adaptive hyperparameter tuning algorithm.
* **deepspeed.yaml**: Train the model with DeepSpeed with constant hyperparameter values. Feel free to modify this 
file to enable adaptive hyperparameter tuning algorithm.

Deepspeed configurations files are located in `ds_configs` and include:
* **ds_config_stage_1.json**: Optimizer state partitioning (ZeRO stage 1).
* **ds_config_stage_2.json**: Gradient partitioning (ZeRO stage 2).
* **ds_config_stage_2_cpu_offload.json**: Gradient partitioning and CPU offloading (ZeRO stage 2).
* **ds_config_stage_3.json**: Parameter partitioning (ZeRO stage 3).

To learn more about DeepSpeed, see [DeepSpeed docs](https://deepspeed.readthedocs.io/en/latest/) and 
[HF DeepSpeed integration](https://huggingface.co/docs/transformers/main_classes/deepspeed).

## Data
This example uses [beans dataset](https://huggingface.co/datasets/beans).

## To Run
If you have not yet installed Determined, installation instructions can be found
under `docs/install-admin.html` or at https://docs.determined.ai/latest/index.html


* To train Vision Transformer with Trainer API and Determined run the following command: 
```
det experiment create const.yaml .
``` 
The other configuration can be run by specifying the appropriate configuration file in place 
of `const.yaml`.


* To train Vision Transformer with Trainer API, DeepSpeed and Determined run the following command:
```
det experiment create deepspeed.yaml .
```
To select DeepSpeed optimization, modify line 35 in `deepspeed.yaml` to point to your preferred DeepSpeed configuration
file. By default, `deepspeed.yaml` is using `ds_configs/ds_config_stage_1.json`.

## Results
Training the model with the hyperparameter settings in `const.yaml` should yield
a validation accuracy of ~96% after 3 epochs.
