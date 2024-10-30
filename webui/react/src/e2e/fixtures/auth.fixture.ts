import { Page } from '@playwright/test';

import { expect } from 'e2e/fixtures/global-fixtures';
import { DefaultRoute } from 'e2e/models/pages/DefaultRoute';
import { SignIn } from 'e2e/models/pages/SignIn';
import { password, username } from 'e2e/utils/envVars';

export class AuthFixture {
  readonly #page: Page;
  readonly #USERNAME: string;
  readonly #PASSWORD: string;
  readonly signInPage: SignIn;

  constructor(readonly page: Page) {
    this.#USERNAME = username();
    this.#PASSWORD = password();
    this.#page = page;
    this.signInPage = new SignIn(page);
  }

  async login({
    expectedURL,
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

    const defaultPage = new DefaultRoute(this.#page);
    await this.#page.waitForURL(expectedURL ?? defaultPage.url);
  }

  async logout(): Promise<void> {
    await (await this.signInPage.nav.sidebar.headerDropdown.open()).signOut.pwLocator.click();
    await expect.soft(this.#page).toHaveDeterminedTitle(this.signInPage.title);
    await this.signInPage.waitForURL();
  }
}
