import { BaseComponent } from 'e2e/models/BaseComponent';

/**
 * Returns a representation of the Mondal component from Hew.
 * This constructor represents the contents in hew/src/kit/Dropdown.tsx.
 * @param {object} obj
 * @param {CanBeParent} obj.parent - The parent used to locate this Dropdown
 * @param {string} obj.selector - Used instead of `defaultSelector`
 */
export class Dropdown extends BaseComponent {
  /**
   * Returns a templated selector for children components.
   * @param {string} id - menu item id
   */
  static selectorTemplate(id: string): string {
    return `li.ant-dropdown-menu-item[data-menu-id$="${id}"]`;
  }
  readonly _menu = new BaseComponent({
    parent: this.root,
    selector: '.ant-dropdown ul.ant-dropdown-menu:visible',
  });
}
