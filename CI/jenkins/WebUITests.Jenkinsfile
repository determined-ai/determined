describeNode = "echo \"Running on \${NODE_NAME} (executor: \${EXECUTOR_NUMBER}) (build tag: \${BUILD_TAG})\""

pipeline {
  agent any
    environment {
      DET_SEGMENT_MASTER_KEY = credentials('dev-determinedai-segment-master-key')
      DET_SEGMENT_WEBUI_KEY = credentials('dev-determinedai-segment-webui-key')
      GOBIN = "${env.WORKSPACE}/gobin"
      INTEGRATIONS_HOST_PORT = sh(script: 'python ./CI/integrations/get_port.py --run-number $EXECUTOR_NUMBER', , returnStdout: true).trim()
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
            python webui/tests/bin/e2e-tests.py post-e2e-tests --integrations-host-port ${INTEGRATIONS_HOST_PORT}
            python webui/tests/bin/e2e-tests.py pre-e2e-tests --integrations-host-port ${INTEGRATIONS_HOST_PORT}
            '''
        }
      }
      stage('E2E Tests') {
        steps {
          sh "${describeNode}"
          // To avoid permission issues during Jenkins cleanup we instruct Cypress to put its output
          // in /tmp as the Cypress docker container is running as root.
          sh script: '''
            . venv/bin/activate
            python webui/tests/bin/e2e-tests.py docker-run-e2e-tests \
            --integrations-host-port ${INTEGRATIONS_HOST_PORT} \
            --cypress-default-command-timeout 30000 \
            --output-dir /tmp/cypress/${BUILD_TAG}
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
