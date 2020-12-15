import argparse
import math
from distutils.version import LooseVersion
from typing import Any

import tensorflow as tf
from tensorflow.keras import backend as K
from tensorflow.python.keras.optimizer_v2 import optimizer_v2

try:
    import horovod.tensorflow.keras as hvd
except ModuleNotFoundError:
    pass


_PRE_TF_2_4_0 = LooseVersion(tf.__version__) < LooseVersion("2.4.0")


class CustomOptimizer(optimizer_v2.OptimizerV2):
    def get_config(self) -> Any:
        config = super(CustomOptimizer, self).get_config()
        return config

    def _create_slots(self, var_list: Any) -> None:
        pass

    def _resource_apply_dense(self, grad: Any, var: Any, apply_state: Any = None) -> Any:
        return var.assign_add(grad)


def check_tf_1(aggregation_frequency: int, average_aggregated_gradients: bool) -> None:
    config = tf.compat.v1.ConfigProto()
    config.gpu_options.allow_growth = True
    config.gpu_options.visible_device_list = str(hvd.local_rank())

    K.set_session(tf.Session(config=config))
    session = tf.compat.v1.keras.backend.get_session(op_input_list=())

    hvd_optimizer = hvd.DistributedOptimizer(
        optimizer=CustomOptimizer("mine"),
        backward_passes_per_step=aggregation_frequency,
        average_aggregated_gradients=average_aggregated_gradients,
    )
    iterations = hvd_optimizer.iterations
    session.run(iterations.initializer)

    grads = [tf.constant([float(hvd.rank())])]
    variables = [tf.Variable([0.0])]
    session.run(variables[0].initializer)

    allreduce_op = hvd_optimizer._allreduce(grads)
    grads_and_vars = [(allreduce_op[0], variables[0])]
    apply_grads_op = hvd_optimizer.apply_gradients(grads_and_vars)

    for idx in range(10):
        _ = session.run(apply_grads_op)

        expected_value = compute_expected_variable_value(
            idx, aggregation_frequency, average_aggregated_gradients
        )

        assert idx + 1 == session.run(hvd_optimizer.iterations)
        assert expected_value == session.run(variables[0].read_value())


def check_tf_2(aggregation_frequency: int, average_aggregated_gradients: bool) -> None:
    gpus = tf.config.experimental.list_physical_devices("GPU")
    for gpu in gpus:
        tf.config.experimental.set_memory_growth(gpu, True)
    if gpus:
        tf.config.experimental.set_visible_devices(gpus[hvd.local_rank()], "GPU")

    hvd_optimizer = hvd.DistributedOptimizer(
        optimizer=CustomOptimizer("mine"),
        backward_passes_per_step=aggregation_frequency,
        average_aggregated_gradients=average_aggregated_gradients,
    )
    _ = hvd_optimizer.iterations

    gradients = [tf.constant([float(hvd.rank())])]
    variables = [tf.Variable([0.0])]
    for idx in range(10):
        if _PRE_TF_2_4_0:
            # In TF < 2.4 `_aggregate_gradients()` is called outside of `apply_gradients()`.
            updated_gradients = hvd_optimizer._aggregate_gradients(zip(gradients, variables))
            hvd_optimizer.apply_gradients(
                zip(updated_gradients, variables), experimental_aggregate_gradients=False
            )
        else:
            hvd_optimizer.apply_gradients(zip(gradients, variables))

        updated_variable_value = variables[0][0].numpy()
        expected_value = compute_expected_variable_value(
            idx, aggregation_frequency, average_aggregated_gradients
        )

        assert expected_value == updated_variable_value
        assert idx + 1 == hvd_optimizer.iterations.numpy()


def compute_expected_variable_value(
    batch_id: int,
    aggregation_frequency: int,
    average_aggregated_gradient: bool,
) -> float:
    """
    Computes the expected current value of variables based on how we are aggregating gradients.
    """
    aggregations_completed = math.floor((batch_id + 1) / aggregation_frequency)
    sum_per_aggregation = 0.0
    for _ in range(aggregation_frequency):
        grads_for_batch = 0.0
        for rank in range(hvd.size()):
            grads_for_batch += rank
        if average_aggregated_gradient:
            grads_for_batch /= float(aggregation_frequency)
        sum_per_aggregation += grads_for_batch / float(hvd.size())

    return aggregations_completed * sum_per_aggregation


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
        check_tf_1(args.aggregation_frequency, args.average_aggregated_gradients)
    else:
        check_tf_2(args.aggregation_frequency, args.average_aggregated_gradients)


if __name__ == "__main__":
    main()
