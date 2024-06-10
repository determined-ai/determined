import { BaseComponent, ComponentBasics } from 'e2e/models/BaseComponent';
import { BasePage } from 'e2e/models/BasePage';

interface requiresRoot {
  root: BasePage;
}

interface PopoverArgsWithoutChildNode extends requiresRoot {
  childNode?: never;
  openMethod: () => Promise<void>;
}

interface PopoverArgsWithChildNode extends requiresRoot {
  childNode: ComponentBasics;
  openMethod?: () => Promise<void>;
}

type PopoverArgs = PopoverArgsWithoutChildNode | PopoverArgsWithChildNode;

/**
 * Represents the Popver component from antd/es/popover/index.js
 */
export class Popover extends BaseComponent {
  readonly openMethod: () => Promise<void>;
  readonly childNode: ComponentBasics | undefined;
  constructor({ root, childNode, openMethod }: PopoverArgs) {
    super({
      parent: root,
      selector: '.ant-popover .ant-popover-content .ant-popover-inner-content:visible',
    });
    if (childNode !== undefined) {
      this.childNode = childNode;
    }
    this.openMethod =
      openMethod ||
      (async () => {
        if (this.childNode === undefined) {
          // We should never be able to throw this error. In the constructor, we
          // either provide a childNode or replace this method.
          throw new Error('This popover does not have a child node to click on.');
        }
        await this.childNode.pwLocator.click();
      });
  }

  /**
   * Opens the popover.
   * @returns {Promise<this>} - the popover for further actions
   */
  async open(): Promise<this> {
    await this.openMethod();
    return this;
  }

  /**
   * Closes the popover.
   */
  async close(): Promise<void> {
    // [ET-284] Popover click handle doesn't work unless we wait
    await this.root._page.waitForTimeout(500);
    // Popover has no close button and doesn't respect Escape key
    await this.root.nav.sidebar.header.pwLocator.click();
    await this.pwLocator.waitFor({ state: 'hidden' });
  }
}
