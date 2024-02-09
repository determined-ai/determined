#!/bin/sh
echo "Starting master in the background"
/test_scripts/master-binary --config-file=/test_scripts/master-config.yaml &

echo "Waiting for migrations results and uploading them"
python3 /test_scripts/wait_for_perf_migration_upload_results.py

echo "Checking if need to take a snapshot"
if [ ! -f /tmp/no-migrations-needed ]; then
    echo "/tmp/no-migrations-needed does not exist checking branch"

    if [ "${BRANCH}" = "devbranchfake" ]; then # TODO SWAP TO MAIN!!!!!!
        echo "On main branch, taking snapshot"
        aws rds create-db-snapshot \
            --region="us-west-2" \
            --db-instance-identifier="${PERF_DB_AWS_NAME}" \
            --db-snapshot-identifier="ci-snapshot-commit-${COMMIT}" \
            --tags "Key=ci-snapshot"
        echo "Waiting for snapshot to become completed"
        aws rds wait db-snapshot-completed \
            -region="us-west-2" \
            --db-snapshot-identifier="ci-snapshot-commit-${COMMIT}"
        echo "Snapshot completed"
    else
        echo "Not on main branch, skipping snapshot"
    fi
else
    echo "/tmp/no-migrations-needed does exist skipping snapshot"
fi

echo "Starting performance test"
k6 run /test_scripts/api_performance_tests.js --summary-export=/test_scripts/summary.json

echo "Reporting performance test result"
python3 /test_scripts/upload_perf_results.py /test_scripts/summary.json

echo "Performance test completed"
