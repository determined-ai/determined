class DeterminedDeployment:
    def __init__(self, template_path, parameters):
        self.template_path = template_path
        self.parameters = parameters

    def deploy(self):
        raise NotImplementedError()

    def print(self):
        with open(self.template_path) as f:
            print(f.read())
