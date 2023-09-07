import numpy as np
import ray

ds = ray.data.range(5)
result = ds.map(lambda x: {"id": x["id"] * 2}).sort("id").take_all()
print("Result:", result)
assert result[-1]["id"] == 8
