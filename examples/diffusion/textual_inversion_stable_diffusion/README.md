# Textual Inversion with Stable Diffusion

This example demonstrates how to incorporate your own images into AI-generated art via
[Textual Inversion](https://textual-inversion.github.io).

The development of [Latent Diffusive Models](https://arxiv.org/abs/2112.10752) has made
it possible to run (and fine-tune) diffusion-based models on consumer-grade GPUs. Such tasks are
made even easier by the release
of [Stable Diffusion](https://stability.ai/blog/stable-diffusion-announcement) and the
development of the 🤗 [Huggingface 🧨 Diffusers](https://huggingface.co/docs/diffusers/index) library.

The present code uses Determined's Core API to seamlessly incorporate 🧨 Diffusers
and 🚀 [Accelerate](https://huggingface.co/docs/transformers/accelerate) into the
Determined framework.

## Before You Start: 🤗 Account, Access Token, and License

In order to use this repository's implementation of Stable Diffusion, you must:

* Have a [Huggingface account](https://huggingface.co/join).
* Have a [Huggingface User Access Token](https://huggingface.co/docs/hub/security-tokens).
* Accept the Stable Diffusion license (click on _Access
  repository_  [in this link)](https://huggingface.co/CompVis/stable-diffusion-v1-4).

## Walkthrough: Basic Usage

Below we walk through the Textual Inversion workflow, first fine-tuning Stable Diffusion on a set of
user-provided training images featuring a new concept, and then incorporating representations of the
concept into generated art.

### Fine-Tuning

After including your user access token in the `finetune_finetune_const.yaml` config file by
replacing `YOUR_HF_AUTH_TOKEN_HERE` where it reads

```yaml
environment:
  environment_variables:
    - HF_AUTH_TOKEN=YOUR_HF_AUTH_TOKEN_HERE
```

a ready-to-go fine-tuning experiment can be run by executing the following in the present directory:

```bash
det -m MASTER_URL_WITH_PORT e create finetune_const.yaml .
```

Above, `MASTER_URL_WITH_PORT` should be replaced with the appropriate url for your Determined
cluster.

This will submit an experiment which introduces a new embedding vector into the world of Stable
Diffusion which we will fine-tune to correspond to the concept of the Determined AI logo, as
represented
through training images found in `det_logos/`, such as the example found below (placed on a
background for
improved results):

![det-logo](./det_logos/on_a_dark_blue_oil_painting_of_ocean_waves.jpg)

A corresponding concept token, chosen to be `<det-logo>` as specified in the `concept_tokens` field
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
det -m MASTER_URL_WITH_PORT notebook start --config-file detsd-notebook.yaml --context .
```

replacing `MASTER_URL_WITH_PORT` as before. A new notebook window will be launched in which
`textual_inversion.ipynb` can be opened and run.

New concepts can be loaded into the notebook by specifying the `uuid`s of their
corresponding Determined checkpoints in the relevant `uuids` list under the _Load Determined
Checkpoints_ section. Then simply run the notebook from top to bottom. Further instructions may be
found in the notebook itself.

### Cluster Generation

Notebooks are excellent for quick experimentation with prompt-selections and tuning of parameters,
but are limited in their capacity
to generate images at scale. Parallelized generation using arbitrary resources can be accomplished
by submitting
an experiment with the `generate_grid.yaml` config file, as in

```bash
det -m MASTER_URL_WITH_PORT e create generate_grid.yaml .
```

where one must again modify the `HF_AUTH_TOKEN=YOUR_HF_AUTH_TOKEN_HERE` line in `generate_grid.yaml`
as previously.

The provided configuration will perform multi-GPU generation, scanning across
multiple
prompts and settings of the `guidance_scale` parameter, logging all images to Tensorboard for easy
retrieval and
organization.

#### Typical Results

***To be added!***

## Customization

The basic `finetune_const.yaml` config can be easily customized to accommodate your own concepts.

The relevant parts of the `hyperparameters` section read:

```yaml
hyperparameters:
#...
concepts:
  learnable_properties: # One of 'object' or 'style' 
    - object
  concept_tokens: # Special tokens representing new concepts. Must not exist in tokenizer.  
    - <det-logo>
  initializer_tokens: # Phrases which are closely related to added concepts.
    - orange brain logo, connected circles, concept art
  img_dirs:
    - det_logos
#...
inference:
  inference_prompts:
    - a photo of a <det-logo>
```

To fine-tune on a new concept:

1) Add your training images in a new directory and list it under `img_dirs`.
2) Set `learnable_properties` to `object` or `style`, according to which facet of the images you
   wish
   to capture.
3) Choose an entry for `concept_tokens`, which is the stand-in for your object in prompts,
   replacing `<det-logo>` above.
4) Choose the `initializer_tokens`, which should be a short, descriptive phrase closely related to
   your images.
5) All prompts included in `inference_prompts` will be periodically generated by the model and saved
   to the checkpoint directory.

If you wish to fine-tune on multiple concepts a once, simply add the
relevant entries under the
`img_dirs`, `learnable_properties`, `concept_tokens`, and `initializer_tokens` fields,
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
* Optionally add regularization to training by setting `hidden_reg_weight` to a non-zero value. When
  used, this penalizes the model for moving the encoder hidden states far from their initialization
  point.

## Some Tips

Generating results of the desired quality is often a balancing act:

* Training images are resized to 512 x 512 pixels by default. Resize your images accordingly will
  lead to
  the most consistent results.
* The provided config files do not use many SGD steps and are intended for quick demonstration.
  Increase the `max_length` field, and adjust other hyperparameters, for more finely tuned
  results.
* There is generally a tradeoff between how faithfully the training images are reproduced and how
  well they can be incorporated into the desired scene. Over-training may lead to perfect-likeness,
  while also overwhelming all other elements in your chosen prompt.
* Prompts are very sensitive to word order. Stable Diffusion pays much more attention to words at
  the
  beginning of a prompt than it does to words at the end and longer prompts often give better
  results. See the _Prompt Development_  section of
  this [Reddit guide](https://www.reddit.com/r/StableDiffusion/comments/xcq819/dreamers_guide_to_getting_started_w_stable/)
  for more detailed tips on prompt-engineering.
* When generating using new concepts trained with Textual Inversion, it often seems helpful to set
  the `guidance_scale` to a lower value than one would normally do, e.g., setting it as low
  as `guidance_scale = 1.1` for a very highly-trained concept (the prompt is ignored entirely
  when `guidance_scale <= 1.`).

## The Code

The code for this example based on a mix of
Huggingface's [own implementation](https://github.com/huggingface/diffusers/tree/main/examples/textual_inversion)
of Textual Inversion, ideas
drawn from the original [Textual Inversion](https://github.com/rinongal/textual_inversion) repo, and
from the #community-research channel on the
official [Stable Diffusion Discord Server](https://www.diffusion.gg).

# TODO

A very incomplete list:

* Write a launch script which uses the `accelerate` launcher?
* fp16 training
* lr scheduler
* Link to blog post, when published.
