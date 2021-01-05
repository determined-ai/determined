import abc
from typing import Union

import numpy as np


class MetricWriter(abc.ABC):
    @abc.abstractmethod
    def add_scalar(self, name: str, value: Union[int, float, np.number], step: int) -> None:
        pass

    @abc.abstractmethod
    def reset(self) -> None:
        pass
