cat task_definition.json | \
  jq '.containerDefinitions[0].image = "your-new-image:latest"' \
  > /tmp/task_definition.json
VERSION=$(aws ecs register-task-definition --cli-input-json file:///tmp/task_definition.json)
