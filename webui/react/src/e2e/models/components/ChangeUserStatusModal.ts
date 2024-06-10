import { Modal } from 'e2e/models/hew/Modal';
import { Select } from 'e2e/models/hew/Select';

/**
 * Represents the ChangeUserStatusModal component in src/components/ChangeUserStatusModal.tsx
 */
export class ChangeUserStatusModal extends Modal {
  readonly status = new StatusSelect({
    parent: this.body,
    selector: '[data-testid="status"]',
  });
}

/**
 * Represents the status Select from the ChangeUserStatusModal component
 */
class StatusSelect extends Select {
  readonly activate = this.menuItem('Activate');
  readonly deactivate = this.menuItem('Deactivate');
}
