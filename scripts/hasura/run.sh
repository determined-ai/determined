#!/bin/bash

DOCKER_NETWORK=determined
DB_HOST=determined-db
DB_PASSWORD="${DET_DB_PASSWORD:-postgres}"
HASURA_SECRET="${DET_HASURA_SECRET:-hasura}"

docker run -p 127.0.0.1:8081:8080 --rm \
  --network "$DOCKER_NETWORK" \
  --name hasura \
  -e HASURA_GRAPHQL_ADMIN_SECRET="$HASURA_SECRET" \
  -e HASURA_GRAPHQL_DATABASE_URL=postgres://postgres:"$DB_PASSWORD"@"$DB_HOST":5432/determined \
  -e HASURA_GRAPHQL_ENABLE_CONSOLE=true \
  -e HASURA_GRAPHQL_ENABLE_TELEMETRY=false \
  -e HASURA_GRAPHQL_CONSOLE_ASSETS_DIR=/srv/console-assets \
  hasura/graphql-engine:v1.1.0
