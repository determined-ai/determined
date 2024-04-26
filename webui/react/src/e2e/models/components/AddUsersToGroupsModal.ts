import { BaseComponent } from 'e2e/models/BaseComponent';
import { Modal } from 'e2e/models/hew/Modal';

/**
 * Returns a representation of the AddUsersToGroupsModal component.
 * This constructor represents the contents in src/components/AddUsersToGroupsModal.tsx.
 * @param {object} obj
 * @param {CanBeParent} obj.parent - The parent used to locate this AddUsersToGroupsModal
 * @param {string} [obj.selector] - Used instead of `defaultSelector`
 */
export class AddUsersToGroupsModal extends Modal {
  readonly groups = new BaseComponent({
    parent: this.body,
    selector: '[data-testid="groups"]',
  });
}
