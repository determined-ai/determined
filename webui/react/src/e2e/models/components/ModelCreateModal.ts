import { BaseComponent } from 'playwright-page-model-base/BaseComponent';

import { Modal } from 'e2e/models/common/hew/Modal';

/**
 * Represents the ModelCreateModal component in src/components/ModelCreateModal.tsx
 */
export class ModelCreateModal extends Modal {
  readonly modelName = new BaseComponent({
    parent: this.body,
    selector: '[id="modelName"]',
  });
}
