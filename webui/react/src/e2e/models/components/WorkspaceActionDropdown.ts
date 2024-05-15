import { DropdownMenu } from 'e2e/models/hew/Dropdown';

/**
 * Returns a representation of the Action Menu Dropdown component.
 * @param {object} obj
 * @param {implementsGetLocator} obj.parent - The parent used to locate this dropdown. Normally dropdowns need to be the root.
 * @param {string} [obj.selector] - Used instead of `defaultSelector`
 */
export class WorkspaceActionDropdown extends DropdownMenu {
  readonly pin = this.menuItem('switchPin');
  readonly edit = this.menuItem('edit');
  readonly archive = this.menuItem('switchArchive');
  readonly delete = this.menuItem('delete');
}
