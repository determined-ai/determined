import json
import math
from typing import Literal, Union

import determined as det
import torch
import torch.nn.functional as F
from accelerate.logging import get_logger
from accelerate import Accelerator
from determined.pytorch import TorchData
from diffusers import (
    AutoencoderKL,
    DDPMScheduler,
    PNDMScheduler,
    StableDiffusionPipeline,
    UNet2DConditionModel,
)
from diffusers.pipelines.stable_diffusion import StableDiffusionSafetyChecker
from torch.utils.data import DataLoader
from transformers import CLIPFeatureExtractor, CLIPTextModel, CLIPTokenizer

import data


class TextualInversionTrainer:
    def __init__(
        self,
        use_auth_token: str,
        latest_checkpoint: Union[str, None],
        train_img_dir: str,
        output_dir: str,
        placeholder_token: str,
        initializer_token: int,
        learnable_property: Literal["object", "style"] = "object",
        pretrained_model_name_or_path: str = "CompVis/stable-diffusion-v1-4",
        train_batch_size: int = 1,
        gradient_accumulation_steps: int = 4,
        learning_rate: float = 5e-04,
        scale_lr: bool = True,
        checkpoint_step_freq: int = 100,
        beta_start: float = 0.00085,
        beta_end: float = 0.012,
        beta_schedule: Literal["linear", "scaled_linear", "squaredcos_cap_v2"] = "scaled_linear",
        num_train_timesteps: int = 1000,
        size: int = 512,
        interpolation: Literal["nearest", "bilinear", "bicubic"] = "bicubic",
        flip_p: float = 0.5,
        center_crop: bool = False,
    ) -> None:
        self.use_auth_token = use_auth_token
        self.latest_checkpoint = latest_checkpoint
        self.pretrained_model_name_or_path = pretrained_model_name_or_path
        self.learnable_property = learnable_property
        self.placeholder_token = placeholder_token
        self.initializer_token = initializer_token
        self.train_batch_size = train_batch_size
        self.gradient_accumulation_steps = gradient_accumulation_steps
        self.learning_rate = learning_rate
        self.output_dir = output_dir
        self.scale_lr = scale_lr
        self.checkpoint_step_freq = checkpoint_step_freq
        self.beta_start = beta_start
        self.beta_end = beta_end
        self.beta_schedule = beta_schedule
        self.num_train_timesteps = num_train_timesteps
        self.train_img_dir = train_img_dir
        self.size = size
        self.interpolation = interpolation
        self.flip_p = flip_p
        self.center_crop = center_crop

        self.logger = get_logger(__name__)
        self.accelerator = Accelerator(
            gradient_accumulation_steps=self.gradient_accumulation_steps,
        )
        self.steps_completed = 0

        self.effective_batch_size = (
            self.gradient_accumulation_steps
            * self.train_batch_size
            * self.accelerator.num_processes
        )
        # If scale_lr, we linearly scale the bare learning rate by the effective batch size
        if scale_lr:
            self.learning_rate *= self.effective_batch_size
            self.logger.info(f"Using scaled learning rate {self.learning_rate}")

        # attrs below instantiated in _setup
        self.tokenizer = None
        self.text_encoder = None
        self.vae = None
        self.unet = None
        self.placeholder_token_id = None
        self.train_dataset = None
        self.train_dataloader = None
        self.optimizer = None
        self.train_noise_scheduler = None

        self._setup()

    def train(self) -> None:
        distributed = det.core.DistributedContext.from_torch_distributed()
        with det.core.init(distributed=distributed) as core_context:
            self._restore_latest_checkpoint(core_context)
            # There will be a single op of len max_length, as defined in the searcher config.
            for op in core_context.searcher.operations():
                print(80 * "$")
                print(f"Step {self.steps_completed} of {op.length}")
                while self.steps_completed < op.length:
                    for batch in self.train_dataloader:
                        print("batch")
                        # Use the accumulate method for efficient gradient accumulation.
                        with self.accelerator.accumulate(self.text_encoder):
                            print("getting loss")
                            loss = self._train_one_batch_and_get_loss(batch)

                        # Check if gradients have been synced, in which case we have performed a step
                        if self.accelerator.sync_gradients:
                            print(80 * "$")
                            print(f"Sync gradients at Step {self.steps_completed} of {op.length}")
                            self.steps_completed += 1
                            if self.steps_completed % self.checkpoint_step_freq == 0:
                                self._save_train_checkpoint(core_context)
                            if core_context.preempt.should_preempt():
                                return
                            if self.steps_completed == op.length:
                                break
                self.accelerator.wait_for_everyone()
                if self.accelerator.is_main_process:
                    print(80 * "$")
                    print(f"reporting complete")
                    op.report_completed(loss.detach().item())
                    self._save_pipeline(core_context)

    def _setup(self):
        """Combined setup steps per HF Accelerator best practices."""
        # Set model attrs using deferred execution:
        # https://huggingface.co/docs/accelerate/concept_guides/deferring_execution
        self._build_models()
        self._add_new_tokens_and_freeze()
        self._build_dataset_and_dataloader()
        self._build_optimizer()
        self._build_train_noise_scheduler()
        self._wrap_and_prepare()

    def _build_models(self) -> None:
        with self.accelerator.main_process_first():
            self.tokenizer = CLIPTokenizer.from_pretrained(
                self.pretrained_model_name_or_path,
                subfolder="tokenizer",
                use_auth_token=self.use_auth_token,
            )
            self.text_encoder = CLIPTextModel.from_pretrained(
                self.pretrained_model_name_or_path,
                subfolder="text_encoder",
                use_auth_token=self.use_auth_token,
            )
            self.vae = AutoencoderKL.from_pretrained(
                self.pretrained_model_name_or_path,
                subfolder="vae",
                use_auth_token=self.use_auth_token,
            )
            self.unet = UNet2DConditionModel.from_pretrained(
                self.pretrained_model_name_or_path,
                subfolder="unet",
                use_auth_token=self.use_auth_token,
            )

    def _add_new_tokens_and_freeze(self) -> None:
        # Convert the initializer_token, placeholder_token to ids
        token_ids = self.tokenizer.encode(self.initializer_token, add_special_tokens=False)
        # Check if initializer_token is a single token or a sequence of tokens
        if len(token_ids) > 1:
            raise ValueError("The initializer token must be a single token.")

        initializer_token_id = token_ids[0]
        self.placeholder_token_id = self.tokenizer.convert_tokens_to_ids(self.placeholder_token)

        # Extend the size of the self.text_encoder to account for the new placeholder_token
        self.text_encoder.resize_token_embeddings(len(self.tokenizer))
        # Initalize the placeholder_token vector to coincide with the initializer_token vector
        token_embeds = self.text_encoder.get_input_embeddings().weight.data
        token_embeds[self.placeholder_token_id] = token_embeds[initializer_token_id]

        # Freeze the vae and unet completely, and everything in the text encoder except the
        # embedding layer
        self._freeze_params(self.vae.parameters())
        self._freeze_params(self.unet.parameters())
        for p in (
            self.text_encoder.text_model.encoder.parameters(),
            self.text_encoder.text_model.final_layer_norm.parameters(),
            self.text_encoder.text_model.embeddings.position_embedding.parameters(),
        ):
            self._freeze_params(p)

    def _freeze_params(self, params) -> None:
        for param in params:
            param.requires_grad = False

    def _build_dataset_and_dataloader(self) -> None:
        self.train_dataset = data.TextualInversionDataset(
            train_img_dir=self.train_img_dir,
            tokenizer=self.tokenizer,
            placeholder_token=self.placeholder_token,
            learnable_property=self.learnable_property,
            size=self.size,
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
        self.optimizer = torch.optim.AdamW(
            embedding_params,  # only optimize the embeddings
            lr=self.learning_rate,
        )

    def _build_train_noise_scheduler(self) -> None:
        self.train_noise_scheduler = DDPMScheduler(
            beta_start=self.beta_start,
            beta_end=self.beta_end,
            beta_schedule="scaled_linear",
            num_train_timesteps=self.num_train_timesteps,
            tensor_format="pt",
        )

    def _wrap_and_prepare(self) -> None:
        """Wrap necessary modules for distributed training and set unwrapped modules appropriately."""
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
                with open(path.joinpath("metadata.json"), "r") as f:
                    metadata_dict = json.load(f)
                self.steps_completed = metadata_dict["steps_completed"]
                with self.accelerator.main_process_first():
                    self.accelerator.load_state(path)

    def _save_train_checkpoint(self, core_context: det.core.Context) -> None:
        if self.accelerator.is_main_process:
            checkpoint_metadata = {
                "steps_completed": self.steps_completed,
            }
            with core_context.checkpoint.store_path(checkpoint_metadata) as (path, storage_id):
                self.accelerator.save_state(path)

    def _train_one_batch_and_get_loss(self, batch: TorchData) -> None:
        self.optimizer.zero_grad()
        # Convert images to latent space
        latents = self.vae.encode(batch["pixel_values"]).sample().detach()
        latents = latents * 0.18215  # Why?

        # Sample noise that we'll add to the latents
        noise = torch.randn(latents.shape).to(latents.device)
        # Sample a random timestep for each image
        timesteps = torch.randint(
            0,
            self.num_train_timesteps,
            (self.train_batch_size,),
            device=latents.device,
        ).long()

        # Add noise to the latents according to the noise magnitude at each timestep
        # (this is the forward diffusion process)
        noisy_latents = self.train_noise_scheduler.add_noise(latents, noise, timesteps)

        # Get the text embedding for conditioning
        encoder_hidden_states = self.text_encoder(batch["input_ids"])[0]

        # Predict the noise residual
        noise_pred = self.unet(noisy_latents, timesteps, encoder_hidden_states)["sample"]
        loss = F.mse_loss(noise_pred, noise)
        self.accelerator.backward(loss)

        # Zero out the gradients for all token embeddings except the newly added
        # embeddings for the concept, as we only want to optimize the concept embeddings
        # DDP inserts an extra module attr which we need to draw from.
        grads = self.text_encoder.module.get_input_embeddings().weight.grad
        # try:
        #     grads = self.text_encoder.module.get_input_embeddings().weight.grad
        # except AttributeError:
        #     grads = self.text_encoder.get_input_embeddings().weight.grad
        # Get the index for tokens that we want to zero the grads for
        index_grads_to_zero = torch.arange(len(self.tokenizer)) != self.placeholder_token_id
        grads.data[index_grads_to_zero] = 0.0
        self.optimizer.step()

        return loss

    def _save_pipeline(self, core_context: det.core.Context) -> None:
        pipeline = StableDiffusionPipeline(
            text_encoder=self.accelerator.unwrap_model(self.text_encoder),
            vae=self.vae,
            unet=self.unet,
            tokenizer=self.tokenizer,
            # Use faster PNDMScheduler for inference
            scheduler=PNDMScheduler(
                beta_start=self.beta_start,
                beta_end=self.beta_end,
                beta_schedule=self.beta_schedule,
                skip_prk_steps=True,
            ),
            safety_checker=StableDiffusionSafetyChecker.from_pretrained(
                "CompVis/stable-diffusion-safety-checker"
            ),
            feature_extractor=CLIPFeatureExtractor.from_pretrained("openai/clip-vit-base-patch32"),
        )
        with core_context.checkpoint.store_path({"steps_completed": 1}) as (path, storage_id):
            print(80 * "=", f"Saving pipeline", 80 * "=", sep="\n")
            pipeline.save_pretrained(path)
            # Also save the newly trained embeddings
            learned_embeds = (
                self.accelerator.unwrap_model(self.text_encoder)
                .get_input_embeddings()
                .weight[self.placeholder_token_id]
            )
            learned_embeds_dict = {self.placeholder_token: learned_embeds.detach().cpu()}
            print(80 * "=", f"Saving learned_embeds", 80 * "=", sep="\n")
            torch.save(learned_embeds_dict, path.joinpath("learned_embeds.bin"))
