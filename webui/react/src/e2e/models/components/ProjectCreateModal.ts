import { BaseComponent } from 'playwright-page-model-base/BaseComponent';

import { Modal } from 'e2e/models/common/hew/Modal';

/**
 * Represents the ProjectCreateModal component in src/components/ProjectCreateModal.tsx
 */
export class ProjectCreateModal extends Modal {
  readonly projectName = new BaseComponent({
    parent: this,
    selector: 'input[id="projectName"]',
  });

  readonly description = new BaseComponent({
    parent: this,
    selector: 'input[id="description"]',
  });
}
