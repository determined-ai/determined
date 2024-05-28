import {
  APIRequest,
  APIRequestContext,
  Browser,
  BrowserContext,
  Page,
} from '@playwright/test';
import { v4 } from 'uuid';

export class ApiAuthFixture {
  apiContext: APIRequestContext | undefined; // we can't get this until login, so may be undefined
  readonly request: APIRequest;
  readonly browser: Browser;
  readonly baseURL: string;
  readonly testId = v4();
  _page: Page | undefined;
  get page(): Page {
    if (this._page === undefined) {
      throw new Error('Accessing page object before initialization in authentication');
    }
    return this._page;
  }
  readonly #STATE_FILE_SUFFIX = 'state.json';
  readonly #USERNAME: string;
  readonly #PASSWORD: string;
  context: BrowserContext | undefined;
  readonly #stateFile = `${this.testId}-${this.#STATE_FILE_SUFFIX}`;

  constructor(
    request: APIRequest,
    browser: Browser,
    baseURL: string | undefined,
    existingPage: Page | undefined = undefined,
  ) {
    if (process.env.PW_USER_NAME === undefined) {
      throw new Error('username must be defined');
    }
    if (process.env.PW_PASSWORD === undefined) {
      throw new Error('password must be defined');
    }
    if (baseURL === undefined) {
      throw new Error('baseURL must be defined in playwright config to use API requests.');
    }
    this.#USERNAME = process.env.PW_USER_NAME;
    this.#PASSWORD = process.env.PW_PASSWORD;
    this.request = request;
    this.browser = browser;
    this.baseURL = baseURL;
    this._page = existingPage;
  }

  async getBearerToken(): Promise<string> {
    const cookies = (await this.apiContext?.storageState())?.cookies ?? [];
    const authToken = cookies.find((cookie) => {
      return cookie.name === 'auth';
    })?.value;
    if (authToken === undefined) {
      throw new Error(
        'Attempted to retrieve the auth token from the PW apiContext, but it does not exist. Have you called apiAuth.login() yet?',
      );
    }
    return `Bearer ${authToken}`;
  }

  /**
   * Logs in via the API. If there is a browser context already assosciated with the
   * fixture, the bearer token will be attached to that context. If not a new
   * browser ontext will be created with the cookie.
   */
  async login(): Promise<void> {
    this.apiContext = await this.request.newContext();
    const resp = await this.apiContext.post('/api/v1/auth/login', {
      data: {
        isHashed: false,
        password: this.#PASSWORD,
        username: this.#USERNAME,
      },
    });
    if (resp.status() !== 200) {
      throw new Error(`Login API request has failed with status code ${resp.status()}`);
    }
    // Save cookie state into the file.
    const state = await this.apiContext.storageState({ path: this.#stateFile });
    if (this._page !== undefined) {
      // add cookies to current page's existing context
      this.context = this._page.context();
      await this.context.addCookies(state.cookies);
    } else {
      // Create a new context for the browser with the saved token.
      this.context = await this.browser.newContext({ storageState: this.#stateFile });
      this._page = await this.context.newPage();
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
    await this.context?.close();
  }
}
