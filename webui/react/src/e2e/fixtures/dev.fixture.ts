import { Page } from '@playwright/test';
import {
  BaseComponent,
  ComponentBasics,
  ComponentContainer,
} from 'playwright-page-model-base/BaseComponent';

import { expect } from 'e2e/fixtures/global-fixtures';
import { DeterminedPage } from 'e2e/models/common/base/BasePage';
import { apiUrl } from 'e2e/utils/envVars';

export class DevFixture {
  setServerAddress = async (page: Page): Promise<void> => {
    // Tells the frontend where to find the backend if built for a different url.
    // Incidentally reloads and logs out of Determined.
    await page.goto('/');
    await page.evaluate(`dev.setServerAddress("${apiUrl()}")`);
    await page.reload();
    // dev.setServerAddress fires a logout request in the background, so we will wait until no network traffic is happening.
    await page.waitForLoadState('networkidle');
  };

  /**
   * Attempts to locate each element in the locator tree. If there is an error at this step,
   * the last locator in the error message is the locator that couldn't be found and needs
   * to be debugged. If there is no error message, the component could be located and this
   * debug line can be removed.
   * @param {BaseComponent} component - The component to debug
   */
  debugComponentVisible(component: BaseComponent): void {
    const componentTree: ComponentContainer[] = [];
    let root: ComponentContainer = component;
    while (!(root instanceof DeterminedPage)) {
      componentTree.unshift(root);
      root = (root as ComponentBasics)._parent;
    }
    componentTree.forEach(async (node) => {
      await expect(node.pwLocator).toBeVisible();
    });
  }
}
