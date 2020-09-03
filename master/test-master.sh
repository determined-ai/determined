set -x

curl-da.sh "http://localhost:8080/api/v1/master/logs?limit=-1"
curl-da.sh "http://localhost:8080/api/v1/master/logs?limit=0"
curl-da.sh "http://localhost:8080/api/v1/master/logs?limit=1"
curl-da.sh "http://localhost:8080/api/v1/master/logs?offset=0&limit=2"
curl-da.sh "http://localhost:8080/api/v1/master/logs?offset=1&limit=1"
curl-da.sh "http://localhost:8080/api/v1/master/logs?offset=-1&limit=1"

