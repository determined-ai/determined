import { BaseComponent } from 'playwright-page-model-base/BaseComponent';
import { BaseOverlay, OverlayArgs } from 'playwright-page-model-base/BaseOverlay';

/**
 * Represents the Dropdown component from antd/es/dropdown/index.js
 * Until the dropdown component supports test ids, this model will match any open dropdown.
 */
export class Dropdown extends BaseOverlay {
  constructor(args: OverlayArgs) {
    super({
      ...args,
      selector: '.ant-dropdown ul.ant-dropdown-menu:visible',
    });
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

  /**
   * Closes the dropdown.
   */
  async close(): Promise<void> {
    await this.pwLocator.press('Escape', { timeout: 500 });
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
