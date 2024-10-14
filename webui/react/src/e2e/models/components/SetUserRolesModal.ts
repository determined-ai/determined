import { BaseComponent } from 'playwright-page-model-base/BaseComponent';

import { Modal } from 'e2e/models/common/hew/Modal';

/**
 * Represents the SetUserRolesModal component in src/components/SetUserRolesModal.tsx
 */
export class SetUserRolesModal extends Modal {
  readonly roles = new BaseComponent({
    parent: this,
    selector: '[data-testid="roles"]',
  });
}
