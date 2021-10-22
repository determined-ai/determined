import inspect

from determined.common.api.fastapi_client import models
# from determined.common.api.fastapi_client import *
from determined.common.schemas import register_str_type

for model in inspect.getmembers(models, inspect.isclass):
    if model[1].__module__ == "determined.common.api.fastapi_client.models":
        model_class = model[1]
        register_str_type(model_class.__name__, model_class)
        if hasattr(model_class, "update_forward_refs"):
            model_class.update_forward_refs()
