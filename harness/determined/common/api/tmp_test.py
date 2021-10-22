from determined.common.api.fastapi_client import models as m
# from determined.common.schemas import SchemaBase 

# o = m.V1Metrics(**{"num_inputs": 3, "state": "STATE_RUNNING"})
# m.V1Metrics.update_forward_refs()
# print(m.V1Metrics.__annotations__.items())
# o = m.V1Metrics.from_dict({"num_inputs": 3}, prevalidated=True)
o = m.V1ModelVersion.from_dict({"model": {"name": "xyz", "metadata": 3}}, prevalidated=True)
# t = m.Trialv1Trial()

# print(t.state)
# # print(dir(t))
# # print(dir(m.Trialv1Trial))
# print(t.id)

print(o.model and o.model.name)
# print(o.num_inputs)
