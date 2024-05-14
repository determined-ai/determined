import { BaseComponent, NamedComponent } from 'e2e/models/BaseComponent';

/**
 * Returns a representation of the Select component from Ant.
 * This constructor represents the contents in antd/es/select/index.d.ts.
 * @param {object} obj
 * @param {implementsGetLocator} obj.parent - The parent used to locate this Select
 * @param {string} obj.selector - Used instead of `defaultSelector`
 */
export class Select extends BaseComponent {
  readonly _menu = new BaseComponent({
    parent: this.root,
    selector: '.ant-select-dropdown .rc-virtual-list-holder-inner',
  });

  readonly search = new BaseComponent({
    parent: this,
    selector: '.ant-select-selection-search input',
  });

  readonly selectionItem = new BaseComponent({
    parent: this,
    selector: '.ant-select-selection-item',
  });

  readonly #selectionOverflow = new BaseComponent({
    parent: this,
    selector: '.ant-select-selection-overflow',
  });

  readonly selectedOverflowItems = new selectionOverflowItem({
    parent: this.#selectionOverflow,
  });

  /**
   * Returns a representation of a select item with the specified title.
   * @param {string} title - the title of the select item
   */
  selectedMenuOverflowItem(title: string): selectionOverflowItem {
    return new selectionOverflowItem({
      attachment: `[title="${title}"]`,
      parent: this.#selectionOverflow,
    });
  }

  /**
   * Shortcut to get all selected items.
   */
  async getSelectedMenuOverflowItemTitles(): Promise<string[]> {
    return await this.selectedOverflowItems.pwLocator.allTextContents();
  }

  /**
   * Shortcut open the menu for the select element.
   */
  async openMenu(): Promise<Select> {
    if (await this._menu.pwLocator.isVisible()) {
      return this;
    }
    await this.pwLocator.click();
    await this._menu.pwLocator.waitFor();
    return this;
  }

  /**
   * Returns a representation of a select dropdown menu item with the specified title.
   * @param {string} title - the title of the menu item
   */
  menuItem(title: string): BaseComponent {
    return new BaseComponent({
      parent: this._menu,
      selector: `div.ant-select-item[title$="${title}"]`,
    });
  }

  /**
   * Returns a representation of a select dropdown menu item. Since order is not
   * guaranteed, make sure to verify the contents of the menu item.
   * @param {number} n - the number of the menu item
   */
  nthMenuItem(n: number): BaseComponent {
    return new BaseComponent({
      parent: this._menu,
      selector: `div.ant-select-item:nth-of-type(${n})`,
    });
  }

  /**
   * Selects a menu item with the specified title.
   * @param {string} title - title of the item to select
   */
  async selectMenuOption(title: string): Promise<void> {
    await this.openMenu();
    await this.menuItem(title).pwLocator.click();
  }
}

/**
 * Returns a representation of the Selection Overflow Item component from Ant.
 * This constructor represents the contents in antd/es/select/index.d.ts.
 * @param {object} obj
 * @param {implementsGetLocator} obj.parent - The parent used to locate this selectionOverflowItem
 * @param {string} obj.selector - Used instead of `defaultSelector`
 */
class selectionOverflowItem extends NamedComponent {
  defaultSelector = '.ant-select-selection-overflow-item ant-select-selection-item';

  readonly removeButton = new BaseComponent({
    parent: this,
    selector: '[aria-label="close"]',
  });
}
