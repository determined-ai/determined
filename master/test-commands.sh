#!/bin/bash -x

id=$1

# create a notebook
curl-da.sh 'http://localhost:8080/notebooks' --data-binary '{"config":{"resources":{"slots":0}},"context":null}';

id=$(curl-da.sh http://localhost:8080/api/v1/notebooks?sort_by=SORT_BY_START_TIME | jq '.notebooks[-1].id' | head -n1 | sed 's/"//g');
echo start asking for logs;
echo

curl-da.sh "http://localhost:8080/api/v1/notebooks/$id/logs?limit=-1"
curl-da.sh "http://localhost:8080/api/v1/notebooks/$id/logs?limit=0"
curl-da.sh "http://localhost:8080/api/v1/notebooks/$id/logs?limit=1"
curl-da.sh "http://localhost:8080/api/v1/notebooks/$id/logs?offset=0&limit=2"
curl-da.sh "http://localhost:8080/api/v1/notebooks/$id/logs?offset=1&limit=2"
curl-da.sh "http://localhost:8080/api/v1/notebooks/$id/logs?offset=-1&limit=2"
curl-da.sh "http://localhost:8080/api/v1/notebooks/$id/logs?follow=true"

