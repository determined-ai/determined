import { BaseComponent } from 'e2e/models/BaseComponent';
import { Dropdown } from 'e2e/models/hew/Dropdown';

/**
 * Returns a representation of the Action Menu Dropdown component.
 * @param {object} obj
 * @param {implementsGetLocator} obj.parent - The parent used to locate this dropdown. Normally dropdowns need to be the root.
 * @param {string} [obj.selector] - Used instead of `defaultSelector`
 */
export class ProjectActionDropdown extends Dropdown {
  readonly edit: BaseComponent = new BaseComponent({
    parent: this._menu,
    selector: Dropdown.selectorTemplate('edit'),
  });
  readonly move: BaseComponent = new BaseComponent({
    parent: this._menu,
    selector: Dropdown.selectorTemplate('move'),
  });
  readonly archive: BaseComponent = new BaseComponent({
    parent: this._menu,
    selector: Dropdown.selectorTemplate('switchArchive'),
  });
  readonly delete: BaseComponent = new BaseComponent({
    parent: this._menu,
    selector: Dropdown.selectorTemplate('delete'),
  });
}
