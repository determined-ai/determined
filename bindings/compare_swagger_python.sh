#!/bin/bash -e
rm ./diff.txt
if [ -f ./build/python/determined.swagger.client/__init__.py ]
then
    cp ./build/python/determined.swagger.client/* ./build/python/determined/swagger/client
    cp ./build/python/determined.swagger.client/api/* ./build/python/determined/swagger/client/api
    cp ./build/python/determined.swagger.client/models/* ./build/python/determined/swagger/client/models
else 
    echo "Swagger files are missing"
    exit 1
fi

diff -rq -x .DS* ./build/python/determined/swagger/client ../harness/determined/swagger/client > ./diff.txt

if [ -s ./diff.txt ]
then
    echo "Generated swagger files are different from harness/determined/swagger/client."
    echo "Copy new files in harness/determined/swagger/client."
    cat diff.txt
    exit 1
else
    echo "Generated swagger files are same as harness/determined/swagger/client."
    exit 0
fi
