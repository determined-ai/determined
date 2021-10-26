import inspect

from determined.common.api.fastapi_client import models
from determined.common.schemas._schema_base import register_str_type

for model in inspect.getmembers(models, inspect.isclass):
    if model[1].__module__ == "determined.common.api.fastapi_client.models":
        model_class = model[1]
        register_str_type(model_class.__name__, model_class) # could we lazy register?
        if hasattr(model_class, "update_forward_refs"):
            model_class.update_forward_refs()
