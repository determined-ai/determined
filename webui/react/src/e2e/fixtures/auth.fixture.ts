import { Page } from '@playwright/test';
import { DeterminedAuth } from 'e2e/models/components/DeterminedAuth';
import { SignIn } from 'e2e/models/pages/SignIn';

export class AuthFixture {
  readonly #page: Page;
  readonly #USERNAME: string;
  readonly #PASSWORD: string;
  readonly signInPage: SignIn;

  constructor(readonly page: Page) {
    if (typeof process.env.PW_USER_NAME === "undefined") {
      throw new Error('username must be defined')
    }
    if (typeof process.env.PW_PASSWORD === "undefined") {
      throw new Error('username must be defined')
    }
    this.#USERNAME = process.env.PW_USER_NAME;
    this.#PASSWORD = process.env.PW_PASSWORD;
    this.#page = page;
    this.signInPage = new SignIn(page);
  }

  async login(waitForURL: string | RegExp | ((url: URL) => boolean)): Promise<void> {
    const auth = this.signInPage.sc.get('determinedAuth') as DeterminedAuth
    // TODO subcomponents - kind of annoying to need to cast here. maybe a helper can help
    
    await this.#page.getByPlaceholder('username').fill(this.#USERNAME);
    await this.#page.getByPlaceholder('password').fill(this.#PASSWORD);
    await this.#page.getByRole('button', { name: 'Sign In' }).click();
    await this.#page.waitForURL(waitForURL);
  }

  async logout(): Promise<void> {
    await this.#page.locator('header').getByText(this.#USERNAME).click();
    await this.#page.getByRole('link', { name: 'Sign Out' }).click();
    await this.#page.waitForURL(/login/);
  }
}
