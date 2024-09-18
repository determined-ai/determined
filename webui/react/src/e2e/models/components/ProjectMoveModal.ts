import { Modal } from 'e2e/models/common/hew/Modal';
import { Select } from 'e2e/models/common/hew/Select';

/**
 * Represents the ProjectMoveModal component in src/components/ProjectMoveModal.tsx
 */
export class ProjectMoveModal extends Modal {
  readonly destinationWorkspace = new Select({
    parent: this,
    selector: 'input[id="workspace"]',
  });
}
