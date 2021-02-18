# React WebUI

## Brief Architecture

The **React** webapp was bootstrapped using [Create React App](https://github.com/facebook/create-react-app#create-react-app--) because it brings a lot to the table with minimal setup and package management. Such as DOM router, Typescript, bundle analyzer, linting, style normalizer and webpack config (gzip, minify, hasing, source maps, tree shaking, code splitting, etc).

The following are the notable main packages and libraries we are using:

- [Ant Design](https://ant.design/) - UI component library
- [Storybook](https://storybook.js.org/) - UI component testing and organization
- [CSS Modules](https://create-react-app.dev/docs/adding-a-css-modules-stylesheet/) - CSS Modules for CSS organization
- [Axios](https://github.com/axios/axios) - Promise based HTTP client
- [io-ts](https://github.com/gcanti/io-ts) - Runtime type checking library

## Development

To get started, install all the dependencies for the React webapp.

```sh
npm install
```

You won't have to do this unless the dependencies change under `package.json`.
For example, if a new package was added to the project, simply run command above
again.

Before proceeding, check to make sure you have a database, an instance of master (which serves the WebUI via Go webserver) and an agent running. Follow the instructions at [https://github.com/determined-ai/determined](https://github.com/determined-ai/determined) to get them up and running first if you haven't already.

To start a local development environment for the React webapp, run the command below.

```sh
npm start
```

This will spin up a nodeJS webserver at [localhost:3000](http://localhost:3000). If the page is a blank, do a page refresh and it should take you the landing page for the WebUI.

The page will automatically load and display new changes via [Hot Module Replacement](https://webpack.js.org/concepts/hot-module-replacement/) when you modify the project code. You will also see any lint errors in the console.

## Environment Variables

- `SERVER_ADDRESS`: If set, directs the WebUI to find the Determined cluster at this address.
This allows users to host the WebUI on a separate server from Determined. This would need the target
server to allow requests coming from the domain hosting the WebUI, aka CORS.
- `PUBLIC_URL`: Indicates where the React assets are being served from relative to the root of the webserver.
This is set to `/det` by default for typical workflows in this project.  [More info](https://create-react-app.dev/docs/using-the-public-folder/)
- `DET_NODE_ENV`: set this to `development` to skip some build optimizations when developing and building
locally to lower build time.

## Developing Against a Remote Cluster

If the remote cluster has `enable_cors` set or allows CORS requests set `SERVER_ADDRESS` to point
to the cluster address. If that's not he case use the bundled `local-cors-proxy` package to run a 
proxy pointing to the target server with `npx lcp --proxyUrl <REMOTE_SERVER_URL>` and then build the
webui or the dev server with `SERVER_ADDRESS` pointing to this local proxy.

## Testing

### Style and JS Linting

We check Javascript linting with [eslint](http://eslint.org/) and CSS linting with [stylelint](https://stylelint.io/).

```sh
# check both CSS and JS linting
npm run lint

# check JS linting
npm run lint:js

# check CSS linting
npm run lint:css
```

Our Javascript linting rules and CSS linting rules can be found in `.eslintrc.js` and `.stylelintrc.js` respectively.

### Unit Testing

To launch the unit test runner in the interactive watch mode.

```sh
npm run test
```

See the section about [running tests](https://facebook.github.io/create-react-app/docs/running-tests) for more information.

To skip the interactive mode and run all unit tests.

```sh
npm run test -- --watchAll=false
```

### End-to-end Testing

We are using [Gauge](https://gauge.org/) and [Taiko](https://taiko.dev/) to run end-to-end tests for both of our webapps. Refer to [this section](https://github.com/determined-ai/determined/blob/master/webui/README.md#end-to-end-testing) for details on how to run e2e tests.

### Visual Testing with Storybook

To run a local instance of storybook, run the following command:

```sh
npm run storybook
```

Point the browser to [localhost:9009](http://localhost:9009) to view storybook.

## Deployment

Generally the deployment process from the project repo will handle all of the project build steps including the **React** webapp. However, if you are looking to build a production webapp and seeing it served from the **master** directly, you can follow these steps to manually build production code.

To build the **React** webapp for deployment:

```sh
# build production code into "build"" directory
make build

# copy the production code into where master looks to serve the webapp
make copy-to-build
```

**Create React App** builds and bundles the app properly in production mode with optimizations. The build is minified with hashed filenames.

## Analyze Project Bundle

It is good practice to check the impact of the library you are adding to the project in terms of file size. To run the bundle analysis:

```sh
# Build the project first if you haven't already
npm run build

# Run a bundle analysis
npm run analyze
```

The bundle analyzer will look at the generated source maps for the `build` directory to calculate sizes of the bundle all the different libraries and frameworks make up.

## Webpack Customization

We are heavily leveraging a lot of goodness from **Create React App** discussed above. To continue benefitting from it, we need to avoid ejecting the project. Meaning we do not want to start managing the webpack configuration. The `npm run eject` command is a one-way operation and once you do it, **there is no going back**! The following describes what exactly happens when you do eject.

>If you aren’t satisfied with the build tool and configuration choices, you can `eject` at any time. This command will remove the single build dependency from your project.

> Instead, it will copy all the configuration files and the transitive dependencies (Webpack, Babel, ESLint, etc) right into your project so you have full control over them. All of the commands except `eject` will still work, but they will point to the copied scripts so you can tweak them. At this point you’re on your own.

> You don’t have to ever use `eject`. The curated feature set is suitable for small and middle deployments, and you shouldn’t feel obligated to use this feature. However we understand that this tool wouldn’t be useful if you couldn’t customize it when you are ready for it.

All that being said, we do require some customization for library support, so we have a way around it described in the next section.

### Customize CRA

With [customize-cra](https://github.com/arackaf/customize-cra) we are able to patch our webpack configuration. `customize-cra` provides an overriding config file called `config-override.js` that allows us to inject webpack config we need.

This is not a silver bullet, as there are certain **Create React App** config that can not be overridden still. For example, changing the `build` directory to a different path or name is still not doable.
