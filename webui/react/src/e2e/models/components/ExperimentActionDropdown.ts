import { DropdownMenu } from 'e2e/models/common/hew/Dropdown';

import ExperimentEditModal from './ExperimentEditModal';

/**
 * Represents the ExperimentActionDropdown component in src/components/ExperimentActionDropdown.tsx
 */

export class ExperimentActionDropdown extends DropdownMenu {
  readonly edit = this.menuItem('Edit');

  readonly editModal = new ExperimentEditModal({
    root: this.root,
  });
}
