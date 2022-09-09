import json
import pathlib
import shutil
import tempfile
from typing import Literal, Union


import determined as det
import torch
import torch.nn.functional as F
import torchmetrics
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
    """Class for training a textual inversion model. Assumes GPU training."""

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
        metric_report_step_freq: int = 100,
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
        self.metric_report_step_freq = metric_report_step_freq
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
        self.mean_loss_metric = torchmetrics.MeanMetric()
        self.mean_loss_metric.cuda()
        # Save the current mean loss as an attr.s
        self.mean_loss = 0.0

        self.effective_global_batch_size = (
            self.gradient_accumulation_steps
            * self.train_batch_size
            * self.accelerator.num_processes
        )
        # If scale_lr, we linearly scale the bare learning rate by the effective batch size
        if scale_lr:
            self.learning_rate *= self.effective_global_batch_size
            self.logger.info(f"Using scaled learning rate {self.learning_rate}")

        # The below are instantiated in _setup
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
        assert self.text_encoder.training, "Text encoder should be in training mode"
        assert not self.vae.training, "VAE should be in eval mode"
        assert not self.unet.training, "UNet should be in eval mode"

        """Run the full latent inversion training loop."""
        self.logger.info("--------------- Starting training ---------------")
        self.logger.info(f"Effective global batch size: {self.effective_global_batch_size}")
        self.logger.info(f"Learning rate: {self.learning_rate}")
        self.logger.info(f"Train dataset size: {len(self.train_dataset)}")

        distributed = det.core.DistributedContext.from_torch_distributed()
        with det.core.init(distributed=distributed) as core_context:
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
                            # Report metrics, if appropriate.
                            if self._should_report_metrics():
                                self._report_train_metrics(core_context)
                            # Save checkpoint and/or preempt, if appropriate.
                            is_end_of_training = self.steps_completed == op.length
                            if (
                                is_end_of_training
                                or self.steps_completed % self.metric_report_step_freq == 0
                            ):
                                self._save(core_context, is_end_of_training)
                                if core_context.preempt.should_preempt():
                                    return
                            if is_end_of_training:
                                break
                if self.accelerator.is_main_process:
                    # Report the final mean loss.
                    op.report_completed(self.mean_loss)

    def _train_one_batch(self, batch: TorchData) -> torch.Tensor:
        """Train on a single batch, returning the loss and updating internal metrics."""
        # Convert images to latent space
        latents = self.vae.encode(batch["pixel_values"]).sample().detach()
        # In 2112.10752, it was found that the latent space variance plays a large role in image
        # quality.  The following scale factor helps to maintain unit latent variance.  See
        # https://github.com/huggingface/diffusers/issues/437 for more details.
        scale_factor = 0.18215
        latents = latents * scale_factor

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

        # Update the mean loss metric
        self.mean_loss_metric.update(loss.detach().item())

        self.accelerator.backward(loss)

        # Get the gradients. An extra .module attr is needed due to the .prepare() call
        grads = self.text_encoder.module.get_input_embeddings().weight.grad
        # Zero out the gradients for all token embeddings except the newly added
        # embeddings for the concept, as we only want to train the concept embeddings
        index_grads_to_zero = torch.arange(len(self.tokenizer)) != self.placeholder_token_id
        grads.data[index_grads_to_zero] = 0.0
        print(grads[1:3])
        self.optimizer.step()
        self.optimizer.zero_grad()

        return loss

    def _setup(self):
        """Combined setup steps per HF Accelerator best practices."""
        self._build_models()
        self._add_new_tokens()
        self._freeze_layers()
        self._build_dataset_and_dataloader()
        self._build_optimizer()
        self._build_train_noise_scheduler()
        self._wrap_and_prepare()

    def _build_models(self) -> None:
        """Download the relevant models using deferred execution:
        https://huggingface.co/docs/accelerate/concept_guides/deferring_execution
        """
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

    def _add_new_tokens(self) -> None:
        """
        Add new concept tokens to the tokenizer.
        """
        # Convert the initializer_token, placeholder_token to ids.
        token_ids = self.tokenizer.encode(self.initializer_token, add_special_tokens=False)
        # Check if initializer_token is a single token or a sequence of tokens.
        if len(token_ids) > 1:
            raise ValueError("The initializer token must be a single token.")

        initializer_token_id = token_ids[0]
        self.placeholder_token_id = self.tokenizer.convert_tokens_to_ids(self.placeholder_token)

        # Extend the size of the self.text_encoder to account for the new placeholder_token.
        self.text_encoder.resize_token_embeddings(len(self.tokenizer))
        # Initalize the placeholder_token vector to coincide with the initializer_token vector.
        token_embeds = self.text_encoder.get_input_embeddings().weight.data
        token_embeds[self.placeholder_token_id] = token_embeds[initializer_token_id]

    def _freeze_layers(self) -> None:
        """Freeze all non-trained layers."""

        def freeze_params(params) -> None:
            """Helper function for freezing parameters."""
            for param in params:
                param.requires_grad = False

        for params in (
            self.vae.parameters(),
            self.unet.parameters(),
            self.text_encoder.text_model.encoder.parameters(),
            self.text_encoder.text_model.final_layer_norm.parameters(),
            self.text_encoder.text_model.embeddings.position_embedding.parameters(),
        ):
            freeze_params(params)

    def _build_dataset_and_dataloader(self) -> None:
        """Build the dataset and dataloader."""
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
                with open(path.joinpath("metadata.json"), "r") as f:
                    metadata_dict = json.load(f)
                self.steps_completed = metadata_dict["steps_completed"]
                with self.accelerator.main_process_first():
                    self.accelerator.load_state(path)

    def _save(self, core_context: det.core.Context, is_end_of_training: bool) -> None:
        """Checkpoints the training state and pipeline."""
        self.logger.info(f"Saving checkpoint at step {self.steps_completed}.")
        self.accelerator.wait_for_everyone()
        train_checkpoint_path = self._write_train_checkpoint_and_get_dir(core_context)
        if self.accelerator.is_main_process:
            checkpoint_metadata = {
                "steps_completed": self.steps_completed,
                "placeholder_tokens": self.placeholder_token,
            }
            with core_context.checkpoint.store_path(checkpoint_metadata) as (path, storage_id):
                # TODO: Avoid this copy
                shutil.copytree(train_checkpoint_path, path, dirs_exist_ok=True)
                self._write_pipline_to_path(path)
                shutil.rmtree(train_checkpoint_path)

    def _write_train_checkpoint_and_get_dir(self, core_context: det.core.Context) -> pathlib.Path:
        """Accelerator's save_state method requires all workers to write to disk. Writes to a
        temp dir and returns the corresponding path object."""
        # Have the chief create a temp dir
        if self.accelerator.is_main_process:
            dirpath = tempfile.mkdtemp()
        else:
            dirpath = None
        # Broadcast to all workers
        dirpath = core_context.distributed.broadcast(dirpath)
        dirpath = pathlib.Path(dirpath)
        print("saving to dirpath")
        self.accelerator.save_state(dirpath)
        self.accelerator.wait_for_everyone()
        return dirpath

    def _write_pipline_to_path(self, path: pathlib.Path) -> None:
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
        pipeline.save_pretrained(path)
        # Also save the newly trained embeddings
        learned_embeds = (
            self.accelerator.unwrap_model(self.text_encoder)
            .get_input_embeddings()
            .weight[self.placeholder_token_id]
        )
        learned_embeds_dict = {self.placeholder_token: learned_embeds.detach().cpu()}
        torch.save(learned_embeds_dict, path.joinpath("learned_embeds.bin"))

    def _report_train_metrics(self, core_context: det.core.Context) -> None:
        """Report training metrics to the Determined master."""
        self.mean_loss = self.mean_loss_metric.compute().item()
        if self.accelerator.is_main_process:
            core_context.train.report_training_metrics(
                steps_completed=self.steps_completed,
                metrics={"loss": self.mean_loss},
            )
        self.mean_loss_metric.reset()

    def _should_report_metrics(self) -> bool:
        return self.steps_completed % self.metric_report_step_freq == 0
