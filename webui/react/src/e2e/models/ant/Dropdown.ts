import { BaseComponent, ComponentBasics } from 'e2e/models/BaseComponent';
import { BasePage } from 'e2e/models/BasePage';

interface RequiresRoot {
  root: BasePage;
}

interface DropdownArgsWithoutChildNode extends RequiresRoot {
  childNode?: never;
  openMethod: () => Promise<void>;
}

interface DropdownArgsWithChildNode extends RequiresRoot {
  childNode: ComponentBasics;
  openMethod?: () => Promise<void>;
}

type DropdownArgs = DropdownArgsWithoutChildNode | DropdownArgsWithChildNode;

/**
 * Represents the Dropdown component from antd/es/dropdown/index.js
 * Until the dropdown component supports test ids, this model will match any open dropdown.
 */
export class Dropdown extends BaseComponent {
  readonly openMethod: (args: { timeout?: number }) => Promise<void>;
  readonly childNode: ComponentBasics | undefined;

  /**
   * Constructs a new Dropdown component.
   * The dropdown can be opened by calling the open method. By default, the open
   * method clicks on the child node. Sometimes you might even need to provide
   * both optional arguments, like when a child node is present but impossible to
   * click on due to being blocked by another element behavior.
   * @param {object} obj
   * @param {BasePage} obj.root - root of the page
   * @param {ComponentBasics} [obj.childNode] - optional if `openMethod` is present. It's the element we click on to open the dropdown.
   * @param {Function} [obj.openMethod] - optional if `childNode` is present. It's the method to open the dropdown.
   */
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
      (async (args = {}) => {
        if (this.childNode === undefined) {
          // We should never be able to throw this error. In the constructor, we
          // either provide a childNode or replace this method.
          throw new Error('This dropdown does not have a child node to click on.');
        }
        await this.childNode.pwLocator.click(args); // refreshing with 10s intervals, this should give us enough time for everything to be stable even if a refresh occurs in the first 5 seconds.
      });
  }

  /**
   * Opens the dropdown.
   * @returns {Promise<this>} - the dropdown for further actions
   */
  async open(args: { timeout?: number } = { timeout: 15_000 }): Promise<this> {
    await this.openMethod(args);
    return this;
  }

  /**
   * Returns a MenuItem with the specified id
   * @param {string} id - the id of the menu item
   */
  menuItem(id: string): MenuItem {
    return new MenuItem({
      parent: this,
      selector: `li.ant-dropdown-menu-item[data-menu-id$="${id}"]`,
    });
  }

  /**
   * Selects a MenuItem with the specified id
   * @param {string} id - id of the item to select
   */
  async selectMenuOption(id: string): Promise<void> {
    await this.open();
    await this.menuItem(id).pwLocator.click();
    await this.pwLocator.waitFor({ state: 'hidden' });
  }
}

/**
 * Represents a menu item from the Dropdown component
 */
class MenuItem extends BaseComponent {
  override readonly _parent: Dropdown;
  constructor({ parent, selector }: { parent: Dropdown; selector: string }) {
    super({ parent, selector });
    this._parent = parent;
  }

  /**
   * Selects the menu item
   * @param {object} clickArgs - arguments to pass to the click method
   */
  async select(clickArgs: object = {}): Promise<void> {
    await this._parent.open();
    await this.pwLocator.click(clickArgs);
    await this._parent.pwLocator.waitFor({ state: 'hidden' });
  }
}
