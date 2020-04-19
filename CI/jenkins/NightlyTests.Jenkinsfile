describeNode = "echo \"Running on \${NODE_NAME} (executor: \${EXECUTOR_NUMBER})\""

pipeline {
  agent { label 'test' }
  triggers {
    cron('H 1 * * *')
  }
  environment {
    DET_SEGMENT_MASTER_KEY = credentials('dev-determinedai-segment-master-key')
    DET_SEGMENT_WEBUI_KEY = credentials('dev-determinedai-segment-webui-key')
    INTEGRATIONS_RESOURCE_SUFFIX = "-${env.BUILD_TAG}"
  }
  stages {
    stage('Nightly tests') {
      options {
        timeout(time: 2, unit: 'HOURS')
      }
      environment {
        AWS_DEFAULT_REGION = "us-west-2"
        PYTEST_MARKS = "nightly"
        REPORT_ROOT = "${env.WORKSPACE}/build"
        SHORT_GIT_HASH = sh(script: 'git rev-parse --short HEAD', returnStdout: true).trim()
        CLUSTER_NAME = "determined-${env.SHORT_GIT_HASH}-nightly"
      }
      steps {
        sh "${describeNode}"
        sh 'virtualenv --python="$(command -v python3.6)" venv'
        sh "venv/bin/python -m pip install -r combined-reqs.txt"
        sh ". venv/bin/activate && det-deploy aws up --cluster-id $CLUSTER_NAME --det-version `git rev-parse HEAD` --keypair integrations-test"
        script {
          env.MASTER_HOST = sh(script: "venv/bin/python CI/integrations/get_address.py $CLUSTER_NAME", returnStdout: true).trim()
        }
        sh "venv/bin/python CI/integrations/wait_for_master.py http://$MASTER_HOST:8080"
        sh ". venv/bin/activate && make test-python-integrations"
      }
      post {
        always {
          sh ". venv/bin/activate && det-deploy aws down --cluster-id $CLUSTER_NAME"
          junit "**/build/test-reports/*.xml"
        }
      }
    }
  }
}
