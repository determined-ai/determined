import json
from typing import Any, Dict, List, Union

from pydantic import BaseModel

# def to_dict(o: BaseModel) -> Dict:
#     rv = o
#     if isinstance(o, List):
#         return [to_dict(i) for i in o]
#     elif hasattr(o, "dict"):
#         rv = o.dict()  # type: Dict[str, Any]
#         if isinstance(o, dict):
#             for k, v in o.items():
#                 rv[k] = to_dict(v)
#     return rv


def to_json(o: Union[BaseModel, List[BaseModel]], **dumps_kwargs):
    if isinstance(o, List):  # FIXME is this enough?
        return [to_json(i) for i in o]
    assert hasattr(o, "json")
    return json.loads(o.json(**dumps_kwargs))  # CHECK do we need this?
