import json
import logging
import os
import pathlib
from typing import Any, Dict, List, Literal, Optional, Sequence, Tuple, Union

import determined as det
import torch
from determined.experimental import client
from diffusers.pipelines.stable_diffusion import (
    StableDiffusionPipeline,
    StableDiffusionPipelineOutput,
)
from torch.utils.tensorboard import SummaryWriter
from torchvision.transforms.functional import pil_to_tensor

from detsd import utils, defaults


class DetSDTextualInversionPipeline:
    """Class for generating images from a Stable Diffusion checkpoint trained using Determined
    AI. Initialize with no arguments in order to run plain Stable Diffusion without any trained
    textual inversion embeddings. Can optionally be run on a Determined cluster through the
    .generate_on_cluster() method for large-scale generation. Only intended for use with a GPU.
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
        use_fp16: bool = True,
        disable_progress_bar: bool = False,
    ) -> None:
        # We assume that the Huggingface User Access token has been stored as the HF_AUTH_TOKEN
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
            other_scheduler_kwargs or defaults.DEFAULT_SCHEDULER_KWARGS_DICT[scheduler_name]
        )
        self.pretrained_model_name_or_path = pretrained_model_name_or_path
        self.device = device
        self.use_fp16 = use_fp16
        self.disable_progress_bar = disable_progress_bar

        scheduler_kwargs = {
            "beta_start": self.beta_start,
            "beta_end": self.beta_end,
            "beta_schedule": self.beta_schedule,
            **self.other_scheduler_kwargs,
        }
        self.scheduler = defaults.NOISE_SCHEDULER_DICT[self.scheduler_name](**scheduler_kwargs)

        # The below attrs are non-trivially instantiated as necessary through appropriate methods.
        self.all_checkpoint_dirs = []
        self.learned_embeddings_dict = {}
        self.concept_to_dummy_tokens_map = {}
        self.all_added_concepts = []

        self.steps_completed = None
        self.num_generated_imgs = None
        self.image_history = []

        self._build_hf_pipeline(disable_progress_bar=self.disable_progress_bar)

    @classmethod
    def generate_on_cluster(cls) -> None:
        """Creates a DetSDTextualInversionPipeline instance on the cluster, drawing hyperparameters
        and other needed information from the Determined master, and then generates images. Expects
        the `hyperparameters` section of the config to be broken into the following sections:
        - `batch_size`: The batch size to use for generation.
        - `main_process_generator_seed`: an integer specifying the seed used by the chief worker for
        generation. Other workers add their process index to this value.
        - `save_freq`: an integer specifying how often to write images to tensorboard.
        - `pipeline`: containing all __init__ args
        - `uuids`: a (possibly empty) array of any checkpoint UUIDs which are to be loaded into
        the pipeline.
        - `local_checkpoint_paths`: a (possibly empty) array of any checkpoint paths which are to be
        loaded into the pipeline.
        - `call_kwargs`: all arguments which are to be passed to the `__call__` method.
        """
        info = det.get_cluster_info()
        assert info is not None, "generate_on_cluster() must be called on a Determined cluster."
        hparams = info.trial.hparams
        trial_id = info.trial.trial_id
        latest_checkpoint = info.latest_checkpoint

        # Extract relevant groups from hparams.
        batch_size = hparams["batch_size"]
        main_process_generator_seed = hparams["main_process_generator_seed"]
        save_freq = hparams["save_freq"]
        pipeline_init_kwargs = hparams["pipeline"]
        uuid_list = hparams["uuids"]
        local_checkpoint_paths_list = hparams["local_checkpoint_paths"]
        call_kwargs = hparams["call_kwargs"]

        logger = logging.getLogger(__name__)

        # Get the distributed context, as needed.
        try:
            distributed = det.core.DistributedContext.from_torch_distributed()
        except KeyError:
            distributed = None

        with det.core.init(
            distributed=distributed, tensorboard_mode=det.core.TensorboardMode.MANUAL
        ) as core_context:
            # Get worker data.
            process_index = core_context.distributed.get_rank()
            is_main_process = process_index == 0
            local_process_index = core_context.distributed.get_local_rank()
            is_local_main_process = local_process_index == 0

            device = f"cuda:{process_index}" if distributed is not None else "cuda"
            pipeline_init_kwargs["device"] = device

            # Instantiate the pipeline and load in any checkpoints by uuid.
            pipeline = cls(**pipeline_init_kwargs)
            # Only the local chief worker performs the download.
            if uuid_list:
                if is_local_main_process:
                    paths = pipeline.load_from_uuids(uuid_list)
                else:
                    paths = None
                paths = core_context.distributed.broadcast_local(paths)
                if not is_local_main_process:
                    for path in paths:
                        pipeline.load_from_checkpoint_dir(path)

            if local_checkpoint_paths_list:
                for path_str in local_checkpoint_paths_list:
                    path = pathlib.Path(path_str)
                    pipeline.load_from_checkpoint_dir(
                        checkpoint_dir=path.parent, learned_embeddings_filename=path.name
                    )

            # Create the Tensorboard writer.
            tb_dir = core_context.train.get_tensorboard_path()
            tb_writer = SummaryWriter(log_dir=tb_dir)
            # Include relevant __call__ args in the tensorboard tag.
            important_generation_params = ("prompt", "guidance_scale", "num_inference_steps")
            tb_tag = "/".join([f"{k}: {call_kwargs[k]}" for k in important_generation_params])

            # Use unique seeds, to avoid repeated images, and add the corresponding generator to the
            # call_kwargs.
            seed = main_process_generator_seed + process_index
            generator = torch.Generator(device=pipeline.device).manual_seed(seed)
            call_kwargs["generator"] = generator
            # Add seed information to the tensorboard tag.
            tb_tag += f"/seed: {seed}"
            # Update the call_kwargs with the batch size, if needed.
            if batch_size > 1:
                call_kwargs["prompt"] = [call_kwargs["prompt"]] * batch_size

            pipeline.steps_completed = 0
            pipeline.num_generated_imgs = 0
            pipeline.image_history = []

            # Restore from a checkpoint, if necessary.
            if latest_checkpoint is not None:
                pipeline._restore_latest_checkpoint(
                    core_context=core_context,
                    latest_checkpoint=latest_checkpoint,
                    generator=generator,
                    trial_id=trial_id,
                )
                if is_main_process:
                    logger.info(f"Resumed from checkpoint at step {pipeline.steps_completed}")

            if is_main_process:
                logger.info("--------------- Generating Images ---------------")

            # There will be a single op of len max_length, as defined in the searcher config.
            for op in core_context.searcher.operations():
                while pipeline.steps_completed < op.length:
                    pipeline.image_history.extend(pipeline(**call_kwargs).images)
                    pipeline.steps_completed += 1

                    # Write to tensorboard and checkpoint at the specified frequency.
                    if (
                        pipeline.steps_completed % save_freq == 0
                        or pipeline.steps_completed == op.length
                    ):
                        pipeline._write_tb_imgs(
                            core_context=core_context, tb_writer=tb_writer, tb_tag=tb_tag
                        )

                        # Checkpointing.
                        devices_and_generators = core_context.distributed.gather(
                            (device, generator.get_state())
                        )
                        if is_main_process:
                            logger.info(f"Saving at step {pipeline.steps_completed}")
                            # Save the state of the generators as the checkpoint.
                            pipeline._save(
                                core_context=core_context,
                                devices_and_generators=devices_and_generators,
                                trial_id=trial_id,
                            )
                            op.report_progress(pipeline.steps_completed)

                        # Only preempt after a checkpoint has been saved.
                        if core_context.preempt.should_preempt():
                            return

                if is_main_process:
                    # Report zero upon completion.
                    op.report_completed(0)

    def load_from_checkpoint_dir(
        self,
        checkpoint_dir: Union[str, pathlib.Path],
        learned_embeddings_filename: Optional[str] = None,
    ) -> None:
        """Load concepts from a checkpoint directory which is expected contain a file with the name
        matching the provided `learned_embeddings_filename` or, if omitted, the __init__
        `learned_embeddings_filename`. The file is expected to contain a dictionary
        whose keys are the `concept_str`s and whose values are dictionaries containing an
        `initializer_token` key and a `learned_embeddings` whose corresponding values are the
        initializer string and learned embedding tensors, respectively.
        """
        if not checkpoint_dir:
            return

        learned_embeddings_filename = (
            learned_embeddings_filename or self.learned_embeddings_filename
        )
        if isinstance(checkpoint_dir, str):
            checkpoint_dir = pathlib.Path(checkpoint_dir)
        learned_embeddings_dict = torch.load(checkpoint_dir.joinpath(learned_embeddings_filename))

        # Update embedding matrix and attrs.
        for concept_str, embedding_dict in learned_embeddings_dict.items():
            if concept_str in self.learned_embeddings_dict:
                raise ValueError(f"Checkpoint concept conflict: {concept_str} already exists.")
            initializer_strs = embedding_dict["initializer_strs"]
            learned_embeddings = embedding_dict["learned_embeddings"]
            (_, dummy_placeholder_ids, dummy_placeholder_strs,) = utils.add_new_tokens_to_tokenizer(
                concept_str=concept_str,
                initializer_strs=initializer_strs,
                tokenizer=self.hf_pipeline.tokenizer,
            )

            self.hf_pipeline.text_encoder.resize_token_embeddings(len(self.hf_pipeline.tokenizer))
            token_embeddings = self.hf_pipeline.text_encoder.get_input_embeddings().weight.data
            # Sanity check on length.
            assert len(dummy_placeholder_ids) == len(
                learned_embeddings
            ), "dummy_placeholder_ids and learned_embeddings must have the same length"
            for d_id, tensor in zip(dummy_placeholder_ids, learned_embeddings):
                token_embeddings[d_id] = tensor
            self.learned_embeddings_dict[concept_str] = embedding_dict
            self.all_added_concepts.append(concept_str)
            self.concept_to_dummy_tokens_map[concept_str] = dummy_placeholder_strs
        self.all_checkpoint_dirs.append(checkpoint_dir)

    def load_from_uuids(
        self,
        uuids: Union[str, Sequence[str]],
    ) -> List[pathlib.Path]:
        """Load concepts from one or more Determined checkpoint uuids and returns a list of all
        downloaded checkpoint paths. Must be logged into the Determined cluster to use this method.
        If not logged-in, call determined.experimental.client.login first.
        """
        if isinstance(uuids, str):
            uuids = [uuids]
        checkpoint_paths = []
        for u in uuids:
            checkpoint = client.get_checkpoint(u)
            checkpoint_paths.append(pathlib.Path(checkpoint.download()))
        for path in checkpoint_paths:
            self.load_from_checkpoint_dir(path)
        return checkpoint_paths

    def _build_hf_pipeline(self, disable_progress_bar: bool = True) -> None:
        revision = "fp16" if self.use_fp16 else "main"
        torch_dtype = torch.float16 if self.use_fp16 else None
        self.hf_pipeline = StableDiffusionPipeline.from_pretrained(
            pretrained_model_name_or_path=self.pretrained_model_name_or_path,
            scheduler=self.scheduler,
            use_auth_token=self.use_auth_token,
            revision=revision,
            torch_dtype=torch_dtype,
        ).to(self.device)
        # Disable the progress bar.
        if disable_progress_bar:
            self.hf_pipeline.set_progress_bar_config(disable=True)

    def _replace_concepts_with_dummies(self, text: str) -> str:
        for concept_str, dummy_tokens in self.concept_to_dummy_tokens_map.items():
            text = text.replace(concept_str, dummy_tokens)
        return text

    def __call__(self, **kwargs) -> StableDiffusionPipelineOutput:
        """Return the results of the HF pipeline's StableDiffusionPipeline __call__ method. Only
        accepts keyword arguments. See the HF docs for information on all available args.
        """
        if isinstance(kwargs["prompt"], str):
            kwargs["prompt"] = self._replace_concepts_with_dummies(kwargs["prompt"])
        else:
            kwargs["prompt"] = [self._replace_concepts_with_dummies(p) for p in kwargs["prompt"]]
        output = self.hf_pipeline(**kwargs)
        return output

    def __repr__(self) -> str:
        attr_dict = {
            "scheduler_name": self.scheduler_name,
            "beta_start": self.beta_start,
            "beta_end": self.beta_end,
            "beta_schedule": self.beta_schedule,
            "other_scheduler_kwargs": self.other_scheduler_kwargs,
            "pretrained_model_name_or_path": self.pretrained_model_name_or_path,
            "device": self.device,
            "use_fp16": self.use_fp16,
            "all_added_concepts": self.all_added_concepts,
        }
        attr_dict_str = ", ".join([f"{key}={value}" for key, value in attr_dict.items()])
        return f"{self.__class__.__name__}({attr_dict_str})"

    def _write_tb_imgs(
        self, core_context: det.core.Context, tb_writer: SummaryWriter, tb_tag: str
    ) -> None:
        for idx, img in enumerate(self.image_history):
            img_t = pil_to_tensor(img)
            global_step = self.num_generated_imgs + idx
            tb_writer.add_image(
                tb_tag,
                img_tensor=img_t,
                global_step=global_step,
            )
        tb_writer.flush()
        core_context.train.upload_tensorboard_files()
        self.num_generated_imgs += len(self.image_history)
        self.image_history = []

    def _restore_latest_checkpoint(
        self,
        core_context: det.core.Context,
        latest_checkpoint: str,
        generator: torch.Generator,
        trial_id: int,
    ) -> None:
        with core_context.checkpoint.restore_path(latest_checkpoint) as path:
            # Restore the state per the docs:
            # https://docs.determined.ai/latest/training/apis-howto/api-core/checkpoints.html
            with open(path.joinpath("metadata.json"), "r") as f:
                checkpoint_metadata_dict = json.load(f)
                if trial_id == checkpoint_metadata_dict["trial_id"]:
                    self.steps_completed = checkpoint_metadata_dict["steps_completed"]
                else:
                    self.steps_completed = 0
                generator_state_dict = torch.load(
                    path.joinpath("generator_state_dict.pt"),
                )
                self.num_generated_imgs = checkpoint_metadata_dict["num_generated_imgs"]
                generator.set_state(generator_state_dict[self.device])
                self.image_history = torch.load(path.joinpath("self.image_history.pt"))

    def _save(
        self,
        core_context: det.core.Context,
        devices_and_generators: List[Tuple[str, torch.ByteTensor]],
        trial_id: int,
    ) -> None:
        checkpoint_metadata_dict = {
            "steps_completed": self.steps_completed,
            "num_generated_imgs": self.num_generated_imgs,
            "trial_id": trial_id,
        }
        with core_context.checkpoint.store_path(checkpoint_metadata_dict) as (
            path,
            storage_id,
        ):
            generator_state_dict = {device: state for device, state in devices_and_generators}
            torch.save(generator_state_dict, path.joinpath("generator_state_dict.pt"))
            torch.save(
                self.image_history,
                path.joinpath("self.image_history.pt"),
            )
