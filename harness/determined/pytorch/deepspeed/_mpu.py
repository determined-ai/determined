from dataclasses import dataclass

from deepspeed.runtime.pipe import topology

from determined import core


@dataclass
class ModelParallelUnit:
    """
    This class contains the functions we expect in order to accurately carry out parallel training.
    For custom model parallel training, you need to subclass and override the functions before
    passing it to the :class:`~determined.pytorch.deepspeed.DeepSpeedTrialContext` by calling
    ``context.wrap_mpu(mpu)``.
    """

    data_parallel_rank: int
    data_parallel_world_size: int
    # Whether the rank should return metrics from training and evaluation steps.
    # This is usually true for ranks in the last stage of pipeline parallel or
    # model parallel training.
    should_report_metrics: bool
    # Whether the rank should build a data loader.
    # This is usually true for ranks in the first or last stage of
    # pipeline parallel or model parallel training.
    should_build_data_loader: bool


def make_data_parallel_mpu(dist_context: core.DistributedContext) -> ModelParallelUnit:
    return ModelParallelUnit(
        data_parallel_rank=dist_context.get_rank(),
        data_parallel_world_size=dist_context.get_size(),
        should_report_metrics=True,
        should_build_data_loader=True,
    )


def make_deepspeed_mpu(topology: topology.PipelineParallelGrid) -> ModelParallelUnit:
    is_first_pipeline_stage = topology.get_pipe_parallel_rank() == 0
    last_stage = topology.get_pipe_parallel_world_size() - 1
    is_last_pipeline_stage = topology.get_pipe_parallel_rank() == last_stage
    should_build_data_loader = topology.get_slice_parallel_rank() == 0 and (
        is_first_pipeline_stage or is_last_pipeline_stage
    )
    return ModelParallelUnit(
        data_parallel_rank=topology.get_data_parallel_rank(),
        data_parallel_world_size=topology.get_data_parallel_world_size(),
        should_report_metrics=True,
        should_build_data_loader=should_build_data_loader,
    )
