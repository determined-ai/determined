import json

class AverageMeter(object):
    """From https://github.com/pytorch/examples/blob/master/imagenet/main.py"""

    def __init__(self):
        self.val = 0
        self.avg = 0
        self.sum = 0
        self.count = 0

    def __repr__(self):
        return f'{self.avg:.2e}'

    def update(self, val, n=1):
        self.val = val
        self.sum += val * n
        self.count += n
        self.avg = self.sum / self.count


class dotdict(dict):
    def __getattr__(self, name):
        return self[name]


def read_json(path, name, required = False):
    try:
        with open(path.joinpath(f'{name}.json'), 'r') as json_dict:
            return json.load(json_dict)
    except Exception as err:
        if required:
            raise err
    return {}

def write_json(path, name, dict):
    with open(path.joinpath(f'{name}.json'), 'w') as file:
        file.write(json.dumps(dict))
