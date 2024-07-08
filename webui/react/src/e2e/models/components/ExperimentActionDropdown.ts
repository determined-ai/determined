import { BaseComponent } from 'e2e/models/common/base/BaseComponent';
import { DropdownMenu } from 'e2e/models/common/hew/Dropdown';
import { Modal } from 'e2e/models/common/hew/Modal';

/**
 * Represents the ExperimentActionDropdown component in src/components/ExperimentActionDropdown.tsx
 */

class ExperimentEditModal extends Modal {
  readonly nameInput = new BaseComponent({
    parent: this,
    selector: 'input[id="experimentName"]',
  });
}

export class ExperimentActionDropdown extends DropdownMenu {
  readonly edit = this.menuItem('Edit');

  readonly editModal = new ExperimentEditModal({
    root: this.root,
  });
}
