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
  environment {
    INTEGRATIONS_RESOURCE_SUFFIX="-${env.BUILD_TAG}"
    GOBIN="${env.WORKSPACE}/gobin"
  }
  stages {
    stage('Build and Push') {
      agent { label 'general' }
      steps {
        sh "${describeNode}"
        sh "${dockerLogin}"
        sh script: '''
virtualenv --python="$(command -v python3.6)" --no-site-packages venv
. venv/bin/activate
make get-deps
'''
        sh script: '''
. venv/bin/activate
make build-docker
make -C CI/integrations build
'''
        sh script: '''
. venv/bin/activate
make publish-dev
make -C CI/integrations publish-dev
'''
        script {
          dockerStrs = sh(script: '. venv/bin/activate && make -s -C CI/integrations get-images', returnStdout: true)
          testTag = sh(script: 'printf \${DOCKER_REGISTRY}determinedai/determined-dev:determined-test-harness-\$(git rev-parse HEAD)', returnStdout: true)
        }
      }
    }
    stage('Integrations') {
      options {
        timeout(time: 2, unit: 'HOURS')
      }
      environment {
        TEST_TAG = "${testTag}"
      }
      parallel {
        stage('Master Integration Tests') {
          agent { label 'general' }
          steps {
            sh "${describeNode}"
            sh "${dockerLogin}"
            sh "docker pull ${testTag}"
            sh "make -C CI/integrations run-master-integration-tests"
          }
        }
        stage("Python Integration Tests Split 1") {
          agent { label 'integrations' }
          environment {
            PYTEST_MARKS = "integ1"
            ETC_ROOT = "${env.WORKSPACE}/build/cluster_config/"
            REPORT_ROOT = "${env.WORKSPACE}/build"
          }
          steps {
            sh "${describeNode}"
            sh "make -C CI/integrations cleanup"
            sh "${dockerLogin}"
            sh "${dockerStrs}"
            sh "mkdir -p $ETC_ROOT"
            sh "aws s3 cp --recursive ${config_uri} $ETC_ROOT"
            sh "make -C CI/integrations run-python-integration-tests"
          }
          post {
            always {
              sh "make -C CI/integrations cleanup"
              junit "**/build/test-reports/*.xml"
            }
          }
        }
        stage("Python Integration Tests Split 2") {
          agent { label 'integrations' }
          environment {
            PYTEST_MARKS = "integ2"
            ETC_ROOT = "${env.WORKSPACE}/build/cluster_config/"
            REPORT_ROOT = "${env.WORKSPACE}/build"
          }
          steps {
            sh "${describeNode}"
            sh "make -C CI/integrations cleanup"
            sh "${dockerLogin}"
            sh "${dockerStrs}"
            sh "mkdir -p $ETC_ROOT"
            sh "aws s3 cp --recursive ${config_uri} $ETC_ROOT"
            sh "make -C CI/integrations run-python-integration-tests"
          }
          post {
            always {
              sh "make -C CI/integrations cleanup"
              junit "**/build/test-reports/*.xml"
            }
          }
        }
        stage("Python Integration Tests Split 3") {
          agent { label 'integrations' }
          environment {
            PYTEST_MARKS = "integ3"
            ETC_ROOT = "${env.WORKSPACE}/build/cluster_config/"
            REPORT_ROOT = "${env.WORKSPACE}/build"
          }
          steps {
            sh "${describeNode}"
            sh "make -C CI/integrations cleanup"
            sh "${dockerLogin}"
            sh "${dockerStrs}"
            sh "mkdir -p $ETC_ROOT"
            sh "aws s3 cp --recursive ${config_uri} $ETC_ROOT"
            sh "make -C CI/integrations run-python-integration-tests"
          }
          post {
            always {
              sh "make -C CI/integrations cleanup"
              junit "**/build/test-reports/*.xml"
            }
          }
        }
        stage("Python Integration Tests Split 4") {
          agent { label 'integrations' }
          environment {
            PYTEST_MARKS = "integ4"
            ETC_ROOT = "${env.WORKSPACE}/build/cluster_config/"
            REPORT_ROOT = "${env.WORKSPACE}/build"
          }
          steps {
            sh "${describeNode}"
            sh "make -C CI/integrations cleanup"
            sh "${dockerLogin}"
            sh "${dockerStrs}"
            sh "mkdir -p $ETC_ROOT"
            sh "aws s3 cp --recursive ${config_uri} $ETC_ROOT"
            sh "make -C CI/integrations run-python-integration-tests"
          }
          post {
            always {
              sh "make -C CI/integrations cleanup"
              junit "**/build/test-reports/*.xml"
            }
          }
        }
        stage("Python Integration Tests Parallel Training") {
          agent { label 'parallel' }
          environment {
            PYTEST_MARKS = "parallel"
            ETC_ROOT = "${env.WORKSPACE}/build/cluster_config/"
            REPORT_ROOT = "${env.WORKSPACE}/build"
          }
          steps {
            sh "${describeNode}"
            sh "make -C CI/integrations cleanup"
            sh "${dockerLogin}"
            sh "${dockerStrs}"
            sh "mkdir -p $ETC_ROOT"
            sh "aws s3 cp --recursive ${config_uri} $ETC_ROOT"
            sh "make -C CI/integrations run-python-integration-tests"
          }
          post {
            always {
              sh "make -C CI/integrations cleanup"
              junit "**/build/test-reports/*.xml"
            }
          }
        }
      }
    }
  }
}
