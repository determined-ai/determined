import { BaseComponent } from 'e2e/models/BaseComponent';
import { Modal } from 'e2e/models/hew/Modal';

/**
 * Returns a representation of the ChangeUserStatusModal component.
 * This constructor represents the contents in src/components/ChangeUserStatusModal.tsx.
 * @param {object} obj
 * @param {implementsGetLocator} obj.parent - The parent used to locate this ChangeUserStatusModal
 * @param {string} [obj.selector] - Used instead of `defaultSelector`
 */
export class ChangeUserStatusModal extends Modal {
  readonly status: BaseComponent = new BaseComponent({ parent: this.body, selector: "[data-testid='status']" });
}
