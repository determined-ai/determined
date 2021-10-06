import json
import jsonpickle
from determined.common.api.fastapi_client import models as m

# aModel = m.Determinedexperimentv1State.PAUSED
aModel = m.V1GetExperimentResponse()
aModel.experiment = m.V1Experiment()
aModel.experiment.name = 'golabi'

class DetEncoder(json.JSONEncoder):
    def default(self, obj):
        print('obje', obj)
        # raise TypeError()
        if isinstance(obj, m.BaseModel):
            return super().default(obj.__dict__)
        # Let the base class default method raise the TypeError
        return super().default(obj)

# print(aModel.__dict__)

# print(jsonpickle.encode(aModel))
# print(json.dumps(aModel, cls=json.JSONEncoder))
# print(DetEncoder().encode(aModel))
# print(json.dumps(aModel, cls=DetEncoder))
print(jsonpickle.encode(aModel, unpicklable=False))
# print(json.dumps(aModel.__dict__))

