#!/bin/sh

# We excpect these variables

# We expect the following environment variables to be set as secrets.

echo "Starting master in the background"
/test_scripts/master-binary --config-file=/test_scripts/master-config.yaml

#echo "Waiting for migrations results and uploading them"
#python .circleci/scripts/wait_for_perf_migration_upload_results.py

#ENV DET_MASTER="http://localhost:8080"
#ENTRYPOINT ["k6"]
#CMD ["run", "/test_scripts/api_performance_tests.js"]
