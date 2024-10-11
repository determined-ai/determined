import { BaseComponent } from 'playwright-page-model-base/BaseComponent';

import { Modal } from 'e2e/models/common/hew/Modal';

/**
 * Represents the CreateUserModal component in src/components/CreateUserModal.tsx
 */
export class CreateUserModal extends Modal {
  readonly username = new BaseComponent({
    parent: this,
    selector: '[data-testid="username"]',
  });
  readonly displayName = new BaseComponent({
    parent: this,
    selector: '[data-testid="displayName"]',
  });
  readonly adminToggle = new BaseComponent({
    parent: this,
    selector: '[data-testid="isAdmin"] button',
  });
  readonly remoteToggle = new BaseComponent({
    parent: this,
    selector: '[data-testid="isRemote"] button',
  });
  readonly password = new BaseComponent({
    parent: this,
    selector: '[data-testid="password"]',
  });
  readonly confirmPassword = new BaseComponent({
    parent: this,
    selector: '[data-testid="confirmPassword"]',
  });
  readonly roleSelect = new BaseComponent({
    parent: this,
    selector: '[data-testid="roles"]',
  });
}
