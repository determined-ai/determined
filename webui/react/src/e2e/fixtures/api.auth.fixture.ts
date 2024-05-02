import {
  APIRequest,
  APIRequestContext,
  Browser,
  BrowserContext,
} from '@playwright/test';

export class ApiAuthFixture {
    apiContext: APIRequestContext | undefined; // DNJ TODO - how to not have undefined
    readonly request: APIRequest;
    readonly #STATE_FILE = 'state.json';
    readonly #USERNAME: string;
    readonly #PASSWORD: string;
    context: BrowserContext | undefined;
    constructor(request: APIRequest) {
        if (process.env.PW_USER_NAME === undefined) {
            throw new Error('username must be defined');
        }
        if (process.env.PW_PASSWORD === undefined) {
            throw new Error('password must be defined');
        }
        this.#USERNAME = process.env.PW_USER_NAME;
        this.#PASSWORD = process.env.PW_PASSWORD;
        this.request = request;
    }

    async login(browser: Browser) {

        this.apiContext = await this.request.newContext({
            httpCredentials: {
                username: this.#USERNAME,
                password: this.#PASSWORD
            }
        });
        await this.apiContext.get(`/login`);
        // Save cookie state into the file.
        await this.apiContext.storageState({ path: this.#STATE_FILE });
        // Create a new context for the browser with the saved token.
        this.context = await browser.newContext({ storageState: this.#STATE_FILE });
    }

    async useContext(){

    }
    async logout() {
        this.apiContext?.dispose();
        this.context?.close();
    }
}