import { Page } from '@playwright/test';

export class AuthFixture {
  readonly #page: Page;
  readonly #USERNAME = process.env.PW_USER_NAME ?? '';
  readonly #PASSWORD = process.env.PW_PASSWORD ?? '';
  constructor(readonly page: Page) {
    this.#page = page;
  }

  async login(waitForURL: string | RegExp | ((url: URL) => boolean)): Promise<void> {
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
