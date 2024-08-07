import { BaseComponent } from 'playwright-page-model-base/BaseComponent';

import { Modal } from 'e2e/models/common/hew/Modal';

/**
 * Represents the WorkspaceDeleteModal component in src/components/WorkspaceDeleteModal.tsx
 */
export class WorkspaceDeleteModal extends Modal {
  readonly nameConfirmation = new BaseComponent({
    parent: this.body,
    selector: 'input[id="workspaceName"]',
  });
}
