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
 * @param {BasePage} root - root of the page
 */
export class Dropdown extends BaseComponent {
  readonly childNode: ComponentBasics | undefined;
  constructor({ root, childNode, openMethod }: DropdownArgs) {
    super({
      parent: root,
      selector: '.ant-dropdown ul.ant-dropdown-menu:visible',
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

  /**
   * Returns a representation of a dropdown menu item. Since order is not
   * guaranteed, make sure to verify the contents of the menu item.
   * @param {number} n - the number of the menu item
   */
  nthMenuItem(n: number): BaseComponent {
    return new BaseComponent({
      parent: this,
      selector: `li.ant-dropdown-menu-item:nth-of-type(${n})`,
    });
  }
}
