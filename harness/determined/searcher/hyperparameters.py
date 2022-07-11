from typing import Dict, List

from determined.common.api import bindings


class HparamValue:
    def __init__(self, name: str):
        self.name = name

    def _to_hyperparameter(self) -> bindings.v1Hyperparameter:
        pass


class IntHparamValue(HparamValue):
    def __init__(self, name: str, value: int):
        super().__init__(name)
        self.value = value

    def _to_hyperparameter(self) -> bindings.v1Hyperparameter:
        return bindings.v1Hyperparameter(
            integerHyperparam=bindings.v1IntegerHyperparameter(str(self.value)),
        )


class DoubleHparamValue(HparamValue):
    def __init__(self, name: str, value: float):
        super().__init__(name)
        self.value = value

    def _to_hyperparameter(self) -> bindings.v1Hyperparameter:
        return bindings.v1Hyperparameter(
            doubleHyperparam=bindings.v1DoubleHyperparameter(self.value),
        )


class CategoricalHparamValue(HparamValue):
    def __init__(self, name: str, value: str):
        super().__init__(name)
        self.value = value

    def _to_hyperparameter(self) -> bindings.v1Hyperparameter:
        return bindings.v1Hyperparameter(
            categoricalHyperparam=bindings.v1CategoricalHyperparameter(self.value),
        )


class HparamSample:
    def __init__(self, values: List[HparamValue]):
        self.values = values

    def _to_hyperparameters(self) -> Dict[str, bindings.v1Hyperparameter]:
        return {hpv.name: hpv._to_hyperparameter() for hpv in self.values}
