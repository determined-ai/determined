import { expect as baseExpect, test as baseTest, Page } from '@playwright/test';

import { detExec, fullPath } from 'e2e/utils/detCLI';
import { apiUrl, isEE } from 'e2e/utils/envVars';
import { safeName } from 'e2e/utils/naming';
import {
  V1PostProjectRequest,
  V1PostProjectResponse,
  V1PostUserRequest,
  V1PostUserResponse,
  V1PostWorkspaceRequest,
  V1PostWorkspaceResponse,
} from 'services/api-ts-sdk/api';

// eslint-disable-next-line no-restricted-imports

import { ApiAuthFixture } from './api.auth.fixture';
import { ApiProjectFixture } from './api.project.fixture';
import { ApiUserFixture } from './api.user.fixture';
import { ApiWorkspaceFixture } from './api.workspace.fixture';
import { AuthFixture } from './auth.fixture';
import { DevFixture } from './dev.fixture';
import { UserFixture } from './user.fixture';

type ExperimentOpts = {
  configFile: string;
  paused?: boolean;
};

type CustomFixtures = {
  devSetup: Page;
  auth: AuthFixture;
  apiAuth: ApiAuthFixture;
  user: UserFixture;
  apiUser: ApiUserFixture;
  apiWorkspace: ApiWorkspaceFixture;
  apiProject: ApiProjectFixture;
  authedPage: Page;
  testExperiments: (count: number, opts: ExperimentOpts) => Promise<string[]>;
};

type CustomWorkerFixtures = {
  dev: DevFixture;
  newAdmin: { request: V1PostUserRequest; response: V1PostUserResponse };
  newWorkspace: { request: V1PostWorkspaceRequest; response: V1PostWorkspaceResponse };
  newProject: { request: V1PostProjectRequest; response: V1PostProjectResponse };
  backgroundApiAuth: ApiAuthFixture;
  backgroundApiUser: ApiUserFixture;
  backgroundApiWorkspace: ApiWorkspaceFixture;
  backgroundApiProject: ApiProjectFixture;
  backgroundAuthedPage: Page;
};

// https://playwright.dev/docs/test-fixtures
export const test = baseTest.extend<CustomFixtures, CustomWorkerFixtures>({
  // get the auth but allow yourself to log in through the api manually.
  apiAuth: async ({ playwright, browser, newAdmin, devSetup }, use) => {
    const apiAuth = new ApiAuthFixture(playwright.request, browser, apiUrl(), devSetup);
    await apiAuth.loginApi({
      creds: {
        password: newAdmin.request.password!,
        username: newAdmin.response.user!.username,
      },
    });
    await apiAuth.apiContext?.post('/api/v1/users/setting', {
      data: {
        settings: [
          {
            key: 'flat_runs',
            storagePath: 'global-features',
            value: 'false',
          },
        ],
      },
    });
    await use(apiAuth);
  },

  apiProject: async ({ apiAuth }, use) => {
    const apiProject = new ApiProjectFixture(apiAuth);
    await use(apiProject);
  },

  apiUser: async ({ apiAuth }, use) => {
    const apiUser = new ApiUserFixture(apiAuth);
    await use(apiUser);
  },

  apiWorkspace: async ({ apiAuth }, use) => {
    const apiWorkspace = new ApiWorkspaceFixture(apiAuth);
    await use(apiWorkspace);
  },

  auth: async ({ page }, use) => {
    const auth = new AuthFixture(page);
    await use(auth);
  },

  // get the existing page but with auth cookie already logged in
  authedPage: async ({ apiAuth }, use) => {
    await apiAuth.loginBrowser(apiAuth.page);
    await use(apiAuth.page);
  },

  /**
   * Does not require the pre-existing Playwright page and does not login so this can be called in beforeAll.
   * Generally use another api fixture instead if you want to call an api. If you just want a logged-in page,
   * use apiAuth in beforeEach().
   */
  backgroundApiAuth: [
    async ({ playwright, browser }, use) => {
      const backgroundApiAuth = new ApiAuthFixture(playwright.request, browser, apiUrl());
      await backgroundApiAuth.loginApi();
      await backgroundApiAuth.apiContext?.post('/api/v1/users/setting', {
        data: {
          settings: [
            {
              key: 'flat_runs',
              storagePath: 'global-features',
              value: 'false',
            },
          ],
        },
      });
      await use(backgroundApiAuth);
      await backgroundApiAuth.dispose();
    },
    { scope: 'worker' },
  ],

  backgroundApiProject: [
    async ({ backgroundApiAuth }, use) => {
      const backgroundApiProject = new ApiProjectFixture(backgroundApiAuth);
      await use(backgroundApiProject);
    },
    { scope: 'worker' },
  ],
  /**
   * Allows calling the user api without a page so that it can run in beforeAll(). You will need to get a bearer
   * token by calling backgroundApiUser.apiAuth.loginAPI(). This will also provision a page in the background which
   * will be disposed of logout(). Before using the page,you need to call dev.setServerAddress() manually and
   * then login() again, since setServerAddress logs out as a side effect.
   */
  backgroundApiUser: [
    async ({ backgroundApiAuth }, use) => {
      const backgroundApiUser = new ApiUserFixture(backgroundApiAuth);
      await use(backgroundApiUser);
    },
    { scope: 'worker' },
  ],
  backgroundApiWorkspace: [
    async ({ backgroundApiAuth }, use) => {
      const backgroundApiWorkspace = new ApiWorkspaceFixture(backgroundApiAuth);
      await use(backgroundApiWorkspace);
    },
    { scope: 'worker' },
  ],

  /**
   * API authenticated page for use in beforeAll()
   */
  backgroundAuthedPage: [
    async ({ browser, dev, backgroundApiAuth }, use) => {
      const page = await browser.newPage();
      await dev.setServerAddress(page);
      await backgroundApiAuth.loginBrowser(page);
      await use(page);
      await page.close();
    },
    { scope: 'worker' },
  ],
  dev: [
    // eslint-disable-next-line no-empty-pattern
    async ({}, use) => {
      const dev = new DevFixture();
      await use(dev);
    },
    { scope: 'worker' },
  ],
  devSetup: [
    async ({ dev, page }, use) => {
      await dev.setServerAddress(page);
      await use(page);
    },
    { auto: true },
  ],
  /**
   * Creates an admin and logs in as that admin for the duraction of the test suite
   */
  newAdmin: [
    async ({ backgroundApiUser }, use, workerInfo) => {
      const request = backgroundApiUser.new({
        userProps: {
          user: {
            active: true,
            admin: true,
            username: safeName(`test-admin-${workerInfo.workerIndex}`),
          },
        },
      });
      const adminUser = await backgroundApiUser.createUser(request);
      await use({ request, response: adminUser });
      await backgroundApiUser.patchUser(adminUser.user!.id!, { active: false });
    },
    { scope: 'worker' },
  ],

  /**
   * Creates a project and deletes it after the test suite
   */
  newProject: [
    async ({ backgroundApiProject, newWorkspace }, use, workerInfo) => {
      const workspaceId = newWorkspace.response.workspace!.id!;
      const request = backgroundApiProject.new({
        projectProps: {
          name: safeName(`test-project-${workerInfo.workerIndex}`),
          workspaceId,
        },
      });
      const newProject = await backgroundApiProject.createProject(workspaceId, request);
      await use({ request, response: newProject });
      await backgroundApiProject.deleteProject(newProject.project!.id!);
    },
    { scope: 'worker' },
  ],

  /**
   * Creates a workspace and deletes it after the test suite
   */
  newWorkspace: [
    async ({ backgroundApiWorkspace }, use, workerInfo) => {
      const request = backgroundApiWorkspace.new({
        workspaceProps: {
          name: safeName(`test-workspace-${workerInfo.workerIndex}`),
        },
      });
      const newWorkspace = await backgroundApiWorkspace.createWorkspace(request);
      await use({ request, response: newWorkspace });
      await backgroundApiWorkspace.deleteWorkspace(newWorkspace.workspace!.id!);
    },
    { scope: 'worker' },
  ],
  testExperiments: [
    async ({ newProject }, use) => {
      const expIds: string[] = [];
      const makeExperiments = (count: number, { configFile, paused }: ExperimentOpts) => {
        const projectId = newProject.response.project.id;
        const pausedFlag = paused ? '--paused' : '';
        return Promise.all(
          Array(count)
            .fill(null)
            .map(async () => {
              const output = await detExec(
                `e create ${fullPath(configFile)} ${pausedFlag} --project_id ${projectId}`,
              );
              const expId = output.match(/(\d+)[\r\n]/)?.[1];
              if (!expId) {
                throw new Error('experiment id not found');
              }
              expIds.push(expId);
              return expId;
            }),
        );
      };
      await use(makeExperiments);
      return Promise.all(expIds.map((id) => detExec(`e delete --yes ${id}`)));
    },
    { scope: 'test' },
  ],
  user: async ({ page }, use) => {
    const user = new UserFixture(page);
    await use(user);
  },
});

export const expect = baseExpect.extend({
  async toHaveDeterminedTitle(page: Page, titleOrRegExp: string | RegExp, options?: object) {
    let message: () => string;
    let pass: boolean;
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    let matcherResult: any;

    const appTitle = isEE() ? 'HPE Machine Learning Development Environment' : 'Determined';

    const getFullTitle = (prefix: string = '') => {
      if (prefix === '') {
        return appTitle;
      }
      return `${prefix} - ${appTitle}`;
    };

    try {
      if (typeof titleOrRegExp === 'string') {
        const fullTitle = getFullTitle(titleOrRegExp);
        await baseExpect(page).toHaveTitle(fullTitle, options);
      } else {
        const fullTitle = new RegExp(getFullTitle(titleOrRegExp.source));
        await baseExpect(page).toHaveTitle(fullTitle, options);
      }
      message = () => `expected page to have title ${titleOrRegExp}, but it did not`;
      pass = true;
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
    } catch (e: any) {
      matcherResult = e.matcherResult;
      pass = false;
      const actualTitle = await page.title();
      message = () =>
        `expected page to have title matching ${titleOrRegExp}, but received ${actualTitle}`;
    }

    return {
      actual: matcherResult?.actual,
      expected: titleOrRegExp,
      message,
      name: 'toHaveDeterminedTitle',
      pass,
    };
  },
});
