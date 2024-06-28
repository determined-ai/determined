import { APIRequest, APIRequestContext, Browser, BrowserContext, Page } from '@playwright/test';

import { baseUrl, password, username } from 'e2e/utils/envVars';

export class ApiAuthFixture {
  apiContext?: APIRequestContext; // we can't get this until login, so may be undefined
  readonly request: APIRequest;
  readonly browser: Browser;
  readonly baseURL: string;
  _page?: Page;
  get page(): Page {
    if (this._page === undefined) {
      throw new Error('Accessing page object before initialization in authentication');
    }
    return this._page;
  }
  browserContext?: BrowserContext;

  constructor(request: APIRequest, browser: Browser, baseURL: string, existingPage?: Page) {
    this.request = request;
    this.browser = browser;
    this.baseURL = baseURL;
    this._page = existingPage;
  }

  async getBearerToken(noBearer = false): Promise<string> {
    const cookies = (await this.apiContext?.storageState())?.cookies ?? [];
    const authToken = cookies.find((cookie) => {
      return cookie.name === 'auth';
    })?.value;
    if (authToken === undefined) {
      throw new Error(
        'Attempted to retrieve the auth token from the PW apiContext, but it does not exist. Have you called apiAuth.login() yet?',
      );
    }
    if (noBearer) return authToken;
    return `Bearer ${authToken}`;
  }

  /**
   * Logs in via the API. If there is a browser context already assosciated with the
   * fixture, the bearer token will be attached to that context. If not a new
   * browser context will be created with the cookie.
   */
  async loginApi({ creds = { password: password(), username: username() } } = {}): Promise<void> {
    this.apiContext = this.apiContext || (await this.request.newContext({ baseURL: this.baseURL }));
    const resp = await this.apiContext.post('/api/v1/auth/login', {
      data: {
        ...creds,
        isHashed: false,
      },
    });
    if (resp.status() !== 200) {
      throw new Error(`Login API request has failed with status code ${resp.status()}`);
    }
  }

  async loginBrowser(page: Page): Promise<void> {
    if (this.apiContext === undefined) {
      throw new Error('Cannot login browser without first logging in API');
    }
    // Save cookie state into the file.
    if (this._page !== undefined) {
      const state = await this.apiContext.storageState();
      // add cookies to current page's existing context
      this.browserContext = this._page.context();
      // replace the domain of api base url with browser base url
      state.cookies.forEach((cookie) => {
        if (cookie.name === 'auth' && cookie.domain === new URL(this.baseURL).hostname) {
          cookie.domain = new URL(baseUrl()).hostname;
        }
      });
      await this.browserContext.addCookies(state.cookies);
      const token = JSON.stringify(await this.getBearerToken(true));
      await page.evaluate((token) => localStorage.setItem('global/auth-token', token), token);
    }
  }

  /**
   * This disposes of all API resources. It should only be called
   * in afterAll. In that case new contexts will have been manually
   * provisioned. If you dispose of these contexts mid-test any
   * tests still using them test will fail.
   */
  async dispose(): Promise<void> {
    await this.apiContext?.dispose();
    await this.browserContext?.close();
  }
}
