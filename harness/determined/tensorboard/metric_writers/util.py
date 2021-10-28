from typing import Any


def is_numerical_scalar(n: Any) -> bool:
    """
    Check if the argument is a numerical scalar that is writeable to TensorBoard.

    There are two cases of numpy "scalars". The first is a true numpy scalar,
    and the second is an array scalar that has 0 dimensions. Both of these
    cases are mathematically scalars but represented differently in numpy for
    historical reasons [1]. [2] is another useful reference.

    [1] https://docs.scipy.org/doc/numpy/user/basics.types.html#array-scalars
    [2] https://docs.scipy.org/doc/numpy/reference/arrays.scalars.html
    """
    import numpy as np

    if isinstance(n, (int, float)):
        return True

    if isinstance(n, np.number):
        return True

    if isinstance(n, np.ndarray) and n.ndim == 0 and np.issubdtype(n.dtype, np.number):
        return True

    return False
