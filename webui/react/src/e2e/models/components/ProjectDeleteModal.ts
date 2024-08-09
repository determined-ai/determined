import { BaseComponent } from 'playwright-page-model-base/BaseComponent';

import { Modal } from 'e2e/models/common/hew/Modal';

/**
 * Represents the ProjectDeleteModal component in src/components/ProjectDeleteModal.tsx
 */
export class ProjectDeleteModal extends Modal {
  readonly nameConfirmation = new BaseComponent({
    parent: this,
    selector: 'input[id="projectName"]',
  });
}
