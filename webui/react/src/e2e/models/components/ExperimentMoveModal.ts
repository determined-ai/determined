import { Modal } from 'e2e/models/common/hew/Modal';
import { Select } from 'e2e/models/common/hew/Select';

/**
 * Represents the ExperimentMoveModal component in src/components/ExperimentMoveModal.tsx
 */
export default class ExperimentMoveModal extends Modal {
  readonly destinationWorkspace = new Select({
    parent: this,
    selector: 'input[id="workspace"]',
  });
  readonly destinationProject = new Select({
    parent: this,
    selector: 'input[id="projectId"]',
  });
}
