# Determined experiment spinning off a flask server

This example includes two experiments:

1. `hello-server`, a flask-based "hello world" web app.
2. `hello-client`, which launches `hello-server`, waits for the server to stand up, makes a request to it, then kills it and shuts down.

To launch this example:

    det e create hello-client.yaml . -f

Upon successful completion, you should see the following in the experiment logs:

    Got server response:  {'data': 'Hello World'}
    SUCCESS!
    Killed experiment <hello-server experiment id>
    hello-server is killed.
