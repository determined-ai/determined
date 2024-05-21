import { DropdownMenu } from 'e2e/models/hew/Dropdown';

/**
 * Returns a representation of the Action Menu Dropdown component.
 * @param {object} obj
 * @param {BasePage} obj.root - root of the page
 * @param {ComponentBasics} [obj.childNode] - optional if `openMethod` is present. It's the element we click on to open the dropdown.
 * @param {Function} [obj.openMethod] - optional if `childNode` is present. It's the method to open the dropdown.
 */
export class ProjectActionDropdown extends DropdownMenu {
  readonly edit = this.menuItem('edit'); 
  readonly move = this.menuItem('move'); 
  readonly archive = this.menuItem('switchArchive'); 
  readonly delete = this.menuItem('delete'); 
}
