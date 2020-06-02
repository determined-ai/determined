# WebUI End-to-end Tests

We use [Cypress](https://www.cypress.io/) to enable end-to-end testing of our
two SPAs, Elm and React, and in some sense the whole det cluster.

We provide a test script to take care of depdencies around Cypress commands.
Let's call this the "test script" for the rest of the this document.

## Requirements

When the tester loads up it loads the WebUI through a given master address
(`$HOST`) and port (`$PORT`)

Based the tester requirements and ease of test development the following
assumptions are made:

- It's a brand new cluster
- The cluster is isolated. Meaning this tester is the only entity interacting
with the cluster

Once the cluster is ready and accessible run the tests:

```
./bin/e2e-tests.py --det-port $PORT pre-e2e-tests && \
./bin/e2e-tests.py --det-port $PORT run-e2e-tests
```

Note that it's is important that one immediately follows the other since the
pre-e2e-tests target starts some experiments.

### Using the bundled cluster manager

For ease of use and quicker local testing we provide an option to set up and
tear down such a cluster through the test script.

Before the tests can be started we need to build the cluster including the WebUI
to make sure the served WebUI and cluster in general are up to date.

Issue the following command:
`./bin/e2e-tests.py e2e-tests` which in turn will:

1. Set up a test cluster
2. Run the Cyrpess tests `Cypress run`
3. Tear down the cluster and clean up in case of errors
