"""Following the SD textual inversion notebook example from HF
https://github.com/huggingface/notebooks/blob/main/diffusers/sd_textual_inversion_training.ipynb"""
import os

import attrdict
import determined as det
from diffusers import (
    AutoencoderKL,
    DDPMScheduler,
    PNDMScheduler,
    StableDiffusionPipeline,
    UNet2DConditionModel,
)
from transformers import CLIPFeatureExtractor, CLIPTextModel, CLIPTokenizer
from torch.utils.data import DataLoader


import data


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
        # Initalize the placeholder_token vector to coincide with teh initializer_token vector
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

        def create_dataloader(train_batch_size=1):
            return DataLoader(train_dataset, batch_size=train_batch_size, shuffle=True)

        noise_scheduler = DDPMScheduler(
            beta_start=0.00085,
            beta_end=0.012,
            beta_schedule="scaled_linear",
            num_train_timesteps=1000,
            tensor_format="pt",
        )

        hyperparameters = {
            "learning_rate": 5e-04,
            "scale_lr": True,
            "max_train_steps": 3000,
            "train_batch_size": 1,
            "gradient_accumulation_steps": 4,
            "seed": 42,
            "output_dir": "sd-concept-output",
        }
