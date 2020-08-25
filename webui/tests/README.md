# WebUI End-to-end Tests

We use [Cypress](https://www.cypress.io/) to enable end-to-end testing of our
React SPA, and in some sense the whole determined cluster.

We provide a test script to take care of dependencies around Cypress commands.
Let's call this the "test script" for the rest of the this document.

## Running the Tests

When the tester loads up it loads the WebUI through a given master address
(`$HOST`) and port (`$PORT`)

Based the tester requirements and ease of test development the following
assumptions are made:

- It's a brand new cluster
- The cluster is isolated. Meaning this tester is the only entity interacting
with the cluster

Once the cluster is ready and accessible run the tests:

```bash
./bin/e2e-tests.py --det-port $PORT pre-e2e-tests && \
./bin/e2e-tests.py --det-port $PORT run-e2e-tests
```

Note that it's is important that one immediately follows the other since the
pre-e2e-tests target starts some experiments.

### Using the Bundled Cluster Manager

For ease of use and quicker local testing we provide an option to set up and
tear down such a cluster through the test script.

Before the tests can be started we need to build the cluster including the WebUI
to make sure the served WebUI and cluster in general are up to date.

Issue the following command:
`./bin/e2e-tests.py e2e-tests` (or `make test`) which in turn will:

1. Set up a test cluster
2. Run the Cyrpess tests `Cypress run`
3. Tear down the cluster and clean up in case of errors

## Test Development

Use `make dev-tests` to set up for test development and then proceed to add new
tests suites or update and rebuild the WebUI artifacts to see changes in tests.

### Debugging Test Issues

For reproducing and catching test flakes there is a simple helper script `./bin/try-for-flake.sh`.
Just executing the script will run `make test` over and over until it hits a error.

To speed up this process you can:

1. try to avoid the whole cluster set up and tear down on each iteration. If the suspected tests are
re-runable or you have a way of setting up and tearing down the cluster that is faster than the provided
`test-cluster` refer to the "Re-runnable Tests" section.
2. limit the test scope:
  - use `.skip` on each unwanted test suite or `.only` on the target suite.
  - temporarily delete the unwanted test suites

#### Re-runnable Tests

By default the tests are not re-runnable without meeting the conditions in the "Running the Tests" section.
However if you can get the tests to a state where they are re-runnable without the need for the cluster to be
reset, pass in `true` to the test flake script to instruct it to skip the cluster set up and tear down and
greatly speed up the process
