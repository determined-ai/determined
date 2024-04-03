import { BaseComponent, NamedComponent } from 'e2e/models/BaseComponent';

/**
 * Returns a representation of the Modal component from Hew.
 * This constructor represents the contents in hew/src/kit/Modal.tsx.
 * @param {object} obj
 * @param {implementsGetLocator} obj.parent - The parent used to locate this Modal
 * @param {string} obj.selector - Used instead of `defaultSelector`
 */
export class Modal extends NamedComponent {
  readonly defaultSelector = '.ant-modal-content';
  readonly body: BaseComponent = new BaseComponent({ parent: this, selector: '.ant-modal-body' });
  readonly footer: ModalFooter = new ModalFooter({ parent: this, selector: '.ant-modal-footer' });
}

/**
 * Returns a representation of the Modal's Footer component from Hew.
 * This constructor represents the footer in hew/src/kit/Modal.tsx.
 * @param {object} obj
 * @param {implementsGetLocator} obj.parent - The parent used to locate this Modal
 * @param {string} obj.selector - Used instead of `defaultSelector`
 */
class ModalFooter extends BaseComponent {
  readonly submit: BaseComponent = new BaseComponent({ parent: this, selector: '[type="submit"]' });
  readonly cancel: BaseComponent = new BaseComponent({ parent: this, selector: '[type="cancel"]' });
}
