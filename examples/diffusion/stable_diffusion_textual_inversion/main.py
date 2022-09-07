"""Following the SD textual inversion notebook example from HF
https://github.com/huggingface/notebooks/blob/main/diffusers/sd_textual_inversion_training.ipynb"""
import os

import attrdict
import determined as det
import logging
import math
import os

import torch
import torch.nn.functional as F
import torch.utils.checkpoint
from accelerate import Accelerator
from accelerate.logging import get_logger
from diffusers import (
    AutoencoderKL,
    DDPMScheduler,
    PNDMScheduler,
    StableDiffusionPipeline,
    UNet2DConditionModel,
)
from diffusers.pipelines.stable_diffusion import StableDiffusionSafetyChecker
from torch import autocast
from torch.utils.data import DataLoader
from tqdm.auto import tqdm
from transformers import CLIPFeatureExtractor, CLIPTextModel, CLIPTokenizer

import data
from train import train

logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)


if __name__ == "__main__":
    HF_AUTH_TOKEN = os.environ["HF_AUTH_TOKEN"]
    info = det.get_cluster_info()
    hparams = attrdict.AttrDict(info.trial.hparams)
    distributed = det.core.DistributedContext.from_torch_distributed()

    with det.core.init(distributed=distributed) as core_context:
        rank = core_context.distributed.rank
        local_rank = core_context.distributed.local_rank
        size = core_context.distributed.size
        is_distributed = size > 1
        is_chief = rank == 0
        is_local_chief = local_rank == 0

        # Build the tokenizer and add the new token
        tokenizer = CLIPTokenizer.from_pretrained(
            hparams.pretrained_model_name_or_path,
            subfolder="tokenizer",
            use_auth_token=HF_AUTH_TOKEN,
        )

        num_added_tokens = tokenizer.add_tokens(hparams.placeholder_token)
        if num_added_tokens == 0:
            raise ValueError(
                f"The tokenizer already contains the token {hparams.placeholder_token}. "
                "Please pass: a different `placeholder_token` that is not already in the tokenizer."
            )

        # Convert the initializer_token, placeholder_token to ids
        token_ids = tokenizer.encode(hparams.initializer_token, add_special_tokens=False)
        # Check if initializer_token is a single token or a sequence of tokens
        if len(token_ids) > 1:
            raise ValueError("The initializer token must be a single token.")

        initializer_token_id = token_ids[0]
        placeholder_token_id = tokenizer.convert_tokens_to_ids(hparams.placeholder_token)

        # Load models and create wrapper for stable diffusion
        text_encoder = CLIPTextModel.from_pretrained(
            hparams.pretrained_model_name_or_path,
            subfolder="text_encoder",
            use_auth_token=HF_AUTH_TOKEN,
        )
        vae = AutoencoderKL.from_pretrained(
            hparams.pretrained_model_name_or_path, subfolder="vae", use_auth_token=HF_AUTH_TOKEN
        )
        unet = UNet2DConditionModel.from_pretrained(
            hparams.pretrained_model_name_or_path, subfolder="unet", use_auth_token=HF_AUTH_TOKEN
        )

        # Extend the size of the text_encoder to account for the new placeholder_token
        text_encoder.resize_token_embeddings(len(tokenizer))
        # Initalize the placeholder_token vector to coincide with the initializer_token vector
        token_embeds = text_encoder.get_input_embeddings().weight.data
        token_embeds[placeholder_token_id] = token_embeds[initializer_token_id]

        # Freeze everything apart from the newly added embedding vector
        def freeze_params(params):
            for param in params:
                param.requires_grad = False

        # Freeze vae and unet
        freeze_params(vae.parameters())
        freeze_params(unet.parameters())
        # Freeze all parameters except for the token embeddings in text encoder
        for p in (
            text_encoder.text_model.encoder.parameters(),
            text_encoder.text_model.final_layer_norm.parameters(),
            text_encoder.text_model.embeddings.position_embedding.parameters(),
        ):
            freeze_params(p)

        # Create the training dataset
        train_dataset = data.TextualInversionDataset(
            data_root=hparams.save_path,
            tokenizer=tokenizer,
            size=hparams.size,
            placeholder_token=hparams.placeholder_token,
            repeats=hparams.repeats,
            learnable_property=hparams.what_to_teach,
            center_crop=hparams.center_crop,
            split="train",
        )

        print(80 * "=", "TRAINING", 80 * "=", sep="\n")

        train(
            train_dataset=train_dataset,
            placeholder_token=hparams.placeholder_token,
            placeholder_token_id=placeholder_token_id,
            text_encoder=text_encoder,
            tokenizer=tokenizer,
            vae=vae,
            unet=unet,
            train_batch_size=hparams.train_batch_size,
            gradient_accumulation_steps=hparams.gradient_accumulation_steps,
            learning_rate=hparams.learning_rate,
            max_train_steps=hparams.max_train_steps,  # Original 3000, edited for speed
            output_dir=hparams.output_dir,
            scale_lr=True,
            core_context=core_context,
        )

        print(80 * "=", "INFERENCE", 80 * "=", sep="\n")
        #
        # pipe = StableDiffusionPipeline.from_pretrained(
        #     hparams.output_dir, torch_dtype=torch.float16
        # ).to("cuda")
        #
        # with autocast("cuda"):
        #     images = pipe(
        #         list(hparams.prompts),
        #         num_inference_steps=hparams.num_inference_steps,
        #         guidance_scale=hparams.guidance_scale,
        #     )["sample"]
        #     print(images)
