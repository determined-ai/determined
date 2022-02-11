import abc
from typing import cast

from deepspeed.runtime.pipe import topology

from determined import _core


class ModelParallelUnit:
    """
    This class contains the functions we expect in order to accurately carry out parallel training.
    For custom model parallel training, you need to subclass and override the functions before
    passing it to the :class:`~determined.pytorch.deepspeed.DeepSpeedTrialContext` by calling
    ``context.wrap_mpu(mpu)``.
    """

    @abc.abstractmethod
    def __init__(self) -> None:
        pass

    @abc.abstractmethod
    def get_data_parallel_rank(self) -> int:
        pass

    @abc.abstractmethod
    def get_data_parallel_world_size(self) -> int:
        pass

    @abc.abstractmethod
    def should_report_metrics(self) -> bool:
        # Whether the rank should return metrics from training and evaluation steps.
        # This is usually true for ranks in the last stage of pipeline parallel or
        # model parallel training.
        pass

    @abc.abstractmethod
    def should_build_data_loader(self) -> bool:
        # Whether the rank should build a data loader.
        # This is usually true for ranks in the first or last stage of
        # pipeline parallel or model parallel training.
        pass


class DeterminedModelParallelUnit(ModelParallelUnit):
    """
    ModelParallelUnit for standard data parallel training.  Data parallel information derived from
    Determined's :class:`~determined._core.DistributedContext`.
    """

    def __init__(self, dist_context: _core.DistributedContext):
        self.dist_context = dist_context

    def get_data_parallel_rank(self) -> int:
        # Which pipeline this rank resides in.
        return self.dist_context.get_rank()

    def get_data_parallel_world_size(self) -> int:
        # The number of pipelines.
        return self.dist_context.get_size()

    def should_report_metrics(self) -> bool:
        # Whether the rank should return metrics from training and evaluation steps.
        # This is usually true for ranks in the last stage of pipeline parallel or
        # model parallel training.
        return True

    def should_build_data_loader(self) -> bool:
        # Whether the rank should build a data loader.
        # This is usually true for ranks in the first or last stage of
        # pipeline parallel or model parallel training.
        return True


class DeepSpeedMPU(ModelParallelUnit):
    """
    ModelParallelUnit for pipeline parallelism when using DeepSpeed's PipelineEngine.
    Data and model parallel information derived from the model engine's mpu.
    """

    def __init__(self, mpu: topology.PipelineParallelGrid) -> None:
        self.mpu = mpu

    def get_data_parallel_rank(self) -> int:
        return cast(int, self.mpu.get_data_parallel_rank())

    def get_data_parallel_world_size(self) -> int:
        return cast(int, self.mpu.get_data_parallel_world_size())

    def is_first_pipeline_stage(self) -> bool:
        return cast(int, self.mpu.get_pipe_parallel_rank()) == 0

    def is_last_pipeline_stage(self) -> bool:
        return cast(int, self.mpu.get_pipe_parallel_rank()) == (
            cast(int, self.mpu.get_pipe_parallel_world_size()) - 1
        )

    def should_report_metrics(self) -> bool:
        return self.is_last_pipeline_stage()

    def should_build_data_loader(self) -> bool:
        return cast(int, self.mpu.get_slice_parallel_rank()) == 0 and (
            self.is_first_pipeline_stage() or self.is_last_pipeline_stage()
        )
