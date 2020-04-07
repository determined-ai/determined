describeNode = "echo \"Running on \${NODE_NAME} (executor: \${EXECUTOR_NUMBER})\""

pipeline {
  agent any
    environment {
      INTEGRATIONS_HOST_PORT = sh(script: 'python ./CI/integrations/get_port.py --run-number $EXECUTOR_NUMBER', , returnStdout: true).trim()
      GOBIN="${env.WORKSPACE}/gobin"
    }
    stages {
      stage('Environment Setup') {
        steps {
          sh "${describeNode}"
          sh 'virtualenv --python="$(command -v python3.6)" --no-site-packages venv'
          sh script: '''
          . venv/bin/activate
          make get-deps
          '''
          sh script: '''
          . venv/bin/activate
          make -C webui/tests get-deps
          '''
          sh script: '''
          . venv/bin/activate
          make build-docker
          '''
        }
      }
      stage('Cluster Setup') {
        steps {
          sh "${describeNode}"
          sh script: '''
            . venv/bin/activate
            python webui/tests/bin/e2e-tests.py pre-e2e-tests --integrations-host-port ${INTEGRATIONS_HOST_PORT}
            '''
        }
      }
      stage('E2E Tests') {
        steps {
          sh "${describeNode}"
          sh script: '''
            . venv/bin/activate
            python webui/tests/bin/e2e-tests.py docker-run-e2e-tests --integrations-host-port ${INTEGRATIONS_HOST_PORT} --cypress-default-command-timeout 30000
            '''
        }
      }
    }
    post {
      always {
        sh '''
          . venv/bin/activate
          python webui/tests/bin/e2e-tests.py post-e2e-tests --integrations-host-port ${INTEGRATIONS_HOST_PORT}
          '''
      }
    }
}
