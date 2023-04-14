# E2E

## Framework

Deteremined AI uses [Playwright 🎭](https://playwright.dev/).

## How to run locally

- Create `.env` file in `webui/react` like `webui/react/.env`
- Add env variables `PW_USER_NAME`, `PW_PASSWORD`, and `PW_SERVER_ADDRESS` (`PW_` prefix stands for Playwright)
  - `PW_USER_NAME`: user name for determined account
  - `PW_PASSWORD`: password for determined account
  - `PW_SERVER_ADDRESS`: API server address
- Run `make build` in `webui/react`
- Run `make e2e` in `webui/react`

## CI

CI is setup as `test-e2e-react` in `.circleci/config.yml`
