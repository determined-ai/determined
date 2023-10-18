#!/usr/bin/env python3

import torch

from determined.experimental import core_v2


def main() -> None:
    core_v2.init(
        defaults=core_v2.DefaultConfig(
            name="pytorch-profiler-sync-test",
        ),
    )

    with torch.profiler.profile(
        activities=[torch.profiler.ProfilerActivity.CPU],
        with_flops=True,
        on_trace_ready=torch.profiler.tensorboard_trace_handler(
            str(core_v2.train.get_tensorboard_path())
        ),
    ):
        state = torch.rand(100, 100)
        for _i in range(1000):
            state = torch.matmul(state, torch.rand(100, 100))
            norm = torch.linalg.norm(state)
            state = state / norm

    core_v2.close()


if __name__ == "__main__":
    main()
