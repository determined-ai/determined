:orphan:

**Breaking Changes**

-  API: Remove model_hub library from determined.

   -  Starting with this release, MMDetTrial and BaseTransformerTrial are removed. HuggingFace users
      should look at provided `HuggingFace TrainerAPI
      examples<https://github.com/determined-ai/determined/tree/main/examples/hf_trainer_api>_`,
      which use a custom callback in place of BaseTransformerTrial. Users of MMDetTrial can refer to
      :ref:`Core API <api-core-ug>`.
