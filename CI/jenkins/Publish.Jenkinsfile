describeNode = "echo \"Running on \${NODE_NAME} (executor: \${EXECUTOR_NUMBER})\""

pipeline {
  agent { label 'test' }
  environment {
    DET_SEGMENT_MASTER_KEY = credentials('prod-determinedai-segment-master-key')
    DET_SEGMENT_WEBUI_KEY = credentials('prod-determinedai-segment-webui-key')
    DET_DOCKERHUB_CREDS = credentials('dockerhub-determinedai-dev')
    DET_TWINE_CREDS = credentials('determined-twine-credentials')
    TWINE_USERNAME = "${env.DET_TWINE_CREDS_USR}"
    TWINE_PASSWORD = "${env.DET_TWINE_CREDS_PSW}"
    GITHUB_TOKEN = credentials('determined-ci-github-access')
    DOCKER_REGISTRY = ""
  }
  stages {
    stage('Publish') {
      steps {
        sh "${describeNode}"
        sh "docker login -u ${env.DET_DOCKERHUB_CREDS_USR} -p ${env.DET_DOCKERHUB_CREDS_PSW}"
        sh 'virtualenv --python="$(command -v python3.6)" venv'
        sh '. venv/bin/activate && make get-deps'
        sh '. venv/bin/activate && make publish'
      }
    }
  }
}
