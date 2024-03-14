#!/bin/bash

version=$1

set -x
det deploy aws up --cluster-id deleteme --det-version ${version} --keypair hamid --region us-east-2
echo "\n\n###########\n\n"
read -p "Press enter to continue"
det deploy gcp up --det-version ${version} --cluster-id deleteme --project-id bogus
echo "\n\n###########\n\n"
read -p "Press enter to continue"
det deploy gke-experimental up --det-version ${version} --cluster-id deleteme
echo "\n\n###########\n\n"
read -p "Press enter to continue"
det deploy --no-preflight-checks local cluster-up --det-version ${version}
echo "\n\n###########\n\n"
read -p "Press enter to continue"
det deploy local agent-up --det-version ${version} localhost
echo "\n\n###########\n\n"
read -p "Press enter to continue"
det deploy local master-up --det-version ${version}
echo "\n\n###########\n\n"
echo "done"
