from typing import Any, ClassVar, Dict, List  # , get_origin new in 3.8

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
        t = self.__annotations__[name]
        # if type(val) != t:
        #     raise AttributeError(f'bad input {name} type. expected {t}')

        if isinstance(val, Dict):
            return default  # unsupported
        elif isinstance(val, Dict):
            return default

        return val
        # alias = kwargs['alias']
        # print(default)

    return field_value


class BaseModel:
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

    def json(self) -> str:
        return ""

    # def __getattribute__(self, name: str):
    #     # print('getattr', args, kwargs)
    #     # return self[name]
    #     return super().__getattribute__(name)


def parse_obj_as(t, model: Any) -> Any:
    return
