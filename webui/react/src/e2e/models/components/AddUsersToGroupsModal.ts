import { BaseComponent } from 'e2e/models/BaseComponent';
import { Modal } from 'e2e/models/hew/Modal';

/**
 * Represents the AddUsersToGroupsModal component in src/components/AddUsersToGroupsModal.tsx
 */
export class AddUsersToGroupsModal extends Modal {
  readonly groups = new BaseComponent({
    parent: this.body,
    selector: '[data-testid="groups"]',
  });
}
