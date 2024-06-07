import { BaseComponent } from 'e2e/models/BaseComponent';
import { Modal } from 'e2e/models/hew/Modal';

/**
 * Represents the ProjectDeleteModal component in src/components/ProjectDeleteModal.tsx
 */
export class ProjectDeleteModal extends Modal {
  readonly nameConfirmation: BaseComponent = new BaseComponent({
    parent: this,
    selector: 'input[id="projectName"]',
  });
}
