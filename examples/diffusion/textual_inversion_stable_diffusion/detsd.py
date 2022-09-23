import json
import os
import pathlib
from contextlib import nullcontext
from datetime import datetime
from PIL import Image
from typing import Any, Dict, List, Literal, Optional, Sequence, Tuple, Union


import accelerate
import attrdict
import determined as det
import torch
import torch.nn as nn
import torch.nn.functional as F
from determined.pytorch import TorchData
from determined.experimental import client
from diffusers import (
    AutoencoderKL,
    DDPMScheduler,
    DDIMScheduler,
    LMSDiscreteScheduler,
    PNDMScheduler,
    StableDiffusionPipeline,
    UNet2DConditionModel,
)
from diffusers.pipelines.stable_diffusion import StableDiffusionSafetyChecker
from torch.utils.data import DataLoader
from torch.utils.tensorboard import SummaryWriter
from transformers import CLIPFeatureExtractor, CLIPTextModel, CLIPTokenizer

import data

# TODO: additional default kwargs for ddim and lms-discrete
NOISE_SCHEDULER_DICT = {
    "ddim": DDIMScheduler,
    "lms-discrete": LMSDiscreteScheduler,
    "pndm": PNDMScheduler,
}
DEFAULT_SCHEDULER_KWARGS_DICT = {
    "pndm": {"skip_prk_steps": True},
    "ddim": {},
    "lms-discrete": {},
}


class DetSDTextualInversionTrainer:
    """Class for training a textual inversion model on a Determined cluster."""

    def __init__(
        self,
        train_img_dirs: Union[str, Sequence[str]],
        concept_tokens: Union[str, Sequence[str]],
        initializer_tokens: Union[str, Sequence[str]],
        learnable_properties: Sequence[Literal["object", "style"]],
        pretrained_model_name_or_path: str = "CompVis/stable-diffusion-v1-4",
        train_batch_size: int = 1,
        gradient_accumulation_steps: int = 4,
        optimizer_name: Literal["adam", "adamw", "sgd"] = "adam",
        learning_rate: float = 5e-04,
        other_optimizer_kwargs: Optional[dict] = None,
        scale_lr: bool = True,
        norm_penalty: float = 1.0,
        checkpoint_freq: int = 100,
        metric_report_freq: int = 100,
        beta_start: float = 0.00085,
        beta_end: float = 0.012,
        beta_schedule: Literal["linear", "scaled_linear", "squaredcos_cap_v2"] = "scaled_linear",
        num_train_timesteps: int = 1000,
        train_seed: int = 2147483647,
        img_size: int = 512,
        interpolation: Literal["nearest", "bilinear", "bicubic"] = "bicubic",
        flip_p: float = 0.0,
        center_crop: bool = False,
        generate_training_images: bool = True,
        inference_prompts: Optional[Union[str, Sequence[str]]] = None,
        inference_scheduler_name: Literal["ddim", "lms-discrete", "pndm"] = "ddim",
        num_inference_steps: int = 50,
        guidance_scale: float = 7.5,
        generator_seed: int = 2147483647,
        other_inference_scheduler_kwargs: Optional[dict] = None,
        latest_checkpoint: Optional[str] = None,
    ) -> None:
        # We assume that the Huggingface User Access token has been stored as a HF_AUTH_TOKEN
        # environment variable. See https://huggingface.co/docs/hub/security-tokens
        try:
            self.use_auth_token = os.environ["HF_AUTH_TOKEN"]
        except KeyError:
            raise KeyError(
                "Please set your HF User Access token as the HF_AUTH_TOKEN environment variable."
            )
        self.logger = accelerate.logging.get_logger(__name__)

        self.latest_checkpoint = latest_checkpoint
        self.pretrained_model_name_or_path = pretrained_model_name_or_path

        if isinstance(learnable_properties, str):
            learnable_properties = [learnable_properties]
        self.learnable_properties = learnable_properties
        if isinstance(concept_tokens, str):
            concept_tokens = [concept_tokens]
        self.concept_tokens = concept_tokens
        if isinstance(initializer_tokens, str):
            initializer_tokens = [initializer_tokens]
        self.initializer_tokens = initializer_tokens
        self.img_size = img_size
        self.interpolation = interpolation
        self.flip_p = flip_p
        self.center_crop = center_crop

        self.train_batch_size = train_batch_size
        self.gradient_accumulation_steps = gradient_accumulation_steps
        self._optim_dict = {
            "adam": torch.optim.Adam,
            "adamw": torch.optim.AdamW,
            "sgd": torch.optim.SGD,
        }
        if optimizer_name not in self._optim_dict:
            raise ValueError(f"Optimizer must be one of {list(self._optim_dict.keys())}.")
        self.optimizer_name = optimizer_name
        self.learning_rate = learning_rate
        self.other_optimizer_kwargs = other_optimizer_kwargs or {}
        self.scale_lr = scale_lr
        self.norm_penalty = norm_penalty
        self.checkpoint_freq = checkpoint_freq
        self.metric_report_freq = metric_report_freq
        self.beta_start = beta_start
        self.beta_end = beta_end
        self.beta_schedule = beta_schedule
        self.num_train_timesteps = num_train_timesteps
        if isinstance(train_img_dirs, str):
            train_img_dirs = [train_img_dirs]
        self.train_img_dirs = train_img_dirs
        self.train_seed = train_seed

        self.accelerator = accelerate.Accelerator(
            gradient_accumulation_steps=self.gradient_accumulation_steps,
        )
        accelerate.utils.set_seed(self.train_seed)

        assert inference_scheduler_name in NOISE_SCHEDULER_DICT, (
            f"inference_scheduler must be one {list(NOISE_SCHEDULER_DICT.keys())},"
            f" but got {inference_scheduler_name}"
        )
        if not generate_training_images and inference_prompts is not None:
            self.logger.warning(
                "Inference prompts were provided, but are being skipped, as generate_training"
                " images was set to False"
            )
        self.generate_training_images = generate_training_images
        if isinstance(inference_prompts, str):
            inference_prompts = [inference_prompts]
        self.inference_scheduler_name = inference_scheduler_name
        self.inference_prompts = inference_prompts
        self.num_inference_steps = num_inference_steps
        self.guidance_scale = guidance_scale
        self.generator_seed = generator_seed

        if other_inference_scheduler_kwargs is None:
            other_inference_scheduler_kwargs = DEFAULT_SCHEDULER_KWARGS_DICT[
                self.inference_scheduler_name
            ]
        self.other_inference_scheduler_kwargs = other_inference_scheduler_kwargs

        self.steps_completed = 0
        self.loss_history = []
        self.last_mean_loss = None

        self.effective_global_batch_size = (
            self.gradient_accumulation_steps
            * self.train_batch_size
            * self.accelerator.num_processes
        )
        # If scale_lr, we linearly scale the bare learning rate by the effective batch size
        if scale_lr:
            self.learning_rate *= self.effective_global_batch_size
            self.logger.info(f"Using scaled learning rate {self.learning_rate}")

        # The below are instantiated as needed through private methods.
        self.tokenizer = None
        self.text_encoder = None
        self.vae = None
        self.unet = None
        self.safety_checker = None
        self.feature_extractor = None
        self.train_dataset = None
        self.train_dataloader = None
        self.optimizer = None
        self.train_scheduler = None
        self.original_embedding_idxs = None
        self.original_embedding_tensors = None
        self.original_embedding_mean_norm = None
        self.new_embedding_idxs = None

        self.concept_to_initializer_tokens_map = {}
        self.concept_to_dummy_tokens_map = {}
        self.concept_to_dummy_ids_map = {}

        self._build_models()
        self._add_new_tokens()
        with torch.no_grad():
            init_norm = torch.linalg.vector_norm(self._get_new_token_embeddings(), dim=1).mean()
            print("NEW NORMS AT INIT", init_norm)
        self._freeze_layers()
        self._build_dataset_and_dataloader()
        self._build_optimizer()
        self._build_train_scheduler()
        self._wrap_and_prepare()

        # Pipeline construction is deferred until the _save call, as it may not be required at all.
        self.inference_scheduler_kwargs = None
        self.pipeline = None

    @classmethod
    def init_on_cluster(cls) -> "DetSDTextualInversionTrainer":
        """Creates a DetStableDiffusion instance on the cluster, drawing hyperparameters and other
        needed information from the Determined master."""
        info = det.get_cluster_info()
        hparams = attrdict.AttrDict(info.trial.hparams)
        latest_checkpoint = info.latest_checkpoint

        return cls(
            latest_checkpoint=latest_checkpoint,
            **hparams.model,
            **hparams.data,
            **hparams.trainer,
            **hparams.inference,
        )

    def train_on_cluster(self) -> None:
        """Run the full textual inversion training loop. Training must be performed on a
        Determined cluster."""
        self.logger.info("--------------- Starting Training ---------------")
        self.logger.info(f"Effective global batch size: {self.effective_global_batch_size}")
        self.logger.info(f"Learning rate: {self.learning_rate}")
        self.logger.info(f"Train dataset size: {len(self.train_dataset)}")
        try:
            distributed = det.core.DistributedContext.from_torch_distributed()
        except KeyError:
            distributed = None
        with det.core.init(
            distributed=distributed, tensorboard_mode=det.core.TensorboardMode.MANUAL
        ) as core_context:
            self._restore_latest_checkpoint(core_context)
            # There will be a single op of len max_length, as defined in the searcher config.
            for op in core_context.searcher.operations():
                while self.steps_completed < op.length:
                    for batch in self.train_dataloader:
                        # Use the accumulate method for efficient gradient accumulation.
                        with self.accelerator.accumulate(self.text_encoder):
                            self._train_one_batch(batch)
                        # An SGD step has been taken when self.accelerator.sync_gradients is True.
                        took_sgd_step = self.accelerator.sync_gradients
                        if took_sgd_step:
                            self.steps_completed += 1
                            self.logger.info(f"Step {self.steps_completed} completed")

                            is_end_of_training = self.steps_completed == op.length
                            time_to_report = self.steps_completed % self.metric_report_freq == 0
                            time_to_ckpt = self.steps_completed % self.checkpoint_freq == 0

                            # Report metrics, checkpoint, and preempt as appropriate.
                            if is_end_of_training or time_to_report or time_to_ckpt:
                                self._report_train_metrics(core_context)
                            if is_end_of_training or time_to_ckpt:
                                self._save(core_context)
                                if core_context.preempt.should_preempt():
                                    return

                            if is_end_of_training:
                                break
                if self.accelerator.is_main_process:
                    # Report the final mean loss.
                    op.report_completed(self.last_mean_loss)

    def _train_one_batch(self, batch: TorchData) -> torch.Tensor:
        """Train on a single batch, returning the loss and updating internal metrics."""
        # Convert images to latent space.
        latent_dist = self.vae.encode(batch["pixel_values"]).latent_dist
        latents = latent_dist.sample().detach()

        # In 2112.10752, it was found that the latent space variance plays a large role in image
        # quality.  The following scale factor helps to maintain unit latent variance.  See
        # https://github.com/huggingface/diffusers/issues/437 for more details.
        scale_factor = 0.18215
        latents = latents * scale_factor

        # Sample noise that we'll add to the latents.
        noise = torch.randn(latents.shape).to(self.accelerator.device)
        # Sample a random timestep for each image in the batch.
        rand_timesteps = torch.randint(
            0,
            self.num_train_timesteps,
            (self.train_batch_size,),
            device=self.accelerator.device,
        ).long()

        # Add noise to the latents according to the noise magnitude at each timestep. This is the
        # forward diffusion process.
        noisy_latents = self.train_scheduler.add_noise(latents, noise, rand_timesteps)

        # Get the text embedding for the prompt.
        encoder_hidden_states = self.text_encoder(batch["input_ids"])[0]

        # Predict the noise residual.
        noise_pred = self.unet(noisy_latents, rand_timesteps, encoder_hidden_states).sample
        mse_loss = F.mse_loss(noise_pred, noise)
        self.accelerator.backward(mse_loss)
        print("MSE LOSS: ", mse_loss)

        # Add a norm penalty to the loss. It is more memory efficient to perform this computation
        # as a separate forward and backward pass, rather than combining it with the loss above.
        # TODO: Clean this up.
        new_token_embeddings_norms = torch.linalg.vector_norm(
            self._get_new_token_embeddings(), dim=1
        )

        # Introduce a squared-loss for the size of the new embedding norms, otherwise they are
        # driven to be much larger than the original embedding norms by SGD and dominate the art.
        # We take the sum rather than the mean, as this should make the optimal norm_penalty value
        # less sensitive to the number of newly added tokens.
        norm_loss = (
            self.norm_penalty
            * ((new_token_embeddings_norms - self.original_embedding_mean_norm) ** 2).sum()
        )
        self.accelerator.backward(norm_loss)

        # Add the total loss to the loss history for metric tracking.
        loss = (mse_loss + norm_loss).detatch()
        self.loss_history.append(loss)

        # For textual inversion, we only update the embeddings of the newly added concept tokens.
        # This is most safely implemented by copying the original embeddings, rather than zeroing
        # out their gradients, as L2 regularization (for instance) will still modify weights whose
        # gradient is zero. See link below for a discussion:
        # https://discuss.pytorch.org/t/how-to-freeze-a-subset-of-weights-of-a-layer/97498
        self.optimizer.step()
        # Only overwrite after the step has actually been taken:
        if self.accelerator.sync_gradients:
            token_embeddings = self._get_token_embedding_weight_data()
            token_embeddings[
                self.original_embedding_idxs
            ] = self.original_embedding_tensors.detach().clone()
            if self.accelerator.is_main_process:
                print(
                    80 * "$",
                    f"MEAN NEW EMBEDDING NORM STEP {self.steps_completed}",
                    new_token_embeddings_norms.detach().mean().item(),
                    f"MAX NEW EMBEDDING NORM STEP {self.steps_completed}",
                    new_token_embeddings_norms.detach().max().item(),
                    print("NORM LOSS: ", norm_loss),
                    80 * "$",
                    sep="\n",
                )
        self.optimizer.zero_grad()

        return loss

    def _build_models(self) -> None:
        """Download the relevant models using deferred execution:
        https://huggingface.co/docs/accelerate/concept_guides/deferring_execution
        """
        with self.accelerator.main_process_first():
            self.tokenizer = CLIPTokenizer.from_pretrained(
                pretrained_model_name_or_path=self.pretrained_model_name_or_path,
                subfolder="tokenizer",
                use_auth_token=self.use_auth_token,
            )
            self.text_encoder = CLIPTextModel.from_pretrained(
                pretrained_model_name_or_path=self.pretrained_model_name_or_path,
                subfolder="text_encoder",
                use_auth_token=self.use_auth_token,
            )
            self.vae = AutoencoderKL.from_pretrained(
                pretrained_model_name_or_path=self.pretrained_model_name_or_path,
                subfolder="vae",
                use_auth_token=self.use_auth_token,
            )
            self.unet = UNet2DConditionModel.from_pretrained(
                pretrained_model_name_or_path=self.pretrained_model_name_or_path,
                subfolder="unet",
                use_auth_token=self.use_auth_token,
            )
        # Modules for StableDiffusionPipeline only required by chief worker.
        if self.accelerator.is_main_process:
            self.safety_checker = StableDiffusionSafetyChecker.from_pretrained(
                pretrained_model_name_or_path="CompVis/stable-diffusion-safety-checker"
            )
            self.feature_extractor = CLIPFeatureExtractor.from_pretrained(
                pretrained_model_name_or_path="openai/clip-vit-base-patch32"
            )

    def _add_new_tokens(self) -> None:
        """
        Add new concept tokens to the tokenizer and the corresponding embedding tensors to the
        text encoder.
        """
        for concept_token, initializer_tokens in zip(self.concept_tokens, self.initializer_tokens):
            dummy_placeholder_tokens, dummy_placeholder_ids = add_new_tokens(
                concept_token=concept_token,
                initializer_tokens=initializer_tokens,
                tokenizer=self.tokenizer,
                text_encoder=self.text_encoder,
            )
            self.concept_to_initializer_tokens_map[concept_token] = initializer_tokens
            self.concept_to_dummy_tokens_map[concept_token] = dummy_placeholder_tokens
            self.concept_to_dummy_ids_map[concept_token] = dummy_placeholder_ids
            self.logger.info(
                f"Added {len(dummy_placeholder_tokens)} new tokens for {concept_token}."
            )

        # Take a snapshot of the original embedding weights.  Used in the update step to ensure that
        # we only train the newly added concept vectors.
        all_dummy_placeholder_token_ids = []
        for ids in self.concept_to_dummy_ids_map.values():
            all_dummy_placeholder_token_ids.extend(ids)
        self.original_embedding_idxs = torch.isin(
            torch.arange(len(self.tokenizer)),
            torch.tensor(all_dummy_placeholder_token_ids),
            invert=True,
        )
        token_embeddings = self._get_token_embedding_weight_data()
        self.original_embedding_tensors = (
            token_embeddings[self.original_embedding_idxs]
            .detach()
            .clone()
            .to(self.accelerator.device)
        )
        with torch.no_grad():
            self.original_embedding_mean_norm = (
                torch.linalg.vector_norm(self.original_embedding_tensors, dim=1).mean().item()
            )
            print("ORIGINAL EMBEDDING MEAN NORM: ", self.original_embedding_mean_norm)
        self.new_embedding_idxs = torch.isin(
            torch.arange(len(self.tokenizer)),
            torch.tensor(all_dummy_placeholder_token_ids),
        )

    def _freeze_layers(self) -> None:
        """Freeze all not-to-be-trained layers."""
        # Freeze everything and then unfreeze only the layers we want to train.
        for model in (
            self.vae,
            self.unet,
            self.text_encoder,
        ):
            for param in model.parameters():
                param.requires_grad = False

        for param in self.text_encoder.text_model.embeddings.token_embedding.parameters():
            param.requires_grad = True

    def _tokenizer_fn(self, text: str) -> torch.Tensor:
        """Helper function for turning text directly into a tensor."""
        dummy_text = self._replace_concepts_with_dummies(text)
        tokenized_dummy_text = self.tokenizer(
            dummy_text,
            padding="max_length",
            truncation=True,
            max_length=self.tokenizer.model_max_length,
            return_tensors="pt",
        ).input_ids[0]
        return tokenized_dummy_text

    def _replace_concepts_with_dummies(self, text: str) -> str:
        """Helper function for replacing concepts with dummy placeholders."""
        for concept_token, d_tokens in self.concept_to_dummy_tokens_map.items():
            text = text.replace(concept_token, " ".join(d_tokens))
        return text

    def _build_dataset_and_dataloader(self) -> None:
        """Build the dataset and dataloader."""
        self.train_dataset = data.TextualInversionDataset(
            train_img_dirs=self.train_img_dirs,
            tokenizer_fn=self._tokenizer_fn,
            concept_tokens=self.concept_tokens,
            learnable_properties=self.learnable_properties,
            img_size=self.img_size,
            interpolation=self.interpolation,
            flip_p=self.flip_p,
            center_crop=self.center_crop,
        )
        self.train_dataloader = DataLoader(
            self.train_dataset, batch_size=self.train_batch_size, shuffle=True
        )

    def _build_optimizer(self) -> None:
        """Construct the optimizer, recalling that only the embedding vectors are to be trained."""
        embedding_params = self.text_encoder.get_input_embeddings().parameters()
        self.optimizer = self._optim_dict[self.optimizer_name](
            embedding_params,  # only optimize the embeddings
            lr=self.learning_rate,
            **self.other_optimizer_kwargs,
        )

    def _build_train_scheduler(self) -> None:
        self.train_scheduler = DDPMScheduler(
            beta_start=self.beta_start,
            beta_end=self.beta_end,
            beta_schedule="scaled_linear",
            num_train_timesteps=self.num_train_timesteps,
            tensor_format="pt",
        )

    def _wrap_and_prepare(self) -> None:
        """Wrap necessary modules for distributed training and set unwrapped, non-trained modules
        to the appropriate eval state."""

        # Freeze the vae and unet completely, and everything in the text encoder except the
        # embedding layer

        self.text_encoder, self.optimizer, self.train_dataloader = self.accelerator.prepare(
            self.text_encoder, self.optimizer, self.train_dataloader
        )
        self.vae.to(self.accelerator.device)
        self.unet.to(self.accelerator.device)
        self.text_encoder.train()
        self.vae.eval()
        self.unet.eval()

    def _restore_latest_checkpoint(self, core_context: det.core.Context) -> None:
        """Restores the experiment state to the latest saved checkpoint, if it exists."""
        if self.latest_checkpoint is not None:
            with core_context.checkpoint.restore_path(self.latest_checkpoint) as path:
                with self.accelerator.local_main_process_first():
                    with open(path.joinpath("metadata.json"), "r") as f:
                        checkpoint_metadata_dict = json.load(f)
                        self.steps_completed = checkpoint_metadata_dict["steps_completed"]
                    optimizer_state_dict = torch.load(path.joinpath("optimizer_state_dict.pt"))
                    self.optimizer.load_state_dict(optimizer_state_dict)
                    learned_embeddings_dict = torch.load(
                        path.joinpath("learned_embeddings_dict.pt")
                    )
                    token_embeddings = self._get_token_embedding_weight_data()
                    for concept_token, d_ids in self.concept_to_dummy_ids_map.items():
                        learned_embeddings = learned_embeddings_dict[concept_token][
                            "learned_embeddings"
                        ]
                        # Sanity check on length.
                        # TODO: replace with strict=True in zip after upgrade to py >= 3.10
                        assert len(d_ids) == len(
                            learned_embeddings
                        ), 'Length of "d_ids" and "learned_embeddings" must be equal.'
                        for idx, tensor in zip(
                            d_ids,
                            learned_embeddings,
                        ):
                            token_embeddings[idx] = tensor

    def _save(self, core_context: det.core.Context) -> None:
        """Checkpoints the training state and pipeline."""
        self.logger.info(f"Saving checkpoint at step {self.steps_completed}.")
        self.accelerator.wait_for_everyone()
        if self.accelerator.is_main_process:
            checkpoint_metadata_dict = {
                "steps_completed": self.steps_completed,
                "pretrained_model_name_or_path": self.pretrained_model_name_or_path,
            }
            if self.generate_training_images:
                self._build_pipeline()
                self._generate_and_write_tb_imgs(core_context)
            with core_context.checkpoint.store_path(checkpoint_metadata_dict) as (path, storage_id):
                self._write_optimizer_state_dict_to_path(path)
                self._write_learned_embeddings_to_path(path)

    def _write_optimizer_state_dict_to_path(self, path: pathlib.Path) -> None:
        optimizer_state_dict = self.optimizer.state_dict()
        self.accelerator.save(optimizer_state_dict, path.joinpath("optimizer_state_dict.pt"))

    def _write_learned_embeddings_to_path(self, path: pathlib.Path) -> None:
        learned_embeddings_dict = {}
        for concept_token, d_ids in self.concept_to_dummy_ids_map.items():
            token_embeddings = self._get_token_embedding_weight_data()
            learned_embeddings = token_embeddings[d_ids].detach().cpu()
            initializer_tokens = self.concept_to_initializer_tokens_map[concept_token]
            learned_embeddings_dict[concept_token] = {
                "initializer_tokens": initializer_tokens,
                "learned_embeddings": learned_embeddings,
            }
        self.accelerator.save(learned_embeddings_dict, path.joinpath("learned_embeddings_dict.pt"))

    def _build_pipeline(self) -> None:
        """Build the pipeline for the chief worker only."""
        if self.accelerator.is_main_process:
            inference_scheduler = NOISE_SCHEDULER_DICT[self.inference_scheduler_name]
            self.inference_scheduler_kwargs = {
                "beta_start": self.beta_start,
                "beta_end": self.beta_end,
                "beta_schedule": self.beta_schedule,
                **self.other_inference_scheduler_kwargs,
            }
            self.pipeline = StableDiffusionPipeline(
                text_encoder=self.accelerator.unwrap_model(self.text_encoder),
                vae=self.vae,
                unet=self.unet,
                tokenizer=self.tokenizer,
                scheduler=inference_scheduler(**self.inference_scheduler_kwargs),
                safety_checker=self.safety_checker,
                feature_extractor=self.feature_extractor,
            ).to(self.accelerator.device)

    def _generate_and_write_tb_imgs(self, core_context: det.core.Context) -> None:
        """Generates images using the current pipeline and logs them to Tensorboard."""
        self.logger.info("Generating sample images")
        tb_dir = core_context.train.get_tensorboard_path()
        tb_writer = SummaryWriter(log_dir=tb_dir)
        for prompt in self.inference_prompts:
            dummy_prompt = self._replace_concepts_with_dummies(prompt)
            # Fix generator for reproducibility.
            generator = torch.Generator(device=self.accelerator.device).manual_seed(
                self.generator_seed
            )
            # Set output_type to anything other than `pil` to get numpy arrays out. Needed
            # for tensorboard logging.
            generated_img = self.pipeline(
                prompt=dummy_prompt,
                num_inference_steps=self.num_inference_steps,
                guidance_scale=self.guidance_scale,
                generator=generator,
                output_type="np",
            ).images[0]
            tb_writer.add_image(
                prompt,
                img_tensor=generated_img,
                global_step=self.steps_completed,
                dataformats="HWC",
            )
        core_context.train.upload_tensorboard_files()

    def _report_train_metrics(self, core_context: det.core.Context) -> None:
        """Report training metrics to the Determined master."""
        self.accelerator.wait_for_everyone()
        local_mean_loss = torch.tensor(self.loss_history, device=self.accelerator.device).mean()
        # reduction = 'mean' seems to return the sum rather than the mean:
        self.last_mean_loss = (
            self.accelerator.reduce(local_mean_loss, reduction="sum").item()
            / self.accelerator.num_processes
        )
        self.loss_history = []
        print("self.last_mean_loss", self.last_mean_loss)
        if self.accelerator.is_main_process:
            core_context.train.report_training_metrics(
                steps_completed=self.steps_completed,
                metrics={"loss": self.last_mean_loss},
            )

    def _get_token_embedding_weight_data(self) -> torch.Tensor:
        """Returns the data tensor from the `weight` parameter of the embedding matrix, accounting
        for the possible insertion of a .module attr insertion due to the .prepare() call.
        """
        try:
            token_embeddings = self.text_encoder.module.get_input_embeddings().weight.data
        except AttributeError:
            token_embeddings = self.text_encoder.get_input_embeddings().weight.data
        return token_embeddings

    def _get_new_token_embeddings(self) -> torch.Tensor:
        """Returns the tensor of newly-added token embeddings."""
        try:
            token_embedding_layer = self.text_encoder.module.text_model.embeddings.token_embedding
        except AttributeError:
            token_embedding_layer = self.text_encoder.text_model.embeddings.token_embedding
        all_concept_tokens = " ".join(list(self.concept_tokens))
        all_dummy_tokens = self._replace_concepts_with_dummies(all_concept_tokens)
        # TODO: Can change the device call to self.accelerator.device after deleting print statement
        # checks.
        all_dummy_tokens_t = torch.tensor(
            self.tokenizer.encode(all_dummy_tokens, add_special_tokens=False),
            device=token_embedding_layer.weight.device,
        )
        new_token_embeddings = token_embedding_layer(all_dummy_tokens_t)
        return new_token_embeddings


class DetSDTextualInversionPipeline:
    """Class for generating images from a Stable Diffusion checkpoint pre-trained using Determined
    AI.  Initialize with no arguments in order to run plan Stable Diffusion without any trained
    textual inversion embeddings.
    """

    def __init__(
        self,
        learned_embeddings_filename: str = "learned_embeddings_dict.pt",
        scheduler_name: Literal["ddim", "lms-discrete", "pndm"] = "pndm",
        beta_start: float = 0.00085,
        beta_end: float = 0.012,
        beta_schedule: Literal["linear", "scaled_linear", "squaredcos_cap_v2"] = "scaled_linear",
        other_scheduler_kwargs: Optional[Dict[str, Any]] = None,
        pretrained_model_name_or_path: str = "CompVis/stable-diffusion-v1-4",
        device: str = "cuda",
        use_autocast: bool = True,
        use_fp16: bool = True,
    ) -> None:
        # We assume that the Huggingface User Access token has been stored as a HF_AUTH_TOKEN
        # environment variable. See https://huggingface.co/docs/hub/security-tokens
        try:
            self.use_auth_token = os.environ["HF_AUTH_TOKEN"]
        except KeyError:
            raise KeyError(
                "Please set your HF User Access token as the HF_AUTH_TOKEN environment variable."
            )
        self.learned_embeddings_filename = learned_embeddings_filename
        self.scheduler_name = scheduler_name
        self.beta_start = beta_start
        self.beta_end = beta_end
        self.beta_schedule = beta_schedule
        self.other_scheduler_kwargs = (
            other_scheduler_kwargs or DEFAULT_SCHEDULER_KWARGS_DICT[scheduler_name]
        )
        self.pretrained_model_name_or_path = pretrained_model_name_or_path
        self.device = device
        if use_fp16 and not use_autocast:
            raise ValueError("If use_fp16 is True, use_autocast must also be True.")
        self.use_autocast = use_autocast
        self.use_fp16 = use_fp16

        scheduler_kwargs = {
            "beta_start": self.beta_start,
            "beta_end": self.beta_end,
            "beta_schedule": self.beta_schedule,
            **self.other_scheduler_kwargs,
        }
        self.scheduler = NOISE_SCHEDULER_DICT[self.scheduler_name](**scheduler_kwargs)

        # The below attrs are non-trivially instantiated as necessary through private methods.
        self.all_checkpoint_paths = []
        self.learned_embeddings_dict = {}
        self.concept_to_dummy_tokens_map = {}
        self.all_added_concepts = []

        self._build_models()
        self._build_pipeline()

    def load_from_checkpoint_paths(
        self, checkpoint_paths: Union[Union[str, pathlib.Path], List[Union[str, pathlib.Path]]]
    ) -> None:
        """Load concepts from one or more checkpoint paths, each of which is expected contain a
        file with the name specified by the `learned_embeddings_filename` init arg. The file is
        expected to contain a dictionary whose keys are concept_token names and whose values are
        dictionaries containing an `initializer_token` key and a `learned_embeddings` whose
        corresponding values are the initializer string and learned embedding tensors, respectively.
        """
        if not checkpoint_paths:
            return
        # Get data from all checkpoints.
        if isinstance(checkpoint_paths, str):
            checkpoint_paths = [pathlib.Path(checkpoint_paths)]
        if isinstance(checkpoint_paths, pathlib.Path):
            checkpoint_paths = [checkpoint_paths]

        for path in checkpoint_paths:
            if isinstance(path, str):
                path = pathlib.Path(path)
            # TODO: Check that the same pretrained_model_name_or_path is used for all ckpts.
            learned_embeddings_dict = torch.load(path.joinpath(self.learned_embeddings_filename))
            # Update embedding matrix and attrs.
            for concept_token, embedding_dict in learned_embeddings_dict.items():
                if concept_token in self.learned_embeddings_dict:
                    raise ValueError(
                        f"Checkpoint concept conflict: {concept_token} already exists."
                    )
                initializer_tokens = embedding_dict["initializer_tokens"]
                learned_embeddings = embedding_dict["learned_embeddings"]
                dummy_placeholder_tokens, dummy_placeholder_ids = add_new_tokens(
                    concept_token=concept_token,
                    initializer_tokens=initializer_tokens,
                    tokenizer=self.tokenizer,
                    text_encoder=self.text_encoder,
                )
                self.concept_to_dummy_tokens_map[concept_token] = dummy_placeholder_tokens
                token_embeddings = self.text_encoder.get_input_embeddings().weight.data
                # Sanity check on length.
                # TODO: replace with strict=True in zip after upgrade to py >= 3.10
                assert len(dummy_placeholder_ids) == len(
                    learned_embeddings
                ), "dummy_placeholder_ids and learned_embeddings must have the same length"
                for d_id, tensor in zip(dummy_placeholder_ids, learned_embeddings):
                    token_embeddings[d_id] = tensor
                self.learned_embeddings_dict[concept_token] = embedding_dict
            self.all_checkpoint_paths.append(path)

        self.all_added_concepts = list(self.concept_to_dummy_tokens_map.keys())
        print(f"Successfully loaded checkpoints. All loaded concepts: {self.all_added_concepts}")

    def load_from_uuids(
        self,
        uuids: Union[str, Sequence[str]],
    ) -> None:
        """Load concepts from one or more Determined checkpoint uuids. Must be logged into the
        Determined cluster to use this method.  If not logged-in, call
        determined.experimental.client.login first.
        """
        if isinstance(uuids, str):
            uuids = [uuids]
        checkpoint_paths = []
        for u in uuids:
            checkpoint = client.get_checkpoint(u)
            checkpoint_paths.append(pathlib.Path(checkpoint.download()))
        self.load_from_checkpoint_paths(checkpoint_paths)

    def _build_models(self) -> None:
        print(80 * "-", "Downloading pre-trained models...", 80 * "-", sep="\n")
        revision = "fp16" if self.use_fp16 else "main"
        self.tokenizer = CLIPTokenizer.from_pretrained(
            pretrained_model_name_or_path=self.pretrained_model_name_or_path,
            subfolder="tokenizer",
            use_auth_token=self.use_auth_token,
            revision=revision,
        )
        self.text_encoder = CLIPTextModel.from_pretrained(
            pretrained_model_name_or_path=self.pretrained_model_name_or_path,
            subfolder="text_encoder",
            use_auth_token=self.use_auth_token,
            revision=revision,
        )
        self.vae = AutoencoderKL.from_pretrained(
            pretrained_model_name_or_path=self.pretrained_model_name_or_path,
            subfolder="vae",
            use_auth_token=self.use_auth_token,
            revision=revision,
        )
        self.unet = UNet2DConditionModel.from_pretrained(
            pretrained_model_name_or_path=self.pretrained_model_name_or_path,
            subfolder="unet",
            use_auth_token=self.use_auth_token,
            revision=revision,
        )
        self.safety_checker = StableDiffusionSafetyChecker.from_pretrained(
            pretrained_model_name_or_path="CompVis/stable-diffusion-safety-checker"
        )
        self.feature_extractor = CLIPFeatureExtractor.from_pretrained(
            pretrained_model_name_or_path="openai/clip-vit-base-patch32"
        )

        for model in (
            self.text_encoder,
            self.vae,
            self.unet,
        ):
            model.to(self.device)
            model.eval()

    def _build_pipeline(self) -> None:
        print(
            80 * "-",
            "Building the pipeline...",
            80 * "-",
            sep="\n",
        )
        self.pipeline = StableDiffusionPipeline(
            text_encoder=self.text_encoder,
            vae=self.vae,
            unet=self.unet,
            tokenizer=self.tokenizer,
            scheduler=self.scheduler,
            safety_checker=self.safety_checker,
            feature_extractor=self.feature_extractor,
        ).to(self.device)
        print("Done!")

    def _replace_concepts_with_dummies(self, text: str) -> str:
        for concept_token, d_tokens in self.concept_to_dummy_tokens_map.items():
            text = text.replace(concept_token, " ".join(d_tokens))
        return text

    def _create_image_grid(self, images: List[Image.Image], rows: int, cols: int) -> Image.Image:
        w, h = images[0].size
        image_grid = Image.new("RGB", size=(cols * w, rows * h))
        for idx, img in enumerate(images):
            image_grid.paste(img, box=(idx % cols * w, idx // cols * h))
        return image_grid

    def _save_image(self, image: Image.Image, filename: str, saved_img_dir: str) -> None:
        """Saves the image as a time-stamped png file."""
        saved_img_dir = pathlib.Path(saved_img_dir)
        save_path = saved_img_dir.joinpath(filename)
        image.save(save_path)

    def __call__(
        self,
        prompt: str,
        rows: int = 1,
        cols: int = 1,
        num_inference_steps: int = 50,
        guidance_scale: int = 7.5,
        saved_img_dir: Optional[str] = None,
        seed: int = 2147483647,
        parallelize_factor: int = 1,
        other_pipeline_call_kwargs: Optional[dict] = None,
    ) -> Image.Image:
        """Generates an image from the provided prompt and optionally writes the results to disk."""
        other_pipeline_call_kwargs = other_pipeline_call_kwargs or {}
        num_samples = rows * cols
        # Could insert a check that num_samples % parallelize_factor == 0, else wasting compute

        images = []
        generated_samples = 0
        # The dummy prompts are what actually get fed into the pipeline
        dummy_prompt = self._replace_concepts_with_dummies(prompt)
        generator = torch.Generator(device="cuda").manual_seed(seed)
        while generated_samples < num_samples:
            context = torch.autocast("cuda") if self.use_autocast else nullcontext()
            with context:
                out = self.pipeline(
                    [dummy_prompt] * parallelize_factor,
                    num_inference_steps=num_inference_steps,
                    guidance_scale=guidance_scale,
                    generator=generator,
                    **other_pipeline_call_kwargs,
                )
            for nsfw, image in zip(out["nsfw_content_detected"], out["sample"]):
                # Re-try, if nsfw_content_detected
                if nsfw:
                    continue
                images.append(image)
                generated_samples += 1
        image_grid = self._create_image_grid(images[:num_samples], rows, cols)
        if saved_img_dir is not None:
            generation_details = f"_{num_inference_steps}_steps_{guidance_scale}_gs_{seed}_seed_"
            timestamp = "_".join(f"_{datetime.now().strftime('%c')}.png".split())
            file_suffix = generation_details + timestamp + ".png"
            joined_split_prompt = "_".join(prompt.split())
            filename = joined_split_prompt[: 255 - len(file_suffix)] + file_suffix
            self._save_image(image_grid, filename, saved_img_dir)
        return image_grid

    def __repr__(self) -> str:
        attr_dict = {
            "scheduler_name": self.scheduler_name,
            "beta_start": self.beta_start,
            "beta_end": self.beta_end,
            "beta_schedule": self.beta_schedule,
            "other_scheduler_kwargs": self.other_scheduler_kwargs,
            "pretrained_model_name_or_path": self.pretrained_model_name_or_path,
            "device": self.device,
            "use_autocast": self.use_autocast,
            "use_fp16": self.use_fp16,
            "all_added_concepts": self.all_added_concepts,
        }
        attr_dict_str = ", ".join([f"{key}={value}" for key, value in attr_dict.items()])
        return f"{self.__class__.__name__}({attr_dict_str})"


def add_new_tokens(
    concept_token: str,
    initializer_tokens: Sequence[str],
    tokenizer: nn.Module,
    text_encoder: nn.Module,
) -> Tuple[List[str], List[int]]:
    """Helper function for adding new tokens to the tokenizer and extending the corresponding
    embeddings appropriately, given a single concept token and its sequence of corresponding
    initializer tokens.  Returns the dummy representation of the initializer tokens and their
    ids."""
    initializer_ids = tokenizer(
        initializer_tokens,
        padding="max_length",
        truncation=True,
        max_length=tokenizer.model_max_length,
        return_tensors="pt",
        add_special_tokens=False,
    ).input_ids
    # Get all non-special tokens
    try:
        special_token_ids = tokenizer.all_special_ids
    except AttributeError:
        special_token_ids = []
    non_special_initializer_locations = torch.isin(
        initializer_ids, torch.tensor(special_token_ids), invert=True
    )
    non_special_initializer_ids = initializer_ids[non_special_initializer_locations]
    if len(non_special_initializer_ids) == 0:
        raise ValueError(
            f'"{initializer_tokens}" maps to trivial tokens, please choose a different initializer.'
        )
    # Add a new randomly generated dummy placeholder token for every token in the initializer.
    dummy_placeholder_tokens = [
        f"{concept_token}_{n}" for n in range(len(non_special_initializer_ids))
    ]
    num_added_tokens = tokenizer.add_tokens(dummy_placeholder_tokens)
    if num_added_tokens != len(dummy_placeholder_tokens):
        raise ValueError(f"Subset of {dummy_placeholder_tokens} tokens already exist in tokenizer.")
    # Get the ids of the new placeholders.
    dummy_placeholder_ids = tokenizer.convert_tokens_to_ids(dummy_placeholder_tokens)
    # Sanity check
    assert len(dummy_placeholder_ids) == len(
        non_special_initializer_ids
    ), "dummy placeholder token ids and non-special initializer token count doesn't match"

    text_encoder.resize_token_embeddings(len(tokenizer))
    try:
        token_embeddings = text_encoder.module.get_input_embeddings().weight.data
    except AttributeError:
        token_embeddings = text_encoder.get_input_embeddings().weight.data
    # Initialize the placeholder vectors to coincide with their initializer vectors.
    # Sanity check on length. TODO: replace with strict=True in zip after upgrade to py >= 3.10
    assert len(dummy_placeholder_ids) == len(
        non_special_initializer_ids
    ), 'Length of "dummy_placeholder_ids" and "non_special_initializer_ids" must match.'
    for p_id, i_id in zip(dummy_placeholder_ids, non_special_initializer_ids):
        token_embeddings[p_id] = token_embeddings[i_id]
    return dummy_placeholder_tokens, dummy_placeholder_ids
