import determined_deploy
from determined_deploy.aws import constants


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
            {"ParameterKey": k, "ParameterValue": str(self.parameters[k])}
            for k in self.parameters.keys()
            if self.parameters[k] and k in self.template_parameter_keys
        ]

    def before_deploy_print(self):
        cluster_id = self.parameters[constants.cloudformation.CLUSTER_ID]
        aws_region = self.parameters[constants.cloudformation.BOTO3_SESSION].region_name
        version = (
            self.parameters[constants.cloudformation.VERSION]
            if self.parameters[constants.cloudformation.VERSION]
            else determined_deploy.__version__
        )
        keypair = self.parameters[constants.cloudformation.KEYPAIR]

        print(f"Determined Version: {version}")
        print(f"Stack Name: {cluster_id}")
        print(f"AWS Region: {aws_region}")
        print(f"Keypair: {keypair}")
