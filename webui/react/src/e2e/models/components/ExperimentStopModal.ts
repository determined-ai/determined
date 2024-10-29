import { Modal } from 'e2e/models/common/hew/Modal';
import { Select } from 'e2e/models/common/hew/Select';

/**
 * Represents the ExperimentStopModal component in src/components/ExperimentStopModal.tsx
 */
export default class ExperimentStopModal extends Modal {
  readonly targetExperiment = new Select({
    parent: this,
    selector: '[data-test="experiment"]',
  });
}
