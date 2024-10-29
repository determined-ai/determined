import { BaseComponent, NamedComponent } from 'playwright-page-model-base/BaseComponent';
import { BaseOverlay } from 'playwright-page-model-base/BaseOverlay';

/**
 * Represents the Select component from antd/es/select/index.js
 */
export class Select extends BaseComponent {
  readonly search = new BaseComponent({
    parent: this,
    selector: '.ant-select-selection-search input',
  });
  readonly _menu = new SelectMenu({
    clickThisComponentToOpen: this,
    root: this.root,
    selector: ':not(.ant-select-dropdown-hidden).ant-select-dropdown .rc-virtual-list-holder-inner',
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
   * Returns a selectedMenuOverflowItem in the Select component
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
      try {
        await this._menu.pwLocator.press('Escape', { timeout: 500 });
        await this._menu.pwLocator.waitFor({ state: 'hidden' });
      } catch (e) {
        // it's fine if this fails, we are just ensuring they are all closed.
      }
    }
    await this._menu.open();
    await this.root._page.waitForTimeout(500); // ant/Select - menus may reset input shortly after opening [ET-283]
    return this;
  }

  menuItem = this._menu.menuItem.bind(this._menu);
  menuItemByIndex = this._menu.menuItemByIndex.bind(this._menu);

  /**
   * Selects a menu item with the specified title.
   * @param {string} title - title of the item to select
   */
  async selectMenuOption(title: string): Promise<void> {
    await this.openMenu();
    await this.menuItem(title).pwLocator.click();
    await this._menu.pwLocator.waitFor({ state: 'hidden' });
  }
  /**
   * Selects a menu item at the specified index.
   * @param {number} index - index of the item to select
   */
  async selectMenuOptionByIndex(index: number): Promise<void> {
    await this.openMenu();
    await (await this.menuItemByIndex(index))?.pwLocator.click();
    await this._menu.pwLocator.waitFor({ state: 'hidden' });
  }
}

/**
 * Represents a selectionOverflowItem from the Select component
 */
class selectionOverflowItem extends NamedComponent {
  defaultSelector = '.ant-select-selection-overflow-item ant-select-selection-item';

  readonly removeButton = new BaseComponent({
    parent: this,
    selector: '[aria-label="close"]',
  });
}

/**
 * Represents a menu in the Select component
 */
class SelectMenu extends BaseOverlay {
  #menuItems = new BaseComponent({
    parent: this,
    selector: 'div.ant-select-item',
  });

  get menuItems() {
    return this.#menuItems.pwLocator
      .all()
      .then(async (items) => await Promise.all(items.flatMap((item) => item.textContent())))
      .catch(() => []);
  }

  /**
   * Returns a menuItem in the Select component
   * @param {string} title - the title of the menu item
   */
  menuItem(title: string): BaseComponent {
    return new BaseComponent({
      parent: this,
      selector: `div.ant-select-item[title="${title}"]`,
    });
  }

  /**
   * Returns a menuItem in the Select component
   * @param {number} index - the index of the menu item
   */
  async menuItemByIndex(index: number): Promise<BaseComponent | undefined> {
    const itemText = (await this.menuItems).at(index);
    if (itemText === null || itemText === undefined) return undefined;
    return this.menuItem(itemText);
  }

  /**
   * Closes the menu.
   */
  async close(): Promise<void> {
    await this.pwLocator.press('Escape', { timeout: 500 });
  }
}
