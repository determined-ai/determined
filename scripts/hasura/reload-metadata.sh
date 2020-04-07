#!/bin/bash

hasura_host=${hasura_host:-http://localhost:8081}
curl "$hasura_host"/v1/query -d '{"type":"reload_metadata","args":{}}'
