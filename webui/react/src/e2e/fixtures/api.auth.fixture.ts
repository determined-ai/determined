import {
  APIRequest,
  APIRequestContext,
  Browser,
  BrowserContext,
  Page,
} from '@playwright/test';

export class ApiAuthFixture {
    apiContext: APIRequestContext | undefined; // DNJ TODO - how to not have undefined
    readonly request: APIRequest;
    readonly browser: Browser;
    _page: Page | undefined;
    get page() {
        if (this._page === undefined) {
            throw new Error("Accessing page object before initialization in authentication")
        }
        return this._page;
    }
    readonly #STATE_FILE = 'state.json';
    readonly #USERNAME: string;
    readonly #PASSWORD: string;
    context: BrowserContext | undefined;
    constructor(request: APIRequest, browser: Browser, existingPage: Page | undefined = undefined) {
        if (process.env.PW_USER_NAME === undefined) {
            throw new Error('username must be defined');
        }
        if (process.env.PW_PASSWORD === undefined) {
            throw new Error('password must be defined');
        }
        this.#USERNAME = process.env.PW_USER_NAME;
        this.#PASSWORD = process.env.PW_PASSWORD;
        this.request = request;
        this.browser = browser;
        this._page = existingPage;
    }

    async login() {
        this.apiContext = await this.request.newContext();
        await this.apiContext.post(`/api/v1/auth/login`, {
            data: {
                username: this.#USERNAME,
                password: this.#PASSWORD,
                isHashed: false
            }
        });
        // Save cookie state into the file.
        const state = await this.apiContext.storageState({ path: this.#STATE_FILE });        
        if (this._page !== undefined){
            // add cookies to current page's existing context
            this.context = this._page.context();
            this.context.addCookies(state.cookies);
        }else{
            // Create a new context for the browser with the saved token.
            this.context = await this.browser.newContext({ storageState: this.#STATE_FILE });
            this._page = await this.context.newPage();
        }
    }

    async logout() {
        await this.apiContext?.dispose();
        await this.context?.close();
    }
}