import argparse
from typing import Any

import tensorflow as tf
from tensorflow.keras import backend as K
from tensorflow.python.keras.optimizer_v2 import optimizer_v2

try:
    import horovod.tensorflow.keras as hvd
except ModuleNotFoundError:
    pass


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
    grads = [tf.constant([hvd.rank() * constant_multiplier])]
    op = hvd_optimizer._allreduce(grads)
    for idx in range(10):
        value = session.run(op)[0][0]
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
    grads_and_vars = [(tf.constant([hvd.rank() * constant_multiplier]), None)]
    for idx in range(10):
        grads = hvd_optimizer._aggregate_gradients(grads_and_vars)
        value = grads[0][0].numpy()
        expected_value = compute_expected_value(
            idx, aggregation_frequency, constant_multiplier, average_aggregated_gradients, True
        )
        assert expected_value == value


def compute_expected_value(
    batch_id: int,
    aggregation_frequency: int,
    multiplier: float,
    average_aggregated_gradient: bool,
    tf2: bool,
) -> float:
    """
    Compute the expected value based on how we are aggregating gradients.
    """
    gradients_aggregated = (batch_id + 1) % aggregation_frequency == 0
    if gradients_aggregated:
        all_reduced_grads = 0.0
        for _ in range(aggregation_frequency):
            grads_for_batch = 0.0
            for rank in range(hvd.size()):
                grads_for_batch += rank * multiplier
            if average_aggregated_gradient:
                grads_for_batch /= float(aggregation_frequency)
            all_reduced_grads += grads_for_batch / float(hvd.size())
        return all_reduced_grads
    else:
        non_aggregated_grads = hvd.rank() * multiplier
        if tf2:
            # In Tf2 we return the sum of the locally aggregated gradients.
            non_aggregated_grads *= (batch_id + 1) % aggregation_frequency
        return non_aggregated_grads


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--tf1", action="store_true")
    parser.add_argument(
        "--aggregation-frequency", dest="aggregation_frequency", default=0, type=int
    )
    parser.add_argument("--average-aggregated-gradients", action="store_true")
    args = parser.parse_args()

    hvd.init()
    if args.tf1:
        test_tf_1(args.aggregation_frequency, args.average_aggregated_gradients)
    else:
        test_tf_2(args.aggregation_frequency, args.average_aggregated_gradients)


if __name__ == "__main__":
    main()
