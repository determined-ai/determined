from typing import Any, Dict, List


def Constant(value: Any) -> Dict:
    return {"type": "const", "val": value}


def Integer(minval: int, maxval: int) -> Dict:
    return {"type": "int", "minval": minval, "maxval": maxval}


def Double(minval: float, maxval: float) -> Dict:
    return {"type": "double", "minval": minval, "maxval": maxval}


def Categorical(vals: List[Any]) -> Dict:
    return {"type": "categorical", "vals": vals}


def Log(minval: float, maxval: float, base: int = 10) -> Any:
    return {"type": "log", "base": base, "minval": minval, "maxval": maxval}
