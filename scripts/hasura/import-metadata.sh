#!/bin/bash

hasura_host=${hasura_host:-http://localhost:8081}
cd "$(dirname "$0")"
curl "$hasura_host"/v1/query -d '{"type":"replace_metadata","args":'"$(<../../master/static/hasura-metadata.json)"'}'
