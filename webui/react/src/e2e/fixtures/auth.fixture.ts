import { Page } from '@playwright/test';

import { expect } from 'e2e/fixtures/global-fixtures';
import { SignIn } from 'e2e/models/pages/SignIn';

export class AuthFixture {
  readonly #page: Page;
  readonly #USERNAME: string;
  readonly #PASSWORD: string;
  readonly signInPage: SignIn;

  constructor(readonly page: Page) {
    if (process.env.PW_USER_NAME === undefined) {
      throw new Error('username must be defined');
    }
    if (process.env.PW_PASSWORD === undefined) {
      throw new Error('password must be defined');
    }
    this.#USERNAME = process.env.PW_USER_NAME;
    this.#PASSWORD = process.env.PW_PASSWORD;
    this.#page = page;
    this.signInPage = new SignIn(page);
  }

  async login({
    expectedURL = /dashboard/,
    username = this.#USERNAME,
    password = this.#PASSWORD,
  }: {
    expectedURL?: string | RegExp | ((url: URL) => boolean);
    username?: string;
    password?: string;
  } = {}): Promise<void> {
    const detAuth = this.signInPage.detAuth;
    if (!(await detAuth.pwLocator.isVisible())) {
      await this.#page.goto('/');
      await expect(detAuth.pwLocator).toBeVisible();
      await expect.soft(this.signInPage.detAuth.submit.pwLocator).toBeDisabled();
    }
    await detAuth.username.pwLocator.fill(username);
    await detAuth.password.pwLocator.fill(password);
    await detAuth.submit.pwLocator.click();
    await this.#page.waitForURL(expectedURL);
  }

  async logout(): Promise<void> {
    await (await this.signInPage.nav.sidebar.headerDropdown.open()).signOut.pwLocator.click();
    await expect.soft(this.#page).toHaveDeterminedTitle(this.signInPage.title);
    await this.signInPage.waitForURL();
  }
}
