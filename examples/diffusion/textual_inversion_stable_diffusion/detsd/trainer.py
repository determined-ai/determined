import json
import os
import pathlib
from typing import List, Literal, Optional, Sequence, Union

import accelerate
import determined as det
import numpy as np
import torch
import torch.nn as nn
import torch.nn.functional as F
from determined.pytorch import TorchData
from diffusers import (
    AutoencoderKL,
    DDPMScheduler,
    StableDiffusionPipeline,
    UNet2DConditionModel,
)
from diffusers.pipelines.stable_diffusion import StableDiffusionSafetyChecker
from torch.utils.data import DataLoader
from torch.utils.tensorboard import SummaryWriter
from transformers import CLIPFeatureExtractor, CLIPTextModel, CLIPTokenizer

from detsd import data, defaults, layers, utils


class DetSDTextualInversionTrainer:
    """Performs Textual Inversion fine-tuning on a Determined cluster."""

    def __init__(
        self,
        img_dirs: Union[str, Sequence[str]],
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
        checkpoint_freq: int = 50,
        metric_report_freq: int = 50,
        beta_start: float = 0.00085,
        beta_end: float = 0.012,
        beta_schedule: Literal["linear", "scaled_linear", "squaredcos_cap_v2"] = "scaled_linear",
        num_train_timesteps: int = 1000,
        train_seed: int = 2147483647,
        hidden_reg_weight: float = 1e-4,
        img_size: int = 512,
        interpolation: Literal["nearest", "bilinear", "bicubic"] = "bicubic",
        flip_p: float = 0.0,
        center_crop: bool = True,
        append_file_name_to_text: bool = False,
        file_name_split_char: str = "_",
        generate_training_images: bool = True,
        images_per_prompt: int = 1,
        inference_prompts: Optional[Union[str, Sequence[str]]] = None,
        inference_scheduler_name: Literal["ddim", "lms-discrete", "pndm"] = "pndm",
        num_inference_steps: int = 50,
        guidance_scale: float = 7.5,
        generator_seed: int = 2147483647,
        other_inference_scheduler_kwargs: Optional[dict] = None,
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
        self.append_file_name_to_text = append_file_name_to_text
        self.file_name_split_char = file_name_split_char

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
        self.checkpoint_freq = checkpoint_freq
        self.metric_report_freq = metric_report_freq
        self.beta_start = beta_start
        self.beta_end = beta_end
        self.beta_schedule = beta_schedule
        self.num_train_timesteps = num_train_timesteps
        if isinstance(img_dirs, str):
            img_dirs = [img_dirs]
        self.img_dirs = img_dirs
        self.train_seed = train_seed
        self.hidden_reg_weight = hidden_reg_weight

        self.accelerator = accelerate.Accelerator(
            gradient_accumulation_steps=self.gradient_accumulation_steps,
        )
        accelerate.utils.set_seed(self.train_seed)

        assert inference_scheduler_name in defaults.NOISE_SCHEDULER_DICT, (
            f"inference_scheduler must be one {list(defaults.NOISE_SCHEDULER_DICT.keys())},"
            f" but got {inference_scheduler_name}"
        )
        if not generate_training_images and inference_prompts is not None:
            self.logger.warning(
                "Inference prompts were provided, but are being skipped, as generate_training"
                " images was set to False."
            )
        if not generate_training_images and images_per_prompt:
            self.logger.warning(
                "images_per_prompt was set to a non-zero value, but no images will be"
                " created, as generate_training images was set to False."
            )
        self.generate_training_images = generate_training_images
        self.images_per_prompt = images_per_prompt
        if isinstance(inference_prompts, str):
            inference_prompts = [inference_prompts]
        self.inference_scheduler_name = inference_scheduler_name
        self.inference_prompts = inference_prompts
        self.num_inference_steps = num_inference_steps
        self.guidance_scale = guidance_scale
        self.generator_seed = generator_seed

        if other_inference_scheduler_kwargs is None:
            other_inference_scheduler_kwargs = defaults.DEFAULT_SCHEDULER_KWARGS_DICT[
                self.inference_scheduler_name
            ]
        self.other_inference_scheduler_kwargs = other_inference_scheduler_kwargs

        self.steps_completed = 0
        self.metrics_history = {"loss": []}
        if self.hidden_reg_weight:
            self.metrics_history["hidden_reg_loss"] = []
        self.last_mean_loss = None

        self.effective_global_batch_size = (
            self.gradient_accumulation_steps
            * self.train_batch_size
            * self.accelerator.num_processes
        )
        # If scale_lr, we linearly scale the bare learning rate by the effective batch size.
        if scale_lr:
            self.learning_rate *= self.effective_global_batch_size
            self.logger.info(f"Using scaled learning rate {self.learning_rate}")

        # The trivial attrs below are instantiated as needed through private methods.
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
        self.inference_scheduler_kwargs = None
        self.pipeline = None

        self.concept_to_initializer_tokens_map = {}
        self.concept_to_non_special_initializer_ids_map = {}
        self.concept_to_dummy_tokens_map = {}
        self.concept_to_dummy_ids_map = {}

        self._build_models()
        self._add_new_tokens_and_update_embeddings()
        self._freeze_layers()
        self._build_dataset_and_dataloader()
        self._build_optimizer()
        self._build_train_scheduler()
        self._wrap_and_prepare()

    @classmethod
    def train_on_cluster(cls) -> None:
        """Creates a DetSDTextualInversionTrainer instance on the cluster, drawing hyperparameters
        and other needed information from the Determined master.  Expects the `hyperparameters`
        section of the config to be organized into the following sections:
        - `model`: specifies which model weights to use.
        - `concepts`: all pertinent information about the to-be-added concepts.
        - `training`: controls details of the training process.
        - `inference`: optionally generate images during training using parameters specified here.
        """
        info = det.get_cluster_info()
        assert info is not None, "init_on_cluster() must be called on a Determined cluster."
        hparams = info.trial.hparams
        latest_checkpoint = info.latest_checkpoint

        # Instantiate the trainer and train.
        trainer = cls(
            **hparams["model"],
            **hparams["concepts"],
            **hparams["training"],
            **hparams["inference"],
        )

        trainer.logger.info("--------------- Starting Training ---------------")
        trainer.logger.info(f"Effective global batch size: {trainer.effective_global_batch_size}")
        trainer.logger.info(
            f"{'(Scaled) ' if trainer.scale_lr else ''}Learning rate: {trainer.learning_rate}"
        )
        trainer.logger.info(f"Train dataset size: {len(trainer.train_dataset)}")

        try:
            distributed = det.core.DistributedContext.from_torch_distributed()
        except KeyError:
            distributed = None

        with det.core.init(
            distributed=distributed, tensorboard_mode=det.core.TensorboardMode.MANUAL
        ) as core_context:
            if latest_checkpoint is not None:
                trainer._restore_latest_checkpoint(
                    core_context=core_context, latest_checkpoint=latest_checkpoint
                )

            # There will be a single op of len max_length, as defined in the searcher config.
            for op in core_context.searcher.operations():
                while trainer.steps_completed < op.length:
                    for batch_idx, batch in enumerate(trainer.train_dataloader):
                        # Use the accumulate method for efficient gradient accumulation.
                        with trainer.accelerator.accumulate(trainer.text_encoder):
                            trainer._train_one_batch(batch)
                        # An SGD step has been taken when trainer.accelerator.sync_gradients is True.
                        took_sgd_step = trainer.accelerator.sync_gradients
                        if took_sgd_step:
                            trainer.steps_completed += 1
                            trainer.logger.info(
                                f"Step {trainer.steps_completed} completed on batch {batch_idx}"
                            )

                            is_end_of_training = trainer.steps_completed == op.length
                            time_to_report = (
                                trainer.steps_completed % trainer.metric_report_freq == 0
                            )
                            time_to_ckpt = trainer.steps_completed % trainer.checkpoint_freq == 0

                            # Report metrics, checkpoint, and preempt as appropriate.
                            if is_end_of_training or time_to_report or time_to_ckpt:
                                trainer._report_train_metrics(core_context)
                                # report_progress for Web UI progress-bar rendering.
                                if trainer.accelerator.is_main_process:
                                    op.report_progress(trainer.steps_completed)
                            if is_end_of_training or time_to_ckpt:
                                trainer._save(core_context)
                                if core_context.preempt.should_preempt():
                                    return
                            if is_end_of_training:
                                break
                if trainer.accelerator.is_main_process:
                    # Report the final mean loss.
                    op.report_completed(trainer.last_mean_loss)

    def _train_one_batch(self, batch: TorchData) -> None:
        """Train on a single batch and update internal metrics."""
        self.text_encoder.train()
        # Convert sample images to latent space.
        prompts, img_tensors = batch
        with torch.no_grad():
            latent_dist = self.vae.encode(img_tensors).latent_dist
            latents = latent_dist.sample()

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
            # Add noise to the latents according to the noise magnitude at each timestep. This is
            # the forward diffusion process.
            noisy_latents = self.train_scheduler.add_noise(latents, noise, rand_timesteps)

        # Process the text for each batch.
        dummy_prompts = [self._replace_concepts_with_dummies(text) for text in prompts]
        dummy_prompts_noise_pred = self._get_noise_pred(
            text=dummy_prompts, noisy_latents=noisy_latents, timesteps=rand_timesteps
        )

        loss = F.mse_loss(dummy_prompts_noise_pred, noise)
        self.metrics_history["loss"].append(loss.item())
        self.accelerator.backward(loss)

        # A similar regularization for the encoder hidden states.
        if self.hidden_reg_weight:
            dummy_hidden_states = self._get_encoder_hidden_states(text=dummy_prompts)
            with torch.no_grad():
                intializer_prompts = [
                    self._replace_concepts_with_initializers(text) for text in prompts
                ]
                initializer_hidden_states = self._get_encoder_hidden_states(text=intializer_prompts)
            hidden_reg_loss = self.hidden_reg_weight * F.mse_loss(
                dummy_hidden_states, initializer_hidden_states
            )
            self.metrics_history["hidden_reg_loss"].append(hidden_reg_loss.item())
            self.accelerator.backward(hidden_reg_loss)

        self.optimizer.step()
        self.optimizer.zero_grad()

    def _get_noise_pred(
        self, text: List[str], noisy_latents: torch.Tensor, timesteps: torch.Tensor
    ) -> torch.Tensor:
        tokenized_text = self.tokenizer(
            text,
            padding="max_length",
            truncation=True,
            max_length=self.tokenizer.model_max_length,
            return_tensors="pt",
        ).input_ids
        encoder_hidden_states = self.text_encoder(tokenized_text).last_hidden_state
        noise_pred = self.unet(noisy_latents, timesteps, encoder_hidden_states).sample
        return noise_pred

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
            self.safety_checker = StableDiffusionSafetyChecker.from_pretrained(
                pretrained_model_name_or_path="CompVis/stable-diffusion-safety-checker"
            )
            self.feature_extractor = CLIPFeatureExtractor.from_pretrained(
                pretrained_model_name_or_path="openai/clip-vit-base-patch32"
            )

    def _add_new_tokens_and_update_embeddings(self) -> None:
        """Add new concept tokens to the tokenizer and update the corresponding embedding layers in
        the text encoder.
        """
        for concept_token, initializer_tokens in zip(self.concept_tokens, self.initializer_tokens):
            (
                non_special_initializer_ids,
                dummy_placeholder_ids,
                dummy_placeholder_tokens,
            ) = utils.add_new_tokens_to_tokenizer(
                concept_token=concept_token,
                initializer_tokens=initializer_tokens,
                tokenizer=self.tokenizer,
            )
            self.concept_to_initializer_tokens_map[concept_token] = initializer_tokens
            self.concept_to_non_special_initializer_ids_map[
                concept_token
            ] = non_special_initializer_ids
            self.concept_to_dummy_tokens_map[concept_token] = dummy_placeholder_tokens
            self.concept_to_dummy_ids_map[concept_token] = dummy_placeholder_ids
            self.logger.info(f"Added {len(dummy_placeholder_ids)} new tokens for {concept_token}.")

        # Create the dummy-to-initializer idx mapping and use the sorted values to generate the
        # updated embedding layer.
        dummy_id_to_initializer_id_map = {}
        for concept_token in self.concept_tokens:
            for dummy_id, initializer_id in zip(
                self.concept_to_dummy_ids_map[concept_token],
                self.concept_to_non_special_initializer_ids_map[concept_token],
            ):
                dummy_id_to_initializer_id_map[dummy_id] = initializer_id
        sorted_dummy_initializer_id_list = sorted(
            [(dummy_id, init_id) for dummy_id, init_id in dummy_id_to_initializer_id_map.items()]
        )
        idxs_to_copy = torch.tensor(
            [init_id for _, init_id in sorted_dummy_initializer_id_list],
            device=self.accelerator.device,
        )
        token_embedding_layer_weight_data = self._get_token_embedding_layer().weight.data
        copied_embedding_weights = (
            token_embedding_layer_weight_data[idxs_to_copy].clone().detach().contiguous()
        )

        # Update the embedding layer.
        original_embedding = self.text_encoder.text_model.embeddings.token_embedding
        self.text_encoder.text_model.embeddings.token_embedding = layers.ExtendedEmbedding(
            original_embedding=original_embedding,
            new_embedding_weights=copied_embedding_weights,
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

        for (
            param
        ) in self.text_encoder.text_model.embeddings.token_embedding.new_embedding.parameters():
            param.requires_grad = True

    def _replace_concepts_with_dummies(self, text: str) -> str:
        """Helper function for replacing concepts with dummy placeholders."""
        for concept_token, dummy_tokens in self.concept_to_dummy_tokens_map.items():
            text = text.replace(concept_token, dummy_tokens)
        return text

    def _replace_concepts_with_initializers(self, text: str) -> str:
        """Helper function for replacing concepts with their initializer tokens."""
        for concept_token, init_tokens in self.concept_to_initializer_tokens_map.items():
            text = text.replace(concept_token, init_tokens)
        return text

    def _build_dataset_and_dataloader(self) -> None:
        """Build the dataset and dataloader."""
        self.train_dataset = data.TextualInversionDataset(
            img_dirs=self.img_dirs,
            concept_tokens=self.concept_tokens,
            learnable_properties=self.learnable_properties,
            img_size=self.img_size,
            interpolation=self.interpolation,
            flip_p=self.flip_p,
            center_crop=self.center_crop,
            append_file_name_to_text=self.append_file_name_to_text,
            file_name_split_char=self.file_name_split_char,
        )
        self.train_dataloader = DataLoader(
            self.train_dataset, batch_size=self.train_batch_size, shuffle=True
        )

    def _build_optimizer(self) -> None:
        token_embedding_layer = self._get_token_embedding_layer()
        # Only optimize the newly-added embedding vectors.
        new_embedding_params = token_embedding_layer.new_embedding.parameters()
        self.optimizer = self._optim_dict[self.optimizer_name](
            new_embedding_params,
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
        self.text_encoder, self.optimizer, self.train_dataloader = self.accelerator.prepare(
            self.text_encoder, self.optimizer, self.train_dataloader
        )
        self.vae.to(self.accelerator.device)
        self.unet.to(self.accelerator.device)
        self.vae.eval()
        self.unet.eval()

    def _restore_latest_checkpoint(
        self, core_context: det.core.Context, latest_checkpoint: str
    ) -> None:
        """Restores the experiment state to the latest saved checkpoint, if it exists."""
        with core_context.checkpoint.restore_path(latest_checkpoint) as path:
            with self.accelerator.local_main_process_first():
                with open(path.joinpath("metadata.json"), "r") as f:
                    checkpoint_metadata_dict = json.load(f)
                    self.steps_completed = checkpoint_metadata_dict["steps_completed"]

                optimizer_state_dict = torch.load(
                    path.joinpath("optimizer_state_dict.pt"),
                    map_location=self.accelerator.device,
                )
                self.optimizer.load_state_dict(optimizer_state_dict)

                learned_embeddings_dict = torch.load(
                    path.joinpath("learned_embeddings_dict.pt"),
                    map_location=self.accelerator.device,
                )

                token_embedding_layer = self._get_token_embedding_layer()
                for concept_token, dummy_ids in self.concept_to_dummy_ids_map.items():
                    learned_embeddings = learned_embeddings_dict[concept_token][
                        "learned_embeddings"
                    ]
                    # Sanity check on length.
                    # TODO: replace with strict=True in zip after upgrade to py >= 3.10.
                    assert len(dummy_ids) == len(
                        learned_embeddings
                    ), 'Length of "dummy_ids" and "learned_embeddings" must be equal.'
                    id_offset = token_embedding_layer.original_embedding.weight.shape[0]
                    new_embedding_layer = token_embedding_layer.new_embedding
                    for dummy_id, tensor in zip(
                        dummy_ids,
                        learned_embeddings,
                    ):

                        new_embedding_layer.weight.data[dummy_id - id_offset] = tensor

    def _save(self, core_context: det.core.Context) -> None:
        """Save the training state, metadata, and any generated images."""
        self.logger.info(f"Saving checkpoint at step {self.steps_completed}.")
        self.accelerator.wait_for_everyone()
        if self.generate_training_images:
            self._build_pipeline()
            self._generate_and_write_tb_imgs(core_context)
        if self.accelerator.is_main_process:
            checkpoint_metadata_dict = {
                "steps_completed": self.steps_completed,
                "pretrained_model_name_or_path": self.pretrained_model_name_or_path,
            }
            with core_context.checkpoint.store_path(checkpoint_metadata_dict) as (path, storage_id):
                self._write_optimizer_state_dict_to_path(path)
                self._write_learned_embeddings_to_path(path)

    def _write_optimizer_state_dict_to_path(self, path: pathlib.Path) -> None:
        optimizer_state_dict = self.optimizer.state_dict()
        self.accelerator.save(optimizer_state_dict, path.joinpath("optimizer_state_dict.pt"))

    def _write_learned_embeddings_to_path(self, path: pathlib.Path) -> None:
        learned_embeddings_dict = {}
        for concept_token, dummy_ids in self.concept_to_dummy_ids_map.items():
            token_embedding_layer = self._get_token_embedding_layer()
            learned_embeddings = token_embedding_layer.new_embedding.weight.data.detach().cpu()
            initializer_tokens = self.concept_to_initializer_tokens_map[concept_token]
            learned_embeddings_dict[concept_token] = {
                "initializer_tokens": initializer_tokens,
                "learned_embeddings": learned_embeddings,
            }
        self.accelerator.save(learned_embeddings_dict, path.joinpath("learned_embeddings_dict.pt"))

    def _build_pipeline(self) -> None:
        inference_scheduler = defaults.NOISE_SCHEDULER_DICT[self.inference_scheduler_name]
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
        self.logger.info("Generating sample images")
        tb_dir = core_context.train.get_tensorboard_path()
        tb_writer = SummaryWriter(log_dir=tb_dir)
        for prompt in self.inference_prompts:
            dummy_prompt = self._replace_concepts_with_dummies(prompt)
            # Fix the seed for reproducibility, unique to each worker.
            generator = torch.Generator(device=self.accelerator.device).manual_seed(
                self.generator_seed + self.accelerator.process_index
            )
            # Set output_type to anything other than `pil` to get numpy arrays out.
            generated_img_array = []
            for _ in range(self.images_per_prompt):
                generated_img_array.append(
                    self.pipeline(
                        prompt=dummy_prompt,
                        num_inference_steps=self.num_inference_steps,
                        guidance_scale=self.guidance_scale,
                        generator=generator,
                        output_type="np",
                    ).images[0]
                )
            generated_img_t = torch.from_numpy(np.concatenate(generated_img_array, axis=1))
            # Gather all images and upload via the chief. In tensorboard images, different rows
            # correspond to different workers, and images_per_prompt sets the number of columns.
            all_generated_img_ts = self.accelerator.gather(
                generated_img_t.to(self.accelerator.device)
            )
            if self.accelerator.is_main_process:
                tb_writer.add_image(
                    prompt,
                    img_tensor=all_generated_img_ts,
                    global_step=self.steps_completed,
                    dataformats="HWC",
                )
        if self.accelerator.is_main_process:
            tb_writer.flush()  # Ensure all images are written to disk.
            core_context.train.upload_tensorboard_files()

    def _report_train_metrics(self, core_context: det.core.Context) -> None:
        self.accelerator.wait_for_everyone()
        # Currently only tracking the loss in self.metrics_history, but the below code generalizes
        # to arbitrarily many tracked metrics.
        mean_metrics = {
            metric_name: torch.tensor(metric_values, device=self.accelerator.device).mean()
            for metric_name, metric_values in self.metrics_history.items()
        }
        # reduction='mean' seems to return the sum rather than the mean.
        # TODO: Verify this apparent problem.
        reduced_mean_metrics = {
            metric_name: self.accelerator.reduce(mean_metric_value, reduction="sum").item()
            / self.accelerator.num_processes
            for metric_name, mean_metric_value in mean_metrics.items()
        }
        self.last_mean_loss = reduced_mean_metrics["loss"]
        # Reset the local metrics history
        self.metrics_history = {metric_name: [] for metric_name in self.metrics_history}
        if self.accelerator.is_main_process:
            core_context.train.report_training_metrics(
                steps_completed=self.steps_completed,
                metrics=reduced_mean_metrics,
            )

    def _get_token_embedding_layer(self) -> nn.Module:
        try:
            token_embedding_layer = self.text_encoder.module.text_model.embeddings.token_embedding
        except AttributeError:
            token_embedding_layer = self.text_encoder.text_model.embeddings.token_embedding
        return token_embedding_layer

    def _get_all_concept_embeddings(
        self, dummy_or_initializer: Literal["dummy", "initializer"]
    ) -> torch.Tensor:
        """Returns the embedding vectors for all added concepts, either using their initializer
        representation or their trained dummy representation.
        """
        concept_replace_fn_dict = {
            "dummy": self._replace_concepts_with_dummies,
            "initializer": self._replace_concepts_with_initializers,
        }
        assert (
            dummy_or_initializer in concept_replace_fn_dict
        ), 'dummy_or_initializer must be "dummy" or "initializer"'
        concept_replace_fn = concept_replace_fn_dict[dummy_or_initializer]

        token_embedding_layer = self._get_token_embedding_layer()
        all_concept_tokens = " ".join(list(self.concept_tokens))
        all_replaced_concept_tokens = concept_replace_fn(all_concept_tokens)
        all_replaced_concept_tokens_t = torch.tensor(
            self.tokenizer.encode(all_replaced_concept_tokens, add_special_tokens=False),
            device=self.accelerator.device,
        )
        all_concept_embeddings = token_embedding_layer(all_replaced_concept_tokens_t)
        return all_concept_embeddings
