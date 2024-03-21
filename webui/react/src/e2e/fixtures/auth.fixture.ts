import { Page } from '@playwright/test';
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
    const detAuth = this.signInPage.detAuth
    await detAuth.username.pwLocator.fill(this.#USERNAME)
    await detAuth.password.pwLocator.fill(this.#PASSWORD);
    await detAuth.submit.pwLocator.click();
    await this.#page.waitForURL(waitForURL);
  }

  async logout(): Promise<void> {
    await this.#page.locator('header').getByText(this.#USERNAME).click();
    await this.#page.getByRole('link', { name: 'Sign Out' }).click();
    await this.#page.waitForURL(/login/);
  }
}
