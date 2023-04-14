import { expect, Page } from '@playwright/test';

export class DevFixture {
  readonly #page: Page;
  constructor(readonly page: Page) {
    this.#page = page;
  }

  async setServerAddress(): Promise<void> {
    await this.#page.goto('/');
    await expect(this.#page).toHaveURL(/localhost:3001/);
    await this.#page.evaluate(`dev.setServerAddress("${process.env.PW_SERVER_ADDRESS}")`);
  }
}
