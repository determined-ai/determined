# E2E

## Framework

Deteremined AI uses [Playwright 🎭](https://playwright.dev/).

## How to run locally

- Create `.env` file in `webui/react` like `webui/react/.env`
- Add env variables `PW_USER_NAME`, `PW_PASSWORD`, and `PW_SERVER_ADDRESS` (`PW_` prefix stands for Playwright)
  - `PW_USER_NAME`: user name for determined account
  - `PW_PASSWORD`: password for determined account
  - `PW_SERVER_ADDRESS`: API server address
- Run `npx playwright install`
- Run `SERVER_ADDRESS={set server address} npm run build` in `webui/react`
  - It is `SERVER_ADDRESS` here. not `PW_SERVER_ADDRESS`, but the both values should be the same
- Run `npm run e2e` in `webui/react`. Use `-- --project=<browsername>` to run a specific browser.

\*\*Avoid using `make` command because it does not take env variables

### Quick start testing using det deploy

If you don't want to use dev cluster, you can use det deploy to initiate the backend. These commands should run and pass tests on chrome:

1. `det deploy local cluster-up --det-version="0.29.0" --no-gpu --master-port=8080`
   - use whatever det-version you want here.
2. `SERVER_ADDRESS="http://localhost:3001" npm run build --prefix webui/react`
3. Optional if you want an experiment created for the test: `det experiment create ./examples/tutorials/mnist_pytorch/const.yaml ./examples/tutorials/mnist_pytorch/`
4. `npm run preview --prefix webui/react` to run the preview app. Not necessary if CI=true
5. To run the tests: `PW_SERVER_ADDRESS="http://localhost:3001"  PW_USER_NAME="admin" PW_PASSWORD="" npm run e2e --prefix webui/react`

## CI

CI is setup as `test-e2e-react` in `.circleci/config.yml`.

We use `mcr.microsoft.com/playwright` for [docker container](https://playwright.dev/docs/docker).
Update the docker image version along with Playwright version.

# TODO
* Page Models
* isDisplayed needs to check all parents, might need to find a way to play nice with expect.to.be.displayed somehow
  how to waitToBeDisplayed()?
* consider proxy over `get locator()`
  * https://stackoverflow.com/questions/1529496/is-there-a-javascript-equivalent-of-pythons-getattr-method
