import { expect } from 'e2e/fixtures/global-fixtures';
import { BaseComponent, CanBeParent } from 'e2e/models/common/base/BaseComponent';
import { BasePage } from 'e2e/models/common/base/BasePage';

export class DevFixture {
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
