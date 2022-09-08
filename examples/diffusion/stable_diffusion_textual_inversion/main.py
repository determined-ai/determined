"""Following the SD textual inversion notebook example from HF
https://github.com/huggingface/notebooks/blob/main/diffusers/sd_textual_inversion_training.ipynb"""

import attrdict
import determined as det
import logging
import os

import trainer

logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)


if __name__ == "__main__":
    HF_AUTH_TOKEN = os.environ["HF_AUTH_TOKEN"]
    info = det.get_cluster_info()
    hparams = attrdict.AttrDict(info.trial.hparams)
    latest_checkpoint = info.latest_checkpoint

    trainer = trainer.TextualInversionTrainer(
        use_auth_token=HF_AUTH_TOKEN,
        latest_checkpoint=latest_checkpoint,
        **hparams.model,
        **hparams.data,
        **hparams.trainer
    )
    trainer.train()

    # print(80 * "=", "INFERENCE", 80 * "=", sep="\n")
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
