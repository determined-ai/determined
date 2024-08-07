import { BaseComponent } from 'playwright-page-model-base/BaseComponent';

import { Modal } from 'e2e/models/common/hew/Modal';

/**
 * Represents the ExperimentEditModal component in src/components/ExperimentEditModal.tsx
 */
export default class ExperimentEditModal extends Modal {
  readonly nameInput = new BaseComponent({
    parent: this,
    selector: 'input[id="experimentName"]',
  });
}
