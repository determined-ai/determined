import { BaseComponent } from 'e2e/models/BaseComponent';
import { Dropdown } from 'e2e/models/hew/Dropdown';

/**
 * Returns a representation of the Mondal component from Hew.
 * This constructor represents the contents in hew/src/kit/Select.tsx.
 * @param {object} obj
 * @param {implementsGetLocator} obj.parent - The parent used to locate this Select
 * @param {string} obj.selector - Used instead of `defaultSelector`
 */
export class Select extends Dropdown {
  /**
   * Returns a templated selector for children components.
   * @param {string} title - menu item id
   */
  static override selectorTemplate(title: string): string {
    return `div.ant-select-item[title$="${title}"]`;
  }

  override readonly _menu: BaseComponent = new BaseComponent({
    parent: this.root,
    selector: '.ant-select-dropdown .rc-virtual-list-holder-inner',
  });
  readonly selectedOption: BaseComponent = new BaseComponent({
    parent: this._menu,
    selector: 'div[role="option"][aria-selected="true"]',
  });
}
