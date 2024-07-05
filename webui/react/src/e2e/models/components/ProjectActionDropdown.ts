import { DropdownMenu } from 'e2e/models/common/hew/Dropdown';

/**
 * Represents the ProjectActionDropdown component in src/components/ProjectActionDropdown.tsx
 */
export class ProjectActionDropdown extends DropdownMenu {
  readonly edit = this.menuItem('edit');
  readonly move = this.menuItem('move');
  readonly archive = this.menuItem('switchArchive');
  readonly delete = this.menuItem('delete');
}
