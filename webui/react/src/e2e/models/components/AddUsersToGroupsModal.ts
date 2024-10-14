import { BaseComponent } from 'playwright-page-model-base/BaseComponent';

import { Modal } from 'e2e/models/common/hew/Modal';

/**
 * Represents the AddUsersToGroupsModal component in src/components/AddUsersToGroupsModal.tsx
 */
export class AddUsersToGroupsModal extends Modal {
  readonly groups = new BaseComponent({
    parent: this,
    selector: '[data-testid="groups"]',
  });
}
