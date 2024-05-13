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

  readonly #selectionOverflow = new BaseComponent({
    parent: this,
    selector: '.ant-select-selection-overflow',
  });

  readonly selectedMenuOptions = new selectionItem({
    parent: this.#selectionOverflow,
  });

  /**
   * Returns a representation of a select item with the specified title.
   * @param {string} title - the title of the select item
   */
  selectedMenuOption(title: string): selectionItem {
    return new selectionItem({
      attachment: `[title="${title}"]`,
      parent: this.#selectionOverflow,
    });
  }

  async getSelectedMenuOptionTitles(): Promise<string[]> {
    return await this.selectedMenuOptions.pwLocator.allTextContents();
  }

  /**
   * Returns a representation of a select dropdown menu item with the specified title.
   * @param {string} title - the title of the menu item
   */
  menuItem(title: string): BaseComponent {
    return new BaseComponent({
      parent: this,
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
      parent: this,
      selector: `div.ant-select-item:nth-of-type(${n})`,
    });
  }
}

class selectionItem extends NamedComponent {
  defaultSelector = '.ant-select-selection-overflow-item ant-select-selection-item';

  readonly removeButton = new BaseComponent({
    parent: this,
    selector: '[aria-label="close"]',
  });
}
