# WebUI

We are currently exploring [React](https://reactjs.org/) as a serious contender for the WebUI framework. React has been the most popular frontend framework and enjoys the benefits of being a mature and stable framework. As a result we see these benefits:

* True live development where new written code instantly transpiles into updated browser experience via Hot Module Replacement (HMR).
* A very large number of supporting libraries and frameworks specific to React.
* Many of the bugs and kinks have been identified and fixed.
* Stable design patterns have mostly been figured out.
* Larger pool of candidates we can hire from due to React's popularity.
* Faster development in general.

## Current WebUI State

We are currently testing React single page app (SPA) in conjunction with the original Elm SPA. The setup involves running both SPAs simultaneously and seamlessly to reduce UX friction. As far as the WebUI user is concerned, the goal is to provide an experience where they can not discern the boundaries between Elm and React.

As part of master, a Go web server serves the built WebUI static files. This includes the static files for docs, Elm SPA and React SPA. Depending on the requested route path, the web server serves from one of three files. The diagram below illustrates which SPA owns which routes.

```
+-----------------------------------------------------------------------------------------------------+
|                                                                                                     |
|    Determined Master                                                                                      |
|                                                                                                     |
|    +-------------------------------------------------------------------------------------------+    |
|    |                                                                                           |    |
|    |    Go Web Server                                                                          |    |
|    |                                                                                           |    |
|    |    +----------------------------+    +----------------------------+    +-------------+    |    |
|    |    |                            |    |                            |    |             |    |    |
|    |    |    Elm SPA Routes          |    |    React SPA Routes        |    |    Docs     |    |    |
|    |    |    /                       |    |    /det                    |    |    /docs    |    |    |
|    |    |    /ui                     |    |      /dashboard            |    |             |    |    |
|    |    |      /login                |    |                            |    |             |    |    |
|    |    |      /logout               |    |                            |    |             |    |    |
|    |    |      /experiments          |    |                            |    |             |    |    |
|    |    |        /<exp id>           |    |                            |    |             |    |    |
|    |    |      /trials/<trial id>    |    |                            |    |             |    |    |
|    |    |      /notebooks            |    |                            |    |             |    |    |
|    |    |      /commands             |    |                            |    |             |    |    |
|    |    |      /tensorboards         |    |                            |    |             |    |    |
|    |    |      /shells               |    |                            |    |             |    |    |
|    |    |      /cluster              |    |                            |    |             |    |    |
|    |    |                            |    |                            |    |             |    |    |
|    |    +----------------------------+    +----------------------------+    +-------------+    |    |
|    |                                                                                           |    |
|    +-------------------------------------------------------------------------------------------+    |
|                                                                                                     |
+-----------------------------------------------------------------------------------------------------+
```

Assuming we are on localhost, when pointing the browser to `http://localhost:8080/det/dashboard`, the web server serves the static files from the React directory. Where as with `http://localhost:8080/ui/experiments`, the web server serves from the Elm directory.

During a `make build` under master, Docs, Elm and React `make build` are automatically triggered to build their respective static files. Upon completion, they are copied over into the build directory.


## Running the WebUI

Before starting this section, please get Determined setup properly by following the [Determined setup instructions](https://github.com/determined-ai/determined).

Starting the master kick-starts a Go web server that serves WebUI static files. There are two ways to start master. The more common way is to run master via Docker. The other method is to run natively (without Docker) via `determined-master`.

1. [Running Master via Docker](https://github.com/determined-ai/determined#local-deployment)
2. [Running Master without Docker (Natively)](https://github.com/determined-ai/determined/wiki/Useful-tools#master)


## Local Development

For local development, our goal is to set up an environment to...

* Auto detect changes in the source code and update the WebUI on the browser to speed up development. Also known as [Hot Module Replacement](https://webpack.js.org/concepts/hot-module-replacement/)
* Provide a debugging environment via source maps. Only applicable to the React SPA.


### Running Elm Live

Run the following from the shell and hit the browser refresh button after any Elm code changes.

```sh
# sometimes elm-live doesn't cleanly exit and blocks the 8000 port
kill $(lsof -i4TCP:8000 | grep node | awk '{print $2}')
# change directory to the elm project directory and run elm-live
cd /PATH/TO/DETERMINED/webui/elm
make live
```

Elm live detects changes to elm files and automatically builds the core Elm file `determined-ui.js` and places it in shared build directory for Go web server to pick.

Couple of things to note:

* This still requires the browser to be manually refreshed to see the changes.
* Changes in the static files such as `public/styles-in.css` and `public/css/wait.css` will NOT get auto updated. A fix is in the works to address this.


### Running React Live

To start React live, simply run the following and point your browser to `http://localhost:3000/`.

```sh
cd /PATH/TO/DETERMINED/webui/react
yarn start
```

If the above fails, it's possible that the project dependencies are not built yet.

```sh
# install all the dependencies according to package.json
yarn install
```

Couple of things to note:

* No need to manually reload the browser page upon code change. The page itself will auto reload upon TypeScript, JavaScript, HTML, CSS, SASS and LESS changes.
* Pointing to `http://localhost:3000` will show a blank page until you refresh because the base route of `/` is owned by the Elm app.

## Testing

To run unit tests for each of the SPAs issue `make test` in their respective directories.

### End-to-end testing

We use [Cypress](https://www.cypress.io/) for end-to-end testing. To run the tests issue `make e2e-tests` or `make docker-e2e-tests` depending on which test runner you'd like to use in the `webui/tests` directory.

#### Dependencies

Note that the e2e tests need a functional webserver (master) to serve the UI and potentially a full cluster to be able to run all the tests.

Be sure to activate the python virtual environment if you haven't already then install webui test dependencies before kicking off the tests.

```sh
# activate python virtual environment
workon determined # or `pipenv shell`

# install webui test dependencies
make -C webui/tests get-deps`
```
