"""RNG state utils for TF frameworks."""

import random
from typing import Any, Dict

import numpy as np
import tensorflow as tf
from packaging import version


def get_rng_state() -> Dict[str, Any]:
    rng_state = {"np_rng_state": np.random.get_state(), "random_rng_state": random.getstate()}

    if version.parse(tf.__version__) < version.parse("2.0.0") or not tf.executing_eagerly():
        rng_state["tf_rng_global_seed"] = tf.compat.v1.random.get_seed(0)[0]
    else:
        generator = tf.random.get_global_generator()
        rng_state["tf2_rng_global_algorithm"] = generator.algorithm
        rng_state["tf2_rng_global_state"] = generator.state

    return rng_state


def set_rng_state(rng_state: Dict[str, Any]) -> None:
    np.random.set_state(rng_state["np_rng_state"])
    random.setstate(rng_state["random_rng_state"])

    if "tf_rng_global_seed" in rng_state:
        tf.compat.v1.random.set_random_seed(rng_state["tf_rng_global_seed"])
    if "tf2_rng_global_algorithm" in rng_state and "tf2_rng_global_state" in rng_state:
        algorithm = rng_state["tf2_rng_global_algorithm"]
        state = rng_state["tf2_rng_global_state"]
        generator = tf.random.Generator.from_state(state, algorithm)
        tf.random.set_global_generator(generator)
