import { BaseComponent } from 'e2e/models/BaseComponent';
import { Modal } from 'e2e/models/hew/Modal';

/**
 * Returns a representation of the CreateUserModal component.
 * This constructor represents the contents in src/components/CreateUserModal.tsx.
 * @param {object} obj
 * @param {implementsGetLocator} obj.parent - The parent used to locate this CreateUserModal
 * @param {string} [obj.selector] - Used instead of `defaultSelector`
 */
export class SetUserRolesModal extends Modal {
  readonly roles: BaseComponent = new BaseComponent({ parent: this.body, selector: "[data-testid='roles']" });
}
