from typing import Any, Dict, List


def Constant(value: Any) -> Dict:
    """
    A hyperparameter configuration for a constant value.

    Arguments:
        value:
            A JSON-serializable value (int, float, str, or a some combination
            of those types in a list or dictionary)

    Returns:
        A dictionary representing the configuration.
    """
    return {"type": "const", "val": value}


def Integer(minval: int, maxval: int) -> Dict:
    """
    A hyperparameter configuration for a discrete uniform distribution over
    integers.

    Arguments:
        minval: Minimum integer value, inclusive.
        maxval: Maximum integer value, inclusive.

    Returns:
        A dictionary representing the configuration.
    """
    return {"type": "int", "minval": minval, "maxval": maxval}


def Double(minval: float, maxval: float) -> Dict:
    """
    A hyperparameter configuration for a continuous uniform distribution over
    float values.

    Arguments:
        minval: Minimum float value, inclusive.
        maxval: Maximum float value, inclusive.

    Returns:
        A dictionary representing the configuration.
    """
    return {"type": "double", "minval": minval, "maxval": maxval}


def Categorical(vals: List[Any]) -> Dict:
    """
    A hyperparameter configuration for a discrete uniform distribution over
    a list of values.

    Arguments:
        vals:
            A list of JSON-serializable values (int, float, str, or a some
            combination of those types in nested lists or dictionaries)

    Returns:
        A dictionary representing the configuration.
    """
    return {"type": "categorical", "vals": vals}


def Log(minval: float, maxval: float, base: int = 10) -> Any:
    """
    A hyperparameter configuration for a log uniform distribution over
    float values.

    Arguments:
        minval:
            The minimum exponent to be used in the distribution. The minimum
            value of the hyperparameter will be base ^ minval.
        maxval:
            The minimum exponent to be used in the distribution. The maximum
            value of the hyperparameter will be base ^ maxval.
        base:
            The logarithm base to use for the distribution (default: 10)

    Returns:
        A dictionary representing the configuration.
    """
    return {"type": "log", "base": base, "minval": minval, "maxval": maxval}
