import { BaseComponent, ComponentBasics } from 'e2e/models/BaseComponent';
import { BasePage } from 'e2e/models/BasePage';

interface requiresRoot {
  root: BasePage;
}

interface DropdownArgsWithoutChildNode extends requiresRoot {
  childNode?: never;
  openMethod: () => Promise<void>;
}

interface DropdownArgsWithChildNode extends requiresRoot {
  childNode: ComponentBasics;
  openMethod?: () => Promise<void>;
}

type DropdownArgs = DropdownArgsWithoutChildNode | DropdownArgsWithChildNode;

/**
 * Returns a representation of the Dropdown component from Ant.
 * Until the dropdown component supports test ids, this model will match any open dropdown.
 * This constructor represents the contents in antd/es/dropdown/index.d.ts.
 *
 * The dropdown can be opened by calling the open method. By default, the open method clicks on the child node. Sometimes you might even need to provide both optional arguments, like when a child node is present but impossible to click on due to canvas behavior.
 * @param {object} obj
 * @param {BasePage} obj.root - root of the page
 * @param {ComponentBasics} [obj.childNode] - optional if `openMethod` is present. It's the element we click on to open the dropdown.
 * @param {Function} [obj.openMethod] - optional if `childNode` is present. It's the method to open the dropdown.
 */
export class Dropdown extends BaseComponent {
  readonly openMethod: () => Promise<void>;
  readonly childNode: ComponentBasics | undefined;
  constructor({ root, childNode, openMethod }: DropdownArgs) {
    super({
      parent: root,
      selector: '.ant-dropdown ul.ant-dropdown-menu:visible',
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
          throw new Error('This dropdown does not have a child node to click on.');
        }
        await this.childNode.pwLocator.click();
      });
  }

  /**
   * Opens the dropdown.
   * @returns {Promise<this>} - the dropdown for further actions
   */
  async open(): Promise<this> {
    await this.openMethod();
    return this;
  }

  /**
   * Returns a representation of a dropdown menu item with the specified id.
   * @param {string} id - the id of the menu item
   */
  menuItem(id: string): BaseComponent {
    return new BaseComponent({
      parent: this,
      selector: `li.ant-dropdown-menu-item[data-menu-id$="${id}"]`,
    });
  }
}
