from determined.common.api.fastapi_client import models as m

o = m.V1Metrics(**{'num_inputs': 3, 'state': 'STATE_RUNNING'})
# t = m.Trialv1Trial()

# print(t.state)
# # print(dir(t))
# # print(dir(m.Trialv1Trial))
# print(t.id)

print(o.num_inputs)
