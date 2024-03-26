import { Page, expect } from '@playwright/test';
import { BaseComponent, parentTypes } from 'e2e/models/BaseComponent';
import { BasePage } from 'e2e/models/BasePage';

export class DevFixture {
  readonly #page: Page;
  constructor(readonly page: Page) {
    this.#page = page;
  }

  async setServerAddress(): Promise<void> {
    await this.#page.goto('/');
    await this.#page.evaluate(`dev.setServerAddress("${process.env.PW_SERVER_ADDRESS}")`);
    await this.#page.reload();
  }

  async debugComponentVisible(component: BaseComponent): Promise<void> {
    let componentTree: parentTypes[] = []
    let root: parentTypes = component;
    for (; !(root instanceof BasePage); root = root._parent) {
      componentTree.unshift(root)
    }
    componentTree.forEach(async branch => {
      await expect(branch.pwLocator).toBeVisible()
    });
  }
}
