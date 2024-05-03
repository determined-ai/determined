import {
  BaseComponent,
  CanBeParent,
} from 'e2e/models/BaseComponent';
import { BasePage } from 'e2e/models/BasePage';

import {
  expect,
  Page,
} from '@playwright/test';

export class DevFixture {
  readonly #page: Page;
  constructor(readonly page: Page) {
    this.#page = page;
  }
  // tells the frontend where to find the backend if built for a different url. Incidentally reloads and logs out of Determined.
  async setServerAddress(): Promise<void> {
    await this.#page.goto('/');
    await this.#page.evaluate(`dev.setServerAddress("${process.env.PW_SERVER_ADDRESS}")`);
    await this.#page.reload();
  }

  /**
   * Attempts to locate each element in the locator tree. If there is an error at this step,
   * the last locator in the error message is the locator that couldn't be found and needs
   * to be debugged. If there is no error message, the component could be located and this
   * debug line can be removed.
   * @param {BaseComponent} component - The component to debug
   */
  debugComponentVisible(component: BaseComponent): void {
    const componentTree: CanBeParent[] = [];
    let root: CanBeParent = component;
    while (!(root instanceof BasePage)) {
      componentTree.unshift(root);
      root = root._parent;
    }
    componentTree.forEach(async (node) => {
      await expect(node.pwLocator).toBeVisible();
    });
  }
}
