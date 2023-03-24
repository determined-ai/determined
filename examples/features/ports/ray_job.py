import numpy as np
import ray

ds = ray.data.range(5)
result = ds.map(lambda x: x * 2).take_all()
print("Result:", result)
assert result[-1] == 8
