dockerStrs = ''
dockerLogin = "\$(AWS_DEFAULT_REGION=\$(ec2metadata --availability-zone | sed 's/.\$//') aws ecr get-login --no-include-email)"
describeNode = "echo \"Running on \${NODE_NAME} (executor: \${EXECUTOR_NUMBER})\""
testTag = ''

/* There are secrets in the cluster config for S3 checkpoint configuration which
   should not be stored in the open-source repository.  We inject the secrets by
   storing the config files in an S3 bucket, which can be downloaded at runtime
   for the integraiton tests.

   To update the configs, create the new config file(s), and save them to a
   directory in an s3 bucket:

       aws s3 cp --recursive master.yaml [agent.yaml ...] \
               s3://integrations-cluster-config/UNIQUE_NAME

   where UNIQUE_NAME is a new name, so you don't break the current integration
   tests.  Today's date is probably fine.

   Then, set config_root = UNIQUE_NAME here and submit your PR. */
config_root = "20200331"
config_uri = "s3://integrations-cluster-config/${config_root}/"

pipeline {
  agent any
  stages {
    stage('Integrations') {
      options {
        timeout(time: 2, unit: 'HOURS')
      }
      parallel {
        stage('Master Integration Tests') {
          agent { label 'test' }
          steps {
            sh "${describeNode}"
          }
        }
      }
    }
  }
}
