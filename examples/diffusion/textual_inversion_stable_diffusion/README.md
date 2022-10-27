# Textual Inversion with Stable Diffusion

This example demonstrates how to incorporate your own images into AI-generated art via
[Textual Inversion](https://textual-inversion.github.io). See the
accompanying [blog post here.](https://www.determined.ai/blog/stable-diffusion-core-api)

The development of [Latent Diffusive Models](https://arxiv.org/abs/2112.10752) has made
it possible to run (and fine-tune) diffusion-based models on consumer-grade GPUs. Such tasks are
made even easier by the release
of [Stable Diffusion](https://stability.ai/blog/stable-diffusion-announcement) (SD) and the
development of the 🤗 [Huggingface 🧨 Diffusers](https://huggingface.co/docs/diffusers/index) library.

The present code uses Determined's Core API to seamlessly incorporate 🧨 Diffusers
and 🚀 [Accelerate](https://huggingface.co/docs/transformers/accelerate) into the
Determined framework.

## Before You Start: 🤗 Account, Access Token, and License

In order to use this repository's implementation of SD, you must:

* Have a [Huggingface account](https://huggingface.co/join).
* Have a [Huggingface User Access Token](https://huggingface.co/docs/hub/security-tokens).
* Accept the SD license (click on _Access
  repository_  [in this link)](https://huggingface.co/CompVis/stable-diffusion-v1-4).

## Walkthrough: Basic Usage

Below we walk through the Textual Inversion workflow, which consists of two stages:

1. Fine-tune SD on a set of
   user-provided training images featuring a new concept
2. Incorporate representations of the
   concept into generated art.

### Fine-Tuning

After including your user access token in the `finetune_const.yaml` config file by
replacing `YOUR_HF_AUTH_TOKEN_HERE` where it reads

```yaml
environment:
  environment_variables:
    - HF_AUTH_TOKEN=YOUR_HF_AUTH_TOKEN_HERE
```

a ready-to-go fine-tuning experiment can be run by executing the following in the present directory:

```bash
det experiment create finetune_const.yaml .
```

This will submit an experiment which introduces a new embedding vector into the world of Stable
Diffusion which we will fine-tune to correspond to the concept of the Determined AI logo, as
represented
through training images found in `det_logos/`, such as the example found below (placed on a
background for
improved results):

![det-logo](./det_logos/on_a_dark_blue_oil_painting_of_ocean_waves.jpg)

A corresponding concept token, chosen to be `det-logo` as specified in the `concept_strs` field
in the config, will then be available for use in our prompts to signify the concept of this logo.

By default, sample images are generated during training which can be viewed by launching a
Tensorboard instance from the experiment in the WebUI.

### Notebook Generation

When fine-tuning is complete, interactive inference can be run by using the included
`textual_inversion.ipynb` on the same Master which performed the Experiment.

In order to launch the
notebook with the requisite files included in its context, first modify
the `HF_AUTH_TOKEN=YOUR_HF_AUTH_TOKEN_HERE` line in the `detsd-notebook.yaml` config file,
analogously to the above, and then run the following command in the root of
this repo:

```bash
det notebook start --config-file detsd-notebook.yaml -i detsd -i startup-hook.sh -i learned_embeddings_dict_demo.pt -i textual_inversion.ipynb
```

where the `-i` flags are short for `--include` and they ensure that the following files will be
included the directory where the Jupyter notebook will be launched:

* `detsd-notebook.yaml`: the notebook config file.
* `detsd`: the python module directory containing the `DetSDTextualInversionPipeline` we will use to
  generate images.
* `learned_embeddings_dict_demo.pt`: pickle file containing a `det-logo-demo` concept, highly
  trained on Determined AI logo images.
* `startup-hook.sh`: a startup script which installs necessary dependencies.
* `textual_inversion.ipynb`: the to-be-launched notebook.

Then, simply run the `textual_inversion.ipynb` notebook from top to bottom. Further instructions may
be
found in the notebook itself.

### Cluster Generation

Notebooks are excellent for quick experimentation with prompt-selections and tuning of parameters,
but are limited in their capacity
to generate images at scale. Parallelized generation using arbitrary resources can be accomplished
by submitting
an experiment with the `generate_grid.yaml` config file, as in

```bash
det experiment create generate_grid.yaml .
```

where one must again modify the `HF_AUTH_TOKEN=YOUR_HF_AUTH_TOKEN_HERE` line in `generate_grid.yaml`
as previously.

`generate_grid.yaml` can either load in trained checkpoints by `uuid` or from local files. The
provided config will load in the highly-trained demo `det-logo-demo` concept saved
in `learned_embeddings_dict_demo.pt` by
default. The configuration performs multi-GPU generation, scanning across
multiple prompts and settings of the `guidance_scale` parameter, logging all images to Tensorboard
for easy
retrieval and organization.

#### Sample Results

Images generated by tuning on the logos in `det_logos/`, chosen for their
diversity.

![det-logo](./readme_imgs/1.png)
![det-logo](./readme_imgs/2.png)
![det-logo](./readme_imgs/3.png)
![det-logo](./readme_imgs/4.png)

## Customization

The basic `finetune_const.yaml` config can be easily customized to accommodate your own concepts.

The relevant parts of the `hyperparameters` section read:

```yaml
hyperparameters:
#...
concepts:
  learnable_properties: # One of 'object' or 'style' 
    - object
  concept_strs: # Individual strings representing new concepts. Must not exist in tokenizer.  
    - det-logo
  initializer_strs: # Strings which describe the added concepts. 
    - brain logo, sharp lines, connected circles, concept art
  img_dirs:
    - det_logos
#...
inference:
  inference_prompts:
    - a watercolor painting on textured paper of a det-logo using soft strokes, pastel colors, incredible composition, masterpiece
```

To fine-tune on a new concept:

1) Add your training images in a new directory and list it under `img_dirs`.
2) Set `learnable_properties` to `object` or `style`, according to which facet of the images you
   wish
   to capture.
3) Choose an entry for `concept_strs`, which is the stand-in for your object in prompts,
   replacing `det-logo` above.
4) Choose the `initializer_strs`, which should be a short, descriptive phrase closely related to
   your images.
5) All prompts included in `inference_prompts` will be periodically generated by the model and saved
   to the checkpoint directory.

If you wish to fine-tune on multiple concepts a once, simply add the
relevant entries under the
`img_dirs`, `learnable_properties`, `concept_strs`, and `initializer_strs` fields,
keeping the same relative ordering across each.

#### Advanced Options

More advanced customizations can be made by modifying the `finetune_const_advanced.yaml` config
file. Examples:

* Images in `det_logos/` use `_`-separated, descriptive filenames, which can be
  appended to the
  training prompts, as controlled by the `append_file_name_to_text` and `file_name_split_char`
  fields
  in
  the `concepts` section of this advanced config. Use similar
  filenames and config settings with your own images for a boost in results.
* There are two forms of regularization which can be used in the `DetSDTextualInversionTrainer` class through its `__init__` args:
  * `norm_reg_weight`: Penalize the tunable concept embeddings for becoming much larger or smaller than the average SD embedding vector.
  * `hidden_reg_weight`: Penalize the `encoder_hidden_states` generated by the `text_encoder` from too far from their initialization values.

## Some Tips

* Training images are resized to 512 x 512 pixels by default. Resizing your images accordingly will
  lead to the most consistent results. 
* There is generally a tradeoff between how faithfully the training images are reproduced and how
  well they can be incorporated into the desired scene. Over-training may lead to perfect-likeness,
  while also overwhelming all other elements in your chosen prompt.
* Prompts are very sensitive to word order. SD pays much more attention to words at
  the
  beginning of a prompt than it does to words at the end and longer prompts often give better
  results. See the _Prompt Development_  section of
  this [Reddit guide](https://www.reddit.com/r/StableDiffusion/comments/xcq819/dreamers_guide_to_getting_started_w_stable/)
  for more detailed tips on prompt-engineering.
* When generating using new concepts trained with Textual Inversion, it often seems helpful to set
  the `guidance_scale` to a lower value than one would normally do, e.g., even setting it as low
  as `guidance_scale = 1.1` for a very highly-trained concept (the prompt is ignored entirely
  when `guidance_scale <= 1.`).
* Scanning across multiple seeds leads to more diverse results. A single fixed seed can lead to
  similar results across different prompts, for instance.
* For increased start-up efficiency, save the vanilla SD weights with
  the `save_pretrained`
  [method](https://huggingface.co/docs/transformers/main/en/main_classes/model#transformers.PreTrainedModel.save_pretrained)
  on a Hugging
  Face [StableDiffusionPipeline](https://huggingface.co/docs/diffusers/api/pipelines/stable_diffusion#diffusers.StableDiffusionPipeline)
  instance and point the `pretrained_model_name_or_path` `__init__` arg to the relevant path, rather
  than downloading the weights anew for each Experiment.

## The Code

The code for this example based on a mix of
Huggingface's [own implementation](https://github.com/huggingface/diffusers/tree/main/examples/textual_inversion)
of Textual Inversion, ideas
drawn from the original [Textual Inversion](https://github.com/rinongal/textual_inversion) repo, and
from the #community-research channel on the
official [Stable Diffusion Discord Server](https://www.diffusion.gg). The code assumes GPU resources
are available.

Summary of important files and directories:

#### Configuration Files

- `finetune_const.yaml`: Basic config file for a fine-tuning Experiment.
- `finetune_const_advanced.yaml`: Advanced config file for a fine-tuning Experiment.
- `generate_grid.yaml`: Config file for a generation Experiment.
- `detsd-notebook.yaml`: Config file for launching the `textual_inversion.ipynb` notebook.

#### Other Files and Directories

- `detsd`: Module containing all code for fine-tuning and generation.
    - `detsd/trainer.py`: Contains the `DetSDTextualInversionTrainer` class used for fine-tuning
      on-cluster.
    - `detsd/pipeline.py`: Contains the `DetSDTextualInversionPipeline` class used for generation in
      both notebooks and on-cluster.
- `textual_inversion.ipynb`: Notebook for interactive generation.
- `learned_embeddings_dict_demo.pt`: Pre-trained demo concept to be used with the above notebook
  or a `generate_grid.yaml` Experiment.
- `startup-hook.sh`: Script for installing necessary dependencies for cluster and notebook
  workflows.

