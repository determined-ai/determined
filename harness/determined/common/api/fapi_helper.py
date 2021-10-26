from determined.common.api.fapi import FApiSchemaBase
import json
from typing import Any, Dict, List, Union, TypeVar

T = TypeVar('T')


# def to_dict(o: Union[FApiSchemaBase, Dict, List]) -> Union[Dict, List[Dict]]:
#     rv = o
#     if isinstance(o, List):
#         return [to_dict(i) for i in o]
#     elif issubclass(o, FApiSchemaBase):
#         rv = o.to_dict()
#         if isinstance(o, dict):
#             for k, v in o.items():
#                 rv[k] = to_dict(v)
#     return rv



# def to_json(o: Union[T, List[T]], **dumps_kwargs):
#     print(o)
#     if isinstance(o, List):  # FIXME is this enough?
#         return [to_json(i) for i in o]
#     # assert hasattr(o, "json")
#     return json.dumps(o, **dumps_kwargs)
#     # return json.loads(o.json(**dumps_kwargs))  # CHECK do we need this?
