import {
  BaseComponent,
  ComponentBasics,
  ComponentContainer,
} from 'playwright-page-model-base/BaseComponent';

import { DeterminedPage } from 'e2e/models/common/base/BasePage';

export function printMap<T1, T2>(map: Map<T1, T2>): string {
  return Array.from(map.entries())
    .map(([key, value]) => `${key}: ${value}`)
    .join('\n');
}

/**
 * Attempts to locate each element in the locator tree. If there is an error at this step,
 * the last locator in the error message is the locator that couldn't be found and needs
 * to be debugged. If there is no error message, the component could be located and this
 * debug line can be removed.
 * @param {BaseComponent} component - The component to debug
 */
export async function debugComponentVisible(component: BaseComponent): Promise<void> {
  const componentTree: ComponentContainer[] = [];
  let root: ComponentContainer = component;
  while (!(root instanceof DeterminedPage)) {
    componentTree.unshift(root);
    root = (root as ComponentBasics)._parent;
  }
  for (const node of componentTree) {
    await node.pwLocator.waitFor();
  }
}
