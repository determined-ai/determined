import inspect

from determined.common.api.fastapi_client import models

for model in inspect.getmembers(models, inspect.isclass):
    if model[1].__module__ == "determined.common.api.fastapi_client.models":
        model_class = model[1]
        if hasattr(model_class, "update_forward_refs"):
            model_class.update_forward_refs()
