# React Helpers

## proxy.js

You can use `proxy.js` to bypass CORS restrictions when communicating with a remote server,
eg a remote Determined master.

For example, to connect the WebUI to a remote cluster with address `MY_SERVER_ADDRESS` you'd
run the proxy with `./proxy.js MY_SERVER_ADDRESS`. This will start a local server which is
by default on port `8100`.  This local server would now behave similar to `MY_SERVER_ADDRESS`.
You can now Use `http://localhost:8100/fixed` wherever you were running into CORS issues with
before, instead of `MY_SERVER_ADDRESS`.

