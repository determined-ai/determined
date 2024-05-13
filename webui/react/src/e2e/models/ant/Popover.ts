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
 * Returns a representation of the Popover component from Ant.
 * Until the popover component supports test ids, this model will match any open popover.
 * This constructor represents the contents in antd/es/popover/index.d.ts.
 * @param {BasePage} root - root of the page
 */
export class Popover extends BaseComponent {
  readonly childNode: ComponentBasics | undefined;
  constructor({ root, childNode, openMethod }: PopoverArgs) {
    super({
      parent: root,
      selector: '.ant-popover .ant-popover-content .ant-popover-inner-content:visible',
    });
    if (childNode !== undefined) {
      this.childNode = childNode;
    }
    if (openMethod !== undefined) {
      this.open = openMethod;
    }
  }

  async open(): Promise<void> {
    if (this.childNode === undefined) {
      // We should never be able to throw this error. In the constructor, we
      // either provide a childNode or replace this method.
      throw new Error('This dropdown does not have a child node to click on.');
    }
    await this.childNode.pwLocator.click();
  }
}
