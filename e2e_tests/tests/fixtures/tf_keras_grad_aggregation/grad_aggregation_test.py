import argparse
from typing import Any

import tensorflow as tf
from tensorflow.keras import backend as K
from tensorflow.python.keras.optimizer_v2 import optimizer_v2

try:
    import horovod.tensorflow.keras as hvd
except ModuleNotFoundError:
    pass


parser = argparse.ArgumentParser()
parser.add_argument("--tf1", action="store_true")
parser.add_argument("--aggregation-frequency", dest="aggregation_frequency", default=0, type=int)
parser.add_argument("--average-aggregated-gradients", action="store_true")
args = parser.parse_args()


class CustomOptimizer(optimizer_v2.OptimizerV2):
    def get_config(self) -> Any:
        config = super(CustomOptimizer, self).get_config()
        return config


def test_tf_1(aggregation_frequency: int, average_aggregated_gradients: bool) -> None:
    config = tf.compat.v1.ConfigProto()
    config.gpu_options.allow_growth = True
    config.gpu_options.visible_device_list = str(hvd.local_rank())
    K.set_session(tf.Session(config=config))
    session = tf.compat.v1.keras.backend.get_session(op_input_list=())

    hvd_optimizer = hvd.DistributedOptimizer(
        optimizer=CustomOptimizer("mine"),
        aggregation_frequency=aggregation_frequency,
        average_aggregated_gradients=average_aggregated_gradients,
    )

    constant_multiplier = 4.0
    op = hvd_optimizer._allreduce([tf.constant([[hvd.rank() * constant_multiplier]])])
    for idx in range(10):
        value = session.run(op)[0][0][0]
        expected_value = compute_expected_value(
            idx, aggregation_frequency, constant_multiplier, average_aggregated_gradients, False
        )
        assert expected_value == value


def test_tf_2(aggregation_frequency: int, average_aggregated_gradients: bool) -> None:
    gpus = tf.config.experimental.list_physical_devices("GPU")
    for gpu in gpus:
        tf.config.experimental.set_memory_growth(gpu, True)
    if gpus:
        tf.config.experimental.set_visible_devices(gpus[hvd.local_rank()], "GPU")

    hvd_optimizer = hvd.DistributedOptimizer(
        optimizer=CustomOptimizer("mine"),
        aggregation_frequency=aggregation_frequency,
        average_aggregated_gradients=average_aggregated_gradients,
    )

    constant_multiplier = 4.0
    for idx in range(10):
        grads = hvd_optimizer._aggregate_gradients(
            [(tf.constant([[hvd.rank() * constant_multiplier]]), None)]
        )
        value = grads[0][0][0].numpy()
        expected_value = compute_expected_value(
            idx, aggregation_frequency, constant_multiplier, average_aggregated_gradients, True
        )
        assert expected_value == value


def compute_expected_value(
    idx: int,
    aggregation_frequency: int,
    multiplier: float,
    average_aggregated_gradient: bool,
    tf2: bool,
) -> float:
    idx += 1
    if idx % aggregation_frequency == 0:
        reduction_sum = 0.0
        for _ in range(aggregation_frequency):
            for rank in range(hvd.size()):
                reduction_sum += rank * multiplier / float(hvd.size())
        if average_aggregated_gradient:
            reduction_sum /= float(aggregation_frequency)
        return reduction_sum
    else:
        result = hvd.rank() * multiplier
        if tf2:
            result *= idx % aggregation_frequency
        return result


def main() -> None:
    hvd.init()
    if args.tf1:
        test_tf_1(args.aggregation_frequency, args.average_aggregated_gradients)
    else:
        test_tf_2(args.aggregation_frequency, args.average_aggregated_gradients)


if __name__ == "__main__":
    main()
