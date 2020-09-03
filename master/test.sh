id=$1

curl-da.sh 'http://localhost:8080/notebooks' --data-binary '{"config":{"resources":{"slots":0}},"context":null}';
id=$(curl-da.sh http://localhost:8080/api/v1/notebooks | jq '.notebooks[-1].id' | head -n1 | sed 's/"//g');
echo start asking for logs;
echo

# curl-da.sh "http://localhost:8080/api/v1/notebooks/$id/logs"

curl-da.sh "http://localhost:8080/api/v1/notebooks/$id/logs?follow=true"

# curl-da.sh "http://localhost:8080/notebooks/$id/events"

