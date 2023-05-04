DS_CONFIG_PATH = "ds_config.json"

AUTOTUNINGS = {"tune", "run"}
METRICS = {"latency", "throughput", "FLOPS_per_gpu"}
TASKS = {"timm", "hf_glue", "hf_clm"}
TUNER_TYPES = {"gridsearch", "random", "model_based"}

DEFAULT_MODELS = {
    "timm": ["efficientnet_b0"],
    "hf_glue": ["distilbert-base-uncased"],
    "hf_clm": ["gpt2"],
}
DEFAULT_DATASETS = {"timm": "mini_imagenet", "hf_glue": "mnli", "hf_clm": "wikitext"}

FLOPS_PROFILER_OUTPUT_PATH = "/run/determined/workdir/flops_profiler_output.txt"

AUTOTUNING_START_PROFILE_STEP = 3
AUTOTUNING_END_PROFILE_STEP = 5
MAX_STEPS = 10
FLOPS_PROFILE_STEP = AUTOTUNING_END_PROFILE_STEP
