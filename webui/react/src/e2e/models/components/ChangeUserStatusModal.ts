import { BaseComponent } from 'e2e/models/BaseComponent';
import { Modal } from 'e2e/models/hew/Modal';
import { Select } from 'e2e/models/hew/Select';

/**
 * Returns a representation of the ChangeUserStatusModal component.
 * This constructor represents the contents in src/components/ChangeUserStatusModal.tsx.
 * @param {object} obj
 * @param {CanBeParent} obj.parent - The parent used to locate this ChangeUserStatusModal
 * @param {string} [obj.selector] - Used instead of `defaultSelector`
 */
export class ChangeUserStatusModal extends Modal {
  readonly status: StatusSelect = new StatusSelect({
    parent: this.body,
    selector: '[data-testid="status"]',
  });
}

class StatusSelect extends Select {
  readonly activate: BaseComponent = new BaseComponent({
    parent: this._menu,
    selector: Select.selectorTemplate('Activate'),
  });
  readonly deactivate: BaseComponent = new BaseComponent({
    parent: this._menu,
    selector: Select.selectorTemplate('Deactivate'),
  });
}
