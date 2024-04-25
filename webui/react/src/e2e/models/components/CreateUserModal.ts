import { BaseComponent } from 'e2e/models/BaseComponent';
import { Modal } from 'e2e/models/hew/Modal';

/**
 * Returns a representation of the CreateUserModal component.
 * This constructor represents the contents in src/components/CreateUserModal.tsx.
 * @param {object} obj
 * @param {CanBeParent} obj.parent - The parent used to locate this CreateUserModal
 * @param {string} [obj.selector] - Used instead of `defaultSelector`
 */
export class CreateUserModal extends Modal {
  readonly username: BaseComponent = new BaseComponent({
    parent: this.body,
    selector: '[data-testid="username"]',
  });
  readonly displayName: BaseComponent = new BaseComponent({
    parent: this.body,
    selector: '[data-testid="displayName"]',
  });
  readonly adminToggle: BaseComponent = new BaseComponent({
    parent: this.body,
    selector: '[data-testid="isAdmin"] button',
  });
  readonly remoteToggle: BaseComponent = new BaseComponent({
    parent: this.body,
    selector: '[data-testid="isRemote"] button',
  });
  readonly password: BaseComponent = new BaseComponent({
    parent: this.body,
    selector: '[data-testid="password"]',
  });
  readonly roleSelect: BaseComponent = new BaseComponent({
    parent: this.body,
    selector: '[data-testid="roles"]',
  });
}
