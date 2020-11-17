# Test Cluster

This directory is here to provide a test cluster for e2e-tests.

It is a direct copy of the `tools` directory from the project root and should
stay around and in sync until we have another way of spinning up a test cluster
alongside a determined dev cluster.

The only changes introduced here are:

- Updated, unique ports: 8081 for master, and 5433 for database
