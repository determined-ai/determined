#!/bin/bash

hasura_host=${hasura_host:-http://localhost:8081}
HASURA_SECRET="${DET_HASURA_SECRET:-hasura}"
cd "$(dirname "$0")"
curl -H "X-Hasura-Admin-Secret: $HASURA_SECRET" "$hasura_host"/v1/query -d '{"type":"export_metadata","args":{}}' | python -m json.tool >../../master/static/hasura-metadata.json
