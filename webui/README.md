# WebUI

We [React](https://reactjs.org/) as a WebUI framework. React has been the most
popular frontend framework and enjoys the benefits of being a mature and stable
framework. As a result we see these benefits:

* True live development where new written code instantly transpiles into updated
browser experience via Hot Module Replacement (HMR).
* A very large number of supporting libraries and frameworks specific to React.
* Many of the bugs and kinks have been identified and fixed.
* Stable design patterns have mostly been figured out.
* Larger pool of candidates we can hire from due to React's popularity.
* Faster development in general.

## Running the WebUI

Before starting this section, please get Determined set up properly by following
the [Determined setup instructions](https://github.com/determined-ai/determined).

Starting the master kick-starts a Go web server that serves WebUI static files.
There are two ways to start master. The more common way is to run master via Docker.
The other method is to run natively (without Docker) via `determined-master`.

1. [Running Master via Docker](https://github.com/determined-ai/determined#local-deployment)
2. [Running Master without Docker (Natively)](https://github.com/determined-ai/determined/wiki/Useful-tools#master)


## Local Development

For local development, our goal is to set up an environment to...

* Auto detect changes in the source code and update the WebUI on the browser to
speed up development. Also known as [Hot Module Replacement](https://webpack.js.org/concepts/hot-module-replacement/)
* Provide a debugging environment via source maps. Only applicable to the React SPA.


### Running React Live

To start React live, simply run the following and point your browser to `http://localhost:3000/`.

```sh
cd /PATH/TO/DETERMINED/webui/react
npm start
```

If the above fails, it's possible that the project dependencies are not built yet.

```sh
# install all the dependencies according to package.json
npm install
```

Couple of things to note:

* No need to manually reload the browser page upon code change. The page itself will auto reload upon TypeScript, JavaScript, HTML, CSS, SASS and LESS changes.
* Pointing to `http://localhost:3000` will show a blank page until you refresh because the base route of `/` is not owned by the React app.

## Testing

To run unit tests for each of the SPAs issue `make test` in their respective directories.

### End-to-end testing

We use [Gauge](https://gauge.org/) and [Taiko](https://taiko.dev/) for end-to-end testing. To run the tests locally with chromium browser, run `make dev-tests` in the `webui/tests` directory. To run the tests in headless mode, run `make test`.

#### Dependencies

Note that the e2e tests need a functional webserver (master) to serve the UI and potentially a full cluster to be able to run all the tests.

Be sure to activate the python virtual environment if you haven't already then install webui test dependencies before kicking off the tests.

```sh
# activate python virtual environment
workon determined # or `pipenv shell`

# install webui test dependencies
make -C webui/tests get-deps`
```
