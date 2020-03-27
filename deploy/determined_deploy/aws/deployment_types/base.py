class DeterminedDeployment:
    template_parameter_keys = []

    def __init__(self, template_path, parameters):
        self.template_path = template_path
        self.parameters = parameters

    def deploy(self):
        raise NotImplementedError()

    def print(self):
        with open(self.template_path) as f:
            print(f.read())

    def consolidate_parameters(self):
        return [
            {"ParameterKey": k, "ParameterValue": self.parameters[k]}
            for k in self.parameters.keys()
            if self.parameters[k] and k in self.template_parameter_keys
        ]
