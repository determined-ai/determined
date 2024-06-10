import { DropdownMenu } from 'e2e/models/hew/Dropdown';

/**
 * Represents the WorkspaceActionDropdown component in src/components/WorkspaceActionDropdown.tsx
 */
export class WorkspaceActionDropdown extends DropdownMenu {
  readonly pin = this.menuItem('switchPin');
  readonly edit = this.menuItem('edit');
  readonly archive = this.menuItem('switchArchive');
  readonly delete = this.menuItem('delete');
}
