import { BaseComponent } from 'e2e/models/BaseComponent';

/**
 * Returns a representation of the Mondal component from Hew.
 * This constructor represents the contents in hew/src/kit/Dropdown.tsx.
 * @param {object} obj
 * @param {implementsGetLocator} obj.parent - The parent used to locate this Dropdown
 * @param {string} obj.selector - Used instead of `defaultSelector`
 */
export class Dropdown extends BaseComponent {
  static selectorTemplate(id: string): string {
    return `li.ant-dropdown-menu-item[data-menu-id$="${id}"]`;
  }
  readonly headerDropdownMenu: BaseComponent = new BaseComponent({
    parent: this.root,
    selector: 'ul.ant-dropdown-menu',
  });
}
