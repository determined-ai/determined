from typing import Any, ClassVar, Dict, List, Type, TypeVar, Optional, Union  # , get_origin new in 3.8
# from determined.common.schemas import SchemaBase

# removing pydantic the startup cost is ~30ms

# potentially slow
def Field(*args, **kwargs) -> Any:
    def field_value(self, name, val) -> Any:
        default = args[0]
        if val is None:
            if default is not Ellipsis:
                return default
            else:
                raise AttributeError(f"missing required param {name}")
        # t = self.__annotations__[name]
        # # if type(val) != t:
        # #     raise AttributeError(f'bad input {name} type. expected {t}')

        # if isinstance(val, Dict):
        #     return default  # unsupported
        # elif isinstance(val, Dict):
        #     return default

        return val
        # alias = kwargs['alias']
        # print(default)

    return field_value


class SchemaBased():
    _id: str = 'abc'
    # @schemas.auto_init
    def __init__(self, *args, **kwargs):
        if self.__annotations__ is None:
            return
        cls_attrs = self.__annotations__.keys()
        for attr in cls_attrs:
            anno = self.__annotations__[attr]
            # self.__annotations__[attr] = eval(anno)
            fvalue = self.__getattribute__(attr)
            # pass the input value to field_value to compute
            # the default and enforce validations
            self.__setattr__(attr, fvalue(attr, kwargs.get(attr)))
        pass

T = TypeVar("T", bound="BaseModel2")
class BaseModel2:
    def __init__(self, *args, **kwargs):
        # print(self.__annotations__)
        print(args, kwargs)
        if self.__annotations__ is None:
            return
        cls_attrs = self.__annotations__.keys()
        for attr in cls_attrs:
            fvalue = self.__getattribute__(attr)
            # pass the input value to field_value to compute
            # the default and enforce validations
            self.__setattr__(attr, fvalue(attr, kwargs.get(attr)))

        # for k, v in kwargs.items():
        #     if k not in self.__annotations__:
        #         raise Exception(f'bad input {k}')
        #     print('setting', k, v)
        #     self.__setattr__(k, v)
        # print(self.__class__.__name__, args, kwargs)

    # @classmethod
    # def from_dict(cls: Type[T], d: dict, prevalidated: bool = False) -> T:
    #     if not isinstance(d, dict) or any(not isinstance(k, str) for k in d):
    #         raise ValueError("from_dict() requires an input dictionary with only string keys")

    #     # Validate before parsing.
    #     if not prevalidated:
    #         errors = expconf.sanity_validation_errors(d, cls._id)
    #         if errors:
    #             raise TypeError(f"incorrect {cls.__name__}:\n" + "\n".join(errors))

    #     init_args = {}

    #     # For every key in the dictionary, get the type from the class annotations.  If it is a
    #     # sublcass of SchemaBase, call from_dict() or from_none() on it based on the value in the
    #     # input.  Otherwise, make sure a primitive type and pass the value to __init__ directly.
    #     for name, value in d.items():
    #         # Special case: drop keys which match the _union_key value of the class.
    #         if name == getattr(cls, "_union_key", None):
    #             continue
    #         anno = cls.__annotations__.get(name)
    #         if anno is None:
    #             raise TypeError(
    #                 f"{cls.__name__}.from_dict() found a key '{name}' input which has no "
    #                 "annotation.  This is a  bug; all SchemaBase subclasses must have annotations "
    #                 "which match the json schema definitions which they correspond to."
    #             )
    #         # Create an instance based on the type annotation.
    #         init_args[name] = schemas._instance_from_annotation(anno, value, prevalidated=True)

    #     return cls(**init_args)

    def json(self) -> str:
        return ""

    # def __getattribute__(self, name: str):
    #     # print('getattr', args, kwargs)
    #     # return self[name]
    #     return super().__getattribute__(name)



# BaseModel = BaseModel2
BaseModel = SchemaBased

def parse_obj_as(t, model: Any) -> Any:
    return

