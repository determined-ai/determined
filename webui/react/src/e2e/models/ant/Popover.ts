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
 *
 * The popover can be opened by calling the open method. By default, the open method clicks on the child node. Sometimes you might even need to provide both optional arguments, like when a child node is present but impossible to click on due to canvas behavior.
 * @param {object} obj
 * @param {BasePage} obj.root - root of the page
 * @param {ComponentBasics} [obj.childNode] - optional if `openMethod` is present. It's the element we click on to open the dropdown.
 * @param {Function} [obj.openMethod] - optional if `childNode` is present. It's the method to open the dropdown.
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
}
