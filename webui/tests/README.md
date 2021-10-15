# WebUI End-to-end Testing

## Overview

We are using [Gauge](https://gauge.org/) and [Taiko](https://taiko.dev/) to run end-to-end tests for both of our webapps.

The end-to-end test begins with a script that starts a new instance of a cluster. The script performs the following:

- Starts a new cluster via [devcluster](https://github.com/determined-ai/devcluster) at [http://localhost:8081](http://localhost:8081).
- Creates a user account with not password (`user-wo-pw`) via the CLI.
- Creates a user account with a password (`user-w-pw`/`special-pw`) via the CLI.
- Creates 4 experiments via the CLI.

Once the script runs its course, Gauge and Taiko start a chromium instance and point to the cluster address and performs a series of tests according to the defined specs here.

## Running WebUI End-to-end Tests

### Running Tests Locally

First get the necessary dependencies to be able to run the tests locally. These are not installed by default during the `make all` process to avoid making that step uncessarily long.

```
# get necessary dependencies to run tests locally
make get-deps
```

You have two ways of running tests. You can run the tests where it will open a chromium browser and run through the tests, where you can visually see chromium follow the automated process, or the headless mode where it will only log in the terminal what it is doing.

```
#  OPTION 1: run tests with chromium
make dev-tests

# OPTION 2: run tests with chromium headless mode
make test
```
