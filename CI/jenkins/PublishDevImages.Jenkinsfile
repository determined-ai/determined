describeNode = "echo \"Running on \${NODE_NAME} (executor: \${EXECUTOR_NUMBER})\""

pipeline {
  agent any
  environment {
    DET_SEGMENT_MASTER_KEY = credentials('dev-determinedai-segment-master-key')
    DET_SEGMENT_WEBUI_KEY = credentials('dev-determinedai-segment-webui-key')
    DET_DOCKERHUB_CREDS = credentials('dockerhub-determinedai-dev')
    DOCKER_REGISTRY = ""
    IMAGE_TYPE = sh(script: "printf ${env.BRANCH_NAME} | sed -r 's/\\//_/g' | sed -r 's/\\./-/g'", returnStdout: true)
  }
  stages {
    stage('Setup') {
      steps {
        sh "${describeNode}"
        sh 'virtualenv --python="$(command -v python3.6)" --no-site-packages venv'
        sh "docker login -u ${env.DET_DOCKERHUB_CREDS_USR} -p ${env.DET_DOCKERHUB_CREDS_PSW}"
      }
    }
    stage('Build') {
      steps {
        sh "${describeNode}"
        sh ". venv/bin/activate && make clean all"
      }
    }
    stage('Push') {
      steps {
        sh "${describeNode}"
        sh ". venv/bin/activate && make publish-dev"
      }
    }
    stage('Deploy') {
      steps {
        sh ". venv/bin/activate && det-deploy aws up --cluster-id determined-${env.BRANCH_NAME} --det-version `git rev-parse HEAD` --keypair integrations-test"
      }
    }
  }
}
